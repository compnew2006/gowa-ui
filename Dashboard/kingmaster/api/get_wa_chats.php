<?php
header('Content-Type: application/json');
session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

require_once '../includes/db.php';

$user_id = $_SESSION['user_id'];
$session_id = isset($_GET['session_id']) ? $_GET['session_id'] : '';

if (empty($session_id)) {
    echo json_encode(['success' => false, 'message' => 'معرف الجلسة مطلوب']);
    exit;
}

try {
    // جلب المحادثات المجموعة حسب الشخص
    $stmt = $conn->prepare("
        SELECT 
            CASE 
                WHEN from_me = 1 THEN tos
                ELSE froms
            END as contact,
            MAX(txt) as last_message,
            MAX(created_at) as last_time,
            COUNT(*) as message_count
        FROM wa_conv
        WHERE user_id = ? AND session_id = ?
        GROUP BY contact
        ORDER BY last_time DESC
    ");
    
    $stmt->bind_param("ss", $user_id, $session_id);
    $stmt->execute();
    $result = $stmt->get_result();
    
    $chats = [];
    while ($row = $result->fetch_assoc()) {
        $chats[] = $row;
    }
    
    echo json_encode([
        'success' => true,
        'chats' => $chats,
        'count' => count($chats)
    ]);
    
} catch (Exception $e) {
    echo json_encode(['success' => false, 'message' => 'خطأ في الخادم: ' . $e->getMessage()]);
}
?>
