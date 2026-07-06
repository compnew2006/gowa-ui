<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');

session_start();
if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح'], JSON_UNESCAPED_UNICODE);
    exit;
}

/**
 * Simple JSON POST helper (calls your api.php wrapper)
 */
function callApiPhp(array $postData): array
{
    $api_url = "https://apis.kingmaster.info/api.php";

    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $api_url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
    curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($postData, JSON_UNESCAPED_UNICODE));
    curl_setopt($ch, CURLOPT_TIMEOUT, 30);
    curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);

    $response = curl_exec($ch);

    if (curl_errno($ch)) {
        $error = curl_error($ch);
        curl_close($ch);
        return [
            'success' => false,
            'message' => 'خطأ في الاتصال: ' . $error
        ];
    }

    $http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    $data = json_decode($response, true);
    if (!is_array($data)) {
        return [
            'success' => false,
            'message' => 'استجابة غير صحيحة من الخادم',
            'http_code' => $http_code,
            'raw' => $response
        ];
    }

    // attach http_code for debugging if needed
    $data['_http_code'] = $http_code;
    return $data;
}

/**
 * Get phone number + pushname from Evolution via api.php wrapper
 */
function getPhoneNumber(string $session_id): array
{
    return callApiPhp([
        "action"  => "get_api__session__get_phone_number",
        "session" => $session_id
    ]);
}

// ---- Input ----
$account_uid = isset($_GET['account_uid']) ? trim($_GET['account_uid']) : '';
if ($account_uid === '') {
    echo json_encode(['success' => false, 'message' => 'معرف الحساب مطلوب'], JSON_UNESCAPED_UNICODE);
    exit;
}

try {
    // 1) Check connection using api.php wrapper
    $data = callApiPhp([
        "action"  => "check_connection",
        "session" => $account_uid
    ]);

    // 2) Determine connected state (supports both old + new shapes)
    $isConnected =
        (isset($data['connected']) && strtolower((string)$data['connected']) === 'connected') || // old style
        (isset($data['status']) && strtoupper((string)$data['status']) === 'CONNECTED') ||
        (isset($data['state']) && strtolower((string)$data['state']) === 'open') ||
        (isset($data['data']['instance']['state']) && strtolower((string)$data['data']['instance']['state']) === 'open');

    if ($isConnected) {
        // 3) Fetch phone number
        $result = getPhoneNumber($account_uid);

        $phone = $result['phoneNumber'] ?? 'غير محدد';
        $pushname = $result['pushname'] ?? 'غير معروف';

        echo json_encode([
            'success'   => true,
            'status_QR' => 'success',
            'phone'     => $pushname . " - " . $phone,
            'message'   => 'الحساب نشط ومتصل بنجاح'
        ], JSON_UNESCAPED_UNICODE);
        exit;
    }

    // Not connected
    echo json_encode([
        'success'   => false,
        'status_QR' => $data['status_QR'] ?? 'failed',
        'message'   => 'الجلسة غير نشطة أو منتهية الصلاحية'
    ], JSON_UNESCAPED_UNICODE);
    exit;

} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
    exit;
}
