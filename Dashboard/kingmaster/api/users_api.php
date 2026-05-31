<?php
require_once '../config/database.php';

applyCorsHeaders('GET, POST, PUT, DELETE, PATCH, OPTIONS');
applySecurityHeaders();
header('Content-Type: application/json; charset=utf-8');
$adminUserId = requireAdminUser();
enforceRateLimit('users_api:' . $adminUserId, 120, 60);
if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
    verifyCsrfToken();
}

// Get the request method and path
$method = $_SERVER['REQUEST_METHOD'];
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$pathSegments = explode('/', trim($path, '/'));

// Remove 'test/api/users_api.php' from path segments
$pathSegments = array_slice($pathSegments, 3);

try {
    switch ($method) {
        case 'GET':
            handleGet($pathSegments);
            break;
        case 'POST':
            handlePost($pathSegments);
            break;
        case 'PUT':
            handlePut($pathSegments);
            break;
        case 'DELETE':
            handleDelete($pathSegments);
            break;
        case 'PATCH':
            handlePatch($pathSegments);
            break;
        default:
            http_response_code(405);
            echo json_encode(['success' => false, 'message' => 'Method not allowed']);
    }
} catch (Exception $e) {
    error_log('users_api error: ' . $e->getMessage());
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Server error']);
}

function handleGet($pathSegments) {
    // جلب جميع المستخدمين
    $query = "
        SELECT 
            u.id,
            u.first_name,
            u.last_name,
            u.email,
            u.user_type,
            u.status,
            u.timezone,
            u.count_s,
            u.created_at,
            u.updated_at,
            us.package_id,
            us.start_date,
            us.end_date as membership_expiry,
            p.name_ar as package_name
        FROM users u
        LEFT JOIN user_subscriptions us ON u.id = us.user_id AND us.status = 'active'
        LEFT JOIN packages p ON us.package_id = p.id
        ORDER BY u.created_at DESC
    ";
    
    $users = fetchAll($query);
    
    if ($users === false) {
        // إنشاء جدول المستخدمين إذا لم يكن موجوداً
        createUsersTable();
        $users = [];
    }
    
    // تحويل البيانات للتنسيق المناسب
    foreach ($users as &$user) {
        $user['user_type'] = $user['user_type'] ?: 'user';
        $user['status'] = $user['status'] ?: 'active';
        $user['timezone'] = $user['timezone'] ?: 'Asia/Riyadh';
    }
    
    echo json_encode([
        'success' => true,
        'data' => $users,
        'count' => count($users),
        'message' => 'تم جلب المستخدمين بنجاح'
    ]);
}

function handlePost($pathSegments) {
    $input = json_decode(file_get_contents('php://input'), true);
    
    if (isset($pathSegments[1]) && $pathSegments[1] === 'assign-package') {
        // تعيين باقة للمستخدم
        $userId = $pathSegments[0];
        assignPackageToUser($userId, $input);
        return;
    }
    
    // إضافة مستخدم جديد
    $required = ['first_name', 'last_name', 'email', 'password'];
    foreach ($required as $field) {
        if (!isset($input[$field]) || empty(trim($input[$field]))) {
            echo json_encode(['success' => false, 'message' => "الحقل {$field} مطلوب"]);
            return;
        }
    }
    
    // التحقق من عدم تكرار البريد الإلكتروني
    $existingUser = fetchRow("SELECT id FROM users WHERE email = ?", [$input['email']]);
    if ($existingUser) {
        echo json_encode(['success' => false, 'message' => 'البريد الإلكتروني مستخدم مسبقاً']);
        return;
    }
    
    // إنشاء المستخدم
    $hashedPassword = password_hash($input['password'], PASSWORD_DEFAULT);
    
    $query = "
        INSERT INTO users (first_name, last_name, email, password, user_type, status, timezone, created_at) 
        VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
    ";
    
    $params = [
        $input['first_name'],
        $input['last_name'],
        $input['email'],
        $hashedPassword,
        $input['user_type'] ?? 'user',
        $input['status'] ?? 'active',
        $input['timezone'] ?? 'Asia/Riyadh'
    ];
    
    $stmt = executeQuery($query, $params);
    
    if ($stmt) {
        $userId = getLastInsertId();
        
        // تعيين باقة إذا كانت محددة
        if (!empty($input['package_id']) && !empty($input['membership_expiry'])) {
            $packageQuery = "
                INSERT INTO user_subscriptions (user_id, package_id, start_date, end_date, status, created_at)
                VALUES (?, ?, NOW(), ?, 'active', NOW())
            ";
            executeQuery($packageQuery, [$userId, $input['package_id'], $input['membership_expiry']]);
        }
        
        echo json_encode(['success' => true, 'message' => 'تم إضافة المستخدم بنجاح', 'user_id' => $userId]);
    } else {
        echo json_encode(['success' => false, 'message' => 'فشل في إضافة المستخدم']);
    }
}

