<?php
header('Content-Type: application/json');
session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

require_once '../config/database.php';

$user_id = $_SESSION['user_id'];
$conversation_id = $_GET['conversation_id'] ?? null;

if (!$conversation_id) {
    echo json_encode(['success' => false, 'message' => 'معرف المحادثة مطلوب']);
    exit;
}

try {
    $conn = getDB();
    
    // Get messages
    $stmt = $conn->prepare("
        SELECT 
            m.id,
            m.sender_id,
            m.receiver_id,
            m.message,
            m.image_path,
            m.is_read,
            m.created_at
        FROM messages m
        WHERE m.conversation_id = :conv_id
        ORDER BY m.created_at ASC
    ");
    $stmt->execute([':conv_id' => $conversation_id]);
    $messages = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    // Mark messages as read
    $stmt = $conn->prepare("
        UPDATE messages 
        SET is_read = 1 
        WHERE conversation_id = :conv_id 
        AND receiver_id = :user_id 
        AND is_read = 0
    ");
    $stmt->execute([':conv_id' => $conversation_id, ':user_id' => $user_id]);
    
    echo json_encode([
        'success' => true,
        'messages' => $messages
    ]);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في جلب الرسائل: ' . $e->getMessage()
    ]);
}
