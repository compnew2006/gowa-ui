<?php
require_once 'config/database.php';

try {
    $pdo = getDB();
    
    $sql = file_get_contents('create_users_table.sql');
    $pdo->exec($sql);
    
    echo "✅ جدول users تم إنشاؤه بنجاح!";
} catch (Exception $e) {
    echo "❌ خطأ: " . $e->getMessage();
}
?>
