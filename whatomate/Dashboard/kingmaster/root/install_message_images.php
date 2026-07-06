<?php
require_once 'config/database.php';

try {
    $conn = getDB();
    
    // التحقق من وجود الحقل مسبقاً
    $stmt = $conn->query("SHOW COLUMNS FROM messages LIKE 'image_path'");
    $exists = $stmt->fetch();
    
    if (!$exists) {
        // إضافة حقل image_path
        $conn->exec("ALTER TABLE `messages` ADD COLUMN `image_path` VARCHAR(500) DEFAULT NULL AFTER `message`");
        
        // إضافة index للبحث السريع
        $conn->exec("ALTER TABLE `messages` ADD INDEX `idx_image_path` (`image_path`)");
        
        echo "<div style='padding: 20px; background: #10b981; color: white; text-align: center; font-family: Cairo, sans-serif; font-size: 18px; border-radius: 10px; margin: 20px;'>";
        echo "<i class='fas fa-check-circle' style='font-size: 50px; margin-bottom: 10px;'></i><br>";
        echo "تم إضافة حقل الصور لجدول الرسائل بنجاح! ✅<br><br>";
        echo "يمكنك الآن رفع ومشاركة الصور في المحادثات.";
        echo "</div>";
    } else {
        echo "<div style='padding: 20px; background: #f59e0b; color: white; text-align: center; font-family: Cairo, sans-serif; font-size: 18px; border-radius: 10px; margin: 20px;'>";
        echo "<i class='fas fa-info-circle' style='font-size: 50px; margin-bottom: 10px;'></i><br>";
        echo "حقل الصور موجود مسبقاً في جدول الرسائل. ⚠️";
        echo "</div>";
    }
    
    echo "<div style='text-align: center; margin-top: 20px;'>";
    echo "<a href='messages.php' style='padding: 12px 30px; background: linear-gradient(135deg, #667eea, #764ba2); color: white; text-decoration: none; border-radius: 10px; font-family: Cairo, sans-serif; font-weight: 700;'>";
    echo "<i class='fas fa-comments'></i> الذهاب إلى صفحة المحادثات";
    echo "</a>";
    echo "</div>";
    
} catch (PDOException $e) {
    echo "<div style='padding: 20px; background: #ef4444; color: white; text-align: center; font-family: Cairo, sans-serif; font-size: 18px; border-radius: 10px; margin: 20px;'>";
    echo "<i class='fas fa-times-circle' style='font-size: 50px; margin-bottom: 10px;'></i><br>";
    echo "خطأ في إضافة الحقل: " . $e->getMessage();
    echo "</div>";
}
?>

<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>تثبيت ميزة الصور في الرسائل</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@400;600;700;800&display=swap" rel="stylesheet">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Cairo', sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
    </style>
</head>
<body>
</body>
</html>
