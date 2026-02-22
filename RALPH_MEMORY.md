# Ralph Memory

## 2026-02-22 03:48 Issue: Git Push Rejected Due to Remote Divergence

- **The Trap:** Assuming `git status` reporting "up to date" means the remote branch hasn't changed since the last fetch.
- **The Reality:** Remote repositories (especially forks) can have forced updates or commits from other contributors that aren't reflected in the local tracking branch until an explicit `git fetch` or `git pull` is performed.
- **The Fix:** Performed `git fetch origin` to reveal the divergence, then `git pull origin main --rebase` to integrate remote changes before pushing successfully.
- **The Law:** Always `fetch` and verify remote state explicitly before assuming a push will succeed, especially on shared or frequently updated branches.

## 2026-02-22 03:50 Issue: Executables and Archives appearing in Git Status

- **The Trap:** Relying on generic or manual ignores for common binary output filenames.
- **The Reality:** Build processes often generate timestamped, architecture-specific, or space-delimited binary filenames that escape simple exact-match patterns.
- **The Fix:** Updated `.gitignore` with wildcard patterns (`whatomate*`, `whatomate_*`) and comprehensive archive extensions (`*.tar.gz`, `*.zip`, `*.7z`) to ensure all variants of build artifacts are ignored.
- **The Law:** Use broad wildcard patterns for build outputs and archive types to prevent accidental inclusion of heavy binaries in the repository history.
