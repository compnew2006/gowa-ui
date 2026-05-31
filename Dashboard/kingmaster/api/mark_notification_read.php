<?php
header('Content-Type: application/json');
session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

require_once '../config/database.php';

$user_id = $_SESSION['user_id'];
$data = json_decode(file_get_contents('php://input'), true);
$notification_id = $data['notification_id'] ?? null;

if (!$notification_id) {
    echo json_encode(['success' => false, 'message' => 'معرف الإشعار مطلوب']);
    exit;
}

try {
    $conn = getDB();
    
    // Mark as read (only for the current user)
    $stmt = $conn->prepare("
        UPDATE notifications 
        SET is_read = 1 
        WHERE id = :id AND user_id = :user_id
    ");
    $stmt->execute([
        ':id' => $notification_id,
        ':user_id' => $user_id
    ]);
    
    echo json_encode([
        'success' => true,
        'message' => 'تم تحديث الإشعار'
    ]);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في تحديث الإشعار: ' . $e->getMessage()
    ]);
}
