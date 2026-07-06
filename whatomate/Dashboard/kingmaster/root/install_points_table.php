<?php
/**
 * سكريبت إنشاء جدول points_packages
 * قم بتشغيل هذا الملف مرة واحدة لإنشاء الجدول
 */

require_once 'config/database.php';

try {
    $conn = getDB();
    
    // إنشاء الجدول
    $sql = "CREATE TABLE IF NOT EXISTS points_packages (
        id INT AUTO_INCREMENT PRIMARY KEY,
        points_count INT NOT NULL,
        price DECIMAL(10, 2) NOT NULL,
        is_active TINYINT(1) DEFAULT 1,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        INDEX idx_is_active (is_active)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci";
    
    $conn->exec($sql);
    echo "✅ تم إنشاء جدول points_packages بنجاح!<br>";
    
    // إدراج بيانات تجريبية
    $insertSql = "INSERT INTO points_packages (points_count, price, is_active) VALUES
        (100, 10.00, 1),
        (500, 45.00, 1),
        (1000, 85.00, 1),
        (2500, 200.00, 1),
        (5000, 375.00, 1),
        (10000, 700.00, 1)";
    
    $conn->exec($insertSql);
    echo "✅ تم إضافة الباقات التجريبية بنجاح!<br>";
    
    echo "🎉 يمكنك الآن استخدام صفحة النقاط<br>";
    echo "<a href='points.php' style='display: inline-block; margin-top: 20px; padding: 10px 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; text-decoration: none; border-radius: 8px; font-family: Arial;'>الانتقال إلى صفحة النقاط</a>";
    
} catch (PDOException $e) {
    echo "❌ خطأ: " . $e->getMessage();
}
?>
