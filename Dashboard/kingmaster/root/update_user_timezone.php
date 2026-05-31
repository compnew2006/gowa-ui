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
    $timezone = isset($_POST['timezone']) ? sanitizeInput($_POST['timezone']) : '';

    if ($userId <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم غير صحيح']);
        exit;
    }

    if (empty($timezone)) {
        echo json_encode(['success' => false, 'message' => 'يرجى تحديد المنطقة الزمنية']);
        exit;
    }

    // تحديث المنطقة الزمنية
    $query = "UPDATE users SET timezone = ? WHERE id = ?";
    $result = executeQuery($query, [$timezone, $userId]);

    if ($result && $result->rowCount() > 0) {
        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث المنطقة الزمنية بنجاح'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل تحديث المنطقة الزمنية'
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
