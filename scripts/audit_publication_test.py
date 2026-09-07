"""Publication-guard regression tests in disposable repositories with fake data."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import audit_publication as subject


class AuditTests(unittest.TestCase):
    def test_synthetic_secret_and_private_path(self):
        fake = ('ghp_' + 'A' * 36).encode()
        self.assertIsNotNone(subject.content_reason(fake, []))
        self.assertIsNotNone(subject.path_reason('outputs/run.json'))
        self.assertIsNotNone(subject.path_reason('data/cookies.json'))
        self.assertIsNone(subject.path_reason('cookies/cookies.go'))
        self.assertIsNone(subject.content_reason(b'example-token', []))
        self.assertIsNotNone(subject.content_reason(b'example-token', [b'example-token']))

    def test_index_and_deleted_history_are_audited(self):
        audit_script = str(Path(subject.__file__).resolve())
        fake = 'ghp_' + 'A' * 36
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)

            def git(*args):
                subprocess.run(['git', *args], cwd=root, check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

            def audit():
                env = dict(os.environ)
                env.pop('XHS_AUDIT_COOKIE_FILE', None)
                return subprocess.run([sys.executable, audit_script, '--audit'], cwd=root,
                                      env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)

            def commit():
                git('-c', 'user.name=Example', '-c', 'user.email=example@example.com',
                    '-c', 'commit.gpgsign=false', 'commit', '-m', 'Synthetic fixture')

            git('init', '-b', 'main')
            fixture = root / 'fixture.txt'
            fixture.write_text(fake)
            git('add', '--', 'fixture.txt')
            fixture.write_text('safe working copy')
            staged = audit()
            self.assertNotEqual(staged.returncode, 0)
            self.assertNotIn(fake, staged.stdout + staged.stderr)
            commit()  # Deliberately commits a fake token only inside this disposable test repo.
            git('add', '--', 'fixture.txt')
            commit()
            history = audit()
            self.assertNotEqual(history.returncode, 0)
            self.assertIn('GitHub credential', history.stderr)
            self.assertNotIn(fake, history.stdout + history.stderr)


if __name__ == '__main__':
    unittest.main()