function handlePut($pathSegments) {
    $input = json_decode(file_get_contents('php://input'), true);
    $userId = $input['id'] ?? null;
    
    if (!$userId) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم مطلوب']);
        return;
    }
    
    // التحقق من وجود المستخدم
    $existingUser = fetchRow("SELECT * FROM users WHERE id = ?", [$userId]);
    if (!$existingUser) {
        echo json_encode(['success' => false, 'message' => 'المستخدم غير موجود']);
        return;
    }
    
    // بناء استعلام التحديث
    $updateFields = [];
    $params = [];
    
    if (isset($input['first_name'])) {
        $updateFields[] = "first_name = ?";
        $params[] = $input['first_name'];
    }
    
    if (isset($input['last_name'])) {
        $updateFields[] = "last_name = ?";
        $params[] = $input['last_name'];
    }
    
    if (isset($input['email'])) {
        // التحقق من عدم تكرار البريد الإلكتروني
        $emailCheck = fetchRow("SELECT id FROM users WHERE email = ? AND id != ?", [$input['email'], $userId]);
        if ($emailCheck) {
            echo json_encode(['success' => false, 'message' => 'البريد الإلكتروني مستخدم مسبقاً']);
            return;
        }
        $updateFields[] = "email = ?";
        $params[] = $input['email'];
    }
    
    if (isset($input['password']) && !empty($input['password'])) {
        $updateFields[] = "password = ?";
        $params[] = password_hash($input['password'], PASSWORD_DEFAULT);
    }
    
    if (isset($input['user_type'])) {
        $updateFields[] = "user_type = ?";
        $params[] = $input['user_type'];
    }
    
    if (isset($input['status'])) {
        $updateFields[] = "status = ?";
        $params[] = $input['status'];
    }
    
    if (isset($input['timezone'])) {
        $updateFields[] = "timezone = ?";
        $params[] = $input['timezone'];
    }
    
    $updateFields[] = "updated_at = NOW()";
    $params[] = $userId;
    
    $query = "UPDATE users SET " . implode(', ', $updateFields) . " WHERE id = ?";
    
    $stmt = executeQuery($query, $params);
    
    if ($stmt) {
        // تحديث الباقة إذا كانت محددة
        if (isset($input['package_id'])) {
            // إلغاء الاشتراكات الحالية
            executeQuery("UPDATE user_subscriptions SET status = 'cancelled' WHERE user_id = ? AND status = 'active'", [$userId]);
            
            // إضافة اشتراك جديد إذا تم تحديد باقة
            if (!empty($input['package_id']) && !empty($input['membership_expiry'])) {
                $packageQuery = "
                    INSERT INTO user_subscriptions (user_id, package_id, start_date, end_date, status, created_at)
                    VALUES (?, ?, NOW(), ?, 'active', NOW())
                ";
                executeQuery($packageQuery, [$userId, $input['package_id'], $input['membership_expiry']]);
            }
        }
        
        echo json_encode(['success' => true, 'message' => 'تم تحديث المستخدم بنجاح']);
    } else {
        echo json_encode(['success' => false, 'message' => 'فشل في تحديث المستخدم']);
    }
}

function handleDelete($pathSegments) {
    $userId = $pathSegments[0] ?? null;
    
    if (!$userId) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم مطلوب']);
        return;
    }
    
    // التحقق من وجود المستخدم
    $existingUser = fetchRow("SELECT * FROM users WHERE id = ?", [$userId]);
    if (!$existingUser) {
        echo json_encode(['success' => false, 'message' => 'المستخدم غير موجود']);
        return;
    }
    
    // حذف الاشتراكات أولاً
    executeQuery("DELETE FROM user_subscriptions WHERE user_id = ?", [$userId]);
    
    // حذف المستخدم
    $stmt = executeQuery("DELETE FROM users WHERE id = ?", [$userId]);
    
    if ($stmt) {
        echo json_encode(['success' => true, 'message' => 'تم حذف المستخدم بنجاح']);
    } else {
        echo json_encode(['success' => false, 'message' => 'فشل في حذف المستخدم']);
    }
}

