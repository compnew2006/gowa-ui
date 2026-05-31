<?php
/**
 * تنفيذ SQL لإنشاء جدول الإشعارات
 * 
 * قم بتشغيل هذا الملف مرة واحدة فقط من المتصفح:
 * http://localhost/test/install_notifications.php
 */

require_once 'config/database.php';

try {
    $conn = getDB();
    
    $sql = "CREATE TABLE IF NOT EXISTS `notifications` (
      `id` int(11) NOT NULL AUTO_INCREMENT,
      `user_id` int(11) NOT NULL,
      `title` varchar(255) NOT NULL,
      `message` text NOT NULL,
      `type` varchar(50) DEFAULT 'info',
      `is_read` tinyint(1) DEFAULT 0,
      `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
      PRIMARY KEY (`id`),
      KEY `user_id` (`user_id`),
      KEY `is_read` (`is_read`),
      KEY `created_at` (`created_at`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;";
    
    $conn->exec($sql);
    
    echo "<div style='font-family: Arial; padding: 20px; background: #10b981; color: white; border-radius: 10px; max-width: 600px; margin: 50px auto; text-align: center;'>";
    echo "<h2>✅ تم إنشاء جدول الإشعارات بنجاح!</h2>";
    echo "<p>يمكنك الآن استخدام نظام الإشعارات.</p>";
    echo "<p><a href='tools.php' style='color: white; text-decoration: underline;'>العودة إلى الصفحة الرئيسية</a></p>";
    echo "</div>";
    
    // إضافة إشعارات تجريبية (اختياري)
    $user_id = 1; // غير هذا إلى user_id الخاص بك
    
    $test_notifications = [
        ['title' => 'مرحباً بك!', 'message' => 'تم تفعيل نظام الإشعارات بنجاح', 'type' => 'success'],
        ['title' => 'تحديث جديد', 'message' => 'تم إضافة ميزة الإشعارات الديناميكية', 'type' => 'info'],
        ['title' => 'تنبيه', 'message' => 'هذه إشعارات تجريبية للاختبار', 'type' => 'warning']
    ];
    
    $stmt = $conn->prepare("
        INSERT INTO notifications (user_id, title, message, type, is_read, created_at)
        VALUES (:user_id, :title, :message, :type, 0, NOW())
    ");
    
    foreach ($test_notifications as $notif) {
        $stmt->execute([
            ':user_id' => $user_id,
            ':title' => $notif['title'],
            ':message' => $notif['message'],
            ':type' => $notif['type']
        ]);
    }
    
    echo "<div style='font-family: Arial; padding: 20px; background: #3b82f6; color: white; border-radius: 10px; max-width: 600px; margin: 20px auto; text-align: center;'>";
    echo "<h3>📬 تم إضافة 3 إشعارات تجريبية</h3>";
    echo "<p>تحقق من أيقونة الجرس في الصفحة الرئيسية</p>";
    echo "</div>";
    
} catch (Exception $e) {
    echo "<div style='font-family: Arial; padding: 20px; background: #ef4444; color: white; border-radius: 10px; max-width: 600px; margin: 50px auto; text-align: center;'>";
    echo "<h2>❌ حدث خطأ!</h2>";
    echo "<p>" . $e->getMessage() . "</p>";
    echo "</div>";
}
?>
