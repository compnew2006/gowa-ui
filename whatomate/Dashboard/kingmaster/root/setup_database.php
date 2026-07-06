<?php
/**
 * إعداد قاعدة البيانات - إنشاء جدول المستخدمين
 */

require_once 'config/database.php';

echo "<!DOCTYPE html>
<html lang='ar' dir='rtl'>
<head>
    <meta charset='UTF-8'>
    <meta name='viewport' content='width=device-width, initial-scale=1.0'>
    <title>إعداد قاعدة البيانات</title>
    <link href='https://fonts.googleapis.com/css2?family=Cairo:wght@400;600;700;800;900&display=swap' rel='stylesheet'>
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
        .setup-container {
            background: white;
            border-radius: 20px;
            padding: 40px;
            max-width: 600px;
            width: 100%;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
        }
        h1 {
            color: #667eea;
            margin-bottom: 20px;
            font-size: 28px;
            text-align: center;
        }
        .message {
            padding: 15px 20px;
            border-radius: 10px;
            margin-bottom: 15px;
            font-weight: 600;
        }
        .success {
            background: #d1fae5;
            color: #065f46;
            border: 2px solid #10b981;
        }
        .error {
            background: #fee2e2;
            color: #991b1b;
            border: 2px solid #ef4444;
        }
        .info {
            background: #dbeafe;
            color: #1e40af;
            border: 2px solid #3b82f6;
        }
        .btn {
            display: inline-block;
            padding: 12px 30px;
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            text-decoration: none;
            border-radius: 10px;
            font-weight: 700;
            margin-top: 20px;
            text-align: center;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 25px rgba(102, 126, 234, 0.4);
        }
    </style>
</head>
<body>
    <div class='setup-container'>
        <h1>🔧 إعداد قاعدة البيانات</h1>";

try {
    $db = getDB();
    
    // حذف الجدول إذا كان موجودًا لتجنب تضارب المفاتيح
    $db->exec("SET FOREIGN_KEY_CHECKS=0");
    $db->exec("DROP TABLE IF EXISTS `users`");
    $db->exec("SET FOREIGN_KEY_CHECKS=1");
    
    // إنشاء جدول users بدون أي Foreign Keys
    $sql = "CREATE TABLE `users` (
      `id` INT(11) NOT NULL AUTO_INCREMENT,
      `user_id` VARCHAR(50) NOT NULL,
      `first_name` VARCHAR(100) NOT NULL,
      `last_name` VARCHAR(100) NOT NULL,
      `email` VARCHAR(255) NOT NULL,
      `password` VARCHAR(255) NOT NULL,
      `phone` VARCHAR(20) NOT NULL,
      `timezone` VARCHAR(100) DEFAULT 'UTC',
      `job` VARCHAR(150) DEFAULT NULL,
      `birth_date` DATE DEFAULT NULL,
      `otp` VARCHAR(6) DEFAULT NULL,
      `otp_created_at` DATETIME DEFAULT NULL,
      `is_verified` TINYINT(1) DEFAULT 0,
      `is_admin` TINYINT(1) DEFAULT 0,
      `package` INT(11) DEFAULT 0,
      `points` INT(11) DEFAULT 100,
      `expiry_date` DATETIME DEFAULT (DATE_ADD(NOW(), INTERVAL 48 HOUR)),
      `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (`id`),
      UNIQUE KEY `unique_user_id` (`user_id`),
      UNIQUE KEY `unique_email` (`email`),
      UNIQUE KEY `unique_phone` (`phone`),
      KEY `idx_is_verified` (`is_verified`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci";
    
    // تنفيذ الاستعلام
    $db->exec($sql);
    
    echo "<div class='message success'>
            ✅ تم إنشاء جدول users بنجاح!
          </div>";
    
    // إنشاء جدول users_wallet
    $walletsSql = "CREATE TABLE IF NOT EXISTS `users_wallet` (
      `id` INT(11) NOT NULL AUTO_INCREMENT,
      `user_id` VARCHAR(50) NOT NULL,
      `balance` DECIMAL(10,2) DEFAULT 0.00,
      `points` INT(11) DEFAULT 100,
      `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (`id`),
      UNIQUE KEY `unique_user_id` (`user_id`),
      KEY `idx_balance` (`balance`),
      KEY `idx_points` (`points`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci";
    
    $db->exec($walletsSql);
    
    echo "<div class='message success'>
            ✅ تم إنشاء جدول users_wallet بنجاح!
          </div>";
    
    echo "<div class='message info'>
            ℹ️ <strong>جدول users</strong> يحتوي على:<br><br>
            • <strong>id</strong> - المعرف التلقائي<br>
            • <strong>user_id</strong> - معرف المستخدم الفريد<br>
            • <strong>first_name, last_name</strong> - الاسم<br>
            • <strong>email</strong> - البريد الإلكتروني<br>
            • <strong>password</strong> - كلمة المرور المشفرة<br>
            • <strong>phone</strong> - رقم الهاتف<br>
            • <strong>timezone</strong> - المنطقة الزمنية<br>
            • <strong>job</strong> - العمل<br>
            • <strong>birth_date</strong> - تاريخ الميلاد<br>
            • <strong>otp, otp_created_at</strong> - رمز التحقق<br>
            • <strong>is_verified</strong> - حالة التحقق<br>
            • <strong>is_admin</strong> - صلاحيات الأدمن<br>
            • <strong>package</strong> - رقم الباقة (افتراضي: 0)<br>
            • <strong>points</strong> - النقاط (افتراضي: 100)<br>
            • <strong>expiry_date</strong> - تاريخ الانتهاء (48 ساعة من التسجيل)<br>
            • <strong>created_at, updated_at</strong> - التواريخ<br><br>
            <strong>جدول users_wallet</strong> يحتوي على:<br><br>
            • <strong>id</strong> - المعرف التلقائي<br>
            • <strong>user_id</strong> - معرف المستخدم (يربط مع users.user_id)<br>
            • <strong>balance</strong> - رصيد الأموال (افتراضي: 0.00)<br>
            • <strong>points</strong> - رصيد النقاط (افتراضي: 100)<br>
            • <strong>created_at, updated_at</strong> - التواريخ
          </div>";
    
    echo "<a href='register.php' class='btn'>الانتقال لصفحة التسجيل</a>";
    
} catch (PDOException $e) {
    echo "<div class='message error'>
            ❌ خطأ في إنشاء الجدول: " . htmlspecialchars($e->getMessage()) . "
          </div>";
    
    echo "<div class='message info'>
            تأكد من:<br>
            • أن XAMPP يعمل بشكل صحيح<br>
            • أن قاعدة البيانات <strong>kingmaster_packages</strong> موجودة<br>
            • أن كلمة المرور في config/database.php صحيحة
          </div>";
}

echo "    </div>
</body>
</html>";
?>