function handlePatch($pathSegments) {
    $userId = $pathSegments[0] ?? null;
    $action = $pathSegments[1] ?? null;
    
    if (!$userId) {
        echo json_encode(['success' => false, 'message' => 'معرف المستخدم مطلوب']);
        return;
    }
    
    if ($action === 'toggle-status') {
        $input = json_decode(file_get_contents('php://input'), true);
        $newStatus = $input['status'] ?? null;
        
        if (!in_array($newStatus, ['active', 'inactive'])) {
            echo json_encode(['success' => false, 'message' => 'حالة المستخدم غير صحيحة']);
            return;
        }
        
        $stmt = executeQuery("UPDATE users SET status = ?, updated_at = NOW() WHERE id = ?", [$newStatus, $userId]);
        
        if ($stmt) {
            echo json_encode(['success' => true, 'message' => 'تم تحديث حالة المستخدم بنجاح']);
        } else {
            echo json_encode(['success' => false, 'message' => 'فشل في تحديث حالة المستخدم']);
        }
    } elseif ($action === 'reset-count') {
        // إعادة تعيين count_s إلى 0
        $stmt = executeQuery("UPDATE users SET count_s = 0, updated_at = NOW() WHERE id = ?", [$userId]);
        
        if ($stmt) {
            echo json_encode(['success' => true, 'message' => 'تم إعادة تعيين العداد بنجاح']);
        } else {
            echo json_encode(['success' => false, 'message' => 'فشل في إعادة تعيين العداد']);
        }
    } else {
        echo json_encode(['success' => false, 'message' => 'إجراء غير مدعوم']);
    }
}

function assignPackageToUser($userId, $input) {
    if (!isset($input['package_id']) || !isset($input['expiry_date'])) {
        echo json_encode(['success' => false, 'message' => 'بيانات الباقة غير كاملة']);
        return;
    }
    
    // التحقق من وجود المستخدم والباقة
    $user = fetchRow("SELECT * FROM users WHERE id = ?", [$userId]);
    $package = fetchRow("SELECT * FROM packages WHERE id = ?", [$input['package_id']]);
    
    if (!$user) {
        echo json_encode(['success' => false, 'message' => 'المستخدم غير موجود']);
        return;
    }
    
    if (!$package) {
        echo json_encode(['success' => false, 'message' => 'الباقة غير موجودة']);
        return;
    }
    
    // إلغاء الاشتراكات الحالية
    executeQuery("UPDATE user_subscriptions SET status = 'cancelled' WHERE user_id = ? AND status = 'active'", [$userId]);
    
    // إضافة اشتراك جديد
    $stmt = executeQuery("
        INSERT INTO user_subscriptions (user_id, package_id, start_date, end_date, status, created_at)
        VALUES (?, ?, NOW(), ?, 'active', NOW())
    ", [$userId, $input['package_id'], $input['expiry_date']]);
    
    if ($stmt) {
        echo json_encode(['success' => true, 'message' => 'تم تعيين الباقة بنجاح']);
    } else {
        echo json_encode(['success' => false, 'message' => 'فشل في تعيين الباقة']);
    }
}

function createUsersTable() {
    $createUsersTable = "
        CREATE TABLE IF NOT EXISTS users (
            id INT AUTO_INCREMENT PRIMARY KEY,
            first_name VARCHAR(100) NOT NULL,
            last_name VARCHAR(100) NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            password VARCHAR(255) NOT NULL,
            user_type ENUM('admin', 'user') DEFAULT 'user',
            status ENUM('active', 'inactive') DEFAULT 'active',
            timezone VARCHAR(100) DEFAULT 'Asia/Riyadh',
            count_s INT DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        )
    ";
    
    $createSubscriptionsTable = "
        CREATE TABLE IF NOT EXISTS user_subscriptions (
            id INT AUTO_INCREMENT PRIMARY KEY,
            user_id INT NOT NULL,
            package_id INT NOT NULL,
            start_date DATE NOT NULL,
            end_date DATE NOT NULL,
            status ENUM('active', 'expired', 'cancelled') DEFAULT 'active',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
            FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE CASCADE
        )
    ";
    
    try {
        executeQuery($createUsersTable);
        executeQuery($createSubscriptionsTable);
        
        // إضافة عمود count_s إذا لم يكن موجوداً
        try {
            executeQuery("ALTER TABLE users ADD COLUMN count_s INT DEFAULT 0");
        } catch (Exception $e) {
            // العمود موجود مسبقاً
        }
        
        // إدراج مدير افتراضي فقط عند تهيئة كلمة مرور صريحة من البيئة
        $adminExists = fetchRow("SELECT id FROM users WHERE email = 'admin@example.com'");
        $defaultAdminPassword = configValue('DEFAULT_ADMIN_PASSWORD', '');
        if (!$adminExists && $defaultAdminPassword !== '') {
            $hashedPassword = password_hash($defaultAdminPassword, PASSWORD_DEFAULT);
            executeQuery("
                INSERT INTO users (first_name, last_name, email, password, user_type, status, timezone, count_s)
                VALUES ('مدير', 'النظام', 'admin@example.com', ?, 'admin', 'active', 'Asia/Riyadh', 0)
            ", [$hashedPassword]);
        } elseif (!$adminExists) {
            error_log('Default admin creation skipped: set DEFAULT_ADMIN_PASSWORD to bootstrap admin@example.com');
        }
    } catch (Exception $e) {
        error_log("Error creating users table: " . $e->getMessage());
    }
}
?>
