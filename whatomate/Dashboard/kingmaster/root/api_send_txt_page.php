<?php

function sendMessage($recipientId, $message, $token) {
    $url = "https://graph.facebook.com/v19.0/me/messages?access_token=" . urlencode($token);

    $payload = [
        //"messaging_type" => "MESSAGE_TAG", // تم تفعيله للضرورة
        //"tag" => "CUSTOMER_FEEDBACK",
        "messaging_type" => "UTILITY",
        "recipient" => ["id" => $recipientId],
        "message" => ["text" => $message],
    ];

    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => ["Content-Type: application/json"],
        CURLOPT_POSTFIELDS => json_encode($payload, JSON_UNESCAPED_UNICODE)
    ]);

    $response = curl_exec($ch);
             
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
                //file_put_contents(__DIR__ . '/a_b_d.log', date('c') . " - Code - " . json_encode($httpCode) . "\n", //FILE_APPEND);
    curl_close($ch);

    // فك التشفير لمعالجة الخطأ 551
    $resData = json_decode($response, true);
    $fbCode = $resData['error']['code'] ?? 0;

    // نجاح إذا كان الكود 200 أو الخطأ 551 (مستخدم غير متاح)
    return ($httpCode === 200 || $fbCode === 551);
}

$token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$txt = $_GET['txt'] ?? null;

header('Content-Type: application/json');

if (sendMessage($id, $txt, $token)) {
    echo json_encode([
        'success' => true,
        "id" => $id, // إضافة id مباشر
        "response" => ["id" => $id] 
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
} else {
    echo json_encode([
        'success' => false, // تغيير error لـ success: false ليفهمها الـ UI
        'id' => $id,
        "response" => ["id" => $id]
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
}