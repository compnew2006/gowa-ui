<?php

if ($_SERVER['REQUEST_METHOD'] == 'GET') {

 
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
function getFbDtsg($cookies) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, "https://www.facebook.com/v2.3/dialog/oauth?redirect_uri=fbconnect://success&response_type=token,code&client_id=356275264482347");
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_COOKIE, implode('; ', array_map(
        function($k, $v) { return "$k=$v"; }, array_keys($cookies), $cookies
    )));
    curl_setopt($ch, CURLOPT_HTTPHEADER, [
        'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36',
        'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/jxl,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7',
        'Accept-Language: vi,en-US;q=0.9,en;q=0.8',
        'Cache-Control: max-age=0',
        'DNT: 1',
        'Sec-Fetch-Dest: document',
        'Sec-Fetch-Mode: navigate',
        'Sec-Fetch-Site: same-origin',
        'Upgrade-Insecure-Requests: 1'
    ]);

    $response = curl_exec($ch);
    curl_close($ch);

    if (preg_match('/DTSGInitialData.*?\\{\"token\":\"(.*?)\"/', $response, $matches)) {
        return $matches[1];
    } else {
        throw new Exception("Unable to fetch fb_dtsg token. Response: " . substr($response, 0, 500));
    }
}
function generate_uuid() {
    return sprintf('%04x%04x-%04x-%04x-%04x-%04x%04x%04x',
        mt_rand(0, 0xffff), mt_rand(0, 0xffff),
        mt_rand(0, 0xffff),
        mt_rand(0, 0x0fff) | 0x4000,
        mt_rand(0, 0x3fff) | 0x8000,
        mt_rand(0, 0xffff), mt_rand(0, 0xffff), mt_rand(0, 0xffff)
    );
}
function run($cookies, $appId) {
    $fbDtsg = getFbDtsg($cookies);
    $cUser = $cookies['c_user'];
    $uuid = generate_uuid();
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, 'https://www.facebook.com/api/graphql/');
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_POSTFIELDS, 'av='.$cUser.'&__user='.$cUser.'&fb_dtsg='.$fbDtsg.'&fb_api_caller_class=RelayModern&fb_api_req_friendly_name=useCometConsentPromptEndOfFlowBatchedMutation&variables=%7B%22input%22%3A%7B%22client_mutation_id%22%3A%224%22%2C%22actor_id%22%3A%22'.$cUser.'%22%2C%22config_enum%22%3A%22GDP_CONFIRM%22%2C%22device_id%22%3Anull%2C%22experience_id%22%3A%22'.$uuid.'%22%2C%22extra_params_json%22%3A%22%7B%5C%22app_id%5C%22%3A%5C%22350685531728%5C%22%2C%5C%22kid_directed_site%5C%22%3A%5C%22false%5C%22%2C%5C%22logger_id%5C%22%3A%5C%22%5C%5C%5C%22'.generate_uuid().'%5C%5C%5C%22%5C%22%2C%5C%22next%5C%22%3A%5C%22%5C%5C%5C%22confirm%5C%5C%5C%22%5C%22%2C%5C%22redirect_uri%5C%22%3A%5C%22%5C%5C%5C%22https%3A%5C%5C%5C%5C%5C%5C%2F%5C%5C%5C%5C%5C%5C%2Fwww.facebook.com%5C%5C%5C%5C%5C%5C%2Fconnect%5C%5C%5C%5C%5C%5C%2Flogin_success.html%5C%5C%5C%22%5C%22%2C%5C%22response_type%5C%22%3A%5C%22%5C%5C%5C%22token%5C%5C%5C%22%5C%22%2C%5C%22return_scopes%5C%22%3A%5C%22false%5C%22%2C%5C%22scope%5C%22%3A%5C%22%5B%5C%5C%5C%22user_subscriptions%5C%5C%5C%22%2C%5C%5C%5C%22user_videos%5C%5C%5C%22%2C%5C%5C%5C%22user_website%5C%5C%5C%22%2C%5C%5C%5C%22user_work_history%5C%5C%5C%22%2C%5C%5C%5C%22friends_about_me%5C%5C%5C%22%2C%5C%5C%5C%22friends_actions.books%5C%5C%5C%22%2C%5C%5C%5C%22friends_actions.music%5C%5C%5C%22%2C%5C%5C%5C%22friends_actions.news%5C%5C%5C%22%2C%5C%5C%5C%22friends_actions.video%5C%5C%5C%22%2C%5C%5C%5C%22friends_activities%5C%5C%5C%22%2C%5C%5C%5C%22friends_birthday%5C%5C%5C%22%2C%5C%5C%5C%22friends_education_history%5C%5C%5C%22%2C%5C%5C%5C%22friends_events%5C%5C%5C%22%2C%5C%5C%5C%22friends_games_activity%5C%5C%5C%22%2C%5C%5C%5C%22friends_groups%5C%5C%5C%22%2C%5C%5C%5C%22friends_hometown%5C%5C%5C%22%2C%5C%5C%5C%22friends_interests%5C%5C%5C%22%2C%5C%5C%5C%22friends_likes%5C%5C%5C%22%2C%5C%5C%5C%22friends_location%5C%5C%5C%22%2C%5C%5C%5C%22friends_notes%5C%5C%5C%22%2C%5C%5C%5C%22friends_photos%5C%5C%5C%22%2C%5C%5C%5C%22friends_questions%5C%5C%5C%22%2C%5C%5C%5C%22friends_relationship_details%5C%5C%5C%22%2C%5C%5C%5C%22friends_relationships%5C%5C%5C%22%2C%5C%5C%5C%22friends_religion_politics%5C%5C%5C%22%2C%5C%5C%5C%22friends_status%5C%5C%5C%22%2C%5C%5C%5C%22friends_subscriptions%5C%5C%5C%22%2C%5C%5C%5C%22friends_videos%5C%5C%5C%22%2C%5C%5C%5C%22friends_website%5C%5C%5C%22%2C%5C%5C%5C%22friends_work_history%5C%5C%5C%22%2C%5C%5C%5C%22ads_management%5C%5C%5C%22%2C%5C%5C%5C%22create_event%5C%5C%5C%22%2C%5C%5C%5C%22create_note%5C%5C%5C%22%2C%5C%5C%5C%22export_stream%5C%5C%5C%22%2C%5C%5C%5C%22friends_online_presence%5C%5C%5C%22%2C%5C%5C%5C%22manage_friendlists%5C%5C%5C%22%2C%5C%5C%5C%22manage_notifications%5C%5C%5C%22%2C%5C%5C%5C%22manage_pages%5C%5C%5C%22%2C%5C%5C%5C%22photo_upload%5C%5C%5C%22%2C%5C%5C%5C%22publish_stream%5C%5C%5C%22%2C%5C%5C%5C%22read_friendlists%5C%5C%5C%22%2C%5C%5C%5C%22read_insights%5C%5C%5C%22%2C%5C%5C%5C%22read_mailbox%5C%5C%5C%22%2C%5C%5C%5C%22read_page_mailboxes%5C%5C%5C%22%2C%5C%5C%5C%22read_requests%5C%5C%5C%22%2C%5C%5C%5C%22read_stream%5C%5C%5C%22%2C%5C%5C%5C%22rsvp_event%5C%5C%5C%22%2C%5C%5C%5C%22share_item%5C%5C%5C%22%2C%5C%5C%5C%22sms%5C%5C%5C%22%2C%5C%5C%5C%22status_update%5C%5C%5C%22%2C%5C%5C%5C%22user_online_presence%5C%5C%5C%22%2C%5C%5C%5C%22video_upload%5C%5C%5C%22%2C%5C%5C%5C%22xmpp_login%5C%5C%5C%22%5D%5C%22%2C%5C%22steps%5C%22%3A%5C%22%7B%7D%5C%22%2C%5C%22tp%5C%22%3A%5C%22%5C%5C%5C%22unspecified%5C%5C%5C%22%5C%22%2C%5C%22cui_gk%5C%22%3A%5C%22%5C%5C%5C%22%5BPASS%5D%3A%5C%5C%5C%22%5C%22%2C%5C%22is_limited_login_shim%5C%22%3A%5C%22false%5C%22%7D%22%2C%22flow_name%22%3A%22GDP%22%2C%22flow_step_type%22%3A%22STANDALONE%22%2C%22outcome%22%3A%22APPROVED%22%2C%22source%22%3A%22gdp_delegated%22%2C%22surface%22%3A%22FACEBOOK_COMET%22%7D%7D&server_timestamps=true&doc_id=6494107973937368');
    curl_setopt($ch, CURLOPT_COOKIE, implode('; ', array_map(
        function($k, $v) { return "$k=$v"; }, array_keys($cookies), $cookies
    )));
    curl_setopt($ch, CURLOPT_HTTPHEADER, [
        'authority: www.facebook.com',
        'accept: */*',
        'accept-language: vi-VN,vi;q=0.9,fr-FR;q=0.8,fr;q=0.7,en-US;q=0.6,en;q=0.5',
        'content-type: application/x-www-form-urlencoded',
        'dnt: 1',
        'origin: https://www.facebook.com',
        'sec-ch-prefers-color-scheme: dark',
        'sec-ch-ua: "Chromium";v="117", "Not;A=Brand";v="8"',
        'sec-ch-ua-full-version-list: "Chromium";v="117.0.5938.157", "Not;A=Brand";v="8.0.0.0"',
        'sec-ch-ua-mobile: ?0',
        'sec-ch-ua-model: ""',
        'sec-ch-ua-platform: "Windows"',
        'sec-ch-ua-platform-version: "15.0.0"',
        'sec-fetch-dest: empty',
        'sec-fetch-mode: cors',
        'sec-fetch-site: same-origin',
        'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36',
        'x-fb-friendly-name: useCometConsentPromptEndOfFlowBatchedMutation',
    ]);

    $response = curl_exec($ch);
    curl_close($ch); 
    $response = json_decode($response, true);
    if (isset($response['data']['run_post_flow_action']['uri'])) {
        $uri = $response['data']['run_post_flow_action']['uri'];
        $parsedUrl = parse_url($uri);
        parse_str($parsedUrl['query'], $queryParams);
        $closeUri = urldecode($queryParams['close_uri'] ?? '');
        $fragmentUrl = parse_url($closeUri);
        parse_str($fragmentUrl['fragment'] ?? '', $fragmentParams);

        $accessToken = $fragmentParams['access_token'] ?? null;
           return array("accessToken" => $accessToken, "fbDtsg" => $fbDtsg);

    }

    throw new Exception("Unable to fetch access token.");
}

