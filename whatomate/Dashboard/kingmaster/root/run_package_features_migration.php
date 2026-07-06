<?php
// إعدادات قاعدة البيانات
$host = 'localhost';
$dbname = 'kingmaster';
$username = 'kingmaster';
$password = 'kingmaster';

try {
    $pdo = new PDO("mysql:host=$host;dbname=$dbname;charset=utf8mb4", $username, $password);
    $pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    
    echo "بدء تحديث قاعدة البيانات...\n";
    
    // قراءة ملف SQL
    $sql = file_get_contents('add_package_features.sql');
    
    // تنفيذ استعلامات ALTER TABLE
    echo "إضافة الحقول الجديدة...\n";
    $pdo->exec("ALTER TABLE packages 
        ADD COLUMN IF NOT EXISTS points INT DEFAULT 0 COMMENT 'عدد النقاط المتاحة في الباقة',
        ADD COLUMN IF NOT EXISTS messages INT DEFAULT 0 COMMENT 'عدد الرسائل المسموحة',
        ADD COLUMN IF NOT EXISTS accounts INT DEFAULT 1 COMMENT 'عدد الحسابات المسموحة',
        ADD COLUMN IF NOT EXISTS supported_platforms JSON COMMENT 'المنصات المدعومة'");
    
    echo "إضافة الفهارس...\n";
    try {
        $pdo->exec("ALTER TABLE packages ADD INDEX idx_points (points)");
    } catch (PDOException $e) {
        if ($e->getCode() != '42000') throw $e; // تجاهل إذا كان الفهرس موجود
    }
    try {
        $pdo->exec("ALTER TABLE packages ADD INDEX idx_messages (messages)");
    } catch (PDOException $e) {
        if ($e->getCode() != '42000') throw $e;
    }
    try {
        $pdo->exec("ALTER TABLE packages ADD INDEX idx_accounts (accounts)");
    } catch (PDOException $e) {
        if ($e->getCode() != '42000') throw $e;
    }
    
    echo "تحديث الباقات الموجودة...\n";
    $pdo->exec("UPDATE packages SET 
        points = CASE WHEN points IS NULL OR points = 0 THEN 1000 ELSE points END,
        messages = CASE WHEN messages IS NULL OR messages = 0 THEN 500 ELSE messages END,
        accounts = CASE WHEN accounts IS NULL OR accounts = 0 THEN 1 ELSE accounts END,
        supported_platforms = CASE WHEN supported_platforms IS NULL THEN JSON_ARRAY('facebook', 'whatsapp', 'instagram') ELSE supported_platforms END
        WHERE id > 0");
    
    echo "تم تحديث قاعدة البيانات بنجاح!\n";
    
    // التحقق من الحقول الجديدة
    $stmt = $pdo->query("DESCRIBE packages");
    $columns = $stmt->fetchAll(PDO::FETCH_COLUMN);
    
    echo "\nالحقول الجديدة المضافة:\n";
    $new_fields = ['points', 'messages', 'accounts', 'supported_platforms'];
    foreach ($new_fields as $field) {
        if (in_array($field, $columns)) {
            echo "✓ $field\n";
        } else {
            echo "✗ $field (لم يتم إضافته)\n";
        }
    }
    
} catch (PDOException $e) {
    echo "خطأ في قاعدة البيانات: " . $e->getMessage() . "\n";
}
?>