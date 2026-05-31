<?php
/**
 * King Master - Database Configuration
 * ملف إعدادات قاعدة البيانات
 */

function configValue($key, $default = '') {
    $value = getenv($key);
    if ($value === false || $value === '') {
        $value = $_ENV[$key] ?? $_SERVER[$key] ?? $default;
    }
    return $value;
}

define('APP_ENV', configValue('APP_ENV', 'production'));
define('DB_HOST', configValue('DB_HOST', '127.0.0.1'));
define('DB_NAME', configValue('DB_NAME', 'kingmaster'));
define('DB_USER', configValue('DB_USER', 'kingmaster'));
define('DB_PASS', configValue('DB_PASS', ''));
define('DB_CHARSET', configValue('DB_CHARSET', 'utf8mb4'));
define('DB_PORT', configValue('DB_PORT', '3306'));

function isProductionEnv() {
    return strtolower((string)APP_ENV) === 'production';
}

function respondJson($payload, $statusCode = 200) {
    if (!headers_sent()) {
        http_response_code($statusCode);
        header('Content-Type: application/json; charset=utf-8');
        header('Cache-Control: no-store');
        applySecurityHeaders();
    }
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function applySecurityHeaders() {
    if (headers_sent()) {
        return;
    }
    header('X-Content-Type-Options: nosniff');
    header('X-Frame-Options: DENY');
    header('X-XSS-Protection: 1; mode=block');
    header('Referrer-Policy: strict-origin-when-cross-origin');
    header('Permissions-Policy: camera=(), microphone=(), geolocation=()');
    if (configValue('APP_ENABLE_CSP', '1') !== '0') {
        $csp = configValue('APP_CSP', "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; object-src 'none'; img-src 'self' data: https:; font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com data:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; script-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; connect-src 'self'");
        header('Content-Security-Policy: ' . $csp);
    }
}

function respondError($message, $statusCode = 400) {
    respondJson(['success' => false, 'message' => $message], $statusCode);
}

function applyCorsHeaders($methods = 'GET, POST, OPTIONS') {
    if (headers_sent()) {
        return;
    }

    $origin = $_SERVER['HTTP_ORIGIN'] ?? '';
    $allowed = array_filter(array_map('trim', explode(',', configValue('APP_ALLOWED_ORIGINS', ''))));

    if ($origin !== '' && (in_array($origin, $allowed, true) || in_array('*', $allowed, true))) {
        header('Access-Control-Allow-Origin: ' . $origin);
        header('Vary: Origin');
        header('Access-Control-Allow-Credentials: true');
        header('Access-Control-Allow-Headers: Content-Type, X-Requested-With, X-CSRF-Token');
        header('Access-Control-Allow-Methods: ' . $methods);
    }

    if (($_SERVER['REQUEST_METHOD'] ?? '') === 'OPTIONS') {
        http_response_code(204);
        exit;
    }
}

function readJsonBody($maxBytes = 1048576) {
    $length = (int)($_SERVER['CONTENT_LENGTH'] ?? 0);
    if ($length > $maxBytes) {
        respondError('حجم الطلب كبير جداً', 413);
    }

    $raw = file_get_contents('php://input');
    if ($raw === false || trim($raw) === '') {
        return [];
    }

    $data = json_decode($raw, true);
    if (!is_array($data) || json_last_error() !== JSON_ERROR_NONE) {
        respondError('JSON غير صالح', 400);
    }
    return $data;
}

function startSecureSession() {
    if (session_status() === PHP_SESSION_ACTIVE) {
        return;
    }

    $secure = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') || configValue('SESSION_COOKIE_SECURE', '') === '1';
    session_set_cookie_params([
        'lifetime' => 0,
        'path' => '/',
        'domain' => '',
        'secure' => $secure,
        'httponly' => true,
        'samesite' => 'Lax',
    ]);
    session_start();
}

function csrfToken() {
    startSecureSession();
    if (empty($_SESSION['csrf_token'])) {
        $_SESSION['csrf_token'] = bin2hex(random_bytes(32));
    }
    return $_SESSION['csrf_token'];
}

function csrfInput() {
    return '<input type="hidden" name="csrf_token" value="' . htmlspecialchars(csrfToken(), ENT_QUOTES, 'UTF-8') . '">';
}

function verifyCsrfToken($token = null) {
    startSecureSession();
    $token = $token ?? ($_POST['csrf_token'] ?? ($_SERVER['HTTP_X_CSRF_TOKEN'] ?? ''));
    if (empty($_SESSION['csrf_token']) || !is_string($token) || !hash_equals($_SESSION['csrf_token'], $token)) {
        respondError('رمز الحماية غير صالح، أعد تحميل الصفحة', 419);
    }
}

function clientIpAddress() {
    $ip = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
    if (!filter_var($ip, FILTER_VALIDATE_IP)) {
        return 'unknown';
    }
    return $ip;
}

function enforceRateLimit($scope, $limit, $windowSeconds) {
    $scope = preg_replace('/[^A-Za-z0-9_.:-]/', '_', (string)$scope);
    $key = hash('sha256', $scope . '|' . clientIpAddress() . '|' . session_id());
    $dir = sys_get_temp_dir() . DIRECTORY_SEPARATOR . 'kingmaster_rate_limits';
    if (!is_dir($dir)) {
        @mkdir($dir, 0700, true);
    }

    $file = $dir . DIRECTORY_SEPARATOR . $key . '.json';
    $now = time();
    $entry = ['start' => $now, 'count' => 0];
    $handle = @fopen($file, 'c+');
    if (!$handle) {
        return;
    }

    flock($handle, LOCK_EX);
    $contents = stream_get_contents($handle);
    if ($contents !== false && trim($contents) !== '') {
        $decoded = json_decode($contents, true);
        if (is_array($decoded)) {
            $entry = $decoded;
        }
    }

    if (($now - (int)($entry['start'] ?? 0)) >= $windowSeconds) {
        $entry = ['start' => $now, 'count' => 0];
    }

    $entry['count'] = (int)$entry['count'] + 1;
    ftruncate($handle, 0);
    rewind($handle);
    fwrite($handle, json_encode($entry));
    fflush($handle);
    flock($handle, LOCK_UN);
    fclose($handle);

    if ($entry['count'] > $limit) {
        respondError('طلبات كثيرة جداً، حاول لاحقاً', 429);
    }
}

function requireAuthenticatedUser() {
    startSecureSession();
    if (empty($_SESSION['user_id'])) {
        respondError('غير مصرح', 401);
    }
    return $_SESSION['user_id'];
}

function requireAdminUser() {
    $userId = requireAuthenticatedUser();
    $pdo = getDB();
    $columns = [];
    try {
        foreach ($pdo->query('SHOW COLUMNS FROM users') as $column) {
            $columns[] = $column['Field'];
        }
    } catch (Throwable $e) {
        error_log('Unable to inspect users columns: ' . $e->getMessage());
    }

    $select = [];
    $select[] = in_array('is_admin', $columns, true) ? 'is_admin' : 'NULL AS is_admin';
    $select[] = in_array('user_type', $columns, true) ? 'user_type' : 'NULL AS user_type';
    $stmt = $pdo->prepare('SELECT ' . implode(', ', $select) . ' FROM users WHERE user_id = ? OR id = ? LIMIT 1');
    $stmt->execute([$userId, $userId]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$user || ((string)($user['is_admin'] ?? '') !== '1' && strtolower((string)($user['user_type'] ?? '')) !== 'admin')) {
        respondError('صلاحيات المدير مطلوبة', 403);
    }
    return $userId;
}

function safeBaseName($name, $fallback = 'file') {
    $name = trim((string)$name);
    $name = str_replace(["\0", '/', '\\'], '', $name);
    $name = preg_replace('/[^\pL\pN._ -]+/u', '_', $name);
    $name = trim($name, ". \t\n\r\0\x0B");
    return $name !== '' ? $name : $fallback;
}

function cleanText($value, $maxLength = 255) {
    $value = trim((string)$value);
    $value = strip_tags($value);
    $value = preg_replace('/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/u', '', $value);
    if (function_exists('mb_substr')) {
        return mb_substr($value, 0, $maxLength, 'UTF-8');
    }
    return substr($value, 0, $maxLength);
}

function requireJsonArray($data, $message = 'بيانات غير صالحة') {
    if (!is_array($data)) {
        respondError($message, 400);
    }
    return $data;
}

/**
 * كلاس الاتصال بقاعدة البيانات
 */
class Database {
    private $host = DB_HOST;
    private $db_name = DB_NAME;
    private $username = DB_USER;
    private $password = DB_PASS;
    private $charset = DB_CHARSET;
    private $pdo;

    /**
     * الاتصال بقاعدة البيانات
     */
    public function connect() {
        if ($this->pdo == null) {
            try {
                $dsn = "mysql:host=" . $this->host . ";port=" . DB_PORT . ";dbname=" . $this->db_name . ";charset=" . $this->charset;
                
                $options = [
                    PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
                    PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
                    PDO::ATTR_EMULATE_PREPARES => false,
                    PDO::ATTR_TIMEOUT => (int)configValue('DB_TIMEOUT', '5'),
                    PDO::MYSQL_ATTR_INIT_COMMAND => "SET NAMES utf8mb4"
                ];

                $this->pdo = new PDO($dsn, $this->username, $this->password, $options);
                
            } catch(PDOException $exception) {
                error_log("Database Connection Error: " . $exception->getMessage());
                respondError('تعذر الاتصال بقاعدة البيانات', 500);
            }
        }

        return $this->pdo;
    }

    /**
     * إغلاق الاتصال
     */
    public function disconnect() {
        $this->pdo = null;
    }

    /**
     * الحصول على اتصال مباشر
     */
    public static function getInstance() {
        static $instance = null;
        if ($instance === null) {
            $instance = new self();
        }
        return $instance->connect();
    }
}

/**
 * دالة مساعدة للحصول على اتصال قاعدة البيانات
 */
function getDB() {
    return Database::getInstance();
}

/**
 * دالة لتنفيذ استعلام آمن
 */
function executeQuery($query, $params = []) {
    try {
        $db = getDB();
        $stmt = $db->prepare($query);
        $stmt->execute($params);
        return $stmt;
    } catch (PDOException $e) {
        error_log("Database Error: " . $e->getMessage());
        return false;
    }
}

/**
 * دالة للحصول على صف واحد
 */
function fetchRow($query, $params = []) {
    $stmt = executeQuery($query, $params);
    return $stmt ? $stmt->fetch() : false;
}

/**
 * دالة للحصول على جميع الصفوف
 */
function fetchAll($query, $params = []) {
    $stmt = executeQuery($query, $params);
    return $stmt ? $stmt->fetchAll() : false;
}

/**
 * دالة للحصول على عدد الصفوف المتأثرة
 */
function getRowCount($query, $params = []) {
    $stmt = executeQuery($query, $params);
    return $stmt ? $stmt->rowCount() : 0;
}

/**
 * دالة للحصول على آخر ID تم إدراجه
 */
function getLastInsertId() {
    return getDB()->lastInsertId();
}

/**
 * تنظيف البيانات من الـ XSS
 */
function sanitizeInput($data) {
    if (is_array($data)) {
        return array_map('sanitizeInput', $data);
    }
    return htmlspecialchars(trim($data), ENT_QUOTES, 'UTF-8');
}

/**
 * التحقق من صحة البريد الإلكتروني
 */
function isValidEmail($email) {
    return filter_var($email, FILTER_VALIDATE_EMAIL) !== false;
}

/**
 * إنشاء رمز عشوائي آمن
 */
function generateSecureToken($length = 32) {
    return bin2hex(random_bytes($length));
}

/**
 * تشفير كلمة المرور
 */
function hashPassword($password) {
    // استخدام ARGON2ID إذا كان متاحًا، وإلا استخدام BCRYPT
    if (defined('PASSWORD_ARGON2ID')) {
        return password_hash($password, PASSWORD_ARGON2ID);
    }
    return password_hash($password, PASSWORD_BCRYPT, ['cost' => 12]);
}

/**
 * التحقق من كلمة المرور
 */
function verifyPassword($password, $hash) {
    return password_verify($password, $hash);
}
?>
