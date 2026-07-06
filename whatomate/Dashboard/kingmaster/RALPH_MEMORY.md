# Ralph Memory - Kingmaster Project Memory
# ذاكرة رالف - ذاكرة مشروع كينج ماستر

## [2026-05-29] Issue: Compliance and Anti-Detection Verification for Facebook & Instagram Operations

- **The Trap:** Assuming that Facebook and Instagram automation scripts execute locally within custom Puppeteer or Playwright instances inside the Kingmaster server codebase, similar to the WhatsApp wppconnect implementation in `sessionManager.js`.
- **The Reality:** The local codebase serves strictly as a management proxy and dashboard. Campaign details are saved locally in MySQL, but all automation and web interaction requests are forwarded via `api/proxy.php` and `root/proxy.php` to a remote backend located at `https://apis.kingmaster.info/api.php`. Consequently, the local codebase lacks any direct Puppeteer configs, proxy bindings, or browser fingerprinting configurations for Facebook and Instagram.
- **The Fix:** Conducted a comprehensive 6-layer anti-detection compliance audit, mapped out specific structural weaknesses (such as timing parameter leakage where Instagram posting files request Facebook timing configurations), and recorded all compliance findings and remediation scripts inside a bilingual (Arabic/English) security artifact.
- **The Law:** Always verify if automation logic is executed locally or offloaded to a remote proxy server before auditing browser-level evasion parameters.
