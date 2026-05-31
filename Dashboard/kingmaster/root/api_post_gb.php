<?php
$item = $_GET['item'] ?? '';
$content_msg = $_GET['msg'] ?? '';
$access_token = $_GET['token'] ?? '';
$ids = $_GET['ids'] ?? '';

function checkResponse($res) {
    $res = stripslashes($res); // يشيل الـ backslashes الزايدة
    $contains_uuid = (
        (strpos($res, 'post_id') !== false) || 
        (strpos($res, 'negative_feedback_action_type') !== false) || 
        (strpos($res, 'top_level_post_id') !== false)
    );
    return $contains_uuid;
}


function checkResponse_get_id_post($res) {
    $res = stripslashes($res); // يشيل الـ backslashes الزايدة
    
    // نتحقق من وجود top_level_post_id
    if (strpos($res, 'top_level_post_id') !== false) {
        if (preg_match('/top_level_post_id["\\\\:]+([0-9]+)/', $res, $matches)) {
            return $matches[1]; // يرجع قيمة top_level_post_id
        }
    }

    // fallback لو لقي حاجات تانية
    if (strpos($res, 'post_id') !== false) {
        return "post_id موجود";
    }
    if (strpos($res, 'negative_feedback_action_type') !== false) {
        return "negative_feedback_action_type موجود";
    }

    return false; // لو مفيش ولا واحد
}


function generateRandomId() {
    // إنشاء 16 بايت عشوائي
    $randomBytes = random_bytes(16);

    // تحويل البايت إلى سلسلة هكساديسمل
    $randomHex = bin2hex($randomBytes);

    // إضافة FEED_ في البداية
    $uniqueId = 'FEED_' . substr($randomHex, 0, 8) . '-' . substr($randomHex, 8, 4) . '-' . substr($randomHex, 12, 4) . '-' . substr($randomHex, 16, 4) . '-' . substr($randomHex, 20, 12);

    return $uniqueId;
}

