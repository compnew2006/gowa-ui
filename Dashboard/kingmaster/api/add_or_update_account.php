<?php
header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

session_start();
if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}

$user_id = $_SESSION['user_id'];
$data = json_decode(file_get_contents('php://input'), true);

$channel = isset($data['channel']) ? trim($data['channel']) : '';
$method_type = isset($data['method']) ? trim($data['method']) : '';
$name = isset($data['name']) ? trim($data['name']) : '';
$account_uid = isset($data['account_uid']) ? trim($data['account_uid']) : '';
$cookies_text = isset($data['cookies_text']) ? trim($data['cookies_text']) : null;
$account_data = isset($data['data']) ? json_encode($data['data']) : null;
$is_reconnect = isset($data['is_reconnect']) ? $data['is_reconnect'] : false;

// التحقق من البيانات الأساسية
if (empty($channel)) {
    echo json_encode(['success' => false, 'message' => 'يرجى تحديد المنصة']);
    exit;
}

try {
    $conn = getDB();
    
    // التحقق من الحد الأقصى للحسابات قبل الإضافة
    if (!$is_reconnect) {
        // جلب معلومات الباقة للمستخدم
        $stmt = $conn->prepare("
            SELECT u.package, u.account_count
            FROM users u
            WHERE u.user_id = :user_id
        ");
        $stmt->execute([':user_id' => $user_id]);
        $user_info = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$user_info) {
            echo json_encode(['success' => false, 'message' => 'لم يتم العثور على بيانات المستخدم']);
            exit;
        }
        
        // جلب الحد الأقصى للحسابات من الباقة
        $stmt = $conn->prepare("
            SELECT accounts_count
            FROM packages
            WHERE id = :package_id
        ");
        $stmt->execute([':package_id' => $user_info['package']]);
        $package_info = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$package_info) {
            echo json_encode(['success' => false, 'message' => 'لم يتم العثور على بيانات الباقة']);
            exit;
        }
        
        // التحقق من عدد الحسابات
        $current_count = (int)$user_info['account_count'];
        $max_count = (int)$package_info['accounts_count'];
        
        if ($current_count >= $max_count) {
            echo json_encode([
                'success' => false, 
                'message' => 'لقد وصلت للحد الأقصى لعدد الحسابات المسموح بها في باقتك',
                'current_count' => $current_count,
                'max_count' => $max_count
            ]);
            exit;
        }
    }
    
    // تحديد الشرط حسب نوع العملية
    if ($is_reconnect) {
        // في حالة إعادة الربط: التحقق بـ user_id و account_uid
        if (empty($account_uid)) {
            echo json_encode(['success' => false, 'message' => 'معرف الحساب مطلوب لإعادة الربط']);
            exit;
        }
        
        $stmt = $conn->prepare("
            SELECT id FROM accounts 
            WHERE user_id = :user_id AND account_uid = :account_uid
        ");
        $stmt->execute([
            ':user_id' => $user_id,
            ':account_uid' => $account_uid
        ]);
        
    } else {
        // في حالة الربط الجديد: التحقق بـ user_id و name
        if (empty($name)) {
            echo json_encode(['success' => false, 'message' => 'اسم الحساب مطلوب']);
            exit;
        }
        
        $stmt = $conn->prepare("
            SELECT id FROM accounts 
            WHERE user_id = :user_id AND name = :name AND channel = :channel
        ");
        $stmt->execute([
            ':user_id' => $user_id,
            ':name' => $name,
            ':channel' => $channel
        ]);
    }
    
    $existing = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if ($existing) {
        // الحساب موجود - تحديث
        $account_id = $existing['id'];
        
        $updateFields = [];
        $params = [':id' => $account_id];
        
        if (!empty($name)) {
            $updateFields[] = 'name = :name';
            $params[':name'] = $name;
        }
        
        if (!empty($account_uid)) {
            $updateFields[] = 'account_uid = :account_uid';
            $params[':account_uid'] = $account_uid;
        }
        
        if (!empty($method_type)) {
            $updateFields[] = 'method = :method';
            $params[':method'] = $method_type;
        }
        
        if ($cookies_text !== null) {
            $updateFields[] = 'cookies_text = :cookies_text';
            $params[':cookies_text'] = $cookies_text;
        }
        
        if ($account_data !== null) {
            $updateFields[] = 'data = :data';
            $params[':data'] = $account_data;
        }
        
        $updateFields[] = 'status = :status';
        $params[':status'] = 'active';
        
        $updateFields[] = 'updated_at = NOW()';
        
        $sql = "UPDATE accounts SET " . implode(', ', $updateFields) . " WHERE id = :id";
        $stmt = $conn->prepare($sql);
        $stmt->execute($params);
        
        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث الحساب بنجاح',
            'action' => 'updated',
            'account_id' => $account_id
        ]);
        
    } else {
        // الحساب غير موجود - إضافة جديد
        $stmt = $conn->prepare("
            INSERT INTO accounts (
                user_id, name, account_uid, channel, 
                method, cookies_text, data, status, created_at
            ) VALUES (
                :user_id, :name, :account_uid, :channel,
                :method, :cookies_text, :data, 'active', NOW()
            )
        ");
        
        $stmt->execute([
            ':user_id' => $user_id,
            ':name' => $name,
            ':account_uid' => $account_uid,
            ':channel' => $channel,
            ':method' => $method_type,
            ':cookies_text' => $cookies_text,
            ':data' => $account_data
        ]);
        
        $account_id = $conn->lastInsertId();
        
        // تحديث عداد الحسابات في جدول users
        $stmt = $conn->prepare("
            UPDATE users 
            SET account_count = account_count + 1
            WHERE user_id = :user_id
        ");
        $stmt->execute([':user_id' => $user_id]);
        
        echo json_encode([
            'success' => true,
            'message' => 'تم إضافة الحساب بنجاح',
            'action' => 'added',
            'account_id' => $account_id
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في العملية: ' . $e->getMessage()
    ]);
}
