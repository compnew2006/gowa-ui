<?php
$configPath = is_file(__DIR__ . '/config/database.php')
    ? __DIR__ . '/config/database.php'
    : __DIR__ . '/../config/database.php';
require_once $configPath;

applyCorsHeaders('GET, POST, OPTIONS');

if (configValue('PROXY_REQUIRE_AUTH', '1') !== '0') {
    requireAuthenticatedUser();
}

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
if (!in_array($method, ['GET', 'POST'], true)) {
    respondError('Method not allowed', 405);
}

$contentLength = (int)($_SERVER['CONTENT_LENGTH'] ?? 0);
if ($contentLength > (int)configValue('PROXY_MAX_BYTES', '1048576')) {
    respondError('حجم الطلب كبير جداً', 413);
}

$apiUrl = rtrim(configValue('PROXY_TARGET_URL', 'https://apis.kingmaster.info/api.php'), '?');
if (!preg_match('#^https://#i', $apiUrl)) {
    respondError('Proxy target must use HTTPS', 500);
}

if (!empty($_SERVER['QUERY_STRING'])) {
    $apiUrl .= '?' . $_SERVER['QUERY_STRING'];
}

if (!function_exists('curl_init')) {
    respondError('cURL غير متاح على الخادم', 500);
}

$payload = file_get_contents('php://input') ?: '';
$contentType = $_SERVER['CONTENT_TYPE'] ?? 'application/json';
$allowedContentTypes = ['application/json', 'application/x-www-form-urlencoded', 'multipart/form-data'];
$forwardContentType = 'application/json';
foreach ($allowedContentTypes as $allowed) {
    if (stripos($contentType, $allowed) === 0) {
        $forwardContentType = $contentType;
        break;
    }
}

$ch = curl_init($apiUrl);
curl_setopt_array($ch, [
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_CUSTOMREQUEST => $method,
    CURLOPT_CONNECTTIMEOUT => (int)configValue('PROXY_CONNECT_TIMEOUT', '5'),
    CURLOPT_TIMEOUT => (int)configValue('PROXY_TIMEOUT', '20'),
    CURLOPT_FOLLOWLOCATION => false,
    CURLOPT_SSL_VERIFYPEER => true,
    CURLOPT_SSL_VERIFYHOST => 2,
    CURLOPT_HTTPHEADER => ['Content-Type: ' . $forwardContentType],
]);

if (defined('CURLOPT_PROTOCOLS') && defined('CURLPROTO_HTTPS')) {
    curl_setopt($ch, CURLOPT_PROTOCOLS, CURLPROTO_HTTPS);
}
if (defined('CURLOPT_REDIR_PROTOCOLS') && defined('CURLPROTO_HTTPS')) {
    curl_setopt($ch, CURLOPT_REDIR_PROTOCOLS, CURLPROTO_HTTPS);
}
if ($method === 'POST') {
    curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);
}

$response = curl_exec($ch);
$httpCode = (int)curl_getinfo($ch, CURLINFO_HTTP_CODE);
$curlError = curl_error($ch);
curl_close($ch);

if ($response === false) {
    error_log('Proxy cURL error: ' . $curlError);
    respondError('تعذر الاتصال بالخدمة الخارجية', 502);
}

http_response_code($httpCode > 0 ? $httpCode : 502);
header('Content-Type: application/json; charset=utf-8');
header('X-Content-Type-Options: nosniff');
echo $response;
