<?php
header('Content-Type: application/json; charset=utf-8');
require_once '../includes/functions.php';

try {
    $db = getDB();
    $q = isset($_GET['q']) ? trim($_GET['q']) : '';
    $sql = "SELECT id, groupId, groupName, groupLink, country, Language, groupDesc, GroupImage, categoryName
            FROM groups_list";
    $params = [];
    if ($q !== '') {
        $sql .= " WHERE groupName LIKE ?";
        $params[] = "%$q%";
    }
    $sql .= " ORDER BY created_at DESC LIMIT 200";
    $stmt = $db->prepare($sql);
    $stmt->execute($params);
    $rows = $stmt->fetchAll(PDO::FETCH_ASSOC);

    echo json_encode([ 'success' => true, 'results' => $rows ], JSON_UNESCAPED_UNICODE);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode([ 'success' => false, 'message' => 'DB error', 'error' => $e->getMessage() ]);
}
