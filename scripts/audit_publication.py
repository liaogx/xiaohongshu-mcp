#!/usr/bin/env python3
"""Conservative publication check. Prints paths/categories, never secret values.

Audits the index, non-ignored files, and reachable or outgoing commit snapshots.
This is not a proof of safety; manually review source, examples and dependencies.
"""
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys

MAX_BYTES = 2 * 1024 * 1024
PRIVATE_PARTS = {
    'data', 'state', 'outputs', 'downloads', 'receipts', 'logs', 'media', 'images',
    'browser-profile', 'profiles', 'user-data', '.ssh', '.aws', '.gnupg', '.codex',
    '.cursor', '.vscode', 'donate',
}
PRIVATE_NAMES = {
    'cookies.json', 'cookies', 'login data', 'local state', 'web data', 'auth.json',
    'accounts.json', 'credentials', 'credentials.json', 'id_rsa', 'id_ed25519',
    'id_ecdsa', 'id_dsa', 'devtoolsactiveport', '.kimi-agent.yml',
}
REVIEW_SUFFIXES = {
    '.pem', '.key', '.p12', '.pfx', '.keystore', '.keychain', '.keychain-db',
    '.db', '.sqlite', '.sqlite3', '.log', '.err', '.zip', '.tar', '.gz', '.xz',
    '.7z', '.mp4', '.mov', '.png', '.gif', '.jpg', '.jpeg', '.webp', '.pdf',
    '.exe', '.dmg', '.heic', '.mp3', '.wav', '.svg',
}
PATTERNS = [
    ('private key', rb'-----BEGIN (?:[A-Z0-9]+ )*PRIVATE KEY-----'),
    ('GitHub credential', rb'\b(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,})\b'),
    ('cloud access key', rb'\b(?:AKIA|ASIA)[A-Z0-9]{16}\b'),
    ('API credential', rb'\bsk-(?:proj-|svcacct-|ant-)?[A-Za-z0-9_-]{32,}\b'),
    ('Slack credential', rb'\bxox[baprs]-[A-Za-z0-9-]{20,}\b'),
    ('JWT credential', rb'\beyJ[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}\b'),
    ('literal bearer credential', rb'(?i)\bBearer\s+[A-Za-z0-9._~-]{32,}'),
    ('assigned secret', rb'''(?i)(?:api[_-]?key|api[_-]?token|access[_-]?token|client[_-]?secret|password|xsec[_-]?token|web_session)["']?\s*[:=]\s*["'][A-Za-z0-9_./+~=-]{24,}["']'''),
    ('signed note URL', rb'(?i)xsec_token=[A-Za-z0-9_+/%=-]{24,}'),
    ('embedded media/session material', rb'(?i)data:(?:image|video|audio)/[a-z0-9.+-]+;base64,[a-z0-9+/=]{100,}'),
    ('private macOS home path', rb'/Users/(?!user/|username/|example/)[A-Za-z0-9._-]+/'),
]
PATTERNS = [(label, re.compile(pattern)) for label, pattern in PATTERNS]


