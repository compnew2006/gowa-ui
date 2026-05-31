<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json');

exit;
 
// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id']) || !isset($_SESSION['is_logged_in'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح بالدخول']);
    exit;
}

// التحقق من وجود معرف المسؤول الأصلي
if (!isset($_SESSION['original_admin_id'])) {
    echo json_encode(['success' => false, 'message' => 'لا يوجد حساب مسؤول للعودة إليه']);
    exit;
}

try {
    $originalAdminId = $_SESSION['original_admin_id'];

    // جلب بيانات المسؤول الأصلي
    $query = "SELECT * FROM users WHERE user_id = ?";
    $stmt = executeQuery($query, [$originalAdminId]);
    $adminUser = $stmt->fetch();

    if (!$adminUser) {
        echo json_encode(['success' => false, 'message' => 'حساب المسؤول الأصلي غير موجود']);
        exit;
    }

    // العودة إلى حساب المسؤول
    $_SESSION['user_id'] = $adminUser['user_id'];
    $_SESSION['first_name'] = $adminUser['first_name'];
    $_SESSION['last_name'] = $adminUser['last_name'];
    $_SESSION['is_admin'] = $adminUser['is_admin'];

    // حذف متغيرات التبديل
    unset($_SESSION['original_admin_id']);
    unset($_SESSION['original_admin_name']);
    unset($_SESSION['is_viewing_as']);

    echo json_encode([
        'success' => true,
        'message' => 'تم العودة إلى حسابك الأصلي',
        'redirect' => 'manage-users.php'
    ]);

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء العودة',
        'error' => $e->getMessage()
    ]);
}
?>
