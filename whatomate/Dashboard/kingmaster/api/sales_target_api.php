<?php
/**
 * API لإدارة تارجت المبيعات والعملاء
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE');
header('Access-Control-Allow-Headers: Content-Type');

require_once '../config/database.php';

$action = $_GET['action'] ?? '';

try {
    $db = getDB();
    
    switch ($action) {
        case 'get_target':
            getTarget($db);
            break;
            
        case 'update_target':
            updateTarget($db);
            break;
            
        case 'get_customers':
            getCustomers($db);
            break;
            
        case 'add_customer':
            addCustomer($db);
            break;
            
        case 'update_customer':
            updateCustomer($db);
            break;
            
        case 'renew_customer':
            renewCustomer($db);
            break;
            
        case 'verify_coupon':
            verifyCoupon($db);
            break;
            
        case 'delete_customer':
            deleteCustomer($db);
            break;
            
        default:
            echo json_encode([
                'success' => false,
                'message' => 'إجراء غير صحيح'
            ]);
    }
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ: ' . $e->getMessage()
    ]);
}

/**
 * الحصول على بيانات التارجت لشهر محدد
 */
function getTarget($db) {
    // السماح باختيار الشهر من GET parameter
    $requestedMonth = $_GET['month'] ?? null;
    $currentMonth = $requestedMonth ?: date('Y-m');
    
    $stmt = $db->prepare("
        SELECT * FROM sales_target 
        WHERE target_month = ?
        LIMIT 1
    ");
    $stmt->execute([$currentMonth]);
    $target = $stmt->fetch();
    
    if (!$target) {
        // إنشاء تارجت جديد للشهر الحالي
        $stmt = $db->prepare("
            INSERT INTO sales_target (target_amount, current_amount, bonus_amount, target_month)
            VALUES (100000.00, 0.00, 0.00, ?)
        ");
        $stmt->execute([$currentMonth]);
        
        $target = [
            'id' => $db->lastInsertId(),
            'target_amount' => 100000.00,
            'current_amount' => 0.00,
            'bonus_amount' => 0.00,
            'target_month' => $currentMonth
        ];
    }
    
    // حساب النسبة المئوية
    $target['percentage'] = $target['target_amount'] > 0 
        ? ($target['current_amount'] / $target['target_amount']) * 100 
        : 0;
    
    // حساب البونص (50% من المبلغ الزائد)
    $target['bonus_percentage'] = 50;
    $target['bonus_earned'] = floatval($target['bonus_amount']) * 0.5;
    
    echo json_encode([
        'success' => true,
        'data' => $target
    ]);
}

/**
 * تحديث التارجت
 */
function updateTarget($db) {
    $input = json_decode(file_get_contents('php://input'), true);
    $targetAmount = $input['target_amount'] ?? 0;
    $currentMonth = date('Y-m');
    
    $stmt = $db->prepare("
        UPDATE sales_target 
        SET target_amount = ?
        WHERE target_month = ?
    ");
    $stmt->execute([$targetAmount, $currentMonth]);
    
    echo json_encode([
        'success' => true,
        'message' => 'تم تحديث التارجت بنجاح'
    ]);
}

/**
 * الحصول على قائمة العملاء
 */
function getCustomers($db) {
    $stmt = $db->query("
        SELECT * FROM sales_customers 
        ORDER BY created_at DESC
    ");
    $customers = $stmt->fetchAll();
    
    echo json_encode([
        'success' => true,
        'data' => $customers
    ]);
}

/**
 * إضافة عميل جديد
 */
function addCustomer($db) {
    $input = json_decode(file_get_contents('php://input'), true);
    
    $name = sanitizeInput($input['name'] ?? '');
    $phone = sanitizeInput($input['phone'] ?? '');
    $email = sanitizeInput($input['email'] ?? '');
    $points = intval($input['points'] ?? 0);
    $purchaseAmount = floatval($input['purchase_amount'] ?? 0);
    $expiryDate = $input['expiry_date'] ?? null;
    $notes = sanitizeInput($input['notes'] ?? '');
    
    if (empty($name) || empty($phone)) {
        echo json_encode([
            'success' => false,
            'message' => 'الاسم ورقم الهاتف مطلوبان'
        ]);
        return;
    }
    
    $db->beginTransaction();
    
    try {
        // إضافة العميل
        $stmt = $db->prepare("
            INSERT INTO sales_customers 
            (name, phone, email, points, purchase_amount, expiry_date, notes)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        ");
        $stmt->execute([$name, $phone, $email, $points, $purchaseAmount, $expiryDate, $notes]);
        
        // تحديث التارجت مع نظام البونص
        $currentMonth = date('Y-m');
        
        // جلب التارجت الحالي
        $stmt = $db->prepare("SELECT * FROM sales_target WHERE target_month = ?");
        $stmt->execute([$currentMonth]);
        $currentTarget = $stmt->fetch();
        
        if ($currentTarget) {
            $targetAmount = floatval($currentTarget['target_amount']);
            $currentAmount = floatval($currentTarget['current_amount']);
            $newTotalAmount = $currentAmount + $purchaseAmount;
            
            // إذا تجاوزنا التارجت
            if ($newTotalAmount > $targetAmount && $currentAmount < $targetAmount) {
                // الجزء الذي يصل للتارجت
                $amountToTarget = $targetAmount - $currentAmount;
                // الجزء الزائد (بونص)
                $bonusAmount = $purchaseAmount - $amountToTarget;
                
                $stmt = $db->prepare("
                    UPDATE sales_target 
                    SET current_amount = ?, bonus_amount = bonus_amount + ?
                    WHERE target_month = ?
                ");
                $stmt->execute([$targetAmount, $bonusAmount, $currentMonth]);
            } elseif ($newTotalAmount > $targetAmount) {
                // كل المبلغ بونص
                $stmt = $db->prepare("
                    UPDATE sales_target 
                    SET bonus_amount = bonus_amount + ?
                    WHERE target_month = ?
                ");
                $stmt->execute([$purchaseAmount, $currentMonth]);
            } else {
                // مازلنا لم نحقق التارجت
                $stmt = $db->prepare("
                    UPDATE sales_target 
                    SET current_amount = current_amount + ?
                    WHERE target_month = ?
                ");
                $stmt->execute([$purchaseAmount, $currentMonth]);
            }
        }
        
        $db->commit();
        
        echo json_encode([
            'success' => true,
            'message' => 'تم إضافة العميل بنجاح',
            'customer_id' => $db->lastInsertId()
        ]);
        
    } catch (Exception $e) {
        $db->rollBack();
        throw $e;
    }
}

/**
 * تحديث بيانات عميل
 */
function updateCustomer($db) {
    $input = json_decode(file_get_contents('php://input'), true);
    
    $id = intval($input['id'] ?? 0);
    $name = sanitizeInput($input['name'] ?? '');
    $phone = sanitizeInput($input['phone'] ?? '');
    $email = sanitizeInput($input['email'] ?? '');
    $points = intval($input['points'] ?? 0);
    $purchaseAmount = floatval($input['purchase_amount'] ?? 0);
    $expiryDate = $input['expiry_date'] ?? null;
    $notes = sanitizeInput($input['notes'] ?? '');
    
    if ($id <= 0 || empty($name) || empty($phone)) {
        echo json_encode([
            'success' => false,
            'message' => 'بيانات غير صحيحة'
        ]);
        return;
    }
    
    $db->beginTransaction();
    
    try {
        // الحصول على المبلغ القديم
        $stmt = $db->prepare("SELECT purchase_amount FROM sales_customers WHERE id = ?");
        $stmt->execute([$id]);
        $oldCustomer = $stmt->fetch();
        $oldAmount = floatval($oldCustomer['purchase_amount'] ?? 0);
        
        // تحديث بيانات العميل
        $stmt = $db->prepare("
            UPDATE sales_customers 
            SET name = ?, phone = ?, email = ?, points = ?, 
                purchase_amount = ?, expiry_date = ?, notes = ?
            WHERE id = ?
        ");
        $stmt->execute([$name, $phone, $email, $points, $purchaseAmount, $expiryDate, $notes, $id]);
        
        // تحديث التارجت مع نظام البونص
        $currentMonth = date('Y-m');
        $difference = $purchaseAmount - $oldAmount;
        
        if ($difference != 0) {
            // حذف المبلغ القديم أولاً
            $stmt = $db->prepare("SELECT * FROM sales_target WHERE target_month = ?");
            $stmt->execute([$currentMonth]);
            $currentTarget = $stmt->fetch();
            
            if ($currentTarget) {
                $targetAmount = floatval($currentTarget['target_amount']);
                $currentAmount = floatval($currentTarget['current_amount']);
                $bonusAmount = floatval($currentTarget['bonus_amount']);
                
                // إزالة المبلغ القديم
                if ($oldAmount <= $currentAmount) {
                    $currentAmount -= $oldAmount;
                } else {
                    $bonusAmount -= ($oldAmount - $currentAmount);
                    $currentAmount = 0;
                }
                
                // إضافة المبلغ الجديد
                $newTotalAmount = $currentAmount + $purchaseAmount;
                
                if ($newTotalAmount > $targetAmount) {
                    if ($currentAmount < $targetAmount) {
                        $amountToTarget = $targetAmount - $currentAmount;
                        $extraBonus = $purchaseAmount - $amountToTarget;
                        $currentAmount = $targetAmount;
                        $bonusAmount += $extraBonus;
                    } else {
                        $bonusAmount += $purchaseAmount;
                    }
                } else {
                    $currentAmount += $purchaseAmount;
                }
                
                $stmt = $db->prepare("
                    UPDATE sales_target 
                    SET current_amount = ?, bonus_amount = ?
                    WHERE target_month = ?
                ");
                $stmt->execute([$currentAmount, max(0, $bonusAmount), $currentMonth]);
            }
        }
        
        $db->commit();
        
        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث بيانات العميل بنجاح'
        ]);
        
    } catch (Exception $e) {
        $db->rollBack();
        throw $e;
    }
}

/**
 * التحقق من صحة الكوبون
 */
function verifyCoupon($db) {
    $input = json_decode(file_get_contents('php://input'), true);
    $code = strtoupper(trim($input['code'] ?? ''));
    
    if (empty($code)) {
        echo json_encode([
            'success' => false,
            'message' => 'يرجى إدخال كود الكوبون'
        ]);
        return;
    }
    
    // جلب بيانات الكوبون
    $stmt = $db->prepare("
        SELECT * FROM coupons 
        WHERE code = ? AND status = 'active'
        LIMIT 1
    ");
    $stmt->execute([$code]);
    $coupon = $stmt->fetch();
    
    if (!$coupon) {
        echo json_encode([
            'success' => false,
            'message' => 'كود الكوبون غير صحيج أو غير مفعّل'
        ]);
        return;
    }
    
    // التحقق من تاريخ الانتهاء
    if ($coupon['expires_at'] && strtotime($coupon['expires_at']) < time()) {
        echo json_encode([
            'success' => false,
            'message' => 'هقد انتهت صلاحية هذا الكوبون',
            'expired' => true
        ]);
        return;
    }
    
    // التحقق من عدد الاستخدامات
    $usedCount = intval($coupon['used_count']);
    $usesLimit = intval($coupon['uses_limit']);
    
    if ($usesLimit > 0 && $usedCount >= $usesLimit) {
        echo json_encode([
            'success' => false,
            'message' => 'لقد وصل هذا الكوبون لحد الاستخدام',
            'limit_reached' => true
        ]);
        return;
    }
    
    // ترجمة نوع الكوبون
    $typeNames = [
        'extra_time' => 'وقت إضافي',
        'points' => 'نقاط',
        'amount' => 'خصم مبلغ',
        'discount' => 'خصم نسبة'
    ];
    
    $coupon['type_name'] = $typeNames[$coupon['type']] ?? $coupon['type'];
    $coupon['remaining_uses'] = $usesLimit > 0 ? ($usesLimit - $usedCount) : -1;
    
    echo json_encode([
        'success' => true,
        'message' => 'كوبون صالح للاستخدام',
        'coupon' => $coupon
    ]);
}

/**
 * حساب قيمة النقاط بالجنيه
 */
function calculatePointsValue($db, $points) {
    if ($points <= 0) return 0;
    
    // جلب جدول الأسعار مرتبة تنازلياً
    $stmt = $db->query("
        SELECT points, price 
        FROM points_pricing 
        WHERE is_active = 1 
        ORDER BY points DESC
    ");
    $pricing = $stmt->fetchAll();
    
    $totalValue = 0;
    $remainingPoints = $points;
    
    foreach ($pricing as $tier) {
        $tierPoints = intval($tier['points']);
        $tierPrice = floatval($tier['price']);
        
        if ($remainingPoints >= $tierPoints) {
            $packages = floor($remainingPoints / $tierPoints);
            $totalValue += $packages * $tierPrice;
            $remainingPoints = $remainingPoints % $tierPoints;
        }
    }
    
    return $totalValue;
}

/**
 * تجديد اشتراك عميل
 */
function renewCustomer($db) {
    $input = json_decode(file_get_contents('php://input'), true);
    
    $id = intval($input['id'] ?? 0);
    $name = sanitizeInput($input['name'] ?? '');
    $phone = sanitizeInput($input['phone'] ?? '');
    $email = sanitizeInput($input['email'] ?? '');
    $points = intval($input['points'] ?? 0);
    $newPurchaseAmount = floatval($input['purchase_amount'] ?? 0);
    $expiryDate = $input['expiry_date'] ?? null;
    $notes = sanitizeInput($input['notes'] ?? '');
    $renewAmount = floatval($input['renew_amount'] ?? 0);
    
    if ($id <= 0 || empty($name) || empty($phone)) {
        echo json_encode([
            'success' => false,
            'message' => 'بيانات غير صحيحة'
        ]);
        return;
    }
    
    $db->beginTransaction();
    
    try {
        // تحديث بيانات العميل
        $stmt = $db->prepare("
            UPDATE sales_customers 
            SET name = ?, phone = ?, email = ?, points = ?, 
                purchase_amount = ?, expiry_date = ?, notes = ?
            WHERE id = ?
        ");
        $stmt->execute([$name, $phone, $email, $points, $newPurchaseAmount, $expiryDate, $notes, $id]);
        
        // إضافة مبلغ التجديد فقط للتارجت (النقاط مجانية)
        $currentMonth = date('Y-m');
        
        $stmt = $db->prepare("SELECT * FROM sales_target WHERE target_month = ?");
        $stmt->execute([$currentMonth]);
        $currentTarget = $stmt->fetch();
        
        if ($currentTarget && $renewAmount > 0) {
            $targetAmount = floatval($currentTarget['target_amount']);
            $currentAmount = floatval($currentTarget['current_amount']);
            $newTotalAmount = $currentAmount + $renewAmount;
            
            // إذا تجاوزنا التارجت
            if ($newTotalAmount > $targetAmount && $currentAmount < $targetAmount) {
                // الجزء الذي يصل للتارجت
                $amountToTarget = $targetAmount - $currentAmount;
                // الجزء الزائد (بونص)
                $bonusAmount = $renewAmount - $amountToTarget;
                
                $stmt = $db->prepare("
                    UPDATE sales_target 
                    SET current_amount = ?, bonus_amount = bonus_amount + ?
                    WHERE target_month = ?
                ");
                $stmt->execute([$targetAmount, $bonusAmount, $currentMonth]);
            } elseif ($newTotalAmount > $targetAmount) {
                // كل المبلغ بونص
                $stmt = $db->prepare("
                    UPDATE sales_target 
                    SET bonus_amount = bonus_amount + ?
                    WHERE target_month = ?
                ");
                $stmt->execute([$renewAmount, $currentMonth]);
            } else {
                // مازلنا لم نحقق التارجت
                $stmt = $db->prepare("
                    UPDATE sales_target 
                    SET current_amount = current_amount + ?
                    WHERE target_month = ?
                ");
                $stmt->execute([$renewAmount, $currentMonth]);
            }
        }
        
        $db->commit();
        
        echo json_encode([
            'success' => true,
            'message' => 'تم تجديد الاشتراك بنجاح'
        ]);
        
    } catch (Exception $e) {
        $db->rollBack();
        throw $e;
    }
}

/**
 * حذف عميل
 */
function deleteCustomer($db) {
    $input = json_decode(file_get_contents('php://input'), true);
    $id = intval($input['id'] ?? 0);
    
    if ($id <= 0) {
        echo json_encode([
            'success' => false,
            'message' => 'معرف غير صحيح'
        ]);
        return;
    }
    
    $db->beginTransaction();
    
    try {
        // الحصول على مبلغ العميل
        $stmt = $db->prepare("SELECT purchase_amount FROM sales_customers WHERE id = ?");
        $stmt->execute([$id]);
        $customer = $stmt->fetch();
        $amount = floatval($customer['purchase_amount'] ?? 0);
        
        // حذف العميل
        $stmt = $db->prepare("DELETE FROM sales_customers WHERE id = ?");
        $stmt->execute([$id]);
        
        // تحديث التارجت مع نظام البونص
        $currentMonth = date('Y-m');
        
        $stmt = $db->prepare("SELECT * FROM sales_target WHERE target_month = ?");
        $stmt->execute([$currentMonth]);
        $currentTarget = $stmt->fetch();
        
        if ($currentTarget && $amount > 0) {
            $currentAmount = floatval($currentTarget['current_amount']);
            $bonusAmount = floatval($currentTarget['bonus_amount']);
            
            // إزالة المبلغ من التارجت أو البونص
            if ($amount <= $currentAmount) {
                $currentAmount -= $amount;
            } else {
                $remainingToDeduct = $amount - $currentAmount;
                $currentAmount = 0;
                $bonusAmount = max(0, $bonusAmount - $remainingToDeduct);
            }
            
            $stmt = $db->prepare("
                UPDATE sales_target 
                SET current_amount = ?, bonus_amount = ?
                WHERE target_month = ?
            ");
            $stmt->execute([$currentAmount, $bonusAmount, $currentMonth]);
        }
        
        $db->commit();
        
        echo json_encode([
            'success' => true,
            'message' => 'تم حذف العميل بنجاح'
        ]);
        
    } catch (Exception $e) {
        $db->rollBack();
        throw $e;
    }
}
?>
