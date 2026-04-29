# POS Foundation Implementation (00-foundation-and-installation)

## Completed: 2026-04-25

## Key Files Created
- `plugins/webkul/POS/composer.json` - Package metadata, namespace `Webkul\Pos`
- `plugins/webkul/POS/src/PosServiceProvider.php` - Extends PackageServiceProvider, registers routes/views/translations
- `plugins/webkul/POS/src/PosPlugin.php` - Filament Plugin registration, discovers resources/pages/clusters/widgets
- `plugins/webkul/POS/routes/web.php` - `/pos` route with `web` + `auth` middleware
- `plugins/webkul/POS/routes/api.php` - `admin/api/v1/pos/health` and `admin/api/v1/pos/bootstrap` with `auth:sanctum`
- `plugins/webkul/POS/resources/views/pos/shell.blade.php` - Basic cashier shell view
- `plugins/webkul/POS/resources/lang/en/app.php` and `ar/app.php` - Translations
- `plugins/webkul/POS/config/filament-shield.php` - Shield permissions config
- `plugins/webkul/POS/database/seeders/DatabaseSeeder.php` - Empty seeder
- Tests: BootTest, AdminAccessTest, CashierShellTest, SecurityTest (14 tests, 18 assertions)

## Key Patterns
- Plugin ID: `pos`, namespace: `Webkul\Pos`, table prefix: `pos_`
- Routes only load when plugin is installed (`is_installed=1` in `plugins` DB table)
- Plugin registered in `bootstrap/providers.php`
- Filament panel registration via `Panel::configureUsing()` in `packageRegistered()`
- Tests use `require_once` for SecurityHelper and TestBootstrapHelper from support plugin

## Gotcha
- The `login` named route doesn't exist in AureusERP (Filament uses `/admin/login`), so auth middleware throws RouteNotFoundException for unauthenticated web requests
