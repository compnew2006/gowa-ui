<?php
require_once __DIR__ . '/config/database.php';

header('Content-Type: application/json');

try {
    $sql = "SELECT 
                id, user_id, first_name, last_name, email, phone, timezone, job,
                birth_date, is_verified, is_admin, package, points, msg_count,
                account_count, expiry_date, created_at, updated_at, img, referrer_id
            FROM users";

    $rows = fetchAll($sql);

    echo json_encode([
        'success' => true,
        'count' => count($rows),
        'data' => $rows
    ]);
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'Error fetching users'
    ]);
}
