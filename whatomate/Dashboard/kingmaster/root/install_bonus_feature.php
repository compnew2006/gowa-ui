<?php
/**
 * تثبيت ميزة البونص 50% لتارجت المبيعات
 */

require_once 'config/database.php';

header('Content-Type: text/html; charset=utf-8');

echo "<!DOCTYPE html>
<html lang='ar' dir='rtl'>
<head>
    <meta charset='UTF-8'>
    <meta name='viewport' content='width=device-width, initial-scale=1.0'>
    <title>تثبيت ميزة البونص</title>
    <style>
        body {
            font-family: 'Cairo', sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 2rem;
            color: #fff;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 16px;
            padding: 2rem;
            backdrop-filter: blur(10px);
        }
        h1 { text-align: center; margin-bottom: 2rem; }
        .status {
            padding: 1rem;
            margin: 1rem 0;
            border-radius: 8px;
            background: rgba(255, 255, 255, 0.2);
        }
        .success { background: rgba(16, 185, 129, 0.3); border-left: 4px solid #10b981; }
        .error { background: rgba(239, 68, 68, 0.3); border-left: 4px solid #ef4444; }
        .info { background: rgba(59, 130, 246, 0.3); border-left: 4px solid #3b82f6; }
        pre {
            background: rgba(0, 0, 0, 0.3);
            padding: 1rem;
            border-radius: 8px;
            overflow-x: auto;
        }
    </style>
</head>
<body>
<div class='container'>
    <h1>🎉 تثبيت ميزة البونص 50%</h1>
";

try {
    $db = getDB();
    
    echo "<div class='status info'>⏳ جاري التحقق من قاعدة البيانات...</div>";
    
    // التحقق من وجود جدول sales_target
    $stmt = $db->query("SHOW TABLES LIKE 'sales_target'");
    if ($stmt->rowCount() == 0) {
        echo "<div class='status error'>❌ جدول sales_target غير موجود. يرجى تشغيل database/sales_target.sql أولاً</div>";
        echo "</div></body></html>";
        exit;
    }
    
    echo "<div class='status success'>✅ جدول sales_target موجود</div>";
    
    // التحقق من وجود حقل bonus_amount
    $stmt = $db->query("SHOW COLUMNS FROM sales_target LIKE 'bonus_amount'");
    
    if ($stmt->rowCount() == 0) {
        echo "<div class='status info'>➕ إضافة حقل bonus_amount...</div>";
        
        $sql = "ALTER TABLE `sales_target` 
                ADD COLUMN `bonus_amount` decimal(10,2) NOT NULL DEFAULT 0.00 
                COMMENT 'المبلغ بعد تحقيق التارجت (50% بونص)' 
                AFTER `current_amount`";
        
        $db->exec($sql);
        
        echo "<div class='status success'>✅ تم إضافة حقل bonus_amount بنجاح</div>";
    } else {
        echo "<div class='status info'>ℹ️ حقل bonus_amount موجود مسبقاً</div>";
    }
    
    // عرض بنية الجدول
    echo "<div class='status info'>
            <strong>📋 بنية جدول sales_target:</strong>
            <pre>";
    
    $stmt = $db->query("DESCRIBE sales_target");
    $columns = $stmt->fetchAll();
    
    foreach ($columns as $col) {
        echo sprintf("%-20s %-20s %s\n", 
            $col['Field'], 
            $col['Type'], 
            $col['Comment'] ?: ''
        );
    }
    
    echo "</pre></div>";
    
    // عرض البيانات الحالية
    $stmt = $db->query("SELECT * FROM sales_target ORDER BY target_month DESC LIMIT 5");
    $targets = $stmt->fetchAll();
    
    if (count($targets) > 0) {
        echo "<div class='status info'>
                <strong>📊 آخر 5 تارجتات:</strong>
                <pre>";
        
        foreach ($targets as $target) {
            echo sprintf(
                "الشهر: %s | التارجت: %.2f | المبلغ الحالي: %.2f | البونص: %.2f\n",
                $target['target_month'],
                $target['target_amount'],
                $target['current_amount'],
                $target['bonus_amount']
            );
        }
        
        echo "</pre></div>";
    }
    
    echo "<div class='status success'>
            <h2>🎊 تم التثبيت بنجاح!</h2>
            <p><strong>الميزات الجديدة:</strong></p>
            <ul>
                <li>✅ عند تحقيق التارجت الشهري → أي مبيعات جديدة تحسب كبونص</li>
                <li>✅ البونص = 50% من المبلغ الزائد بعد التارجت</li>
                <li>✅ في بداية كل شهر جديد → التارجت يبدأ من الصفر</li>
                <li>✅ عرض البونص المكتسب في الواجهة</li>
            </ul>
            <p><a href='sales_target.html' style='color: #60a5fa; text-decoration: underline;'>🔗 انتقل إلى صفحة التارجت</a></p>
          </div>";
    
} catch (PDOException $e) {
    echo "<div class='status error'>
            <strong>❌ خطأ في قاعدة البيانات:</strong>
            <pre>" . htmlspecialchars($e->getMessage()) . "</pre>
          </div>";
}

echo "</div></body></html>";
?>
