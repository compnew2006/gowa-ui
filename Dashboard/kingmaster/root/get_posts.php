<?php
session_start();
require_once 'config/database.php';

header('Content-Type: application/json; charset=UTF-8');

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id']) || !isset($_SESSION['is_logged_in'])) {
    echo json_encode(['success' => false, 'message' => 'يجب تسجيل الدخول أولاً']);
    exit;
}

try {
    // جلب جميع النشرات
    $query = "SELECT id, title, content, typs as type, created_at, updated_at 
              FROM posts 
              ORDER BY created_at DESC";
    
    $posts = fetchAll($query);
    
    if ($posts !== false) {
        echo json_encode([
            'success' => true,
            'posts' => $posts
        ]);
    } else {
        echo json_encode([
            'success' => true,
            'posts' => []
        ]);
    }
} catch (Exception $e) {
    error_log("Get Posts Error: " . $e->getMessage());
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ أثناء جلب النشرات'
    ]);
}
?>
