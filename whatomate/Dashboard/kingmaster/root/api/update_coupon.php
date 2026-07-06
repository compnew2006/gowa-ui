<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';

try {
    $pdo = getDB();

    $id = intval($_POST['id'] ?? 0);
    $code = trim($_POST['code'] ?? '');
    $discount_type = $_POST['discount_type'] ?? '';
    $discount_value = floatval($_POST['discount_value'] ?? 0);
    $uses_limit = intval($_POST['uses_limit'] ?? 0);
    $expires_at = $_POST['expires_at'] ?? null;

    // التحقق من الحقول المطلوبة
    if ($id <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرّف الكوبون غير صالح']);
        exit;
    }

    if (empty($code)) {
        echo json_encode(['success' => false, 'message' => 'كود الكوبون مطلوب']);
        exit;
    }

    if (!in_array($discount_type, ['points', 'duration', 'discount'])) {
        echo json_encode(['success' => false, 'message' => 'نوع الخصم غير صالح']);
        exit;
    }

    if ($discount_value <= 0) {
        echo json_encode(['success' => false, 'message' => 'قيمة الخصم يجب أن تكون أكبر من صفر']);
        exit;
    }

    if ($uses_limit <= 0) {
        echo json_encode(['success' => false, 'message' => 'عدد مرات الاستخدام يجب أن يكون أكبر من صفر']);
        exit;
    }

    // التحقق من عدم تكرار كود الكوبون (مع استثناء الكوبون الحالي)
    $stmt = $pdo->prepare("SELECT id FROM coupons WHERE code = ? AND id != ?");
    $stmt->execute([$code, $id]);
    if ($stmt->fetch()) {
        echo json_encode(['success' => false, 'message' => 'كود الكوبون موجود مسبقاً']);
        exit;
    }

    // تحديث الكوبون
    $stmt = $pdo->prepare("
        UPDATE coupons 
        SET code = ?, discount_type = ?, discount_value = ?, uses_limit = ?, expires_at = ? 
        WHERE id = ?
    ");
    
    $result = $stmt->execute([
        $code,
        $discount_type,
        $discount_value,
        $uses_limit,
        $expires_at,
        $id
    ]);

    if ($result) {
        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث الكوبون بنجاح'
        ]);
    } else {
        echo json_encode(['success' => false, 'message' => 'فشل تحديث الكوبون']);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
