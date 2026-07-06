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
    
    // التحقق من البيانات المطلوبة
    if (empty($input['platform']) || empty($input['settingsName']) || 
        empty($input['intervalFrom']) || empty($input['intervalTo'])) {
        echo json_encode([
            'success' => false,
            'message' => 'جميع الحقول الأساسية مطلوبة'
        ]);
        exit;
    }
    
    // إعداد الاستعلام
    $sql = "INSERT INTO sending_settings (
        user_id, 
        settings_name, 
        platform, 
        interval_from, 
        interval_to, 
        protection_enabled, 
        msg_count, 
        protection_interval_from, 
        protection_interval_to, 
        blacklist_enabled, 
        blacklist
    ) VALUES (
        :user_id, 
        :settings_name, 
        :platform, 
        :interval_from, 
        :interval_to, 
        :protection_enabled, 
        :msg_count, 
        :protection_interval_from, 
        :protection_interval_to, 
        :blacklist_enabled, 
        :blacklist
    )";
    
    $stmt = $conn->prepare($sql);
    
    // ربط المتغيرات
    $stmt->bindParam(':user_id', $user_id, PDO::PARAM_INT);
    $stmt->bindParam(':settings_name', $input['settingsName'], PDO::PARAM_STR);
    $stmt->bindParam(':platform', $input['platform'], PDO::PARAM_STR);
    $stmt->bindParam(':interval_from', $input['intervalFrom'], PDO::PARAM_INT);
    $stmt->bindParam(':interval_to', $input['intervalTo'], PDO::PARAM_INT);
    $stmt->bindParam(':protection_enabled', $input['protectionEnabled'], PDO::PARAM_INT);
    $stmt->bindParam(':msg_count', $input['msgCount'], PDO::PARAM_INT);
    $stmt->bindParam(':protection_interval_from', $input['protectionIntervalFrom'], PDO::PARAM_INT);
    $stmt->bindParam(':protection_interval_to', $input['protectionIntervalTo'], PDO::PARAM_INT);
    $stmt->bindParam(':blacklist_enabled', $input['blacklistEnabled'], PDO::PARAM_INT);
    $stmt->bindParam(':blacklist', $input['blacklist'], PDO::PARAM_STR);
    
    // تنفيذ الاستعلام
    if ($stmt->execute()) {
        echo json_encode([
            'success' => true,
            'message' => 'تم حفظ الإعدادات بنجاح',
            'id' => $conn->lastInsertId()
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل في حفظ الإعدادات'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
