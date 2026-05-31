<?php
/**
 * اختبار اتصال قاعدة البيانات
 */

header('Content-Type: application/json; charset=UTF-8');

try {
    // إعدادات قاعدة البيانات
    $host = 'localhost';
    $dbname = 'kingmaster_packages';
    $username = 'root';
    $password = 'your_new_password';
    
    // محاولة الاتصال
    $dsn = "mysql:host=$host;dbname=$dbname;charset=utf8mb4";
    $pdo = new PDO($dsn, $username, $password, [
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
    ]);
    
    // اختبار وجود الجدول
    $stmt = $pdo->query("SHOW TABLES LIKE 'packages'");
    $tableExists = $stmt->rowCount() > 0;
    
    if ($tableExists) {
        // محاولة جلب الباقات
        $stmt = $pdo->query("SELECT * FROM packages ORDER BY display_order ASC");
        $packages = $stmt->fetchAll();
        
        echo json_encode([
            'success' => true,
            'message' => 'الاتصال ناجح',
            'database' => $dbname,
            'table_exists' => true,
            'packages_count' => count($packages),
            'packages' => $packages
        ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'قاعدة البيانات موجودة لكن الجدول packages غير موجود',
            'database' => $dbname,
            'table_exists' => false,
            'suggestion' => 'يجب تنفيذ ملف kingmaster_packages.sql'
        ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    }
    
} catch (PDOException $e) {
    if ($e->getCode() == 1049) {
        // قاعدة البيانات غير موجودة
        echo json_encode([
            'success' => false,
            'message' => 'قاعدة البيانات kingmaster_packages غير موجودة',
            'error_code' => $e->getCode(),
            'suggestion' => 'يجب إنشاء قاعدة البيانات أولاً من phpMyAdmin'
        ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    } elseif ($e->getCode() == 1045) {
        // خطأ في اسم المستخدم أو كلمة المرور
        echo json_encode([
            'success' => false,
            'message' => 'خطأ في اسم المستخدم أو كلمة المرور',
            'error_code' => $e->getCode(),
            'suggestion' => 'تحقق من كلمة مرور MySQL'
        ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'خطأ في الاتصال: ' . $e->getMessage(),
            'error_code' => $e->getCode(),
            'full_error' => $e
        ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    }
}
?>