function get_page($access_token, $idaccount, $userid, $name_acc) {
    // الاتصال بقاعدة البيانات
    $servername = "localhost";  // اسم الخادم
    $username = "kingmaster_circleapi";         // اسم المستخدم
    $password = "kingmaster_circleapi";             // كلمة المرور
    $dbname = "kingmaster_circleapi";  // اسم قاعدة البيانات
 
    // إنشاء الاتصال
    $conn = new mysqli($servername, $username, $password, $dbname);

    // التحقق من الاتصال
    if ($conn->connect_error) {
        die("Connection failed: " . $conn->connect_error);
                            file_put_contents(
    'loginds.txt', 
    "Error: details: " . $conn->connect_error . "\n", 
    FILE_APPEND
);
    }

    // حذف السجلات السابقة بناءً على fb و id_acc
    $deleteStmt = $conn->prepare("DELETE FROM fb_page WHERE fb = ? AND id_acc = ?");
    $deleteStmt->bind_param("ss", $idaccount, $userid);
    $deleteStmt->execute();
    $deleteStmt->close();
 
     $ch = curl_init("https://102.132.103.8/v2.9/me?fields=accounts.limit(5000){additional_profile_id,name,access_token}&access_token=$access_token");

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

    
    
    
    
    file_put_contents('loginds.txt', "page -- {$response} \n", FILE_APPEND);  
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
            $last =$name . " = " . $name_acc;
            // تحضير الاستعلام لإضافة البيانات إلى قاعدة البيانات
            $stmt = $conn->prepare("INSERT INTO fb_page (fb, id_acc, name, access_token, facebook_id) VALUES (?, ?, ?, ?, ?)");
                      if (!$stmt) {
        // إذا حدث خطأ في تحضير الاستعلام
        file_put_contents('loginds.txt', "Prepare Error: " . $conn->error . "\n", FILE_APPEND);
        continue;
    }


            $stmt->bind_param("sssss", $idaccount, $userid, $last , $access_token, $facebook_id);

            // تنفيذ الاستعلام
            if ($stmt->execute()) {
                echo "تم إضافة الحساب: $name إلى قاعدة البيانات.\n";
            } else {
                       file_put_contents(
            'loginds.txt',
            "Execute Error: " . $stmt->error . " | Data: idaccount=$idaccount, userid=$userid, last=$last, access_token=$access_token, facebook_id=$facebook_id\n",
            FILE_APPEND
        );

            }
            $stmt->close();
        }
    } else {
        echo "لم يتم العثور على بيانات الحسابات.\n";
    }

    // إغلاق الاتصال
    $conn->close();
}
function getfb_dtsg($cookies) {
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
    $ch = curl_init("https://102.132.103.8/v18.0/me?fields=id%2Cname&access_token=$access_token");

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
function getlite($access_token) {
    
        $ch = curl_init("https://b-api.facebook.com/method/auth.getSessionForApp?&access_token=$access_token&format=json&generate_session_cookies=1&new_app_id=275254692598279");

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
function getliteNew($email, $sso_blob, $cookies) {
    $url = "https://b-graph.facebook.com/auth/login";

    // Facebook Lite App Credentials
    $app_id = "275254692598279";
    $app_secret = "0eb922e96d99727409f53874313f898d";

    // Prepare the data for the new Graph API login
    $data = [
        "format" => "json",
        "email" => $email,
        "password" => $sso_blob, // This is the 'blob' from native_sso_approve
        "device_id" => "787c0621-0a03-4e95-8f9d-7a779445f62e",
        "credentials_type" => "browser_to_native_app_sso",
        "app_id" => $app_id,
        "source" => "browser_to_native_app_sso",
        "generate_session_cookies" => "1",
        "locale" => "ar_AR",
        "client_country_code" => "EG",
        "fb_api_req_friendly_name" => "authenticate",
        "fb_api_caller_class" => "AuthOperations\$NativeSSOAuthOperation",
        "api_key" => $app_id,
        "access_token" => "350685531728|62f8ce9f74b12f84c123cc23437a4a32" 
    ];

    // Generate the required 'sig' (Signature)
    ksort($data);
    $sig_string = "";
    foreach ($data as $key => $value) {
        $sig_string .= $key . "=" . $value;
    }
    $data['sig'] = md5($sig_string . $app_secret);

    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => http_build_query($data),
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => [
            "Authorization: OAuth null",
            "Content-Type: application/x-www-form-urlencoded",
           "User-Agent: [FBAN/FB4A;FBAV/400.0.0.0.0;FBBV/45678912;FBDM/{density=3.0,width=1080,height=1920};FBLC/ar_EG;FBCR/Vodafone;FBPN/com.facebook.katana;FBDV/GalaxyS22;FBSV/12;FBOP/1;FBCA/arm64-v8a;]"
            //"User-Agent: [FBAN/FB4A;FBAV/35.0.0.48.273;FBBV/13774109;FBDM/{density=1.5,width=480,height=800};FBLC/en_US//;FBCR/Verizon;FBPN/com.facebook.katana;FBDV/Sch-i545;FBSV/4.4.2;FBOP/1;FBCA/armeabi-v7a:armeabi;]"
        ],
        CURLOPT_COOKIE => $cookies,
        CURLOPT_SSL_VERIFYPEER => false,
        CURLOPT_SSL_VERIFYHOST => false
    ]);

    $response = curl_exec($ch);
    curl_close($ch);

    $responseData = json_decode($response, true);
     @file_put_contents(__DIR__ . '/a_b_d.log',
                date('c') . " response Data old =" . json_encode($responseData, JSON_UNESCAPED_UNICODE) . "\n",
                FILE_APPEND
            );
    // Return exactly like the old function
    if (isset($responseData['access_token'])) {
        return $responseData['access_token'];
    } else {
        return null;
    }
}

