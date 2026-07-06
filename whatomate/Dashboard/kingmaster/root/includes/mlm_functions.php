<?php
/**
 * MLM Commission System Functions
 * نظام حساب العمولات متعدد المستويات
 */

require_once __DIR__ . '/../config/database.php';

/**
 * نسب العمولات حسب المستوى
 */
define('MLM_COMMISSION_RATES', [
    1 => 15.0,   // المستوى الأول: 15%
    2 => 10.0,   // المستوى الثاني: 10%
    3 => 5.0,    // المستوى الثالث: 5%
    4 => 2.5     // المستوى الرابع: 2.5%
]);

/**
 * الحصول على سلسلة الإحالات للمستخدم (حتى 4 مستويات)
 * 
 * @param PDO $conn اتصال قاعدة البيانات
 * @param string $user_id معرف المستخدم
 * @return array مصفوفة من المستخدمين في السلسلة
 */
function getReferralChain($conn, $user_id) {
    $chain = [];
    $current_user_id = $user_id;
    $level = 1;
    
    // البحث عن 4 مستويات كحد أقصى
    while ($level <= 4 && $current_user_id) {
        // البحث عن من قام بإحالة هذا المستخدم
        $stmt = $conn->prepare("
            SELECT referrer_id 
            FROM mlm_referrals 
            WHERE user_id = :user_id
            LIMIT 1
        ");
        $stmt->execute([':user_id' => $current_user_id]);
        $result = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if ($result && $result['referrer_id']) {
            $chain[] = [
                'user_id' => $result['referrer_id'],
                'level' => $level
            ];
            $current_user_id = $result['referrer_id'];
            $level++;
        } else {
            break;
        }
    }
    
    return $chain;
}

/**
 * حساب وتوزيع العمولات على سلسلة الإحالات
 * 
 * @param PDO $conn اتصال قاعدة البيانات
 * @param string $buyer_id معرف المشتري
 * @param int $package_id معرف الباقة
 * @param float $final_amount المبلغ النهائي بعد الخصومات
 * @return array نتيجة العملية
 */
function distributeMLMCommissions($conn, $buyer_id, $package_id, $final_amount) {
    try {
        // الحصول على سلسلة الإحالات
        $chain = getReferralChain($conn, $buyer_id);
        
        if (empty($chain)) {
            return [
                'success' => true,
                'message' => 'لا توجد سلسلة إحالات',
                'commissions' => []
            ];
        }
        
        $commissions = [];
        
        // توزيع العمولات على كل مستوى
        foreach ($chain as $referrer) {
            $level = $referrer['level'];
            $beneficiary_id = $referrer['user_id'];
            
            // حساب العمولة
            $commission_rate = MLM_COMMISSION_RATES[$level];
            $commission_amount = ($final_amount * $commission_rate) / 100;
            
            // إضافة العمولة إلى محفظة المستفيد
            $stmt = $conn->prepare("
                UPDATE commission_wallets 
                SET commission = commission + :commission 
                WHERE user_id = :user_id
            ");
            $stmt->execute([
                ':commission' => $commission_amount,
                ':user_id' => $beneficiary_id
            ]);
            
            // تسجيل العمولة في جدول mlm_commissions
            $stmt = $conn->prepare("
                INSERT INTO mlm_commissions 
                (beneficiary_id, buyer_id, package_id, package_price, commission_rate, commission_amount, level, created_at) 
                VALUES 
                (:beneficiary_id, :buyer_id, :package_id, :package_price, :commission_rate, :commission_amount, :level, NOW())
            ");
            $stmt->execute([
                ':beneficiary_id' => $beneficiary_id,
                ':buyer_id' => $buyer_id,
                ':package_id' => $package_id,
                ':package_price' => $final_amount,
                ':commission_rate' => $commission_rate,
                ':commission_amount' => $commission_amount,
                ':level' => $level
            ]);
            
            $commissions[] = [
                'beneficiary_id' => $beneficiary_id,
                'level' => $level,
                'rate' => $commission_rate,
                'amount' => $commission_amount
            ];
        }
        
        return [
            'success' => true,
            'message' => 'تم توزيع العمولات بنجاح',
            'commissions' => $commissions
        ];
        
    } catch (PDOException $e) {
        return [
            'success' => false,
            'message' => 'خطأ في توزيع العمولات: ' . $e->getMessage(),
            'commissions' => []
        ];
    }
}

/**
 * تسجيل علاقة إحالة جديدة
 * 
 * @param PDO $conn اتصال قاعدة البيانات
 * @param string $user_id معرف المستخدم الجديد
 * @param string $referral_code رمز الإحالة المستخدم
 * @return array نتيجة العملية
 */
function registerReferral($conn, $user_id, $referral_code) {
    try {
        // البحث عن صاحب رمز الإحالة
        $stmt = $conn->prepare("
            SELECT user_id 
            FROM users 
            WHERE user_id = :referral_code
            LIMIT 1
        ");
        $stmt->execute([':referral_code' => $referral_code]);
        $referrer = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$referrer) {
            return [
                'success' => false,
                'message' => 'رمز الإحالة غير موجود'
            ];
        }
        
        $referrer_id = $referrer['user_id'];
        
        // التحقق من عدم وجود إحالة سابقة لهذا المستخدم
        $stmt = $conn->prepare("
            SELECT id FROM mlm_referrals WHERE user_id = :user_id
        ");
        $stmt->execute([':user_id' => $user_id]);
        
        if ($stmt->fetch()) {
            return [
                'success' => false,
                'message' => 'المستخدم مسجل بالفعل تحت إحالة أخرى'
            ];
        }
        
        // تسجيل العلاقة في mlm_referrals
        $stmt = $conn->prepare("
            INSERT INTO mlm_referrals 
            (user_id, referrer_id, referral_code, level, created_at) 
            VALUES 
            (:user_id, :referrer_id, :referral_code, 1, NOW())
        ");
        $stmt->execute([
            ':user_id' => $user_id,
            ':referrer_id' => $referrer_id,
            ':referral_code' => $referral_code
        ]);
        
        // تحديث referrer_id في جدول users
        $stmt = $conn->prepare("
            UPDATE users 
            SET referrer_id = :referrer_id 
            WHERE user_id = :user_id
        ");
        $stmt->execute([
            ':referrer_id' => $referrer_id,
            ':user_id' => $user_id
        ]);
        




        
        return [
            'success' => true,
            'message' => 'تم تسجيل الإحالة بنجاح',
            'referrer_id' => $referrer_id
        ];
        
    } catch (PDOException $e) {
        return [
            'success' => false,
            'message' => 'خطأ في تسجيل الإحالة: ' . $e->getMessage()
        ];
    }
}

/**
 * الحصول على إحصائيات MLM للمستخدم
 * 
 * @param PDO $conn اتصال قاعدة البيانات
 * @param string $user_id معرف المستخدم
 * @return array الإحصائيات
 */
function getMLMStats($conn, $user_id) {
    try {
        // عدد الإحالات المباشرة (المستوى الأول)
        $stmt = $conn->prepare("
            SELECT COUNT(*) as direct_referrals 
            FROM mlm_referrals 
            WHERE referrer_id = :user_id
        ");
        $stmt->execute([':user_id' => $user_id]);
        $result = $stmt->fetch(PDO::FETCH_ASSOC);
        $direct_referrals = $result['direct_referrals'];
        
        // إجمالي العمولات المكتسبة
        $stmt = $conn->prepare("
            SELECT 
                SUM(commission_amount) as total_commissions,
                COUNT(*) as total_transactions
            FROM mlm_commissions 
            WHERE beneficiary_id = :user_id
        ");
        $stmt->execute([':user_id' => $user_id]);
        $result = $stmt->fetch(PDO::FETCH_ASSOC);
        
        // العمولات حسب المستوى
        $stmt = $conn->prepare("
            SELECT 
                level,
                SUM(commission_amount) as level_commissions,
                COUNT(*) as level_transactions
            FROM mlm_commissions 
            WHERE beneficiary_id = :user_id
            GROUP BY level
            ORDER BY level
        ");
        $stmt->execute([':user_id' => $user_id]);
        $level_stats = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        return [
            'success' => true,
            'direct_referrals' => $direct_referrals,
            'total_commissions' => $result['total_commissions'] ?? 0,
            'total_transactions' => $result['total_transactions'] ?? 0,
            'level_stats' => $level_stats
        ];
        
    } catch (PDOException $e) {
        return [
            'success' => false,
            'message' => 'خطأ في جلب الإحصائيات: ' . $e->getMessage()
        ];
    }
}
?>
