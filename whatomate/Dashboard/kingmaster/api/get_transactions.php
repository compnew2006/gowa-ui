<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

$pdo = getDB();

$user_id = isset($_GET['user_id']) ? (int)$_GET['user_id'] : 0;

if (empty($user_id)) {
    echo json_encode(['success' => false, 'message' => 'معرف المستخدم مطلوب']);
    exit;
}

try {
    $stmt = $pdo->prepare("
        SELECT 
            id,
            transaction_type,
            sender_id,
            sender_name,
            receiver_id,
            receiver_name,
            amount,
            note,
            created_at
        FROM wallet_transactions
        WHERE sender_id = ? OR receiver_id = ?
        ORDER BY created_at DESC
        LIMIT 100
    ");
    
    $stmt->execute([$user_id, $user_id]);
    $transactions = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'transactions' => $transactions,
        'total' => count($transactions)
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage(),
        'transactions' => []
    ], JSON_UNESCAPED_UNICODE);
}