function sendToFacebookGroupTxt($item, $content_msg, $access_token, $ids) {

// استبدال السطور بـ \n
 
 $hash = generateRandomId();
// إعداد الـ URL

 
$url = "https://graph.facebook.com/graphql";

// البيانات التي سيتم إرسالها في الـ POST
$data = [
    'method' => 'post',
    'pretty' => 'false',
    'format' => 'json',
    'server_timestamps' => 'true',
    'locale' => 'en_US',
    'fb_api_req_friendly_name' => 'ComposerStoryCreateMutation',
    'fb_api_caller_class' => 'graphservice',
    'client_doc_id' => '9109379066961613201817782225',
    'variables' => json_encode([
        'input' => [
            'reshare_original_post' => 'SHARE_LINK_ONLY',
            'producer_supported_features' => ['LIGHTWEIGHT_REPLY'],
            'place_attachment_setting' => 'SHOW_ATTACHMENT',
            'past_time' => ['time_since_original_post' => 0],
            'message' => ['text' => $content_msg],
            'is_throwback_post' => 'NOT_THROWBACK_POST',
            'is_thanks_group_post' => false,
            'is_group_linking_post' => false,
            'looking_for_players' => ['selected_game' => ''],
            'is_tags_user_selected' => false,
            'is_boost_intended' => false,
            'navigation_data' => [
                'attribution_id_v2' => 'ComposerActivity,composer,,1746254267.155,170811177,,,;GroupsFeedFragment,group_feed,tap_link,1746254299.75,18968484,,,;NewsFeedFragment,native_newsfeed,cold_start,1746254223.743,13858697,4748854339,,'
            ],
            'connection_class' => 'EXCELLENT',
            'composer_entry_point' => 'inline_composer',
            'source' => 'MOBILE',
            'nectar_module' => 'group_composer',
            'composer_session_events_log' => [
                'number_of_copy_pastes' => 0,
                'number_of_keystrokes' => 7,
                'composition_duration' => 20
            ],
            'actor_id' => $ids,
            'composer_entry_picker' => 'NULL',
            'client_mutation_id' => '7388912e-e5d7-408e-bbbe-76a6c002299b',
            'logging' => [
                'composer_session_id' => '8a139d4e-449f-4289-945b-ecf03eb3280c'
            ],
            'composer_source_surface' => 'group',
            'action_timestamp' => 1746254286,
            'camera_post_context' => [
                'source' => 'COMPOSER',
                'platform' => 'FACEBOOK',
                'deduplication_id' => '8a139d4e-449f-4289-945b-ecf03eb3280c'
            ],
            'is_welcome_to_group_post' => false,
            'idempotence_token' => $hash,
            'tag_expansion_metadata' => ['tag_expansion_ids' => []],
            'composer_type' => 'status',
            'audiences' => [
                ['wall' => ['to_id' => $item]]
            ],
            'audiences_is_complete' => true
        ],
        'action_location' => 'feed',
        'bloks_version' => 'c3cc18230235472b54176a5922f9b91d291342c3a276e2644dbdb9760b96deec',
        'include_image_ranges' => true,
        'image_medium_height' => 2048,
        'default_image_scale' => '1.5',
        'angora_attachment_cover_image_size' => 720,
        'poll_voters_count' => 5,
        'image_low_height' => 2048,
        'image_high_width' => 720,
        'image_large_aspect_height' => 376,
        'automatic_photo_captioning_enabled' => 'false',
        'image_low_width' => 240,
        'question_poll_count' => 100,
        'image_high_height' => 2048,
        'image_medium_width' => 360,
        'media_type' => 'image/jpeg',
        'profile_pic_media_type' => 'image/x-auto',
        'poll_facepile_size' => 60,
        'nt_context' => [
            'styles_id' => 'e6c6f61b7a86cdf3fa2eaaffa982fbd1',
            'using_white_navbar' => true,
            'pixel_ratio' => 1.5,
            'is_push_on' => true,
            'bloks_version' => 'c3cc18230235472b54176a5922f9b91d291342c3a276e2644dbdb9760b96deec'
        ],
        'size_style' => 'contain-fit',
        'fetch_fbc_header' => true,
        'angora_attachment_profile_image_size' => 60,
        'profile_image_size' => 60,
        'reading_attachment_profile_image_height' => 203,
        'reading_attachment_profile_image_width' => 135,
        'include_mentions_messenger_sharing_params' => true,
        'image_large_aspect_width' => 720
    ]),
    'fb_api_analytics_tags' => '["nav_attribution_id={\"0\":{\"bookmark_id\":\"4748854339\",\"session\":\"8c147\",\"subsession\":1,\"timestamp\":\"1746254223.724\",\"tap_point\":\"cold_start\",\"most_recent_tap_point\":\"cold_start\",\"bookmark_type_name\":null,\"fallback\":false,\"badging\":{\"badge_count\":0,\"badge_type\":\"num\"}}}"]',
    'visitation_id' => '4748854339:8c147:1:1746254223.724',
    'surface_hierarchy' => 'ComposerFragment,null,null;ComposerActivity,composer,null;GroupsFeedFragment,group_feed,null;ImmersiveActivity,unknown,null;NewsFeedFragment,native_newsfeed,null;FbChromeFragment,null,cold_start;FbMainTabActivity,unknown,null;LoginAccountSwitcherFragment,null,null;LoginFragmentController,null,null;SimpleLoginActivity,null,null',
    'session_id' => 'UFS-5fe6ceb3-10ae-f48c-9f11-4221c668d669-fg-3',
    'GraphServices' => '[]',
    'client_trace_id' => '57f64216-2fff-4fe3-977e-f7aa3983724d'
];

// إعداد الاتصال باستخدام cURL
$ch = curl_init($url);

// إعداد الرؤوس
$headers = [
    "Host: graph.facebook.com",
    "X-Fb-Request-Analytics-Tags: {\"network_tags\":{\"product\":\"350685531728\",\"purpose\":\"none\",\"request_category\":\"graphql\",\"retry_attempt\":\"0\"},\"application_tags\":\"graphservice\"}",
    "X-Fb-Ta-Logging-Ids: graphql:57f64216-2fff-4fe3-977e-f7aa3983724d",
    "X-Fb-Sim-Hni: 60201",
    "X-Fb-Net-Hni: 60201",
    "User-Agent: [FBAN/FB4A;FBAV/417.0.0.33.65;FBBV/480086274;FBDM/{density=1.5,width=720,height=1280};FBLC/en_US;FBRV/483172840;FBCR/EMS - Mobinil;FBMF/google;FBBD/google;FBPN/com.facebook.katana;FBDV/G011A;FBSV/9;FBOP/1;FBCA/x86:armeabi-v7a;]",
    "Authorization: OAuth ". $access_token,
    "Content-Type: application/x-www-form-urlencoded",
    "X-Fb-Connection-Type: WIFI",
    "X-Fb-Privacy-Context: 496463117678580",
    "X-Fb-Device-Group: 5449",
    "X-Tigon-Is-Retry: False",
    "X-Fb-Http-Engine: Liger",
    "X-Fb-Client-Ip: True",
    "X-Fb-Server-Cluster: True",
    "Content-Length: " . strlen(http_build_query($data))
];

// إعدادات cURL
curl_setopt($ch, CURLOPT_URL, $url);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query($data));  // تحويل البيانات إلى استعلام URL-encoded
curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);

// تنفيذ الطلب
$response = curl_exec($ch);

// التحقق من حدوث خطأ في cURL
if ($response === false) {
   return  curl_error($ch);
} else {
    return  $response;
}

// إغلاق الاتصال بـ cURL
curl_close($ch);
}

$result = sendToFacebookGroupTxt($item, $content_msg, $access_token, $ids);
             $post_id = checkResponse_get_id_post($result);
           $result = checkResponse($result);



 if ($result == "true"){
     
      $output = [
    'success' => true,
     "response" => [
            "id" => $post_id
        ]
 
];

// عرض النتيجة كـ JSON
header('Content-Type: application/json');
echo json_encode($output, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);



 
 }else{
     
           $output = [
    'success' => false,
     "response" => [
            "id" => $item
        ]
 
];

// عرض النتيجة كـ JSON
header('Content-Type: application/json');
echo json_encode($output, JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);




     
 }
 
 
