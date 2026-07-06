<?php
/**
 * سكريبت إنشاء جدول sending_settings
 * قم بتشغيل هذا الملف مرة واحدة لإنشاء الجدول
 */

require_once 'config/database.php';

try {
    $conn = getDB();
    
    $sql = "CREATE TABLE IF NOT EXISTS sending_settings (
        id INT AUTO_INCREMENT PRIMARY KEY,
        user_id INT NOT NULL,
        settings_name VARCHAR(255) NOT NULL,
        platform ENUM('facebook', 'whatsapp', 'instagram', 'telegram', 'email') NOT NULL,
        interval_from INT NOT NULL,
        interval_to INT NOT NULL,
        protection_enabled TINYINT(1) DEFAULT 0,
        msg_count INT DEFAULT NULL,
        protection_interval_from INT DEFAULT NULL,
        protection_interval_to INT DEFAULT NULL,
        blacklist_enabled TINYINT(1) DEFAULT 0,
        blacklist JSON DEFAULT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        INDEX idx_user_id (user_id),
        INDEX idx_platform (platform)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci";
    
    $conn->exec($sql);
    
    echo "✅ تم إنشاء جدول sending_settings بنجاح!<br>";
    echo "🎉 يمكنك الآن استخدام صفحة إعدادات الإرسال<br>";
    echo "<a href='sending-settings.php' style='display: inline-block; margin-top: 20px; padding: 10px 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; text-decoration: none; border-radius: 8px; font-family: Arial;'>الانتقال إلى إعدادات الإرسال</a>";
    
} catch (PDOException $e) {
    echo "❌ خطأ في إنشاء الجدول: " . $e->getMessage();
}
?>
