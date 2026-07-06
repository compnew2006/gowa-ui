<?php
session_start();
header('Content-Type: application/json');

// الاتصال بقاعدة البيانات
require_once '../config/database.php';

// الحصول على الاتصال بقاعدة البيانات
$conn = getDB();

// تحقق من تسجيل دخول المستخدم
$user_id = $_SESSION['user_id'] ?? 1; // للتجربة نستخدم 1، في الإنتاج استخدم الجلسة الحقيقية

try {
    // جلب جميع الإعدادات الخاصة بالمستخدم
    $sql = "SELECT * FROM sending_settings WHERE user_id = :user_id ORDER BY created_at DESC";
    $stmt = $conn->prepare($sql);
    $stmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
    $stmt->execute();
    
    $settings = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo json_encode([
        'success' => true,
        'settings' => $settings
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
