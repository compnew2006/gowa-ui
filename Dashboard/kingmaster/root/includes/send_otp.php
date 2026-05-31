<?php
/**
 * إرسال OTP عبر WhatsApp API
 */

// إعدادات API
define('WHATSAPP_API_URL', 'https://king-master.pro/api/send');
define('WHATSAPP_INSTANCE_ID', '6926C9A5115D9');
define('WHATSAPP_ACCESS_TOKEN', '6604ac2316788');

/**
 * إرسال رسالة OTP عبر WhatsApp
 * 
 * @param string $phone رقم الهاتف (مع كود الدولة بدون +)
 * @param string $otp رمز التحقق
 * @return array النتيجة (success و message)
 */
function sendOTP($phone, $otp) {
    // تنظيف رقم الهاتف من أي رموز
  
    // التأكد من وجود كود الدولة
 
    // إنشاء نص الرسالة
    $message = "🔐 *رمز التحقق الخاص بك*\n\n";
    $message .= "الرمز: *{$otp}*\n\n";
    $message .= "⏰ هذا الرمز صالح لمدة 15 دقيقة\n";
    $message .= "⚠️ لا تشارك هذا الرمز مع أي شخص\n\n";
    $message .= "شكراً لاستخدامك Kingmaster 👑";
    
 
    
    try {
        // إرسال الطلب
$payload = json_encode([
    "number" => $phone,
    "type" => "text",
    "message" => $message,
    "instance_id" => "6967AAB9ADA6E",
        "access_token" => "6604ac2316788"
]);

$ch = curl_init();
curl_setopt($ch, CURLOPT_URL, "https://king-master.pro/api/send");
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);

curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Content-Type: application/json',
    'Content-Length: ' . strlen($payload)
]);

curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);

$response = curl_exec($ch);
curl_close($ch);

        // فك تشفير الاستجابة
        $data = json_decode($response, true);
        
        // التحقق من نجاح الإرسال
        if (isset($data['status']) && $data['status'] == 'success') {
            return [
                'success' => true,
                'message' => 'تم إرسال رمز التحقق بنجاح'
            ];
        } else {
            error_log("WhatsApp API Error Response: " . $response);
            return [
                'success' => false,
                'message' => 'فشل إرسال رمز التحقق'
            ];
        }
        
    } catch (Exception $e) {
        error_log("WhatsApp API Exception: " . $e->getMessage());
        return [
            'success' => false,
            'message' => 'حدث خطأ أثناء إرسال الرمز'
        ];
    }
}

/**
 * إرسال رسالة مخصصة عبر WhatsApp
 * 
 * @param string $phone رقم الهاتف
 * @param string $message نص الرسالة
 * @return array النتيجة
 */
function sendWhatsAppMessage($phone, $message) {
    // تنظيف رقم الهاتف
 
    $url = WHATSAPP_API_URL . '?' . http_build_query([
        'number' => $phone,
        'type' => 'text',
        'message' => $message,
        'instance_id' => WHATSAPP_INSTANCE_ID,
        'access_token' => WHATSAPP_ACCESS_TOKEN
    ]);
    
    try {
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 30);
        curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, true);
        
        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);
        
        $data = json_decode($response, true);
        
        if ($httpCode == 200 && isset($data['status']) && $data['status'] == 'success') {
            return ['success' => true, 'message' => 'تم الإرسال بنجاح'];
        }
        
        return ['success' => false, 'message' => 'فشل الإرسال'];
        
    } catch (Exception $e) {
        return ['success' => false, 'message' => 'حدث خطأ'];
    }
}
?>
