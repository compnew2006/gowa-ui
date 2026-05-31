<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';
require_once '../includes/mlm_functions.php';

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

$package_id = isset($data['package_id']) ? intval($data['package_id']) : 0;
$payment_method = isset($data['payment_method']) ? trim($data['payment_method']) : '';
$duration_months = isset($data['duration_months']) ? intval($data['duration_months']) : 1;
$final_amount = isset($data['final_amount']) ? floatval($data['final_amount']) : 0;
$coupon_id = isset($data['coupon_id']) ? intval($data['coupon_id']) : null;
$coupon_code = isset($data['coupon_code']) ? trim($data['coupon_code']) : null;
$referral_code = isset($data['referral_code']) ? trim($data['referral_code']) : null;

$user_id = $_SESSION['user_id'];

// التحقق من البيانات المطلوبة
if ($package_id <= 0) {
    echo json_encode([
        'success' => false,
        'message' => 'معرف الباقة غير صحيح'
    ]);
    exit;
}

if (empty($payment_method)) {
    echo json_encode([
        'success' => false,
        'message' => 'يرجى اختيار طريقة الدفع'
    ]);
    exit;
}

if ($final_amount <= 0) {
    echo json_encode([
        'success' => false,
        'message' => 'المبلغ غير صحيح'
    ]);
    exit;
}

try {
    $conn = getDB();
    $conn->beginTransaction();

    // جلب معلومات المستخدم
    $stmt = $conn->prepare("SELECT * FROM users WHERE user_id = :id");
    $stmt->execute([':id' => $user_id]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!$user) {
        $conn->rollBack();
        echo json_encode([
            'success' => false,
            'message' => 'المستخدم غير موجود'
        ]);
        exit;
    }

    // جلب معلومات الباقة
    $stmt = $conn->prepare("SELECT * FROM packages WHERE id = :id");
    $stmt->execute([':id' => $package_id]);
    $package = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!$package) {
        $conn->rollBack();
        echo json_encode([
            'success' => false,
            'message' => 'الباقة غير موجودة'
        ]);
        exit;
    }

    // إذا كانت طريقة الدفع من الرصيد
    if ($payment_method === 'balance') {
        // جلب رصيد المحفظة
        $stmt = $conn->prepare("SELECT balance FROM users_wallet WHERE user_id = :user_id FOR UPDATE");
        $stmt->execute([':user_id' => $user['user_id']]);
        $wallet = $stmt->fetch(PDO::FETCH_ASSOC);

        if (!$wallet) {
            $conn->rollBack();
            echo json_encode([
                'success' => false,
                'message' => 'المحفظة غير موجودة'
            ]);
            exit;
        }

        // التحقق من كفاية الرصيد
        if ($wallet['balance'] < $final_amount) {
            $conn->rollBack();
            echo json_encode([
                'success' => false,
                'message' => 'رصيدك غير كافٍ لإتمام العملية'
            ]);
            exit;
        }

        // خصم المبلغ من الرصيد وإضافة النقاط
        $new_balance = $wallet['balance'] - $final_amount;
        $stmt = $conn->prepare("UPDATE users_wallet SET balance = :balance, points = points + :points WHERE user_id = :user_id");
        $stmt->execute([
            ':balance' => $new_balance,
            ':user_id' => $user['user_id'],
            ':points' => $package['points']
        ]);
       

        // إضافة معاملة في transactions
        $to_email = 'عملية شراء باقة: ' . $package['name'];
        $stmt = $conn->prepare("
            INSERT INTO transactions (user_id, to_email, transaction_type, amount_type, amount, created_at) 
            VALUES (:user_id, :to_email, 'send', 'money', :amount, NOW())
        ");
        $stmt->execute([
            ':user_id' => $user_id,
            ':to_email' => $to_email,
            ':amount' => $final_amount
        ]);
    }

    // إذا تم استخدام كوبون، تسجيله في user_coupons
    if ($coupon_id && $coupon_code) {
        $stmt = $conn->prepare("
            INSERT INTO user_coupons (user_id, coupon_id, coupon_code, created_at) 
            VALUES (:user_id, :coupon_id, :coupon_code, NOW())
        ");
        $stmt->execute([
            ':user_id' => $user_id,
            ':coupon_id' => $coupon_id,
            ':coupon_code' => $coupon_code
        ]);

        // زيادة عدد استخدامات الكوبون
        $stmt = $conn->prepare("UPDATE coupons SET used_count = used_count + 1 WHERE id = :id");
        $stmt->execute([':id' => $coupon_id]);
    }

    // حساب تاريخ انتهاء الاشتراك
    // إذا كان لديه اشتراك ساري المفعول، يضاف عليه، وإلا يبدأ من اليوم
    $current_expiry = $user['expiry_date'];
    $now = date('Y-m-d H:i:s');
    
    if ($current_expiry && strtotime($current_expiry) > strtotime($now)) {
        // لديه اشتراك ساري، يضاف عليه
        $expiry_date = date('Y-m-d H:i:s', strtotime($current_expiry . " +{$duration_months} months"));
    } else {
        // لا يوجد اشتراك ساري أو منتهي، يبدأ من اليوم
        $expiry_date = date('Y-m-d H:i:s', strtotime("+{$duration_months} months"));
    }

    // تحديث باقة المستخدم وتاريخ الانتهاء
    $stmt = $conn->prepare("
        UPDATE users 
        SET package = :package_id, expiry_date = :expiry_date 
        WHERE user_id = :user_id
    ");
    $stmt->execute([
        ':package_id' => $package_id,
        ':expiry_date' => $expiry_date,
        ':user_id' => $user['user_id']
    ]);

    // إذا كان هناك رمز إحالة، تسجيله في نظام MLM
    if ($referral_code) {
        // تسجيل الإحالة في نظام MLM
        $referralResult = registerReferral($conn, $user_id, $referral_code);
        
        if (!$referralResult['success']) {
            // إذا فشل تسجيل الإحالة، لا نوقف العملية، فقط نسجل الخطأ
            error_log("Referral registration failed: " . $referralResult['message']);
        }
    }else{

        $referral_code = "USRDE066FE7279664E0";
        $referralResult = registerReferral($conn, $user_id, $referral_code);
        
        if (!$referralResult['success']) {
            // إذا فشل تسجيل الإحالة، لا نوقف العملية، فقط نسجل الخطأ
            error_log("Referral registration failed: " . $referralResult['message']);
        }

    }
    
    // توزيع عمولات MLM على سلسلة الإحالات
    $commissionResult = distributeMLMCommissions($conn, $user_id, $package_id, $final_amount);
    
    if (!$commissionResult['success']) {
        // إذا فشل توزيع العمولات، نسجل الخطأ ولكن نكمل العملية
        error_log("Commission distribution failed: " . $commissionResult['message']);
    }

    // تأكيد العملية
    $conn->commit();

    echo json_encode([
        'success' => true,
        'message' => 'تم شراء الباقة بنجاح!',
        'package_name' => $package['name'],
        'duration_months' => $duration_months,
        'expiry_date' => $expiry_date,
        'amount_paid' => $final_amount
    ]);

} catch (PDOException $e) {
    $conn->rollBack();
    echo json_encode([
        'success' => false,
        'message' => 'حدث خطأ في معالجة الطلب: ' . $e->getMessage()
    ]);
}
?>
