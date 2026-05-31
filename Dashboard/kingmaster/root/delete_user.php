<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id']) || !isset($_SESSION['is_logged_in'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح بالدخول']);
    exit;
}

// التحقق من صلاحية المسؤول (اختياري - يمكنك تفعيله إذا أردت)
// if (!isset($_SESSION['is_admin']) || $_SESSION['is_admin'] != 1) {
//     echo json_encode(['success' => false, 'message' => 'ليس لديك صلاحية لحذف المستخدمين']);
//     exit;
// }

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    echo json_encode(['success' => false, 'message' => 'طريقة الطلب غير صحيحة']);
    exit;
}

try {
    $userId = isset($_POST['user_id']) ? (int)$_POST['user_id'] : 0;

    if ($userId <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم غير صحيح']);
        exit;
    }

    // منع حذف المستخدم الحالي
    if ($userId == $_SESSION['user_id']) {
        echo json_encode(['success' => false, 'message' => 'لا يمكنك حذف حسابك الخاص']);
        exit;
    }

    // حذف المستخدم
    $query = "DELETE FROM users WHERE id = ?";
    $result = executeQuery($query, [$userId]);

    if ($result && $result->rowCount() > 0) {
        echo json_encode([
            'success' => true,
            'message' => 'تم حذف المستخدم بنجاح'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'المستخدم غير موجود أو تم حذفه مسبقاً'
        ]);
    }

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء الحذف',
        'error' => $e->getMessage()
    ]);
}
?>