def git(*args):
    result = subprocess.run(['git', *args], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if result.returncode:
        raise RuntimeError('Git inspection failed')
    return result.stdout


def path_reason(path):
    p = PurePosixPath(path.lower())
    if any(part in PRIVATE_PARTS for part in p.parts) or p.name in PRIVATE_NAMES:
        return 'private runtime/configuration path'
    if p.name == '.env' or (p.name.startswith('.env.') and p.name != '.env.example'):
        return 'environment credential file'
    if p.suffix in REVIEW_SUFFIXES:
        return 'media/binary/private file requires separate review'
    if 'qrcode' in p.name or ('cookie' in p.name and p.suffix == '.json'):
        return 'session/QR file'
    return None


def private_cookie_values():
    """Optional in-memory exact-match check; never writes or prints values."""
    path = os.environ.get('XHS_AUDIT_COOKIE_FILE')
    if not path:
        return []
    p = Path(path)
    if p.is_symlink() or not p.is_file() or p.stat().st_size > MAX_BYTES:
        raise RuntimeError('Unsafe private comparison input')
    obj = json.loads(p.read_text())
    records = obj if isinstance(obj, list) else obj.get('cookies', [])
    return [r['value'].encode() for r in records
            if isinstance(r, dict) and isinstance(r.get('value'), str) and len(r['value']) >= 12]


def content_reason(data, private_values):
    if len(data) > MAX_BYTES or b'\0' in data:
        return 'binary/oversized content requires separate review'
    if any(value in data for value in private_values):
        return 'actual private session value'
    for category, pattern in PATTERNS:
        if pattern.search(data):
            return category
    return None


def main():
    os.chdir(git('rev-parse', '--show-toplevel').decode().strip())
    values = private_cookie_values()
    audit = sys.argv[1:] == ['--audit']
    revisions = set()
    if audit:
        revisions.update(git('rev-list', '--all').decode().split())
    else:
        for line in sys.stdin:
            fields = line.split()
            if len(fields) != 4:
                raise RuntimeError('Invalid pre-push input')
            _, local_oid, _, remote_oid = fields
            if not all(re.fullmatch(r'[0-9a-f]{40,64}', oid) for oid in (local_oid, remote_oid)):
                raise RuntimeError('Invalid revision')
            if set(local_oid) == {'0'}:
                continue
            args = ['rev-list', local_oid]
            if set(remote_oid) != {'0'}:
                exists = subprocess.run(['git', 'cat-file', '-e', remote_oid + '^{commit}'],
                                        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                if exists.returncode == 0:
                    args.append('^' + remote_oid)
            revisions.update(git(*args).decode().split())

    findings, checked = set(), {}

    def report(path, reason):
        if reason:
            findings.add((path, reason))

    def inspect_blob(path, oid, mode):
        report(path, path_reason(path))
        if mode not in ('100644', '100755'):
            report(path, 'symlink/submodule requires separate review')
            return
        if oid not in checked:
            size = int(git('cat-file', '-s', oid))
            checked[oid] = ('oversized file requires separate review' if size > MAX_BYTES
                            else content_reason(git('cat-file', 'blob', oid), values))
        report(path, checked[oid])

    for rev in revisions:
        report('commit ' + rev[:12], content_reason(
            git('show', '-s', '--format=%B%n%an%n%ae%n%cn%n%ce', rev), values))
        for entry in git('ls-tree', '-rz', rev).split(b'\0'):
            if entry:
                meta, raw_path = entry.split(b'\t', 1)
                mode, kind, oid = meta.decode().split()
                path = raw_path.decode('utf-8', 'replace')
                if kind != 'blob':
                    report(path, 'submodule requires separate review')
                else:
                    inspect_blob(path, oid, mode)

    if audit:
        for entry in git('ls-files', '--stage', '-z').split(b'\0'):
            if entry:
                meta, raw_path = entry.split(b'\t', 1)
                mode, oid, stage = meta.decode().split()
                path = raw_path.decode('utf-8', 'replace')
                if stage != '0':
                    report(path, 'unresolved index conflict')
                inspect_blob(path, oid, mode)
        for raw_path in git('ls-files', '--cached', '--others', '--exclude-standard', '-z').split(b'\0'):
            if not raw_path:
                continue
            path = raw_path.decode('utf-8', 'replace')
            p = Path(path)
            report(path, path_reason(path))
            if p.is_symlink():
                report(path, 'symlink requires separate review')
            elif p.is_file():
                report(path, 'oversized file requires separate review' if p.stat().st_size > MAX_BYTES
                       else content_reason(p.read_bytes(), values))

    if findings:
        print('BLOCKED: review findings before publishing.', file=sys.stderr)
        for path, reason in sorted(findings):
            print(f'  {path!r}: {reason}', file=sys.stderr)
        return 1
    print(f'Publication check passed: {len(revisions)} commit(s), {len(checked)} distinct blob(s).')
    print('Manual review is still required; web uploads do not trigger Git hooks.')
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except Exception:
        print('Audit could not finish safely; publication stopped. No secret values printed.', file=sys.stderr)
        sys.exit(1)
