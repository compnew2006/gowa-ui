<?php
header('Content-Type: application/json');
session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

require_once '../config/database.php';
require_once '../includes/functions.php';

$user_id = $_SESSION['user_id'];

try {
    $conn = getDB();
    
    // Get conversations with last 3 messages
    $stmt = $conn->prepare("
        SELECT 
            c.id as conversation_id,
            CASE 
                WHEN c.user1_id = :user_id1 THEN c.user2_id 
                ELSE c.user1_id 
            END as other_user_id,
            c.last_message,
            c.last_message_time,
            (SELECT COUNT(*) FROM messages 
             WHERE conversation_id = c.id 
             AND receiver_id = :user_id2 
             AND is_read = 0) as unread_count
        FROM conversations c
        WHERE c.user1_id = :user_id3 OR c.user2_id = :user_id4
        ORDER BY c.last_message_time DESC
        LIMIT 3
    ");
    $stmt->execute([
        ':user_id1' => $user_id,
        ':user_id2' => $user_id,
        ':user_id3' => $user_id,
        ':user_id4' => $user_id
    ]);
    $conversations = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    // Get user details for each conversation
foreach ($conversations as &$conv) {
    $user = getUserByUserId($conv['other_user_id']);

    if (is_array($user)) {
        $conv['other_user_name'] =
            trim(($user['first_name'] ?? '') . ' ' . ($user['last_name'] ?? ''));

        if ($conv['other_user_name'] === '') {
            $conv['other_user_name'] = 'مستخدم';
        }

        $conv['other_user_img'] =
            (!empty($user['img']) && $user['img'] !== '0')
                ? $user['img']
                : 'https://i.pravatar.cc/150?img=33';

    } else {
        // لو المستخدم مش موجود
        $conv['other_user_name'] = 'مستخدم';
        $conv['other_user_img'] = 'https://i.pravatar.cc/150?img=33';
    }
}

    
    // Get total unread count
    $stmt = $conn->prepare("
        SELECT COUNT(*) as total_unread 
        FROM messages 
        WHERE receiver_id = :user_id AND is_read = 0
    ");
    $stmt->execute([':user_id' => $user_id]);
    $unread = $stmt->fetch(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'conversations' => $conversations,
        'unread_count' => $unread['total_unread']
    ]);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في جلب المحادثات: ' . $e->getMessage()
    ]);
}
