<?php
// proxy.php
header("Access-Control-Allow-Origin: *");
header("Access-Control-Allow-Methods: POST, GET, OPTIONS");
header("Access-Control-Allow-Headers: Content-Type, Authorization");

// للتعامل مع طلبات OPTIONS الخاصة بالـ CORS
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(200);
    exit;
}

// رابط الـ API الخارجي
$apiUrl = "https://apis.kingmaster.info/api.php";

// ضم الاستعلامات الموجودة في الـ URL
if (!empty($_SERVER['QUERY_STRING'])) {
    $apiUrl .= '?' . $_SERVER['QUERY_STRING'];
}

// جلب بيانات الـ POST إذا موجودة
$payload = file_get_contents('php://input');

$ch = curl_init($apiUrl);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $_SERVER['REQUEST_METHOD']);
if ($payload) {
    curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);
}

// لو عايز تبعت نفس الهيدرز
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    "Content-Type: application/json",
]);

$response = curl_exec($ch);
$httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);

// رجّع الرد كما هو
http_response_code($httpCode);
echo $response;
