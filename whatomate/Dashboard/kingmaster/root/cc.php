<?php
 
function native_sso_approve($fb_dtsg, $t2_pass, $cookie)
{
    $url = "https://m.facebook.com/login/native_sso/approved/?flow&extra_data=%7B%22after_login%22%3Afalse%2C%22auto%22%3Afalse%7D";

    // POST DATA
    $postData = http_build_query([
        "fb_dtsg" => $fb_dtsg,
        "jazoest" => "25581",
        "app_id" => "350685531728",
        "token" => $t2_pass,
        "action_login" => "auto",
        "action_login" => "" // زي VB بالظبط (مكرر)
    ]);

    $headers = [
        "Content-Type: application/x-www-form-urlencoded",
        "User-Agent: Mozilla/5.0 (Linux; Android 9; ASUS_I005DA Build/PI) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/68.0.3440.70 Mobile Safari/537.36",
        "Cookie: $cookie"
    ];

    $ch = curl_init();

    curl_setopt_array($ch, [
        CURLOPT_URL => $url,
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $postData,
        CURLOPT_HTTPHEADER => $headers,
        CURLOPT_HEADER => true,        // مهم جداً عشان نقرأ الهيدر
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_FOLLOWLOCATION => false // ❌ ممنوع الريدايركت عشان هنقرأ Location
    ]);

    $response = curl_exec($ch);

    if ($response === false) {
        return "Curl Error: " . curl_error($ch);
    }

    // استخراج الهيدر فقط
    $header_size = curl_getinfo($ch, CURLINFO_HEADER_SIZE);
    $headers_text = substr($response, 0, $header_size);

    curl_close($ch);

    // استخراج blob من الهيدر
    if (preg_match('/blob=([^&]+)/', $headers_text, $m)) {
        return $m[1];
    }

    return "blob not found";
}





$data = "locale=ar_AR&client_country_code=EG&method=GET&fb_api_req_friendly_name=browser_to_native_sso_token_fetch&fb_api_caller_class=BrowserToNativeSSOQueue&access_token=256002347743983%7C374e60f8b9bb6b8cbb30f78030438895";

$ch = curl_init();

curl_setopt($ch, CURLOPT_URL, "https://b-graph.facebook.com/browser_to_native_sso_token_fetch?fields=t1,t2");
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, $data);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

$headers = [
    "Authorization: OAuth null",
    "Content-Type: application/x-www-form-urlencoded",
    "User-Agent: Dalvik/2.1.0 (Linux; U; Android 9; ASUS_I005DA Build/PI) [FBAN/Orca-Android;FBAV/391.2.0.20.404;FBPN/com.facebook.orca;FBLC/ar_EG;FBBV/437533963;FBCR/Vodafone-Mirsfone;FBMF/Asus;FBBD/Asus;FBDV/ASUS_I005DA;FBSV/9;FBCA/x86:armeabi-v7a;FBDM/{density=1.5,width=720,height=1280};FB_FW/1;]",
    "Referer: https://b-graph.facebook.com"
];

curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);

$response = curl_exec($ch);
curl_close($ch);

// استخراج t1_email
preg_match('/{"t1":"(.+?)"/', $response, $match1);
$t1_email = $match1[1] ?? "";

// استخراج t2_pass
preg_match('/"t2":"(.+?)"}/', $response, $match2);
$t2_pass = $match2[1] ?? "";

 


// مثال تشغيل:
$fb_dtsg = "NAfsuJlIErUxd7xng622Dd8iHRqoqnLZs3nFAhcZaJdEpz-XG7U5AkQ:28:1757428746";   // من قبل
$cookie = "datr=Wrq2aDGEVbeDYlfd5RcOunRW; sb=Yrq2aLxJRDapIJqE0tZ4KsuK; ps_l=1; ps_n=1; c_user=61561028164092; fr=1udOxmLbkMUjrjpDd.AWf122DoEPGPCt4mkoDUp1GMzkYtB4Eha-mpQPoI6FJqwWfCf50.BpMHo4..AAA.0.0.BpMHo4.AWdJvMtOrZzPTlgfPaTrG0vCi1Q; xs=28%3AP711LfX6qLfTmg%3A2%3A1757428746%3A-1%3A-1%3A%3AAcx77E12maCpWn5eA7PlHeNf5KkLSRvjSFoKvAtXZ-zx; presence=C%7B%22t3%22%3A%5B%5D%2C%22utc3%22%3A1764784897533%2C%22v%22%3A1%7D; wd=1920x653"; // كوكي الحساب

$blob = native_sso_approve($fb_dtsg, $t2_pass, $cookie);



 

function fb_auth_login($t1_email, $token_1)
{
    $url = "https://b-graph.facebook.com/auth/login";

    // POST BODY (نفس VB.NET بالظبط)
    $postData = http_build_query([
        "format" => "json",
        "email" => $t1_email,
        "password" => $token_1,
        "device_id" => "787c0621-0a03-4e95-8f9d-7a779445f62e",
        "credentials_type" => "browser_to_native_app_sso",
        "app_id" => "256002347743983",
        "source" => "browser_to_native_app_sso",
        "generate_session_cookies" => "1",
        "locale" => "ar_AR",
        "client_country_code" => "EG",
        "fb_api_req_friendly_name" => "authenticate",
        "fb_api_caller_class" => "AuthOperations\$NativeSSOAuthOperation",
        "api_key" => "256002347743983",
        "sig" => "01986e9c5832ea91c93a5899e1afad4f",
        "access_token" => "350685531728|62f8ce9f74b12f84c123cc23437a4a32"
    ]);

    $headers = [
        "Authorization: OAuth null",
        "Content-Type: application/x-www-form-urlencoded",
        "User-Agent: Dalvik/2.1.0 (Linux; U; Android 9; ASUS_I005DA Build/PI) [FBAN/Orca-Android;FBAV/391.2.0.20.404;FBPN/com.facebook.orca;FBLC/ar_EG;FBBV/437533963;FBCR/Vodafone-Mirsfone;FBMF/Asus;FBBD/Asus;FBDV/ASUS_I005DA;FBSV/9;FBCA/x86:armeabi-v7a;FBDM/{density=1.5,width=720,height=1280};FB_FW/1;]"
    ];

    $ch = curl_init();

    curl_setopt_array($ch, [
        CURLOPT_URL => $url,
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => $postData,
        CURLOPT_HTTPHEADER => $headers,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_FOLLOWLOCATION => false,
    ]);

    $response = curl_exec($ch);

    if ($response === false) {
        return "Curl Error: " . curl_error($ch);
    }

    curl_close($ch);

    // استخراج access_token
    if (preg_match('/"access_token":"(.+?)"/', $response, $m)) {
        return $m[1];
    }

    return "Access Token Not Found";
}

 
$token_facebook_android = fb_auth_login($t1_email, $blob);

echo "TOKEN: " . $token_facebook_android;




?>
