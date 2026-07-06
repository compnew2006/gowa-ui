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

$parent_referral_code = isset($input['referral_code']) ? strtoupper(trim($input['referral_code'])) : '';
$username = isset($input['username']) ? trim($input['username']) : '';
$email = isset($input['email']) ? trim($input['email']) : '';
$full_name = isset($input['full_name']) ? trim($input['full_name']) : '';
$phone = isset($input['phone']) ? trim($input['phone']) : null;
$package_id = isset($input['package_id']) ? (int)$input['package_id'] : null;

// التحقق من البيانات المطلوبة
if (empty($username) || empty($email) || empty($full_name) || empty($parent_referral_code)) {
    echo json_encode(['success' => false, 'message' => 'يرجى ملء جميع الحقول المطلوبة']);
    exit;
}

// التحقق من صحة البريد الإلكتروني
if (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
    echo json_encode(['success' => false, 'message' => 'البريد الإلكتروني غير صحيح']);
    exit;
}

try {
    // البحث عن المستخدم الأب بكود الإحالة
    $stmt = $pdo->prepare("SELECT id, username, full_name FROM mlm_users WHERE referral_code = ?");
    $stmt->execute([$parent_referral_code]);
    $parent_user = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$parent_user) {
        echo json_encode(['success' => false, 'message' => 'كود الإحالة غير صحيح']);
        exit;
    }
    
    $parent_id = $parent_user['id'];
    
    // التحقق من عدم تكرار اسم المستخدم
    $stmt = $pdo->prepare("SELECT id FROM mlm_users WHERE username = ?");
    $stmt->execute([$username]);
    if ($stmt->fetch()) {
        echo json_encode(['success' => false, 'message' => 'اسم المستخدم موجود بالفعل']);
        exit;
    }
    
    // التحقق من عدم تكرار البريد الإلكتروني
    $stmt = $pdo->prepare("SELECT id FROM mlm_users WHERE email = ?");
    $stmt->execute([$email]);
    if ($stmt->fetch()) {
        echo json_encode(['success' => false, 'message' => 'البريد الإلكتروني مسجل بالفعل']);
        exit;
    }
    
    // توليد كود الإحالة
    $referral_code = strtoupper(substr(md5(uniqid($username, true)), 0, 8));
    
    // التحقق من عدم تكرار كود الإحالة
    $stmt = $pdo->prepare("SELECT id FROM mlm_users WHERE referral_code = ?");
    $stmt->execute([$referral_code]);
    while ($stmt->fetch()) {
        $referral_code = strtoupper(substr(md5(uniqid($username, true)), 0, 8));
        $stmt->execute([$referral_code]);
    }
    
    $pdo->beginTransaction();
    
    // إدخال المستخدم الجديد
    $stmt = $pdo->prepare("
        INSERT INTO mlm_users 
        (username, email, full_name, phone, parent_id, referral_code)
        VALUES (?, ?, ?, ?, ?, ?)
    ");
    $stmt->execute([$username, $email, $full_name, $phone, $parent_id, $referral_code]);
    
    $new_user_id = $pdo->lastInsertId();
    
    // تحديث إحصائيات الأب إذا وجد
    // ملاحظة: نظام المستويات:
    // - المستوى 0: المستخدم الرئيسي + جميع الإحالات المباشرة
    // - المستوى 1: أول مستوى من الإحالات غير المباشرة
    // - المستوى 2+: مستويات أعمق من الإحالات غير المباشرة
    if ($parent_id) {
        // تحديث عدد الإحالات المباشرة للأب المباشر فقط
        $stmt = $pdo->prepare("
            UPDATE mlm_users 
            SET direct_referrals = direct_referrals + 1,
                total_referrals = total_referrals + 1
            WHERE id = ?
        ");
        $stmt->execute([$parent_id]);
        
        // تحديث إجمالي الإحالات حتى 5 مستويات فقط في نفس الفرع
        $current_parent_id = $parent_id;
        $level = 1;
        $max_levels = 5;
        $stmt = $pdo->prepare("SELECT parent_id FROM mlm_users WHERE id = ?");
        
        while ($current_parent_id && $level < $max_levels) {
            $stmt->execute([$current_parent_id]);
            $parent = $stmt->fetch(PDO::FETCH_ASSOC);
            
            if ($parent && $parent['parent_id']) {
                $current_parent_id = $parent['parent_id'];
                // تحديث total_referrals فقط بدون direct_referrals
                $updateStmt = $pdo->prepare("
                    UPDATE mlm_users 
                    SET total_referrals = total_referrals + 1
                    WHERE id = ?
                ");
                $updateStmt->execute([$current_parent_id]);
                $level++;
            } else {
                break;
            }
        }
    }
    
    $pdo->commit();
    
    // حساب العمولات إذا تم تحديد باقة
    $commissions_result = [];
    if ($package_id) {
        $commissions_result = calculatePackageCommissions($pdo, $new_user_id, $package_id);
    }
    
    echo json_encode([
        'success' => true,
        'message' => 'تم إضافة المستخدم بنجاح تحت ' . $parent_user['full_name'],
        'user_id' => $new_user_id,
        'referral_code' => $referral_code,
        'parent_name' => $parent_user['full_name'],
        'commissions' => $commissions_result
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    $pdo->rollBack();
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}

// دالة حساب عمولات الباقة
function calculatePackageCommissions($pdo, $user_id, $package_id) {
    try {
        // جلب بيانات المستخدم
        $stmt = $pdo->prepare("SELECT parent_id FROM mlm_users WHERE id = ?");
        $stmt->execute([$user_id]);
        $user = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$user || !$user['parent_id']) {
            return ['total' => 0, 'details' => []];
        }
        
        // جلب إعدادات الباقة
        $stmt = $pdo->prepare("
            SELECT ps.*, p.monthly_price 
            FROM package_mlm_settings ps
            JOIN packages p ON ps.package_id = p.id
            WHERE ps.package_id = ? AND ps.is_active = 1
        ");
        $stmt->execute([$package_id]);
        $settings = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$settings) {
            return ['total' => 0, 'details' => []];
        }
        
        $commissions = [];
        $total_commissions = 0;
        $current_parent_id = $user['parent_id'];
        $level = 1;
        $max_levels = $settings['max_levels'];
        
        // حساب العمولات لكل مستوى
        while ($current_parent_id && $level <= $max_levels) {
            // جلب بيانات المستلم
            $stmt = $pdo->prepare("SELECT id, username, full_name, parent_id FROM mlm_users WHERE id = ?");
            $stmt->execute([$current_parent_id]);
            $parent = $stmt->fetch(PDO::FETCH_ASSOC);
            
            if (!$parent) break;
            
            $commission_amount = 0;
            $commission_type = '';
            $percentage_value = 0;
            
            if ($level == 1) {
                // عمولة مباشرة (بالمبلغ)
                $commission_amount = $settings['direct_commission_amount'];
                $commission_type = 'direct';
                $percentage_value = 0;
            } else {
                // عمولة غير مباشرة (بالنسبة)
                // المستوى 2 في الشجرة = level_1_percentage في الإعدادات
                $settings_level = $level - 1;
                $percentage_value = $settings["level_{$settings_level}_percentage"] ?? 0;
                $commission_amount = ($settings['monthly_price'] * $percentage_value) / 100;
                $commission_type = 'indirect';
            }
            
            if ($commission_amount > 0) {
                // إدخال العمولة
                $stmt = $pdo->prepare("
                    INSERT INTO mlm_commissions 
                    (user_id, from_user_id, commission_type, level_number, amount, percentage, sale_amount, description)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                ");
                $stmt->execute([
                    $parent['id'],
                    $user_id,
                    $commission_type,
                    $level,
                    $commission_amount,
                    $percentage_value,
                    $settings['monthly_price'],
                    "عمولة من اشتراك باقة"
                ]);
                
                // تحديث أرباح المستلم
                $stmt = $pdo->prepare("
                    UPDATE mlm_users 
                    SET total_earnings = total_earnings + ? 
                    WHERE id = ?
                ");
                $stmt->execute([$commission_amount, $parent['id']]);
                
                $commissions[] = [
                    'level' => $level,
                    'user_id' => $parent['id'],
                    'user_name' => $parent['full_name'],
                    'amount' => $commission_amount,
                    'type' => $commission_type
                ];
                
                $total_commissions += $commission_amount;
            }
            
            $current_parent_id = $parent['parent_id'];
            $level++;
        }
        
        return [
            'total' => $total_commissions,
            'details' => $commissions
        ];
        
    } catch (PDOException $e) {
        return ['error' => $e->getMessage()];
    }
}
