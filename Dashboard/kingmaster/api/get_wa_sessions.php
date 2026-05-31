<?php
header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

session_start();
if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

$user_id = $_SESSION['user_id'];

try {
    $conn = getDB();
    
    // جلب جميع جلسات الواتساب للمستخدم
    $stmt = $conn->prepare("
        SELECT 
            id,
            name,
            account_uid,
            status,
            created_at
        FROM accounts 
        WHERE user_id = :user_id AND channel = 'whatsapp'
        ORDER BY created_at DESC
    ");
    
    $stmt->execute([':user_id' => $user_id]);
    $sessions = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'sessions' => $sessions
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في جلب الجلسات: ' . $e->getMessage()
    ]);
}
