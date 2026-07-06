<?php
require_once __DIR__ . '/../config/database.php';

if (!isProductionEnv() && configValue('ALLOW_PHPINFO', '') === '1') {
    requireAdminUser();
    phpinfo();
    exit;
}

respondError('Not found', 404);
