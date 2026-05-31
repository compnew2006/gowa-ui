<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json; charset=UTF-8');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id']) || !isset($_SESSION['is_logged_in'])) {
    echo json_encode(['success' => false, 'message' => 'يجب تسجيل الدخول أولاً']);
    exit;
}

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    echo json_encode(['success' => false, 'message' => 'طريقة الطلب غير صحيحة']);
    exit;
}

$post_id = $_POST['post_id'] ?? '';

// التحقق من معرف النشرة
if (empty($post_id)) {
    echo json_encode(['success' => false, 'message' => 'معرف النشرة مطلوب']);
    exit;
}

try {
    // حذف النشرة
    $query = "DELETE FROM posts WHERE id = ?";
    $result = executeQuery($query, [$post_id]);
    
    if ($result) {
        echo json_encode([
            'success' => true,
            'message' => 'تم حذف النشرة بنجاح'
        ]);
    } else {
        echo json_encode(['success' => false, 'message' => 'فشل في حذف النشرة']);
    }
} catch (Exception $e) {
    error_log("Delete Post Error: " . $e->getMessage());
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء حذف النشرة'
    ]);
}
?>
