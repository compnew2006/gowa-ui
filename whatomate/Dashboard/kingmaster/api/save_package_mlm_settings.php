<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: POST');
header('Access-Control-Allow-Headers: Content-Type');

require_once '../config/database.php';

$pdo = getDB();

$input = json_decode(file_get_contents('php://input'), true);

if (!$input) {
    echo json_encode(['success' => false, 'message' => 'بيانات غير صالحة']);
    exit;
}

$package_id = isset($input['package_id']) ? (int)$input['package_id'] : 0;
$max_levels = isset($input['max_levels']) ? (int)$input['max_levels'] : 5;
$direct_commission_amount = isset($input['direct_commission_amount']) ? (float)$input['direct_commission_amount'] : 0;

if ($package_id <= 0) {
    echo json_encode(['success' => false, 'message' => 'معرف الباقة غير صحيح']);
    exit;
}

try {
    // التحقق من وجود الباقة
    $stmt = $pdo->prepare("SELECT id FROM packages WHERE id = ?");
    $stmt->execute([$package_id]);
    if (!$stmt->fetch()) {
        echo json_encode(['success' => false, 'message' => 'الباقة غير موجودة']);
        exit;
    }

    // جمع نسب المستويات
    $levels = [];
    for ($i = 1; $i <= 10; $i++) {
        $key = "level_{$i}_percentage";
        $levels[$key] = isset($input[$key]) ? (float)$input[$key] : 0;
    }

    // التحقق من وجود إعدادات سابقة
    $stmt = $pdo->prepare("SELECT id FROM package_mlm_settings WHERE package_id = ?");
    $stmt->execute([$package_id]);
    $exists = $stmt->fetch();

    if ($exists) {
        // تحديث
        $stmt = $pdo->prepare("
            UPDATE package_mlm_settings SET
                max_levels = ?,
                direct_commission_amount = ?,
                level_1_percentage = ?,
                level_2_percentage = ?,
                level_3_percentage = ?,
                level_4_percentage = ?,
                level_5_percentage = ?,
                level_6_percentage = ?,
                level_7_percentage = ?,
                level_8_percentage = ?,
                level_9_percentage = ?,
                level_10_percentage = ?
            WHERE package_id = ?
        ");
        
        $stmt->execute([
            $max_levels,
            $direct_commission_amount,
            $levels['level_1_percentage'],
            $levels['level_2_percentage'],
            $levels['level_3_percentage'],
            $levels['level_4_percentage'],
            $levels['level_5_percentage'],
            $levels['level_6_percentage'],
            $levels['level_7_percentage'],
            $levels['level_8_percentage'],
            $levels['level_9_percentage'],
            $levels['level_10_percentage'],
            $package_id
        ]);
    } else {
        // إدخال جديد
        $stmt = $pdo->prepare("
            INSERT INTO package_mlm_settings (
                package_id, max_levels, direct_commission_amount,
                level_1_percentage, level_2_percentage, level_3_percentage,
                level_4_percentage, level_5_percentage, level_6_percentage,
                level_7_percentage, level_8_percentage, level_9_percentage,
                level_10_percentage
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ");
        
        $stmt->execute([
            $package_id,
            $max_levels,
            $direct_commission_amount,
            $levels['level_1_percentage'],
            $levels['level_2_percentage'],
            $levels['level_3_percentage'],
            $levels['level_4_percentage'],
            $levels['level_5_percentage'],
            $levels['level_6_percentage'],
            $levels['level_7_percentage'],
            $levels['level_8_percentage'],
            $levels['level_9_percentage'],
            $levels['level_10_percentage']
        ]);
    }

    echo json_encode([
        'success' => true,
        'message' => 'تم حفظ الإعدادات بنجاح'
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
