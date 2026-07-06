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
    // الحصول على البيانات من الطلب
    $input = json_decode(file_get_contents('php://input'), true);
    
    if (empty($input['id'])) {
        echo json_encode([
            'success' => false,
            'message' => 'معرف الإعدادات مطلوب'
        ]);
        exit;
    }
    
    // حذف الإعدادات فقط إذا كانت تخص المستخدم الحالي
    $sql = "DELETE FROM sending_settings WHERE id = :id AND user_id = :user_id";
    $stmt = $conn->prepare($sql);
    $stmt->bindParam(':id', $input['id'], PDO::PARAM_INT);
    $stmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
    
    if ($stmt->execute()) {
        if ($stmt->rowCount() > 0) {
            echo json_encode([
                'success' => true,
                'message' => 'تم حذف الإعدادات بنجاح'
            ]);
        } else {
            echo json_encode([
                'success' => false,
                'message' => 'لم يتم العثور على الإعدادات'
            ]);
        }
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل في حذف الإعدادات'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
