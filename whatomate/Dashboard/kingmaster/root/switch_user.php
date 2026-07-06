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
    $targetUserId = isset($_POST['target_user_id']) ? (int)$_POST['target_user_id'] : 0;

    if ($targetUserId <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم غير صحيح']);
        exit;
    }

    // التحقق من وجود المستخدم المستهدف
    $query = "SELECT * FROM users WHERE id = ?";
    $stmt = executeQuery($query, [$targetUserId]);
    $targetUser = $stmt->fetch();

    if (!$targetUser) {
        echo json_encode(['success' => false, 'message' => 'المستخدم غير موجود']);
        exit;
    }

    // حفظ معرف المسؤول الأصلي إذا لم يكن محفوظاً
    if (!isset($_SESSION['original_admin_id'])) {
        $_SESSION['original_admin_id'] = $_SESSION['user_id'];
        $_SESSION['original_admin_name'] = $_SESSION['first_name'] . ' ' . $_SESSION['last_name'];
        $_SESSION['is_viewing_as'] = true;
    }

    // التبديل إلى المستخدم المستهدف
    $_SESSION['user_id'] = $targetUser['user_id'];
    $_SESSION['first_name'] = $targetUser['first_name'];
    $_SESSION['last_name'] = $targetUser['last_name'];
    $_SESSION['is_admin'] = $targetUser['is_admin'];

    echo json_encode([
        'success' => true,
        'message' => 'تم التبديل إلى حساب: ' . $targetUser['first_name'] . ' ' . $targetUser['last_name'],
        'redirect' => 'index.php'
    ]);

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء التبديل',
        'error' => $e->getMessage()
    ]);
}
?>
