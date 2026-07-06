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
$session_id = isset($_GET['session_id']) ? trim($_GET['session_id']) : '';
$contact = isset($_GET['contact']) ? trim($_GET['contact']) : '';

if (empty($session_id)) {
    echo json_encode(['success' => false, 'message' => 'معرف الجلسة مطلوب']);
    exit;
}

if (empty($contact)) {
    echo json_encode(['success' => false, 'message' => 'رقم الجهة مطلوب']);
    exit;
}

try {
    $conn = getDB();
    
    // جلب المحادثات للجلسة المحددة مع شخص معين
    $stmt = $conn->prepare("
        SELECT 
            id,
            txt,
            froms,
            tos,
            from_me,
            created_at
        FROM wa_conv 
        WHERE user_id = :user_id 
            AND session_id = :session_id
            AND (froms = :contact OR tos = :contact)
        ORDER BY created_at ASC
    ");
    
    $stmt->execute([
        ':user_id' => $user_id,
        ':session_id' => $session_id,
        ':contact' => $contact
    ]);
    
    $conversations = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'conversations' => $conversations,
        'count' => count($conversations)
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في جلب المحادثات: ' . $e->getMessage()
    ]);
}
