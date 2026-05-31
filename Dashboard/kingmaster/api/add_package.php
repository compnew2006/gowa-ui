<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';

try {
    $pdo = getDB();

    $name = trim($_POST['name'] ?? '');
    $description = trim($_POST['description'] ?? '');
    $features = $_POST['features'] ?? '[]';
    $price = floatval($_POST['price'] ?? 0);
    $original_price = !empty($_POST['original_price']) ? floatval($_POST['original_price']) : null;
    $currency = trim($_POST['currency'] ?? 'EGP');
    $has_discount = intval($_POST['has_discount'] ?? 0);
    $is_popular = intval($_POST['is_popular'] ?? 0);
    $platforms = $_POST['platforms'] ?? '[]';
    $accounts_count = intval($_POST['accounts_count'] ?? 0);
    $messages_count = intval($_POST['messages_count'] ?? 0);
    $points = intval($_POST['points'] ?? 0);

    // التحقق من الحقول المطلوبة
    if (empty($name)) {
        echo json_encode(['success' => false, 'message' => 'اسم الباقة مطلوب']);
        exit;
    }

    if ($price <= 0) {
        echo json_encode(['success' => false, 'message' => 'السعر يجب أن يكون أكبر من صفر']);
        exit;
    }

    // التحقق من المميزات
    $featuresArray = json_decode($features, true);
    if (empty($featuresArray) || !is_array($featuresArray)) {
        echo json_encode(['success' => false, 'message' => 'يجب إضافة ميزة واحدة على الأقل']);
        exit;
    }

    // التحقق من المنصات
    $platformsArray = json_decode($platforms, true);
    if (empty($platformsArray) || !is_array($platformsArray)) {
        echo json_encode(['success' => false, 'message' => 'يجب اختيار منصة واحدة على الأقل']);
        exit;
    }

    // إدراج الباقة
    $stmt = $pdo->prepare("
        INSERT INTO packages (
            name, description, features, price, original_price, currency,
            has_discount, is_popular, platforms, accounts_count, 
            messages_count, points, created_at
        ) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
    ");
    
    $result = $stmt->execute([
        $name,
        $description,
        $features,
        $price,
        $original_price,
        $currency,
        $has_discount,
        $is_popular,
        $platforms,
        $accounts_count,
        $messages_count,
        $points
    ]);

    if ($result) {
        echo json_encode([
            'success' => true,
            'message' => 'تم إضافة الباقة بنجاح',
            'package_id' => $pdo->lastInsertId()
        ]);
    } else {
        echo json_encode(['success' => false, 'message' => 'فشل إضافة الباقة']);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
