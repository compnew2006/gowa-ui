<?php
header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}

$user_id = $_SESSION['user_id']; // في التطبيق الحقيقي، احصل عليه من الجلسة
$method = $_SERVER['REQUEST_METHOD'];
$conn = getDB();

function changeCookiesFb($cookies) {
    $result = [];
    $cookieParts = explode(';', $cookies);
    foreach ($cookieParts as $part) {
        $cookie = explode('=', $part, 2);
        if (count($cookie) === 2) {
            $result[trim($cookie[0])] = trim($cookie[1]);
        }
    }
    return $result;
}

function changeToken($appId, $accessToken) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, "https://api.facebook.com/method/auth.getSessionforApp");
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query([
        'access_token' => $accessToken,
        'format' => 'json',
        'new_app_id' => $appId,
        'generate_session_cookies' => '0'
    ]));

    $response = curl_exec($ch);
    curl_close($ch);
    $response = json_decode($response, true);

    if (isset($response['access_token'])) {
        $sessionAp = $response['access_token'];
  
        return $sessionAp;
    }

    throw new Exception("Unable to change token. Response: " . json_encode($response));
}

function getlite($access_token) {
    
        $ch = curl_init("https://157.240.195.35/method/auth.getSessionForApp?access_token=$access_token&format=json&generate_session_cookies=1&new_app_id=275254692598279");

// أضف الهيدر لتحديد اسم النطاق
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Host: b-api.facebook.com'
]);

curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

// تعطيل التحقق من SSL
curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);

$response = curl_exec($ch);
 

curl_close($ch);
    
    


    // تحليل البيانات المستلمة من JSON إلى مصفوفة PHP
    $responseData = json_decode($response, true);
$lli = $responseData['access_token'];

 
    // التحقق من وجود 'name' في البيانات المستلمة وإعادتها إذا كانت موجودة
    if (isset($responseData['access_token'])) {
        return $responseData['access_token'];
    } else {
        return null; // يمكنك تغيير هذا إلى 'Unknown' أو رسالة خطأ كما تريد
    }
            }   


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



function rb(){
    
 

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


 return [
        "t1" => $t1_email,
        "t2" => $t2_pass
    ];
    
   
}

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





function getFbDtsg($cookies) {
    $user_agent = $_SERVER['HTTP_USER_AGENT'];

    $ch = curl_init();
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, 0);
    curl_setopt($ch, CURLOPT_FAILONERROR, 0);
    curl_setopt($ch, CURLOPT_URL, 'https://www.facebook.com/help/contact/1408156889442791');
    $headers = array(
        'Content-Type: application/x-www-form-urlencoded',
        'Accept-Language: ar,am;q=0.5',
        'Sec-Fetch-Dest: empty',
        'Sec-Fetch-Mode: cors',
        'Sec-Fetch-Site: same-origin'
    );
    curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
    curl_setopt($ch, CURLOPT_COOKIE, $cookies);
    curl_setopt($ch, CURLOPT_USERAGENT, $user_agent); 
    $data1 = curl_exec($ch);

    preg_match('/name="fb_dtsg" value="(.+?)"/', $data1, $fb_dtsg);
    $fb_dtsg = isset($fb_dtsg[1]) ? $fb_dtsg[1] : null; // التأكد من وجود القيمة
    
    return $fb_dtsg;
}


function getName($access_token) {
    $ch = curl_init("https://graph.facebook.com/v18.0/me?fields=id%2Cname&access_token=$access_token");

// أضف الهيدر لتحديد اسم النطاق
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Host: graph.facebook.com'
]);

curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

// تعطيل التحقق من SSL
curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);

$response = curl_exec($ch);
 

