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
    $stmt = $pdo->prepare("SELECT points_balance, money_balance FROM wallets WHERE user_id = ?");
    $stmt->execute([$user_id]);
    $wallet = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if ($wallet) {
        echo json_encode([
            'success' => true,
            'points_balance' => $wallet['points_balance'],
            'money_balance' => $wallet['money_balance']
        ], JSON_UNESCAPED_UNICODE);
    } else {
        echo json_encode([
            'success' => true,
            'points_balance' => 0,
            'money_balance' => 0
        ], JSON_UNESCAPED_UNICODE);
    }
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
