#!/usr/bin/env python3
"""Build the supported release executables and metadata from a clean checkout.

Requires Python 3.9+, Git and Go. Does not create tags, publish, or deploy.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
from datetime import datetime, timezone

PLATFORMS = (('darwin', 'arm64'), ('linux', 'amd64'), ('windows', 'amd64'))
PROGRAMS = (
    ('xiaohongshu-mcp', '.'),
    ('xiaohongshu-login', './cmd/login'),
    ('xiaohongshu-recover', './cmd/recover'),
)
METADATA_PACKAGE = 'github.com/liaogx/xiaohongshu-mcp/pkg/buildinfo'
REPOSITORY = 'https://github.com/liaogx/xiaohongshu-mcp'


def release_version(tag):
    if not re.fullmatch(r'v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?', tag):
        raise ValueError('Use a release tag such as v1.0.0')
    return tag[1:]


def asset_name(program, goos, goarch):
    return f'{program}-{goos}-{goarch}' + ('.exe' if goos == 'windows' else '')


def json_objects(text):
    decoder = json.JSONDecoder()
    while text.strip():
        obj, end = decoder.raw_decode(text.lstrip())
        yield obj
        text = text.lstrip()[end:]


def run(args, root, env=None):
    return subprocess.check_output(args, cwd=root, env=env, text=True).strip()


def sha256(path):
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def build_environment(goos, goarch):
    env = os.environ.copy()
    env.update(GOOS=goos, GOARCH=goarch, CGO_ENABLED='0', GOWORK='off',
               GOTOOLCHAIN='local', GOFLAGS='')
    # Keep the documented baseline CPU target independent of the developer shell.
    for name in ('GOAMD64', 'GOARM64', 'GOEXPERIMENT'):
        env.pop(name, None)
    return env


def third_party_notices(go, root, env):
    # Collect the union of modules actually linked on any supported platform,
    # rather than unrelated modules only present in dependency go.mod graphs.
    modules = {}
    for goos, goarch in PLATFORMS:
        target_env = dict(env, GOOS=goos, GOARCH=goarch)
        packages = run([go, 'list', '-deps', '-json=Module', '-mod=readonly',
                        *(package for _, package in PROGRAMS)], root, target_env)
        for package in json_objects(packages):
            module = package.get('Module')
            if module and not module.get('Main'):
                modules[module['Path']] = module
    goroot = Path(run([go, 'env', 'GOROOT'], root, env))
    parts = ['Third-party license texts included with this binary distribution.\n',
             '=== Go standard library and runtime ===\n', (goroot / 'LICENSE').read_text()]
    for module in sorted(modules.values(), key=lambda m: m['Path']):
        if module.get('Replace'):
            raise ValueError('Release builds do not accept module replacements')
        if not module.get('Dir'):
            raise ValueError(f"Download dependencies first: {module['Path']}")
        directory = Path(module['Dir'])
        legal_files = sorted(p for p in directory.iterdir() if p.is_file() and
                             re.fullmatch(r'(LICENSE|LICENCE|COPYING|NOTICE)([.-].*)?',
                                          p.name, re.IGNORECASE))
        if not legal_files:
            raise ValueError(f"Review missing dependency license: {module['Path']}")
        parts.append(f"\n=== {module['Path']} {module.get('Version', '')} ===\n")
        for path in legal_files:
            parts.extend([f'--- {path.name} ---\n', path.read_text(errors='strict')])
    return '\n'.join(parts)


def render_notes(source, info):
    marker = '<!-- BUILD_INFO -->'
    if source.count(marker) != 1:
        raise ValueError('Release notes must contain one BUILD_INFO marker')
    rows = [
        ('Version', info['version']), ('Tag', info['tag']),
        ('Commit', f"[`{info['commit'][:12]}`]({REPOSITORY}/commit/{info['commit']})"),
        ('Go', info['go_version']), ('Build time (UTC)', info['build_time']),
        ('Browser distribution', info['browser_version']),
        ('CGO_ENABLED', '0'), ('Build flags', '`-trimpath -buildvcs=true -mod=readonly -ldflags="-s -w …"`'),
    ]
    table = '| 项目 | 信息 |\n| --- | --- |\n' + '\n'.join(
        f'| {name} | {value} |' for name, value in rows)
    return source.replace(marker, table)


def build(args):
    root = Path(__file__).resolve().parents[1]
    version = release_version(args.tag)
    if run(['git', 'status', '--porcelain', '--untracked-files=normal'], root):
        raise ValueError('Commit source changes before building a release')
    default = re.search(r'const DefaultVersion = "([^"]+)"',
                        (root / 'pkg/buildinfo/buildinfo.go').read_text())
    if not default or default[1] != version:
        raise ValueError('Release version must match pkg/buildinfo.DefaultVersion')
    commit = run(['git', 'rev-parse', 'HEAD'], root)
    if not re.fullmatch(r'[0-9a-f]{40,64}', commit):
        raise ValueError('Cannot determine source commit')
    existing_tag = run(['git', 'tag', '--list', args.tag], root)
    if existing_tag and run(['git', 'rev-parse', args.tag + '^{commit}'], root) != commit:
        raise ValueError('Release tag points at a different commit')
    output = (root / (args.output or f'dist/{args.tag}')).resolve()
    if root not in output.parents:
        raise ValueError('Build artifacts must be inside an ignored directory of this checkout')
    subprocess.run(['git', 'check-ignore', '-q', str(output.relative_to(root)) + '/'],
                   cwd=root, check=True)
    if output.exists():
        raise ValueError('Output directory already exists; choose a new empty destination')

    env = build_environment('darwin', 'arm64')
    go = str(Path(args.go).resolve()) if os.path.sep in args.go else args.go
    run([go, 'mod', 'download'], root, env)
    notices = third_party_notices(go, root, env)
    go_version = run([go, 'env', 'GOVERSION'], root, env)
    build_time = datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
    info = dict(version=version, tag=args.tag, commit=commit, go_version=go_version,
                build_time=build_time, browser_version=(root / 'browser/browser_version.txt').read_text().strip(),
                cgo_enabled=False, trimpath=True, vcs_modified=False, binaries=[])
    flags = (f'-s -w -X main.version={version} '
             f'-X {METADATA_PACKAGE}.Commit={commit} '
             f'-X {METADATA_PACKAGE}.BuildTime={build_time}')
    output.mkdir(parents=True)
    for goos, goarch in PLATFORMS:
        for program, package in PROGRAMS:
            name = asset_name(program, goos, goarch)
            print(f'Building {name}', flush=True)
            subprocess.run([go, 'build', '-trimpath', '-buildvcs=true', '-mod=readonly',
                            '-ldflags', flags, '-o', str(output / name), package],
                           cwd=root, env=build_environment(goos, goarch), check=True)
            info['binaries'].append(dict(name=name, goos=goos, goarch=goarch,
                                         size=(output / name).stat().st_size,
                                         sha256=sha256(output / name)))
    if run(['git', 'status', '--porcelain', '--untracked-files=normal'], root):
        raise ValueError('Source changed while building; do not publish these artifacts')
    for name in ('LICENSE', 'NOTICE'):
        shutil.copyfile(root / name, output / name)
    (output / 'THIRD_PARTY_NOTICES.txt').write_text(notices, encoding='utf-8')
    (output / 'BUILD-INFO.json').write_text(json.dumps(info, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    assets = sorted(output.iterdir())
    (output / 'SHA256SUMS').write_text(''.join(f'{sha256(p)}  {p.name}\n' for p in assets), encoding='utf-8')
    notes = (root / f'docs/releases/{args.tag}.md').read_text(encoding='utf-8')
    (output / 'RELEASE_NOTES.md').write_text(render_notes(notes, info), encoding='utf-8')
    print(f'Built {len(info["binaries"])} executables in {output.relative_to(root)}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--tag', required=True)
    parser.add_argument('--go', default='go', help='Go executable; uses this toolchain without auto-upgrading')
    parser.add_argument('--output', help='New ignored output directory inside the repository')
    args = parser.parse_args()
    try:
        build(args)
    except (ValueError, OSError, subprocess.CalledProcessError) as exc:
        parser.exit(1, f'Release build stopped: {exc}\n')


if __name__ == '__main__':
    main()