function get_one_login($country){
    
  // Get Token Instagram
$ch = curl_init();
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_HEADER, 1);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, 0);
curl_setopt($ch, CURLOPT_FAILONERROR, 0);
curl_setopt($ch, CURLOPT_URL, 'https://www.facebook.com/x/oauth/status?client_id=124024574287414&input_token&origin=1&redirect_uri=https%3A%2F%2Fwww.instagram.com%2Faccounts%2Fedit%2F&sdk=joey&wants_cookie_data=true');
$headers = array(
    'Origin: https://www.instagram.com',
    'Content-Type: application/x-www-form-urlencoded',
    'Accept-Language: ar,am;q=0.5',
    'Sec-Fetch-Dest: empty',
    'Sec-Fetch-Mode: cors',
    'Sec-Fetch-Site: same-origin'
);
curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
curl_setopt($ch, CURLOPT_COOKIE, $country);
curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36'); 
$html = curl_exec($ch);

preg_match('/"access_token":"(.+?)"/', $html, $access_token);
$access_token =$access_token[1];

  
  return  $access_token;
} 
 


function decode_once($str) {
    return rawurldecode($str);
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

function fb_auth_login($t1_email, $token_1,$cookies)
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
    CURLOPT_COOKIE => $cookies,
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_FOLLOWLOCATION => false,

    // 🔐 Force TLS 1.2 like VB 3072
    CURLOPT_SSLVERSION => CURL_SSLVERSION_TLSv1_2,

    // 🛡️ Allow old SSL if needed (optional)
    CURLOPT_SSL_VERIFYPEER => false,
    CURLOPT_SSL_VERIFYHOST => false,

    // 🌐 Keep connection alive
    CURLOPT_TCP_KEEPALIVE => 1,
    CURLOPT_TCP_KEEPIDLE => 30
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

    return $response;
}

