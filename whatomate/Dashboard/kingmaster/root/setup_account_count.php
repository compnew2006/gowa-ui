<?php
// هذا الملف يُستخدم مرة واحدة فقط لإضافة حقل account_count وتحديث البيانات

require_once 'config/database.php';

try {
    $conn = getDB();
    
    echo "<h2>تحديث قاعدة البيانات - إضافة حقل account_count</h2>";
    
    // 1. إضافة الحقل إذا لم يكن موجوداً
    echo "<p>1. إضافة حقل account_count...</p>";
    $sql = "ALTER TABLE `users` 
            ADD COLUMN IF NOT EXISTS `account_count` INT(11) NOT NULL DEFAULT 0 
            AFTER `package`";
    
    try {
        $conn->exec($sql);
        echo "<p style='color: green;'>✓ تم إضافة الحقل بنجاح</p>";
    } catch (PDOException $e) {
        if (strpos($e->getMessage(), 'Duplicate column') !== false) {
            echo "<p style='color: orange;'>⚠ الحقل موجود بالفعل</p>";
        } else {
            throw $e;
        }
    }
    
    // 2. تحديث عداد الحسابات الحالي
    echo "<p>2. تحديث عداد الحسابات الحالي...</p>";
    $sql = "UPDATE `users` u
            SET u.account_count = (
                SELECT COUNT(*) 
                FROM `accounts` a 
                WHERE a.user_id = u.user_id
            )";
    
    $stmt = $conn->prepare($sql);
    $stmt->execute();
    $affected = $stmt->rowCount();
    
    echo "<p style='color: green;'>✓ تم تحديث {$affected} مستخدم</p>";
    
    // 3. عرض النتيجة
    echo "<h3>نتيجة التحديث:</h3>";
    echo "<table border='1' cellpadding='10' style='border-collapse: collapse;'>";
    echo "<tr><th>User ID</th><th>Package</th><th>Account Count</th></tr>";
    
    $stmt = $conn->query("SELECT user_id, package, account_count FROM users");
    while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
        echo "<tr>";
        echo "<td>{$row['user_id']}</td>";
        echo "<td>{$row['package']}</td>";
        echo "<td>{$row['account_count']}</td>";
        echo "</tr>";
    }
    echo "</table>";
    
    echo "<hr>";
    echo "<p style='color: green; font-weight: bold;'>✓ تم التحديث بنجاح! يمكنك حذف هذا الملف الآن.</p>";
    
} catch (PDOException $e) {
    echo "<p style='color: red;'>خطأ: " . $e->getMessage() . "</p>";
}
?>