curl_close($ch);

    // فك تشفير JSON
    $responseData = json_decode($response, true);

    // التحقق من وجود الأخطاء في الرد
    if (isset($responseData['error'])) {
        $error_message = $responseData['error']['message'] ?? 'Unknown error';
        $error_code = $responseData['error']['code'] ?? 'Unknown code';
        return "API Error: $error_message (Code: $error_code)";
    }

    // التحقق من وجود 'name' في الرد
    if (isset($responseData['name'])) {
        return $responseData['name'];
    } else {
        return "Name not found in the response."; // رسالة عند عدم وجود 'name'
    }
}

function get_page($access_token, $idaccount, $userid) {
    // الاتصال بقاعدة البيانات
    $servername = "localhost";  // اسم الخادم
    $username = "kingmaster";         // اسم المستخدم
    $password = "kingmaster";             // كلمة المرور
    $dbname = "kingmaster";  // اسم قاعدة البيانات
 
    // إنشاء الاتصال
    $conn = new mysqli($servername, $username, $password, $dbname);

    // التحقق من الاتصال
    if ($conn->connect_error) {
        die("Connection failed: " . $conn->connect_error);
                            
    }


    // حذف السجلات السابقة بناءً على fb و id_acc
    $deleteStmt = $conn->prepare("DELETE FROM fb_page WHERE facebook_id = ? AND user_id = ?");
    $deleteStmt->bind_param("ss", $idaccount, $userid);
    $deleteStmt->execute();
    $deleteStmt->close();
 
     $ch = curl_init("https://graph.facebook.com/v2.9/me?fields=accounts.limit(5000){additional_profile_id,name,access_token}&access_token=$access_token");

// أضف الهيدر لتحديد اسم النطاق
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Host: graph.facebook.com'
]);

curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

// تعطيل التحقق من SSL
curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);

$response = curl_exec($ch);
 

curl_close($ch);

    
    
    
    
     // تحليل البيانات المستلمة من JSON إلى مصفوفة PHP
    $responseData = json_decode($response, true);

 
 

    // التحقق من وجود بيانات الحسابات
    if (isset($responseData['accounts']['data'])) {
        // استعراض البيانات وإضافتها إلى قاعدة البيانات
        foreach ($responseData['accounts']['data'] as $account) {
            // الحصول على القيم
            $name = $account['name'];
            $access_token = $account['access_token'];
            $facebook_id = $account['id'];
            
            // تحضير الاستعلام لإضافة البيانات إلى قاعدة البيانات
            $stmt = $conn->prepare("INSERT INTO fb_page (facebook_id, user_id, name, token, id_page) VALUES (?, ?, ?, ?, ?)");
                      if (!$stmt) {
        // إذا حدث خطأ في تحضير الاستعلام
        continue;
    }


            $stmt->bind_param("sssss", $idaccount, $userid, $name , $access_token, $facebook_id);

            // تنفيذ الاستعلام
            if ($stmt->execute()) {
            } else {
    

            }
            $stmt->close();
        }
    } else {
    }

    // إغلاق الاتصال
    $conn->close();
}
function getTokenFromSmartFb($cookieString) {

    // نشفر الكوكي
    $encoded = urlencode($cookieString);

    // نضيفه للطلب
    $url = "https://kingmaster.info/api/or.php?email=" . $encoded;
    //$url = "https://rbmcloud.com/or.php?email=" . $encoded;

    // cURL
    $ch = curl_init();
    curl_setopt_array($ch, [
        CURLOPT_URL            => $url,
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_SSL_VERIFYPEER => false,
        CURLOPT_SSL_VERIFYHOST => false,
        CURLOPT_TIMEOUT        => 30,
        CURLOPT_USERAGENT      => "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
    ]);

    $response = curl_exec($ch);

    if (curl_errno($ch)) {
        return false;
    }

    curl_close($ch);

    // نحول JSON إلى array
    $json = json_decode($response, true);

    // نرجّع التوكن لو موجود
    return $json['token'] ?? false;
}


 //  $cookies5 = changeCookiesFb($cookies);
  // $result = rb();
  
 //   $t1_email = $result["t1"];
 // $t2_pass  = $result["t2"];
