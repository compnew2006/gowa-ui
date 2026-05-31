<?php
require_once __DIR__ . '/config/database.php';

header('Content-Type: application/json');

try {
    $sql = "SELECT id, user_id, name, account_uid, channel, status, method, cookies_text, data, created_at, updated_at 
            FROM accounts";
    $rows = fetchAll($sql);

    echo json_encode([
        'success' => true,
        'count' => count($rows),
        'data' => $rows
    ]);
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'Error fetching accounts'
    ]);
}


