<?php
/**
 * إنشاء محادثة تجريبية بين مستخدمين
 * 
 * افتح هذا الملف من المتصفح:
 * http://localhost/test/create_test_conversation.php
 */

session_start();
require_once 'config/database.php';

// غير هذه القيم حسب المستخدمين في قاعدة البيانات
$user1_id = "USR9A866D735F9738E2"; // المستخدم الأول
$user2_id = "USR7B53D9E97C0FECEF"; // المستخدم الثاني

try {
    $conn = getDB();
    
    // Check if conversation exists
    $stmt = $conn->prepare("
        SELECT id FROM conversations 
        WHERE (user1_id = :user1 AND user2_id = :user2) 
        OR (user1_id = :user3 AND user2_id = :user4)
    ");
    $stmt->execute([
        ':user1' => $user1_id, 
        ':user2' => $user2_id,
        ':user3' => $user2_id,
        ':user4' => $user1_id
    ]);
    $conversation = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if ($conversation) {
        echo "<div style='font-family: Arial; padding: 20px; background: #f59e0b; color: white; border-radius: 10px; max-width: 600px; margin: 50px auto; text-align: center;'>";
        echo "<h2>⚠️ المحادثة موجودة بالفعل!</h2>";
        echo "<p>يوجد محادثة بين المستخدم {$user1_id} والمستخدم {$user2_id}</p>";
        echo "<p>conversation_id: {$conversation['id']}</p>";
        echo "</div>";
    } else {
        // Create new conversation
        $test_message = "مرحباً! هذه رسالة تجريبية";
        
        $stmt = $conn->prepare("
            INSERT INTO conversations (user1_id, user2_id, last_message, last_message_time)
            VALUES (:user1, :user2, :message, NOW())
        ");
        $stmt->execute([
            ':user1' => min($user1_id, $user2_id),
            ':user2' => max($user1_id, $user2_id),
            ':message' => $test_message
        ]);
        $conversation_id = $conn->lastInsertId();
        
        // Insert test messages
        $messages = [
            ['sender' => $user1_id, 'receiver' => $user2_id, 'text' => 'مرحباً! كيف حالك؟'],
            ['sender' => $user2_id, 'receiver' => $user1_id, 'text' => 'أهلاً! بخير والحمد لله، وأنت؟'],
            ['sender' => $user1_id, 'receiver' => $user2_id, 'text' => 'الحمد لله، كل شيء تمام'],
        ];
        
        $stmt = $conn->prepare("
            INSERT INTO messages (conversation_id, sender_id, receiver_id, message, is_read, created_at)
            VALUES (:conv_id, :sender, :receiver, :message, 0, NOW())
        ");
        
        foreach ($messages as $msg) {
            $stmt->execute([
                ':conv_id' => $conversation_id,
                ':sender' => $msg['sender'],
                ':receiver' => $msg['receiver'],
                ':message' => $msg['text']
            ]);
        }
        
        echo "<div style='font-family: Arial; padding: 20px; background: #10b981; color: white; border-radius: 10px; max-width: 600px; margin: 50px auto; text-align: center;'>";
        echo "<h2>✅ تم إنشاء المحادثة بنجاح!</h2>";
        echo "<p>تم إنشاء محادثة بين المستخدم {$user1_id} والمستخدم {$user2_id}</p>";
        echo "<p>تم إضافة 3 رسائل تجريبية</p>";
        echo "<p>conversation_id: {$conversation_id}</p>";
        echo "<p><a href='messages.php' style='color: white; text-decoration: underline;'>الذهاب إلى الرسائل</a></p>";
        echo "</div>";
    }
    
    // عرض المستخدمين المتاحين
    $stmt = $conn->prepare("SELECT user_id as id, CONCAT(first_name, ' ', last_name) as name FROM users LIMIT 10");
    $stmt->execute();
    $users = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    if ($users) {
        echo "<div style='font-family: Arial; padding: 20px; background: #3b82f6; color: white; border-radius: 10px; max-width: 600px; margin: 20px auto;'>";
        echo "<h3>👥 المستخدمين المتاحين:</h3>";
        echo "<ul style='text-align: right;'>";
        foreach ($users as $user) {
            echo "<li>ID: {$user['id']} - الاسم: {$user['name']}</li>";
        }
        echo "</ul>";
        echo "<p style='font-size: 13px; margin-top: 15px;'>لإنشاء محادثة جديدة، غير قيم <code style='background: rgba(0,0,0,0.2); padding: 2px 5px; border-radius: 3px;'>\$user1_id</code> و <code style='background: rgba(0,0,0,0.2); padding: 2px 5px; border-radius: 3px;'>\$user2_id</code> في الملف</p>";
        echo "</div>";
    }
    
} catch (Exception $e) {
    echo "<div style='font-family: Arial; padding: 20px; background: #ef4444; color: white; border-radius: 10px; max-width: 600px; margin: 50px auto; text-align: center;'>";
    echo "<h2>❌ حدث خطأ!</h2>";
    echo "<p>" . $e->getMessage() . "</p>";
    echo "</div>";
}
?>

<div style="font-family: Arial; padding: 20px; background: #6b7280; color: white; border-radius: 10px; max-width: 600px; margin: 20px auto; text-align: center;">
    <h3>📝 كيفية الاستخدام:</h3>
    <ol style="text-align: right; display: inline-block;">
        <li>افتح الملف <code>create_test_conversation.php</code></li>
        <li>غير قيم <code>$user1_id</code> و <code>$user2_id</code> للمستخدمين المطلوبين</li>
        <li>احفظ الملف وأعد تحميل الصفحة</li>
        <li>اذهب إلى صفحة الرسائل لرؤية المحادثة</li>
    </ol>
</div>