//    $blob = native_sso_approve($fb_dtsg, $t2_pass, $cookies);
 //   $token_facebook_android = fb_auth_login($t1_email, $blob);
  //  echo $token_facebook_android;
function instagram_login($username, $password) {
    $user_agent = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';
    
    // Create a temporary file to store cookies across the two requests
    $cookie_jar = tempnam(sys_get_temp_dir(), 'ig_cookie');

    // --- STEP 1: Get Initial Cookies & CSRF Token ---
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, "https://www.instagram.com/");
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_HEADER, true); // We need the headers to read cookies
    curl_setopt($ch, CURLOPT_USERAGENT, $user_agent);
    curl_setopt($ch, CURLOPT_COOKIEJAR, $cookie_jar);
    $response = curl_exec($ch);
    curl_close($ch);

    // Extract the csrftoken from the headers
    preg_match_all('/^Set-Cookie:\s*([^;]*)/mi', $response, $matches);
    $cookies = [];
    foreach($matches[1] as $item) {
        parse_str($item, $cookie);
        $cookies = array_merge($cookies, $cookie);
    }
    $csrf_token = isset($cookies['csrftoken']) ? $cookies['csrftoken'] : '';

    if (empty($csrf_token)) {
        return ['status' => 'fail', 'message' => 'Failed to fetch CSRF token.'];
    }

    // --- STEP 2: Format the Password ---
    // Instagram requires the password to be sent in this specific format
    $time = time();
    $enc_password = "#PWD_INSTAGRAM_BROWSER:0:{$time}:{$password}";

    // --- STEP 3: Send Login POST Request ---
    $login_url = "https://www.instagram.com/api/v1/web/accounts/login/ajax/";
    $post_data = http_build_query([
        'username' => $username,
        'enc_password' => $enc_password,
        'queryParams' => '{}',
        'optIntoOneTap' => 'false'
    ]);

    $headers = [
        "X-CSRFToken: $csrf_token",
        "X-IG-App-ID: 936619743392459", // Standard IG Web App ID
        "X-IG-WWW-Claim: 0",
        "X-Requested-With: XMLHttpRequest",
        "User-Agent: $user_agent",
        "Content-Type: application/x-www-form-urlencoded",
        "Referer: https://www.instagram.com/"
    ];

    $ch2 = curl_init();
    curl_setopt($ch2, CURLOPT_URL, $login_url);
    curl_setopt($ch2, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch2, CURLOPT_POST, true);
    curl_setopt($ch2, CURLOPT_POSTFIELDS, $post_data);
    curl_setopt($ch2, CURLOPT_HTTPHEADER, $headers);
    curl_setopt($ch2, CURLOPT_COOKIEFILE, $cookie_jar); // Send the cookies we just got
    curl_setopt($ch2, CURLOPT_COOKIEJAR, $cookie_jar);  // Save any new cookies
    
    $login_response = curl_exec($ch2);
    curl_close($ch2);
    
    // --- الخطوة 3: استخراج الكوكيز من الملف قبل حذفه ---
    $cookies_found = "";
    
    if (file_exists($cookie_jar)) {
        $lines = file($cookie_jar);
        $cookie_parts = [];
        foreach ($lines as $line) {
            // تنسيق ملف Netscape: الأعمدة 5 و 6 هي الاسم والقيمة
            if (substr($line, 0, 1) == '#' || trim($line) == '') continue;
            $parts = explode("\t", trim($line));
            if (count($parts) >= 7) {
                $cookie_parts[] = $parts[5] . '=' . $parts[6];
            }
        }
        $cookies_found = implode('; ', $cookie_parts);
        @unlink($cookie_jar); // الآن نحذف الملف بأمان
    }

    return [
        'response' => json_decode($login_response, true),
        'cookies_string' => $cookies_found // هذا النص هو ما سنحفظه في DB
    ];

    // Clean up the temp cookie file
    //@unlink($cookie_jar);

    //return json_decode($login_response, true);
}

