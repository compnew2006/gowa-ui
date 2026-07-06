<?php
header('Content-Type: application/json');
session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

require_once '../config/database.php';

$user_id = $_SESSION['user_id'];

try {
    $conn = getDB();
    
    // Get last 5 notifications
    $stmt = $conn->prepare("
        SELECT id, title, message, type, is_read, created_at 
        FROM notifications 
        WHERE user_id = :user_id 
        ORDER BY created_at DESC 
        LIMIT 5
    ");
    $stmt->execute([':user_id' => $user_id]);
    $notifications = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    // Get unread count
    $stmt = $conn->prepare("
        SELECT COUNT(*) as unread_count 
        FROM notifications 
        WHERE user_id = :user_id AND is_read = 0
    ");
    $stmt->execute([':user_id' => $user_id]);
    $unread = $stmt->fetch(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'notifications' => $notifications,
        'unread_count' => $unread['unread_count']
    ]);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في جلب الإشعارات: ' . $e->getMessage()
    ]);
}
