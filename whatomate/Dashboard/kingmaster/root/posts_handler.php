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

$title = sanitizeInput($_POST['title'] ?? '');
$content = sanitizeInput($_POST['description'] ?? '');
$typs = sanitizeInput($_POST['type'] ?? ''); // type from form -> typs in DB
$post_id = $_POST['post_id'] ?? '';

// التحقق من الحقول المطلوبة
if (empty($title) || empty($content) || empty($typs)) {
    echo json_encode(['success' => false, 'message' => 'جميع الحقول مطلوبة']);
    exit;
}

try {
    if (empty($post_id)) {
        // إضافة نشرة جديدة
        $query = "INSERT INTO posts (title, content, typs) VALUES (?, ?, ?)";
        $result = executeQuery($query, [$title, $content, $typs]);
        
        if ($result) {
            echo json_encode([
                'success' => true,
                'message' => 'تم إضافة النشرة بنجاح'
            ]);
        } else {
            echo json_encode(['success' => false, 'message' => 'فشل في إضافة النشرة']);
        }
    } else {
        // تحديث نشرة موجودة
        $query = "UPDATE posts SET title = ?, content = ?, typs = ? WHERE id = ?";
        $result = executeQuery($query, [$title, $content, $typs, $post_id]);
        
        if ($result) {
            echo json_encode([
                'success' => true,
                'message' => 'تم تحديث النشرة بنجاح'
            ]);
        } else {
            echo json_encode(['success' => false, 'message' => 'فشل في تحديث النشرة']);
        }
    }
} catch (Exception $e) {
    error_log("Posts Handler Error: " . $e->getMessage());
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء معالجة الطلب'
    ]);
}
?>
