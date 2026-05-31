<?php
require_once __DIR__ . '/config/database.php';

startSecureSession();
applySecurityHeaders();

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respondError('طريقة الطلب غير صحيحة', 405);
}

verifyCsrfToken();
enforceRateLimit('login', 5, 300);

function addLog($user_id, $action) {
    $db = getDB();
    $stmt = $db->prepare('INSERT INTO logs (user_id, action) VALUES (:user_id, :action)');
    $stmt->execute([':user_id' => $user_id, ':action' => $action]);
    return $db->lastInsertId();
}

function addLoginNotification($user_id) {
    try {
        $stmt = getDB()->prepare("
            INSERT INTO notifications (user_id, title, message, type, is_read, created_at)
            VALUES (:user_id, :title, :message, :type, 0, NOW())
        ");
        $stmt->execute([
            ':user_id' => $user_id,
            ':title' => 'اشعار امان',
            ':message' => 'تسجيل دخول جديد',
            ':type' => 'info',
        ]);
    } catch (Throwable $e) {
        error_log('Login notification error: ' . $e->getMessage());
    }
}

$email = filter_var($_POST['email'] ?? '', FILTER_SANITIZE_EMAIL);
$password = $_POST['password'] ?? '';
$terms = isset($_POST['terms']) && $_POST['terms'] === 'on';

if ($email === '' || $password === '') {
    respondError('جميع الحقول مطلوبة', 400);
}
if (!$terms) {
    respondError('يجب الموافقة على الشروط والأحكام', 400);
}
if (!isValidEmail($email)) {
    respondError('البريد الإلكتروني غير صحيح', 400);
}

try {
    $user = fetchRow(
        'SELECT user_id, email, password, first_name, last_name, is_verified, is_admin, phone, otp FROM users WHERE email = ? LIMIT 1',
        [$email]
    );

    if (!$user || !verifyPassword($password, $user['password'])) {
        error_log('Failed login for ' . $email . ' from ' . clientIpAddress());
        respondError('البريد الإلكتروني أو كلمة المرور غير صحيحة', 401);
    }

    session_regenerate_id(true);
    $_SESSION['user_id'] = $user['user_id'];
    $_SESSION['email'] = $user['email'];
    $_SESSION['first_name'] = $user['first_name'];
    $_SESSION['last_name'] = $user['last_name'];
    $_SESSION['is_admin'] = $user['is_admin'];
    $_SESSION['is_logged_in'] = true;
    $_SESSION['csrf_token'] = bin2hex(random_bytes(32));

    addLog($user['user_id'], 'تسجيل دخول');
    addLoginNotification($user['user_id']);

    respondJson(['success' => true, 'message' => 'تم تسجيل الدخول بنجاح', 'redirect' => 'index.php']);
} catch (Throwable $e) {
    error_log('Login Error: ' . $e->getMessage());
    respondError('حدث خطأ أثناء تسجيل الدخول', 500);
}
