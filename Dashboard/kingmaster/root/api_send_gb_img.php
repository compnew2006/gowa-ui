<?php


function generateRandomId() {
    // إنشاء 16 بايت عشوائي
    $randomBytes = random_bytes(16);

    // تحويل البايت إلى سلسلة هكساديسمل
    $randomHex = bin2hex($randomBytes);

    // إضافة FEED_ في البداية
    $uniqueId = 'FEED_' . substr($randomHex, 0, 8) . '-' . substr($randomHex, 8, 4) . '-' . substr($randomHex, 12, 4) . '-' . substr($randomHex, 16, 4) . '-' . substr($randomHex, 20, 12);

    return $uniqueId;
}

function upload_photo_to_facebook($token, $img){
    $codes = generateRandomId();

// Facebook Graph API URL
$url = "https://graph.facebook.com/v10.0/me/photos";

// مسار الصورة على الخادم
$image_path =$img;
$image_name = basename($image_path);
// إعداد الـ cURL
$ch = curl_init();

// البيانات التي سيتم إرسالها
$data = array(
    'published' => 'false',
    'audience_exp' => 'true',
    'qn' => 'a948be50-c403-424a-9dbf-4033cdaf620a',
    'composer_session_id' => 'a948be50-c403-424a-9dbf-4033cdaf620a',
    'idempotence_token' => $codes,
    'source_type' => 'group',
    'locale' => 'en_US',
    'client_country_code' => 'EG',
    'fb_api_req_friendly_name' => 'upload-photo',
    'fb_api_caller_class' => 'MultiPhotoUploader',
    'source' => new CURLFile($image_path, 'image/jpeg', $image_name)  // إضافة الصورة
);

// إعداد الـ headers
$headers = array(
    "Authorization: OAuth ".$token,
    "X-Fb-Request-Analytics-Tags: {\"network_tags\":{\"product\":\"350685531728\",\"retry_attempt\":\"0\"},\"application_tags\":\"MULTIMEDIA\"}",
    "X-Fb-Device-Group: 5449",
    "X-Fb-Net-Hni: 60201",
    "Zero-Rated: 0",
    "X-Fb-Sim-Hni: 60201",
    "X-Fb-Connection-Quality: EXCELLENT",
    "X-Zero-Eh: 2,,AXLC68P0_D_ROG3WVSV6y3XeAqA4ppODl8NMEipf0SAaipzNjf1zKr0EfvM2DdwC2E8",
    "X-Fb-Friendly-Name: upload-photo",
    "X-Fb-Connection-Bandwidth: 35261165",
    "User-Agent: [FBAN/FB4A;FBAV/417.0.0.33.65;FBBV/480086274;FBDM/{density=1.5,width=720,height=1280};FBLC/en_US;FBRV/483172840;FBCR/EMS - Mobinil;FBMF/google;FBBD/google;FBPN/com.facebook.katana;FBDV/G011A;FBSV/9;FBOP/1;FBCA/x86:armeabi-v7a;]",
    "X-Fb-Connection-Type: WIFI",
    "X-Tigon-Is-Retry: False",
    "Priority: u=3,i",
    "X-Fb-Http-Engine: Liger",
    "X-Fb-Client-Ip: True",
    "X-Fb-Server-Cluster: True"
);

// إعداد خيارات cURL
curl_setopt($ch, CURLOPT_URL, $url);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, $data);
curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);

 






    $response = curl_exec($ch);

    if (curl_errno($ch)) {
        curl_close($ch);
        return [
            "success" => false,
            "error" => curl_error($ch)
        ];
    }

    curl_close($ch);

    $data = json_decode($response, true);

    if (!is_array($data)) {
        return [
            "success" => false,
            "error" => "Invalid JSON response",
            "raw" => $response
        ];
    }

    if (isset($data['id']) && !empty($data['id'])) {
        return [
            "success" => true,
            "id" => $data['id']
        ];
    }

    return [
        "success" => false,
        "error" => "ID not found",
        "response" => $data
    ];


}


