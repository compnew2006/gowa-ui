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

$from_user_id = isset($input['from_user_id']) ? (int)$input['from_user_id'] : 0;
$sale_amount = isset($input['sale_amount']) ? (float)$input['sale_amount'] : 0;
$description = isset($input['description']) ? trim($input['description']) : 'عمولة من عملية بيع';

if (empty($from_user_id) || $sale_amount <= 0) {
    echo json_encode(['success' => false, 'message' => 'بيانات غير كاملة']);
    exit;
}

try {
    $pdo->beginTransaction();
    
    // جلب إعدادات MLM
    $stmt = $pdo->prepare("
        SELECT level_number, direct_commission_percentage, indirect_commission_percentage
        FROM mlm_settings
        WHERE is_active = 1
        ORDER BY level_number ASC
    ");
    $stmt->execute();
    $settings = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    if (empty($settings)) {
        $pdo->rollBack();
        echo json_encode(['success' => false, 'message' => 'لا توجد إعدادات نشطة']);
        exit;
    }
    
    // جلب بيانات المستخدم
    $stmt = $pdo->prepare("SELECT id, username, full_name, parent_id FROM mlm_users WHERE id = ?");
    $stmt->execute([$from_user_id]);
    $from_user = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if (!$from_user) {
        $pdo->rollBack();
        echo json_encode(['success' => false, 'message' => 'المستخدم غير موجود']);
        exit;
    }
    
    $commissions = [];
    $current_parent_id = $from_user['parent_id'];
    $current_level = 1;
    $max_levels = 5; // حد أقصى 5 مستويات لكل فرع
    
    // حساب العمولات لكل مستوى
    foreach ($settings as $setting) {
        if (!$current_parent_id || $current_level > $max_levels) break; // توقف بعد 5 مستويات
        
        // جلب بيانات المستلم
        $stmt = $pdo->prepare("SELECT id, username, full_name, parent_id, total_earnings FROM mlm_users WHERE id = ?");
        $stmt->execute([$current_parent_id]);
        $parent = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$parent) break;
        
        // حساب نسبة العمولة
        $percentage = $current_level === 1 
            ? $setting['direct_commission_percentage'] 
            : $setting['indirect_commission_percentage'];
        
        if ($percentage > 0) {
            $commission_amount = ($sale_amount * $percentage) / 100;
            $commission_type = $current_level === 1 ? 'direct' : 'indirect';
            
            // إدخال العمولة
            $stmt = $pdo->prepare("
                INSERT INTO mlm_commissions 
                (user_id, from_user_id, commission_type, level_number, amount, percentage, sale_amount, description)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            ");
            $stmt->execute([
                $parent['id'],
                $from_user_id,
                $commission_type,
                $current_level,
                $commission_amount,
                $percentage,
                $sale_amount,
                $description
            ]);
            
            // تحديث إجمالي أرباح المستلم
            $stmt = $pdo->prepare("
                UPDATE mlm_users 
                SET total_earnings = total_earnings + ? 
                WHERE id = ?
            ");
            $stmt->execute([$commission_amount, $parent['id']]);
            
            $commissions[] = [
                'level' => $current_level,
                'user_id' => $parent['id'],
                'user_name' => $parent['full_name'],
                'percentage' => $percentage,
                'amount' => $commission_amount
            ];
        }
        
        // الانتقال للمستوى التالي
        $current_parent_id = $parent['parent_id'];
        $current_level++;
        
        if ($current_level > count($settings)) break;
    }
    
    $pdo->commit();
    
    $total_commission = array_sum(array_column($commissions, 'amount'));
    
    echo json_encode([
        'success' => true,
        'message' => 'تم حساب وتوزيع العمولات بنجاح',
        'commissions' => $commissions,
        'total_commission' => $total_commission,
        'sale_amount' => $sale_amount
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    $pdo->rollBack();
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
