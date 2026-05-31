<?php
require_once __DIR__ . '/config/database.php';

header('Content-Type: application/json');

try {
    $sql = "DELETE FROM users";
    $count = getRowCount($sql);

    echo json_encode([
        'success' => true,
        'deleted_rows' => $count
    ]);
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'Error deleting users'
    ]);
}
