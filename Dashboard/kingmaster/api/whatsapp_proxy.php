<?php
header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST');
header('Access-Control-Allow-Headers: Content-Type');

session_start();

if (!isset($_SESSION['user_id'])) {
    echo json_encode(['success' => false, 'message' => 'غير مصرح']);
    exit;
}



function generateCode() {
    // timestamp الحالي
    $timestamp = time();

    // مجموعة الحروف والأرقام الممكن استخدامها
    $characters = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
    $randomString = '';

    // نختار 5 أحرف/أرقام عشوائية
    for ($i = 0; $i < 5; $i++) {
        $index = rand(0, strlen($characters) - 1);
        $randomString .= $characters[$index];
    }

    // دمج الـ timestamp مع الـ random string
    return $timestamp . $randomString;
}

// اختبار الدالة
 

$WAAPI_KEY = 'VV9D6WL23X';
$WAAPI_BASE_URL = 'https://apis.kingmaster.info/api.php';

$action = $_GET['action'] ?? '';
$instance_id = generateCode();

try {
    switch ($action) {
        case 'create_instance':
            // إنشاء instance جديد
           $response = '{"status":"success","message":"Instance ID generated successfully","instance_id":"'.$instance_id.'"}';

        
            echo $response;
            break;
            
        case 'get_qrcode':
            // جلب QR Code
            if (empty($instance_id)) {
                echo json_encode(['success' => false, 'message' => 'instance_id مطلوب']);
                exit;
            }
            
            $url = "{$WAAPI_BASE_URL}/api.php?action=start_session&session={$instance_id}";

            // بيانات الـ POST
            $data = [
                "webhook" => "https://apis.kingmaster.info/webhook.php",
                "waitQrCode" => true
            ];

            // تحويل المصفوفة إلى JSON
            $jsonData = json_encode($data);

            // إعدادات الـ POST request
            $options = [
                "http" => [
                    "header"  => "Content-Type: application/json\r\n",
                    "method"  => "POST",
                    "content" => $jsonData,
                    "ignore_errors" => true
                ]
            ];

            $context  = stream_context_create($options);
            $response = file_get_contents($url, false, $context);

            if ($response === false) {
                throw new Exception('فشل جلب QR Code');
            }
            
            echo $response;
            break;
             case 'getnum':
            // جلب QR Code
            if (empty($instance_id)) {
                echo json_encode(['success' => false, 'message' => 'instance_id مطلوب']);
                exit;
            }
            
            $url = "{$WAAPI_BASE_URL}/get_phone_number.php?key={$WAAPI_KEY}&instance_id={$instance_id}";
            $response = file_get_contents($url);

            if ($response === false) {
                throw new Exception('فشل جلب QR Code');
            }
            
            echo $response;
            break;
            
        default:
            echo json_encode(['success' => false, 'message' => 'إجراء غير صالح']);
            break;
    }
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => $e->getMessage()
    ]);
}
