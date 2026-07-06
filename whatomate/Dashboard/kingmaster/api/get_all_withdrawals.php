<?php
session_start();
header('Content-Type: application/json');
require_once '../config/database.php';

if (!isset($_SESSION['user_id'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب تسجيل الدخول أولاً'
    ]);
    exit;
}

$conn = getDB();

try {
    $stmt = $conn->prepare("
        SELECT w.*, u.first_name, u.email, u.phone
        FROM withdrawals w
        LEFT JOIN users u ON w.user_id = u.user_id
        ORDER BY w.created_at DESC
    ");
    $stmt->execute();
    $withdrawals = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'withdrawals' => $withdrawals
    ]);
} catch(PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ في جلب البيانات'
    ]);
}
?>
