<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');

// الاتصال بقاعدة البيانات
require_once '../config/database.php';

// معرف المستخدم الحالي (يمكن جلبه من الجلسة)
$current_user_id = isset($_GET['user_id']) ? (int)$_GET['user_id'] : 1;

try {
    $pdo = getDB();
    
    // جلب البوستات مع معلومات التقييم
    $stmt = $pdo->prepare("
        SELECT 
            p.id,
            p.title,
            p.content,
            p.created_at,
            COUNT(pr.id) as total_ratings
        FROM posts p
        LEFT JOIN post_ratings pr ON p.id = pr.post_id
        GROUP BY p.id, p.user_id, p.user_name, p.user_avatar, p.title, p.content, p.image, p.created_at
        ORDER BY p.created_at DESC
    ");
    
    $stmt->execute();
    $posts = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    // جلب تقييم المستخدم الحالي لكل بوست
    $stmt2 = $pdo->prepare("SELECT post_id, rating FROM post_ratings WHERE user_id = ?");
    $stmt2->execute([$current_user_id]);
    $userRatings = [];
    while ($row = $stmt2->fetch(PDO::FETCH_ASSOC)) {
        $userRatings[$row['post_id']] = $row['rating'];
    }
    
    // إضافة تقييم المستخدم لكل بوست
    foreach ($posts as &$post) {
        $post['user_rating'] = isset($userRatings[$post['id']]) ? $userRatings[$post['id']] : null;
    }
    
    echo json_encode([
        'success' => true,
        'posts' => $posts,
        'total' => count($posts)
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage(),
        'posts' => []
    ], JSON_UNESCAPED_UNICODE);
}
