<?php
session_start();
header('Content-Type: application/json');

require_once '../config/database.php';

$user_id = $_SESSION['user_id'] ?? 1;

try {
    $id = $_GET['id'] ?? null;
    
    if (empty($id)) {
        echo json_encode([
            'success' => false,
            'message' => 'معرف الإعدادات مطلوب'
        ]);
        exit;
    }
    
    $conn = getDB();
    
    // جلب الإعدادات فقط إذا كانت تخص المستخدم الحالي
    $sql = "SELECT * FROM sending_settings WHERE id = :id AND user_id = :user_id";
    $stmt = $conn->prepare($sql);
    $stmt->bindParam(':id', $id, PDO::PARAM_INT);
    $stmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
    $stmt->execute();
    
    $setting = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if ($setting) {
        echo json_encode([
            'success' => true,
            'setting' => $setting
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'لم يتم العثور على الإعدادات'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
