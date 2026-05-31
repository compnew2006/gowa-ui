<?php

function sendImage($recipientId, $imageUrl, $token) {
    $url = "https://graph.facebook.com/v17.0/me/messages?access_token=" . urlencode($token);

    $payload = [
        "recipient" => [
            "id" => $recipientId
        ],
        "message" => [
            "attachments" => [
                [
                    "type" => "image",
                    "payload" => [
                        "url" => $imageUrl,
                        "is_reusable" => true
                    ]
                ]
            ]
        ],
        "messaging_type" => "UTILITY"
        //"tag" => "CUSTOMER_FEEDBACK"
    ];

    $ch = curl_init($url);

    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => [
            "Content-Type: application/json"
        ],
        CURLOPT_POSTFIELDS => json_encode($payload, JSON_UNESCAPED_UNICODE)
    ]);

    $response = curl_exec($ch);

    if (curl_errno($ch)) {
        curl_close($ch);
        return false;
    }

    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    return ($httpCode === 200 && !empty($response));
}

// استقبال البيانات من GET
$token = $_GET['token'] ?? '';
$id = $_GET['id'] ?? '';
$imageUrl = $_GET['imageUrl'] ?? '';

header('Content-Type: application/json; charset=UTF-8');

if (sendImage($id, $imageUrl, $token)) {
    echo json_encode([
        'success' => true,
        'response' => [
            'id' => $id
        ]
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
} else {
    echo json_encode([
        'success' => false,
        'response' => [
            'id' => $id
        ]
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
}
