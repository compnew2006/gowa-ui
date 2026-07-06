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
    $isAdmin = isset($_POST['is_admin']) ? (int)$_POST['is_admin'] : 0;

    if ($userId <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم غير صحيح']);
        exit;
    }

    // منع المستخدم من تغيير صلاحية نفسه
    if ($userId == $_SESSION['user_id']) {
        echo json_encode(['success' => false, 'message' => 'لا يمكنك تغيير صلاحيتك الخاصة']);
        exit;
    }

    // تحديد القيمة (0 أو 1 فقط)
    $isAdmin = ($isAdmin == 1) ? 1 : 0;

    // تحديث الصلاحية
    $query = "UPDATE users SET is_admin = ? WHERE id = ?";
    $result = executeQuery($query, [$isAdmin, $userId]);

    if ($result && $result->rowCount() > 0) {
        $roleText = $isAdmin ? 'مسؤول' : 'مستخدم';
        echo json_encode([
            'success' => true,
            'message' => "تم تحديث الصلاحية إلى: {$roleText}"
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل تحديث الصلاحية'
        ]);
    }

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء التحديث',
        'error' => $e->getMessage()
    ]);
}
?>
