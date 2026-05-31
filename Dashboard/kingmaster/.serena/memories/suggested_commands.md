# Suggested Commands

## Development

```bash
# Start PHP dev server (from root/)
php -S localhost:8000 -t root/

# Start WhatsApp Node.js service
cd /path/to/project && npm start
# or with auto-reload:
npm run dev

# Install PHP dependencies
cd root && composer install

# Install Node.js dependencies
npm install
```

## Database

```bash
# Run migration
mysql -u kingmaster -p kingmaster < migrations/2026_05_28_performance_indexes.sql

# Import full schema
mysql -u kingmaster -p kingmaster < root/schema_only.sql
```

## System utilities (Darwin)
- Standard unix commands work normally (`ls`, `grep`, `find`, `git`).
- No Darwin-specific deviations needed.
- PHP is likely from Homebrew: `php` or `php83`/`php84`.
- MySQL/MariaDB: `mysql` client from Homebrew.

## Notes
- No linter, formatter, or test runner configured.
- No `package.json` scripts beyond `start` and `dev`.
- No CI/CD configuration found.
