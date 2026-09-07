# Private data and publication checks

Only source code, synthetic tests, and usage documentation belong in this repository. Never commit real account profiles, cookies, login/security QR codes, tokens, passwords, private keys, personal media, logs, browser profiles, or interaction receipts.

Keep runtime data outside the checkout and configure `COOKIES_PATH` consistently across the service and login/recovery commands. Treat full signed note URLs and image links as potentially sensitive. Use placeholders when reporting a bug.

Before committing or publishing, review the actual staged files and the outgoing history, then run:

```bash
python3 scripts/audit_publication.py --audit
git diff --cached --stat
git diff --cached
```

The check scans the index, non-ignored working files, and every reachable commit, so deleting a secret in the latest version is not enough. It reports paths and categories without printing matched values. Optional `XHS_AUDIT_COOKIE_FILE` points to a local cookie file for in-memory exact-match comparison; never commit that path or its contents.

To run the check automatically before pushes, install a local `.git/hooks/pre-push` script that invokes `python3 scripts/audit_publication.py` from the repository root and forwards its arguments and standard input. Run the same `--audit` check manually before browser uploads, which bypass Git hooks. Hooks are local and are not installed automatically by cloning.

The script is a conservative guard, not a complete secret detector. Manually review examples, build scripts, dependencies, and the exact outgoing changes. Do not force-add ignored runtime files. If a secret reaches a public commit or attachment, revoke/rotate it and address the exposed history; do not assume deleting the latest file removes the exposure.