function send_photo_to_facebook_group($content_msg, $access_token, $photo_id, $group_ids, $token, $my_id ){
      

// استبدال السطور بـ \n
 
$ids = $my_id ;
$hash = generateRandomId();
// إعداد الـ URL

// إعدادات الـ cURL
$ch = curl_init();

// URL الخاص بـ Graph API لفيسبوك
$url = "https://graph.facebook.com/graphql";

// إعداد البيانات التي سيتم إرسالها
$data = array(
    'method' => 'post',
    'pretty' => 'false',
    'format' => 'json',
    'server_timestamps' => 'true',
    'locale' => 'en_US',
    'fb_api_req_friendly_name' => 'ComposerStoryCreateMutation',
    'fb_api_caller_class' => 'graphservice',
    'client_doc_id' => '9109379066961613201817782225',
    'variables' => json_encode(array(
        "input" => array(
            "producer_supported_features" => ["LIGHTWEIGHT_REPLY"],
            "place_attachment_setting" => "SHOW_ATTACHMENT",
            "past_time" => array("time_since_original_post" => 1),
            "navigation_data" => array(
                "attribution_id_v2" => "GroupsFeedFragment,group_feed,tap_link,1746264897.545,142547849,,,;NewsFeedFragment,native_newsfeed,tap_top_jewel_bar,1746264895.260,13858697,4748854339,,"
            ),
            "message" => array("text" => $content_msg),
            "is_throwback_post" => "NOT_THROWBACK_POST",
            "is_thanks_group_post" => false,
            "is_tags_user_selected" => false,
            "is_boost_intended" => false,
            "is_welcome_to_group_post" => false,
            "idempotence_token" => $hash,
            "tag_expansion_metadata" => array("tag_expansion_ids" => []),
            "composer_type" => "status",
            "composer_entry_picker" => "NULL",
            "implicit_with_tags_ids" => [],
            "connection_class" => "EXCELLENT",
            "composer_entry_point" => "inline_composer",
            "source" => "MOBILE",
            "nectar_module" => "group_composer",
            "composer_session_events_log" => array(
                "number_of_copy_pastes" => 0,
                "number_of_keystrokes" => 10,
                "composition_duration" => 53
            ),
            "looking_for_players" => array("selected_game" => ""),
            "is_group_linking_post" => false,
            "actor_id" => $ids,
            "client_mutation_id" => "3c5ff16a-9f8b-42fc-adf3-4a9aeb25b801",
            "logging" => array("composer_session_id" => "a948be50-c403-424a-9dbf-4033cdaf620a"),
            "composer_source_surface" => "group",
            "camera_post_context" => array(
                "source" => "COMPOSER",
                "platform" => "FACEBOOK",
                "deduplication_id" => "a948be50-c403-424a-9dbf-4033cdaf620a"
            ),
            "action_timestamp" => 1746264968,
            "reshare_original_post" => "SHARE_LINK_ONLY",
            "coordinates" => array(
                "longitude" => 121.48789833333333,
                "latitude" => 31.24916,
                "timestamp" => 1746207315000,
                "accuracy" => 10
            ),
            "attachments" => array(
                array(
                    "photo" => array(
                        "unified_stories_media_source" => "CAMERA_ROLL",
                        "story_media_audio_data" => array(
                            "raw_media_type" => "PHOTO"
                        ),
                        "id" => $photo_id
                    )
                )
            ),
            "audiences" => array(
                array("wall" => array("to_id" => $group_ids))
            ),
            "audiences_is_complete" => true
        ),
        "action_location" => "feed",
        "bloks_version" => "c3cc18230235472b54176a5922f9b91d291342c3a276e2644dbdb9760b96deec",
        "include_image_ranges" => true,
        "image_medium_height" => 2048,
        "default_image_scale" => "1.5",
        "angora_attachment_cover_image_size" => 720,
        "poll_voters_count" => 5,
        "image_low_height" => 2048,
        "image_high_width" => 720,
        "image_large_aspect_height" => 376,
        "automatic_photo_captioning_enabled" => "false",
        "image_low_width" => 240,
        "question_poll_count" => 100,
        "image_high_height" => 2048,
        "image_medium_width" => 360,
        "media_type" => "image/jpeg",
        "profile_pic_media_type" => "image/x-auto",
        "poll_facepile_size" => 60,
        "nt_context" => array(
            "styles_id" => "e6c6f61b7a86cdf3fa2eaaffa982fbd1",
            "using_white_navbar" => true,
            "pixel_ratio" => 1.5,
            "is_push_on" => true,
            "bloks_version" => "c3cc18230235472b54176a5922f9b91d291342c3a276e2644dbdb9760b96deec"
        ),
        "size_style" => "contain-fit",
        "fetch_fbc_header" => true,
        "angora_attachment_profile_image_size" => 60,
        "profile_image_size" => 60,
        "reading_attachment_profile_image_height" => 203,
        "reading_attachment_profile_image_width" => 135,
        "include_mentions_messenger_sharing_params" => true,
        "image_large_aspect_width" => 720
    ))
);

// إعداد الـ headers
$headers = array(
    "Authorization: OAuth ". $token,
    "Content-Type: application/x-www-form-urlencoded"
);

// إعداد خيارات الـ cURL
curl_setopt($ch, CURLOPT_URL, $url);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, http_build_query($data));
curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);

// تنفيذ الطلب
$response = curl_exec($ch);

// التحقق من وجود أي أخطاء
if(curl_errno($ch)) {
   return curl_error($ch);
} else {
     
    // طباعة الاستجابة من الخادم
    return $response;
}

// إغلاق الـ cURL
curl_close($ch);
   
}


    $token = $_GET['token'] ?? '';
$content_msg = $_GET['msg'] ?? '';
$group_ids = $_GET['id'] ?? '';
$my_id = $_GET['my_id'] ?? '';

 
$img= $_GET['img'] ?? '';
$result = upload_photo_to_facebook($token, $img);



header('Content-Type: application/json; charset=utf-8');

if ($result['success']) {
    $res = send_photo_to_facebook_group($content_msg, $token, $result['id'], $group_ids, $token, $my_id );
     

     
    echo json_encode([
        "success" =>  true,
        "id" => $result['id'],
        "post_id" => $group_ids. "_".$result['id'],
        "res" => $res
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
} else {
    echo json_encode([
                "success" =>  false,
        "error" => $result['error'],
        "debug" => $result['response'] ?? null
    ], JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE);
}
