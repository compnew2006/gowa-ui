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

$receiver_id = $data['receiver_id'] ?? null;
$message = trim($data['message'] ?? '');
$image_path = $data['image_path'] ?? null;

if (!$receiver_id || (empty($message) && empty($image_path))) {
    echo json_encode(['success' => false, 'message' => 'البيانات غير مكتملة']);
    exit;
}

try {
    $conn = getDB();
    
    // Check if conversation exists
    $stmt = $conn->prepare("
        SELECT id FROM conversations 
        WHERE (user1_id = :user1 AND user2_id = :user2) 
        OR (user1_id = :user3 AND user2_id = :user4)
    ");
    $stmt->execute([
        ':user1' => $user_id, 
        ':user2' => $receiver_id,
        ':user3' => $receiver_id,
        ':user4' => $user_id
    ]);
    $conversation = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$conversation) {
        // Create new conversation
        $stmt = $conn->prepare("
            INSERT INTO conversations (user1_id, user2_id, last_message, last_message_time)
            VALUES (:user1, :user2, :message, NOW())
        ");
        $stmt->execute([
            ':user1' => min($user_id, $receiver_id),
            ':user2' => max($user_id, $receiver_id),
            ':message' => $message
        ]);
        $conversation_id = $conn->lastInsertId();
    } else {
        $conversation_id = $conversation['id'];
        
        // Update conversation
        $stmt = $conn->prepare("
            UPDATE conversations 
            SET last_message = :message, last_message_time = NOW()
            WHERE id = :id
        ");
        $stmt->execute([':message' => $message, ':id' => $conversation_id]);
    }
    
    // Insert message
    $stmt = $conn->prepare("
        INSERT INTO messages (conversation_id, sender_id, receiver_id, message, image_path, is_read, created_at)
        VALUES (:conv_id, :sender, :receiver, :message, :image_path, 0, NOW())
    ");
    $stmt->execute([
        ':conv_id' => $conversation_id,
        ':sender' => $user_id,
        ':receiver' => $receiver_id,
        ':message' => $message,
        ':image_path' => $image_path
    ]);
    
    echo json_encode([
        'success' => true,
        'message' => 'تم إرسال الرسالة بنجاح',
        'conversation_id' => $conversation_id
    ]);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في إرسال الرسالة: ' . $e->getMessage()
    ]);
}