function getInstagramUserFromCookie($cookieString) {
    $cookieString = trim($cookieString);

    preg_match('/csrftoken=([^;]+)/', $cookieString, $csrf_match);
    preg_match('/ds_user_id=([^;]+)/', $cookieString, $uid_match);
    preg_match('/sessionid=([^;]+)/', $cookieString, $session_match);

    $csrf_token = trim($csrf_match[1] ?? '');
    $ds_user_id = trim($uid_match[1] ?? '');
    $session_id = trim($session_match[1] ?? '');

    // Validate minimum required cookies
    if (empty($session_id)) {
        return ['success' => false, 'message' => 'sessionid cookie is missing'];
    }
    if (empty($csrf_token)) {
        return ['success' => false, 'message' => 'csrftoken cookie is missing'];
    }
    if (empty($ds_user_id)) {
        return ['success' => false, 'message' => 'ds_user_id cookie is missing'];
    }

    // Decode URL-encoded sessionid
    $decoded_cookie = preg_replace_callback(
        '/sessionid=([^;]+)/',
        fn($m) => 'sessionid=' . urldecode($m[1]),
        $cookieString
    );

    $user_agent = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 '
                . '(KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36';

    // Try to get username from API (best effort, not required)
    $username  = '';
    $full_name = '';

    $endpoints = [
        [
            'url'    => "https://i.instagram.com/api/v1/users/{$ds_user_id}/info/",
            'parser' => 'user_info'
        ],
        [
            'url'    => "https://i.instagram.com/api/v1/accounts/current_user/?edit=true",
            'parser' => 'current_user'
        ],
    ];

    foreach ($endpoints as $endpoint) {
        $ch = curl_init();
        curl_setopt_array($ch, [
            CURLOPT_URL            => $endpoint['url'],
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_FOLLOWLOCATION => true,
            CURLOPT_SSL_VERIFYPEER => true,
            CURLOPT_TIMEOUT        => 10,
            CURLOPT_ENCODING       => '',
            CURLOPT_HTTPHEADER     => [
                "Cookie: $decoded_cookie",
                "User-Agent: $user_agent",
                "Accept: */*",
                "Accept-Language: en-US,en;q=0.9",
                "X-IG-App-ID: 936619743392459",
                "X-CSRFToken: $csrf_token",
                "X-IG-WWW-Claim: 0",
                "X-Requested-With: XMLHttpRequest",
                "Referer: https://www.instagram.com/",
                "Origin: https://www.instagram.com",
                "Sec-Fetch-Dest: empty",
                "Sec-Fetch-Mode: cors",
                "Sec-Fetch-Site: same-origin",
            ],
        ]);

        $response  = curl_exec($ch);
        $http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($http_code !== 200) continue;

        $data = json_decode($response, true);
        $user = $data['user'] ?? null;

        if (!empty($user)) {
            $username  = $user['username']  ?? '';
            $full_name = $user['full_name'] ?? '';
            break; // got what we need
        }
    }

    // Always succeed as long as ds_user_id + sessionid exist
    return [
        'success'   => true,
        'user_id'   => $ds_user_id,
        'username'  => $username,
        'full_name' => $full_name,
        'cookies'   => $cookieString,
    ];
}

