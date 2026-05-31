<?php
session_start();
header('Content-Type: application/json');

require_once '../config/database.php';

$user_id = $_SESSION['user_id'] ?? 1;

try {
    $input = json_decode(file_get_contents('php://input'), true);
    
    // التحقق من البيانات المطلوبة
    if (empty($input['id']) || empty($input['platform']) || empty($input['settingsName']) || 
        empty($input['intervalFrom']) || empty($input['intervalTo'])) {
        echo json_encode([
            'success' => false,
            'message' => 'جميع الحقول الأساسية مطلوبة'
        ]);
        exit;
    }
    
    $conn = getDB();
    
    // تحديث الإعدادات فقط إذا كانت تخص المستخدم الحالي
    $sql = "UPDATE sending_settings SET 
        settings_name = :settings_name, 
        platform = :platform, 
        interval_from = :interval_from, 
        interval_to = :interval_to, 
        protection_enabled = :protection_enabled, 
        msg_count = :msg_count, 
        protection_interval_from = :protection_interval_from, 
        protection_interval_to = :protection_interval_to, 
        blacklist_enabled = :blacklist_enabled, 
        blacklist = :blacklist
        WHERE id = :id AND user_id = :user_id";
    
    $stmt = $conn->prepare($sql);
    
    // ربط المتغيرات
    $stmt->bindParam(':id', $input['id'], PDO::PARAM_INT);
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
    
    if ($stmt->execute()) {
        if ($stmt->rowCount() > 0) {
            echo json_encode([
                'success' => true,
                'message' => 'تم تحديث الإعدادات بنجاح'
            ]);
        } else {
            echo json_encode([
                'success' => false,
                'message' => 'لم يتم العثور على الإعدادات أو لم يتم تغيير أي بيانات'
            ]);
        }
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل في تحديث الإعدادات'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
