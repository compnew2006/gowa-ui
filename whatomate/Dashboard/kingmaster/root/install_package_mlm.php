<?php
require_once 'config/database.php';
$pdo = getDB();

echo "<!DOCTYPE html>
<html lang='ar' dir='rtl'>
<head>
    <meta charset='UTF-8'>
    <title>تثبيت إعدادات عمولات الباقات</title>
    <style>
        body { font-family: Arial; padding: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; }
        h1 { color: #667eea; }
        .success { color: green; padding: 10px; background: #d4edda; margin: 10px 0; border-radius: 5px; }
        .error { color: red; padding: 10px; background: #f8d7da; margin: 10px 0; border-radius: 5px; }
        .info { color: blue; padding: 10px; background: #d1ecf1; margin: 10px 0; border-radius: 5px; }
    </style>
</head>
<body>
<div class='container'>
<h1>تثبيت جدول إعدادات عمولات الباقات</h1>";

try {
    echo "<div class='info'>جاري إنشاء جدول package_mlm_settings...</div>";
    
    $pdo->exec("
        CREATE TABLE IF NOT EXISTS `package_mlm_settings` (
          `id` INT AUTO_INCREMENT PRIMARY KEY,
          `package_id` INT NOT NULL,
          `max_levels` INT NOT NULL DEFAULT 5,
          `direct_commission_amount` DECIMAL(10,2) NOT NULL DEFAULT 0.00,
          `level_1_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_2_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_3_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_4_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_5_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_6_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_7_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_8_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_9_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `level_10_percentage` DECIMAL(5,2) NOT NULL DEFAULT 0.00,
          `is_active` TINYINT(1) DEFAULT 1,
          `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
          `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
          UNIQUE KEY `unique_package` (`package_id`)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
    ");
    
    echo "<div class='success'>✓ تم إنشاء الجدول بنجاح</div>";
    
    echo "<div class='info'>جاري إضافة إعدادات افتراضية للباقات...</div>";
    
    $stmt = $pdo->query("SELECT id, name_ar, monthly_price FROM packages WHERE is_active = 1");
    $packages = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    foreach ($packages as $package) {
        $insertStmt = $pdo->prepare("
            INSERT IGNORE INTO package_mlm_settings 
            (package_id, max_levels, direct_commission_amount, level_1_percentage, level_2_percentage, level_3_percentage, level_4_percentage, level_5_percentage)
            VALUES (?, 5, 50.00, 5.00, 3.00, 2.00, 1.00, 1.00)
        ");
        $insertStmt->execute([$package['id']]);
        
        echo "<div class='success'>✓ تمت إضافة إعدادات الباقة: {$package['name_ar']}</div>";
    }
    
    echo "<div class='success' style='font-size: 18px; margin-top: 20px;'><strong>✓✓✓ تم التثبيت بنجاح!</strong></div>";
    echo "<p><a href='package_mlm_settings.html'>انتقل إلى صفحة الإعدادات</a></p>";
    
} catch (PDOException $e) {
    echo "<div class='error'>✗ خطأ: " . $e->getMessage() . "</div>";
}

echo "</div></body></html>";
?>
