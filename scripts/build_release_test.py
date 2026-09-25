"""Offline release metadata tests; never build, publish or access an account."""
import json
import unittest
from unittest.mock import patch

import build_release as subject


class ReleaseTests(unittest.TestCase):
    def test_tag_and_platform_assets(self):
        self.assertEqual(subject.release_version('v1.0.0'), '1.0.0')
        for tag in ('1.0.0', '../v1.0.0', 'v1.0.0 x', 'v1.0'):
            with self.assertRaises(ValueError):
                subject.release_version(tag)
        names = {subject.asset_name(p, goos, arch) for p, _ in subject.PROGRAMS
                 for goos, arch in subject.PLATFORMS}
        self.assertEqual(len(names), 9)
        self.assertIn('xiaohongshu-mcp-windows-amd64.exe', names)
        self.assertNotIn('xiaohongshu-mcp-darwin-amd64', names)

    def test_multiple_go_json_objects(self):
        data = '  ' + json.dumps({'Path': 'example.com/a'}) + '\n' + json.dumps({'Path': 'example.com/b'})
        self.assertEqual([m['Path'] for m in subject.json_objects(data)],
                         ['example.com/a', 'example.com/b'])

    @patch.dict('os.environ', {'GOAMD64': 'v4', 'GOFLAGS': '-race', 'GOEXPERIMENT': 'example'})
    def test_portable_build_environment(self):
        env = subject.build_environment('windows', 'amd64')
        self.assertEqual(env['CGO_ENABLED'], '0')
        self.assertEqual(env['GOOS'], 'windows')
        self.assertEqual(env['GOWORK'], 'off')
        self.assertEqual(env['GOFLAGS'], '')
        self.assertNotIn('GOAMD64', env)
        self.assertNotIn('GOEXPERIMENT', env)

    def test_notes_have_actual_build_fields(self):
        info = dict(version='1.0.0', tag='v1.0.0', commit='a' * 40,
                    go_version='go1.27.1', build_time='2026-01-01T00:00:00Z', browser_version='example')
        notes = subject.render_notes('## Build\n<!-- BUILD_INFO -->', info)
        self.assertIn(info['commit'], notes)
        self.assertIn(info['build_time'], notes)
        self.assertNotIn('<!-- BUILD_INFO -->', notes)
        with self.assertRaises(ValueError):
            subject.render_notes('No build marker', info)


if __name__ == '__main__':
    unittest.main()
