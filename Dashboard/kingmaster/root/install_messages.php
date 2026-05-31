<?php
/**
 * تنفيذ SQL لإنشاء جداول الرسائل
 * 
 * قم بتشغيل هذا الملف مرة واحدة فقط من المتصفح:
 * http://localhost/test/install_messages.php
 */

require_once 'config/database.php';

try {
    $conn = getDB();
    
    // Create conversations table
    $sql1 = "CREATE TABLE IF NOT EXISTS `conversations` (
      `id` int(11) NOT NULL AUTO_INCREMENT,
      `user1_id` int(11) NOT NULL,
      `user2_id` int(11) NOT NULL,
      `last_message` text DEFAULT NULL,
      `last_message_time` timestamp NULL DEFAULT NULL,
      `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
      PRIMARY KEY (`id`),
      UNIQUE KEY `unique_conversation` (`user1_id`, `user2_id`),
      KEY `user1_id` (`user1_id`),
      KEY `user2_id` (`user2_id`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;";
    
    $conn->exec($sql1);
    
    // Create messages table
    $sql2 = "CREATE TABLE IF NOT EXISTS `messages` (
      `id` int(11) NOT NULL AUTO_INCREMENT,
      `conversation_id` int(11) NOT NULL,
      `sender_id` int(11) NOT NULL,
      `receiver_id` int(11) NOT NULL,
      `message` text NOT NULL,
      `is_read` tinyint(1) DEFAULT 0,
      `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
      PRIMARY KEY (`id`),
      KEY `conversation_id` (`conversation_id`),
      KEY `sender_id` (`sender_id`),
      KEY `receiver_id` (`receiver_id`),
      KEY `is_read` (`is_read`),
      KEY `created_at` (`created_at`),
      FOREIGN KEY (`conversation_id`) REFERENCES `conversations` (`id`) ON DELETE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;";
    
    $conn->exec($sql2);
    
    echo "<div style='font-family: Arial; padding: 20px; background: #10b981; color: white; border-radius: 10px; max-width: 600px; margin: 50px auto; text-align: center;'>";
    echo "<h2>✅ تم إنشاء جداول الرسائل بنجاح!</h2>";
    echo "<p>تم إنشاء:</p>";
    echo "<ul style='text-align: right; display: inline-block;'>";
    echo "<li>جدول المحادثات (conversations)</li>";
    echo "<li>جدول الرسائل (messages)</li>";
    echo "</ul>";
    echo "<p><a href='messages.php' style='color: white; text-decoration: underline;'>الذهاب إلى الرسائل</a></p>";
    echo "</div>";
    
    // إضافة رسائل تجريبية (اختياري)
    echo "<div style='font-family: Arial; padding: 20px; background: #3b82f6; color: white; border-radius: 10px; max-width: 600px; margin: 20px auto; text-align: center;'>";
    echo "<h3>📝 ملاحظة</h3>";
    echo "<p>لإنشاء محادثة تجريبية، يمكنك:</p>";
    echo "<ul style='text-align: right; display: inline-block;'>";
    echo "<li>الذهاب إلى صفحة الرسائل</li>";
    echo "<li>اختيار مستخدم من قائمة المستخدمين</li>";
    echo "<li>إرسال رسالة جديدة</li>";
    echo "</ul>";
    echo "</div>";
    
} catch (Exception $e) {
    echo "<div style='font-family: Arial; padding: 20px; background: #ef4444; color: white; border-radius: 10px; max-width: 600px; margin: 50px auto; text-align: center;'>";
    echo "<h2>❌ حدث خطأ!</h2>";
    echo "<p>" . $e->getMessage() . "</p>";
    echo "</div>";
}
?>
