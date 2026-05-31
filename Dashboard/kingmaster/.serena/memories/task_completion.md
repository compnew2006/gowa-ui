# Task Completion

No automated quality gates configured. When a coding task is done:

## PHP changes
1. Verify no PHP syntax errors: `php -l root/file.php`
2. Confirm prepared statements used for all DB queries (no string interpolation).
3. Confirm `htmlspecialchars()` applied to all user-supplied data in HTML output.
4. If API endpoint: verify `requireAuthenticatedUser()` or appropriate auth guard is called.
5. If new session page: verify `session_start()` + redirect guard at top.

## CSS changes
1. Check cascade order — `enterprise-polish.css` loads last; new tokens go there.
2. Verify no new `!important` unless overriding base CSS with `position: fixed` media queries.
3. Test RTL layout — Arabic text should not break with new styles.

## JS changes
1. No framework-specific build needed — plain JS files.
2. Verify Chart.js canvas elements have `role="img"` and `aria-label`.

## General
- No test suite exists. Manual browser verification is required.
- No linter or formatter configured.
- No git repo initialized in this extracted project.