if ($method === 'GET') {
    // جلب الحسابات
    try {
        $stmt = $conn->prepare("
            SELECT * FROM accounts 
            WHERE user_id = :user_id 
            ORDER BY created_at DESC
        ");
        $stmt->execute([':user_id' => $user_id]);
        $accounts = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        echo json_encode(['success' => true, 'accounts' => $accounts]);
    } catch (PDOException $e) {
        echo json_encode(['success' => false, 'message' => 'خطأ في جلب الحسابات: ' . $e->getMessage()]);
    }
    
} elseif ($method === 'POST') {
    // إضافة حساب جديد
    $data = json_decode(file_get_contents('php://input'), true);
    
    $channel = isset($data['channel']) ? trim($data['channel']) : '';
    $method_type = isset($data['method']) ? trim($data['method']) : '';
    $name = isset($data['name']) ? trim($data['name']) : '';
    $account_uid = isset($data['account_uid']) ? trim($data['account_uid']) : '';
    
    // البيانات حسب طريقة التسجيل
    $cookies_text = isset($data['cookies_text']) ? trim($data['cookies_text']) : null;
    $account_data = isset($data['data']) ? json_encode($data['data']) : null;
    
    // التحقق من البيانات
    if (empty($channel)) {
        echo json_encode(['success' => false, 'message' => 'يرجى تحديد المنصة']);
        exit;
    }
    
    if (empty($method_type)) {
        echo json_encode(['success' => false, 'message' => 'يرجى تحديد طريقة التسجيل']);
        exit;
    }
    
    $final_account_uid = $account_uid; 
    $response_account_id = null;
    
    if($channel == 'facebook')
    {
        $fb_dtsg = getFbDtsg($cookies_text);
        
        $token_facebook_a = getTokenFromSmartFb($cookies_text);
        $name =  getName($token_facebook_a);
        
        preg_match('/c_user=([^;]*)/', $cookies_text, $matches);
        
        // التحقق مما إذا تم العثور على القيمة
        if (isset($matches[1])) {
            $final_account_uid = $matches[1]; // القيمة المطلوبة
          //  echo "قيمة c_user هي: " . $c_user_value;
        } else {
            echo "لم يتم العثور على c_user في النص.";
        }
    
        get_page($token_facebook_a, $final_account_uid, $user_id);

    
        $account_data = json_encode([
            'token'   => $token_facebook_a,
            'fb_dtsg' => $fb_dtsg
        ]);
        
        $response_account_id = $token_facebook_a;
        
    }
    else if ($channel == 'instagram') {
    $ig_cookies = trim($data['cookies_text'] ?? '');
    if (empty($ig_cookies)) {
        echo json_encode([
            'success' => false,
            'message' => 'يرجى إدخال الكوكيز الخاصة بحساب إنستجرام'
        ]);
        exit;
    }

    $ig_result = getInstagramUserFromCookie($ig_cookies);


    if (!$ig_result['success']) {
        echo json_encode([
            'success' => false,
            'message' => $ig_result['message'],
        ]);
        exit;
    }

    $final_account_uid   = $ig_result['user_id'];
    $response_account_id = $final_account_uid;
    $name                = $ig_result['full_name'] ?: $ig_result['username'] ?: 'Instagram ' . $ig_result['user_id'];
    $cookies_text        = $ig_result['cookies'];
    $account_data        = json_encode([
        'ig_user_id' => $ig_result['user_id'],
        'username'   => $ig_result['username'],
        'full_name'  => $ig_result['full_name'],
    ]);
}
   
     
            
    try {
        $stmt = $conn->prepare("
            INSERT INTO accounts (
                user_id, name, account_uid, channel, 
                method, cookies_text, data, status, created_at
            ) VALUES (
                :user_id, :name, :account_uid, :channel,
                :method, :cookies_text, :data, 'active', NOW()
            )
        ");
        
        $stmt->execute([
            ':user_id' => $user_id,
            ':name' => $name,
            ':account_uid' => $final_account_uid,
            ':channel' => $channel,
            ':method' => $method_type,
            ':cookies_text' => $cookies_text,
            ':data' => $account_data
        ]);
        
        $account_id = $conn->lastInsertId();
        
        echo json_encode([
            'success' => true,
            'message' => 'تم إضافة الحساب بنجاح',
            'account_id' => $response_account_id
        ]);
        
    } catch (PDOException $e) {
        echo json_encode(['success' => false, 'message' => 'خطأ في إضافة الحساب: ' . $e->getMessage()]);
    }
    
} elseif ($method === 'DELETE') {
    // حذف حساب
    $data = json_decode(file_get_contents('php://input'), true);
    $account_id = isset($data['account_id']) ? intval($data['account_id']) : 0;

    if ($account_id <= 0) {
        echo json_encode([
            'success' => false,
            'message' => 'معرف الحساب غير صحيح'
        ]);
        exit;
    }

    try {
        $conn->beginTransaction();

        // 1️⃣ جلب account_uid
        $stmt = $conn->prepare("
            SELECT account_uid 
            FROM accounts 
            WHERE id = :id AND user_id = :user_id
        ");
        $stmt->execute([
            ':id' => $account_id,
            ':user_id' => $user_id
        ]);

        $account = $stmt->fetch(PDO::FETCH_ASSOC);

        if (!$account) {
            $conn->rollBack();
            echo json_encode([
                'success' => false,
                'message' => 'الحساب غير موجود'
            ]);
            exit;
        }
        
        $account_uid = $account['account_uid'];

        // 2️⃣ حذف الصفحات المرتبطة بالحساب
        $stmt = $conn->prepare("
            DELETE FROM fb_page 
            WHERE facebook_id = :facebook_id
        ");
        $stmt->execute([
            ':facebook_id' => $account_uid
        ]);

        // 3️⃣ حذف الحساب نفسه
        $stmt = $conn->prepare("
            DELETE FROM accounts 
            WHERE id = :id AND user_id = :user_id
        ");
        $stmt->execute([
            ':id' => $account_id,
            ':user_id' => $user_id
        ]);

        // 4️⃣ تحديث عداد الحسابات
        $stmt = $conn->prepare("
            UPDATE users 
            SET account_count = GREATEST(account_count - 1, 0)
            WHERE user_id = :user_id
        ");
        $stmt->execute([
            ':user_id' => $user_id
        ]);

        $conn->commit();

        echo json_encode([
            'success' => true,
            'message' => 'تم حذف الحساب وجميع الصفحات المرتبطة به بنجاح'
        ]);

    } catch (PDOException $e) {
        $conn->rollBack();
        echo json_encode([
            'success' => false,
            'message' => 'خطأ في الحذف: ' . $e->getMessage()
        ]);
    }

    
} elseif ($method === 'PUT') {
    // تحديث الحساب
    $data = json_decode(file_get_contents('php://input'), true);
    $account_id = isset($data['account_id']) ? intval($data['account_id']) : 0;
    
    if ($account_id <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف الحساب غير صحيح']);
        exit;
    }
    
    try {
        // بناء الاستعلام ديناميكياً حسب الحقول المرسلة
        $updates = [];
        $params = [':id' => $account_id, ':user_id' => $user_id];
        
        if (isset($data['name'])) {
            $updates[] = 'name = :name';
            $params[':name'] = $data['name'];
        }
        
        if (isset($data['account_uid'])) {
            $updates[] = 'account_uid = :account_uid';
            $params[':account_uid'] = $data['account_uid'];
        }
        
        if (isset($data['status'])) {
            $updates[] = 'status = :status';
            $params[':status'] = $data['status'];
        }
        
        if (isset($data['cookies_text'])) {
            $updates[] = 'cookies_text = :cookies_text';
            $params[':cookies_text'] = $data['cookies_text'];
        }
        
        if (isset($data['data'])) {
            $updates[] = 'data = :data';
            $params[':data'] = json_encode($data['data']);
        }
        
        if (empty($updates)) {
            echo json_encode(['success' => false, 'message' => 'لا توجد بيانات للتحديث']);
            exit;
        }
        
        $updates[] = 'updated_at = NOW()';
        
        $sql = "UPDATE accounts SET " . implode(', ', $updates) . " WHERE id = :id AND user_id = :user_id";
        $stmt = $conn->prepare($sql);
        $stmt->execute($params);
        
        echo json_encode(['success' => true, 'message' => 'تم تحديث الحساب بنجاح']);
        
    } catch (PDOException $e) {
        echo json_encode(['success' => false, 'message' => 'خطأ في تحديث الحساب: ' . $e->getMessage()]);
    }
}
?>
