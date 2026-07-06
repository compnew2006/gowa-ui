<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';

// التحقق من تسجيل الدخول
if (!isset($_SESSION['user_id'])) {
    echo json_encode([
        'success' => false,
        'message' => 'يجب تسجيل الدخول'
    ]);
    exit;
}

// قراءة البيانات المرسلة
$data = json_decode(file_get_contents('php://input'), true);

$coupon_code = isset($data['coupon_code']) ? trim($data['coupon_code']) : '';
$package_id = isset($data['package_id']) ? intval($data['package_id']) : 0;
$package_price = isset($data['package_price']) ? floatval($data['package_price']) : 0;

// التحقق من البيانات المطلوبة
if (empty($coupon_code)) {
    echo json_encode([
        'success' => false,
        'message' => 'يرجى إدخال رمز الكوبون'
    ]);
    exit;
}

if ($package_price <= 0) {
    echo json_encode([
        'success' => false,
        'message' => 'سعر الباقة غير صحيح'
    ]);
    exit;
}

try {
    $conn = getDB();
    
    // البحث عن الكوبون في قاعدة البيانات
    $stmt = $conn->prepare("SELECT * FROM coupons WHERE code = :code");
    $stmt->execute([':code' => $coupon_code]);
    $coupon = $stmt->fetch(PDO::FETCH_ASSOC);
    
    // التحقق من وجود الكوبون
    if (!$coupon) {
        echo json_encode([
            'success' => false,
            'message' => 'الكوبون غير موجود'
        ]);
        exit;
    }
    
    // التحقق من عدد الاستخدامات
    if ($coupon['used_count'] >= $coupon['uses_limit']) {
        echo json_encode([
            'success' => false,
            'message' => 'تم استنفاذ عدد استخدامات هذا الكوبون'
        ]);
        exit;
    }
    
    // التحقق من تاريخ انتهاء الصلاحية
    $now = new DateTime();
    $expires_at = new DateTime($coupon['expires_at']);
    
    if ($now > $expires_at) {
        echo json_encode([
            'success' => false,
            'message' => 'انتهت صلاحية هذا الكوبون'
        ]);
        exit;
    }
    
    // حساب قيمة الخصم
    $discount = 0;
    
    if ($coupon['discount_type'] === 'discount') {
        // الخصم بالنسبة المئوية
        $discount_percentage = floatval($coupon['discount_value']);
        $discount = ($package_price * $discount_percentage) / 100;
        
        // التأكد من أن الخصم لا يتجاوز سعر الباقة
        if ($discount > $package_price) {
            $discount = $package_price;
        }
    } else {
        // أنواع خصم أخرى (إذا كانت موجودة)
        echo json_encode([
            'success' => false,
            'message' => 'نوع الكوبون غير مدعوم'
        ]);
        exit;
    }
    
    // إرجاع النتيجة الناجحة
    echo json_encode([
        'success' => true,
        'message' => 'تم تطبيق الكوبون بنجاح',
        'discount' => $discount,
        'discount_percentage' => floatval($coupon['discount_value']),
        'new_price' => $package_price - $discount,
        'original_price' => $package_price,
        'coupon_id' => $coupon['id']
    ]);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ في التحقق من الكوبون: ' . $e->getMessage()
    ]);
}
?>
