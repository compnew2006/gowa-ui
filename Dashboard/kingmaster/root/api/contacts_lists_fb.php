<?php
session_start();
header('Content-Type: application/json; charset=utf-8');
require_once __DIR__ . '/../config/database.php';

try {
    if (!isset($_SESSION['user_id'])) {
        http_response_code(401);
        echo json_encode(['success'=>false,'message'=>'Unauthorized']);
        exit;
    }
    $uid = $_SESSION['user_id'];
    $db = getDB();
    $stmt = $db->prepare("SELECT id, name FROM contacts WHERE user_id = :uid AND platform = 'facebook' ORDER BY created_at DESC, id DESC");
    $stmt->execute([':uid'=>$uid]);
    $rows = $stmt->fetchAll(PDO::FETCH_ASSOC) ?: [];
    echo json_encode(['success'=>true,'lists'=>$rows], JSON_UNESCAPED_UNICODE);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode(['success'=>false,'message'=>$e->getMessage()]);
}
