<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id']) || !isset($_SESSION['is_logged_in'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح بالدخول']);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    echo json_encode(['success' => false, 'message' => 'طريقة الطلب غير صحيحة']);
    exit;
}

try {
    $userId = isset($_POST['user_id']) ? (int)$_POST['user_id'] : 0;
    $type = isset($_POST['type']) ? sanitizeInput($_POST['type']) : '';

    if ($userId <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم غير صحيح']);
        exit;
    }

    if (!in_array($type, ['accounts', 'messages'])) {
        echo json_encode(['success' => false, 'message' => 'نوع الإعادة غير صحيح']);
        exit;
    }

    // تحديد العمود المطلوب إعادة تعيينه
    $column = ($type === 'accounts') ? 'account_count' : 'msg_count';
    
    // إعادة التعيين
    $query = "UPDATE users SET $column = 0 WHERE id = ?";
    $result = executeQuery($query, [$userId]);

    if ($result && $result->rowCount() > 0) {
        $message = ($type === 'accounts') ? 'تم إعادة تعيين عدد الحسابات بنجاح' : 'تم إعادة تعيين عدد الرسائل بنجاح';
        echo json_encode([
            'success' => true,
            'message' => $message
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل إعادة التعيين'
        ]);
    }

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء إعادة التعيين',
        'error' => $e->getMessage()
    ]);
}
?>