function getNewAccessToken($access_token) {
    //$appId = '275254692598279';
    //$appSecret = '0eb922e96d99727409f53874313f898d';
    $appId = '26289275550723596';
    $appSecret = 'b95bb11399cbe0ceef0f7f6da7e83fba';
    
    $url = "https://graph.facebook.com/v23.0/oauth/access_token?" . http_build_query([
        'grant_type' => 'fb_exchange_token',
        'client_id' => $appId,
        'client_secret' => $appSecret,
        'fb_exchange_token' => $access_token,
    ]);

 @file_put_contents(__DIR__ . '/a_b_d.log',
                    date('c') . "url =" . json_encode($url, JSON_UNESCAPED_UNICODE) . "\n",
                    FILE_APPEND
                );
    $ch = curl_init($url);
    curl_setopt_array($ch, [
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_SSL_VERIFYPEER => false,
        CURLOPT_SSL_VERIFYHOST => false
    ]);

    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    @file_put_contents(__DIR__ . '/a_b_d.log',
                    date('c') . " tested token =" . json_encode($response, JSON_UNESCAPED_UNICODE) . "\n",
                    FILE_APPEND
                );

return $response;
}

$email5 = $_GET['email'];
$email =  decode_once($email5);
    $cookies = changeCookiesFb($email5);
    $result = rb();
    $fb_dtsg = getFbDtsg($cookies);
    $t1_email = $result["t1"];
    $t2_pass  = $result["t2"];
    $blob = native_sso_approve($fb_dtsg, $t2_pass, $email);
    $token_facebook_android = fb_auth_login($t1_email, $blob, $email5);
    
//
$token_facebook_a =  getlite($token_facebook_android);
// Now call the updated getlite with the same SSO credentials
//$token_facebook_a = getlite($t1_email, $blob, $email5);
@file_put_contents(__DIR__ . '/a_b_d.log',
                    date('c') . " user token =" . json_encode($token_facebook_android, JSON_UNESCAPED_UNICODE) . "\n",
                    FILE_APPEND
                );
@file_put_contents(__DIR__ . '/a_b_d.log',
                    date('c') . " full token =" . json_encode($token_facebook_a, JSON_UNESCAPED_UNICODE) . "\n",
                    FILE_APPEND
                );                
 //$new_token = getNewAccessToken($token_facebook_android);
$to_fa = '';
header('Content-Type: application/json');
 

      echo json_encode([
        'token' => $token_facebook_a
                

 
    ]);
}

  

 
  
 