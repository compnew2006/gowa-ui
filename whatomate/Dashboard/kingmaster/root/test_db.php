<?php
header('Content-Type: application/json');

// محاولة الاتصال بقاعدة البيانات
try {
    $pdo = new PDO('mysql:host=localhost;charset=utf8mb4', 'root', 'your_new_password');
    $pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    
    // إنشاء قاعدة البيانات إذا لم تكن موجودة
    $pdo->exec("CREATE DATABASE IF NOT EXISTS kingmaster_packages");
    $pdo->exec("USE kingmaster_packages");
    
    // التحقق من وجود جدول packages
    $stmt = $pdo->query("SHOW TABLES LIKE 'packages'");
    $tableExists = $stmt->rowCount() > 0;
    
    if (!$tableExists) {
        // إنشاء جدول packages
        $createTable = "
            CREATE TABLE packages (
                id INT AUTO_INCREMENT PRIMARY KEY,
                name_ar VARCHAR(255) NOT NULL,
                name_en VARCHAR(255) NOT NULL,
                name_fr VARCHAR(255) NOT NULL,
                badge_ar VARCHAR(100),
                badge_en VARCHAR(100),
                badge_fr VARCHAR(100),
                monthly_price DECIMAL(10,2) NOT NULL,
                yearly_price DECIMAL(10,2) NOT NULL,
                currency VARCHAR(10) DEFAULT 'ريال',
                is_popular BOOLEAN DEFAULT FALSE,
                is_active BOOLEAN DEFAULT TRUE,
                has_discount BOOLEAN DEFAULT FALSE,
                monthly_discount DECIMAL(10,2) DEFAULT 0,
                yearly_discount DECIMAL(10,2) DEFAULT 0,
                monthly_discount_percentage DECIMAL(5,2) DEFAULT 0,
                yearly_discount_percentage DECIMAL(5,2) DEFAULT 0,
                points INT DEFAULT 0,
                messages INT DEFAULT 0,
                accounts INT DEFAULT 1,
                supported_platforms JSON,
                features_ar JSON,
                features_en JSON,
                features_fr JSON,
                display_order INT DEFAULT 0,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
            )
        ";
        $pdo->exec($createTable);
        
        // إدراج بيانات تجريبية
        $insertData = "
            INSERT INTO packages (
                name_ar, name_en, name_fr,
                badge_ar, badge_en, badge_fr,
                monthly_price, yearly_price,
                is_popular, has_discount,
                monthly_discount, yearly_discount,
                monthly_discount_percentage, yearly_discount_percentage,
                points, messages, accounts,
                supported_platforms, features_ar, features_en, features_fr,
                display_order
            ) VALUES 
            (
                'الباقة الأساسية', 'Basic Plan', 'Plan Basique',
                'أساسية', 'Basic', 'Basique',
                99.00, 990.00,
                0, 0,
                0, 0, 0, 0,
                1000, 500, 1,
                '[\"facebook\", \"whatsapp\"]',
                '[\"أدوات واتساب الأساسية\", \"إدارة فيسبوك البسيطة\", \"دعم عبر البريد الإلكتروني\", \"تقارير أساسية\", \"حتى 5 مستخدمين\"]',
                '[\"Basic WhatsApp Tools\", \"Simple Facebook Management\", \"Email Support\", \"Basic Reports\", \"Up to 5 Users\"]',
                '[\"Outils WhatsApp de Base\", \"Gestion Facebook Simple\", \"Support Email\", \"Rapports de Base\", \"Jusqu à 5 Utilisateurs\"]',
                1
            ),
            (
                'الباقة الاحترافية', 'Professional Plan', 'Plan Professionnel',
                'احترافية', 'Professional', 'Professionnel',
                199.00, 1990.00,
                1, 1,
                39.00, 390.00, 19.6, 19.6,
                5000, 2000, 5,
                '[\"facebook\", \"whatsapp\", \"instagram\", \"telegram\"]',
                '[\"أدوات واتساب المتقدمة\", \"إدارة فيسبوك الكاملة\", \"أدوات انستغرام كاملة\", \"تحليلات متقدمة\", \"دعم ذو أولوية\", \"مستخدمين غير محدود\"]',
                '[\"Advanced WhatsApp Tools\", \"Complete Facebook Management\", \"Full Instagram Tools\", \"Advanced Analytics\", \"Priority Support\", \"Unlimited Users\"]',
                '[\"Outils WhatsApp Avancés\", \"Gestion Facebook Complète\", \"Outils Instagram Complets\", \"Analyses Avancées\", \"Support Prioritaire\", \"Utilisateurs Illimités\"]',
                2
            ),
            (
                'الباقة المؤسسية', 'Enterprise Plan', 'Plan Entreprise',
                'مؤسسية', 'Enterprise', 'Entreprise',
                499.00, 4990.00,
                0, 1,
                99.00, 999.00, 19.8, 20.0,
                10000, 5000, 50,
                '[\"facebook\", \"whatsapp\", \"instagram\", \"telegram\", \"b2b\"]',
                '[\"جميع الأدوات والمميزات\", \"تكامل مخصص\", \"دعم مخصص\", \"علامة تجارية مخصصة\", \"وصول كامل للـ API\", \"تدريب مجاني\"]',
                '[\"All Tools and Features\", \"Custom Integration\", \"Dedicated Support\", \"White Label\", \"Full API Access\", \"Free Training\"]',
                '[\"Tous les Outils et Fonctionnalités\", \"Intégration Personnalisée\", \"Support Dédié\", \"Marque Blanche\", \"Accès API Complet\", \"Formation Gratuite\"]',
                3
            )
        ";
        $pdo->exec($insertData);
        
        echo json_encode([
            'success' => true,
            'message' => 'تم إنشاء قاعدة البيانات والجدول والبيانات التجريبية بنجاح',
            'table_created' => true,
            'data_inserted' => true
        ]);
    } else {
        // التحقق من البيانات الموجودة
        $stmt = $pdo->query("SELECT COUNT(*) as count FROM packages WHERE is_active = 1");
        $count = $stmt->fetch()['count'];
        
        echo json_encode([
            'success' => true,
            'message' => 'قاعدة البيانات موجودة مسبقاً',
            'table_exists' => true,
            'active_packages_count' => $count
        ]);
    }
    
} catch(PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>