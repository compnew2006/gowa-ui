<?php

require_once '../config/database.php';
 
require_once '../vendor/autoload.php';

use PhpOffice\PhpSpreadsheet\Spreadsheet;
use PhpOffice\PhpSpreadsheet\Writer\Xlsx;
use PhpOffice\PhpSpreadsheet\Cell\DataType;


if (!isset($_GET['camp_id'])) {
    die("❌ Camp ID is required");
}

$camp_id = $_GET['camp_id'];
$tool = $_GET['tool'];

if ($tool == "wa-extract-contacts") {

    $db = getDB(); // اتصال PDO

    $stmt = $db->prepare("SELECT `name`, `phone`, `pushname` FROM `wa_contacts` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "wa_extract_contacts_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");

    // كتابة BOM (حتى يفتح الملف بشكل سليم في Excel)
    fwrite($output, "\xEF\xBB\xBF");

    // كتابة العناوين
    fputcsv($output, ["الاسم", "الهاتف", "الاسم البديل"]);

    // كتابة الصفوف
    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;
}


elseif ($tool =="Extract-Messages-WA"){

    $db = getDB(); // اتصال PDO

    $stmt = $db->prepare("SELECT `name`, `phone`, `pushname`, `unreadCount` FROM `wa_msg` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "wa_extract_Messages_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");

    // كتابة BOM (حتى يفتح الملف بشكل سليم في Excel)
    fwrite($output, "\xEF\xBB\xBF");

    // كتابة العناوين
    fputcsv($output, ["الاسم", "الهاتف", "الاسم البديل", "عدد الرسائل الغير مقروءة"]);

    // كتابة الصفوف
    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;
}elseif ($tool =="Extract-Groups-WA"){

    $db = getDB(); // اتصال PDO

    $stmt = $db->prepare("SELECT `name`, `gb_id`, `participantsCount` FROM `gb_wa` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "wa_extract_Groups_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");

    // كتابة BOM (حتى يفتح الملف بشكل سليم في Excel)
    fwrite($output, "\xEF\xBB\xBF");

    // كتابة العناوين
    fputcsv($output, ["الاسم", "المعرف", "عدد الاعضاء"]);

    // كتابة الصفوف
    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;
}elseif ($tool =="Extract-Members-WA"){

    $db = getDB(); // اتصال PDO

    $stmt = $db->prepare("SELECT `name`, `phone`, `pushname` FROM `wa_members_gb` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "wa_extract_Members_Groups_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");

    // كتابة BOM (حتى يفتح الملف بشكل سليم في Excel)
    fwrite($output, "\xEF\xBB\xBF");

    // كتابة العناوين
    fputcsv($output, ["الاسم", "المعرف", "الاسم الثاني"]);

    // كتابة الصفوف
    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;
}elseif ($tool =="Extract-Messages-WA-send"){

    $db = getDB(); // اتصال PDO
    $stmt = $db->prepare("SELECT `phone`, `st` FROM `rb_wa` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "wa_report_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");

    // كتابة BOM (حتى يفتح الملف بشكل سليم في Excel)
    fwrite($output, "\xEF\xBB\xBF");

    // كتابة العناوين
    fputcsv($output, ["الرقم","الحاله"]);

    // كتابة الصفوف
    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;

}elseif ($tool =="filter-WA-send"){


    $db = getDB(); // اتصال PDO
    $stmt = $db->prepare("SELECT `phone`, `name`, `ty` FROM `filter_wa` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "wa_report_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");

    // كتابة BOM (حتى يفتح الملف بشكل سليم في Excel)
    fwrite($output, "\xEF\xBB\xBF");

    // كتابة العناوين
    fputcsv($output, ["الرقم","الاسم","الحالة"]);

    // كتابة الصفوف
    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;


}elseif ($tool =="Extract-serch-pg_b"){


$db = getDB(); // اتصال PDO
$stmt = $db->prepare("SELECT `page_id`, `name`, `followers_count` FROM `fb_serch` WHERE `campaign_id` = ?");
$stmt->execute([$camp_id]);
$result = $stmt->fetchAll(PDO::FETCH_ASSOC);

if (!$result) {
    die("❌ No data found for this camp_id");
}

$spreadsheet = new Spreadsheet();
$sheet = $spreadsheet->getActiveSheet();

// عناوين الأعمدة
$sheet->setCellValue('A1', 'المعرف');
$sheet->setCellValue('B1', 'الاسم');
$sheet->setCellValue('C1', 'العدد');

// Freeze header
$sheet->freezePane('A2');

// كتابة البيانات
$rowNum = 2;
foreach ($result as $row) {

    // إجبار page_id كنص (حل نهائي لمشكلة الأرقام)
    $sheet->setCellValueExplicit(
        'A' . $rowNum,
        $row['page_id'],
        DataType::TYPE_STRING
    );

    $sheet->setCellValue('B' . $rowNum, $row['name']);
    $sheet->setCellValue('C' . $rowNum, $row['followers_count']);

    $rowNum++;
}

// Auto size
foreach (['A', 'B', 'C'] as $col) {
    $sheet->getColumnDimension($col)->setAutoSize(true);
}

$filename = "wa_report_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".xlsx";

// Headers للتحميل
header('Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');
header("Content-Disposition: attachment; filename=\"$filename\"");
header('Cache-Control: max-age=0');

$writer = new Xlsx($spreadsheet);
$writer->save('php://output');
exit;


}elseif ($tool =="ddbs"){

 
$db = getDB(); // اتصال PDO
$stmt = $db->prepare("SELECT `fb_id`, `name`, `phone`, `gender`, `birthday`, `location`, `relashan`, `email`, `work`, `educ` FROM `db_camp` WHERE `campaign_id` = ?");
$stmt->execute([$camp_id]);
$result = $stmt->fetchAll(PDO::FETCH_ASSOC);

if (!$result) {
    die("❌ No data found for this camp_id");
}

$spreadsheet = new Spreadsheet();
$sheet = $spreadsheet->getActiveSheet();

// عناوين الأعمدة
$sheet->setCellValue('A1', 'المعرف');
$sheet->setCellValue('B1', 'الاسم');
$sheet->setCellValue('C1', 'الهاتف');
$sheet->setCellValue('D1', 'الجنس');
$sheet->setCellValue('E1', 'تاريخ الميلاد');
$sheet->setCellValue('F1', 'العنوان');
$sheet->setCellValue('G1', 'الحاله');
$sheet->setCellValue('H1', 'البريد');
$sheet->setCellValue('I1', 'العمل');
$sheet->setCellValue('J1', 'التعليم');
// Freeze header
$sheet->freezePane('A2');

// كتابة البيانات
$rowNum = 2;
foreach ($result as $row) {

    // fb_id كنص
    $sheet->setCellValueExplicit(
        'A' . $rowNum,
        $row['fb_id'],
        DataType::TYPE_STRING
    );

    $sheet->setCellValue('B' . $rowNum, $row['name']);
    $sheet->setCellValueExplicit('C' . $rowNum, $row['phone'], DataType::TYPE_STRING);
    $sheet->setCellValue('D' . $rowNum, $row['gender']);
    $sheet->setCellValue('E' . $rowNum, $row['birthday']);
    $sheet->setCellValue('F' . $rowNum, $row['location']);
    $sheet->setCellValue('G' . $rowNum, $row['relashan']);
    $sheet->setCellValue('H' . $rowNum, $row['email']);
    $sheet->setCellValue('I' . $rowNum, $row['work']);
    $sheet->setCellValue('J' . $rowNum, $row['educ']);

    $rowNum++;
}

// Auto size
foreach (['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J'] as $col) {
    $sheet->getColumnDimension($col)->setAutoSize(true);
}

$filename = "wa_report_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".xlsx";

// Headers للتحميل
header('Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');
header("Content-Disposition: attachment; filename=\"$filename\"");
header('Cache-Control: max-age=0');

$writer = new Xlsx($spreadsheet);
$writer->save('php://output');
exit;


}
elseif ($tool == "Extract Posts IG") {

    $db = getDB();
    $stmt = $db->prepare("SELECT `shortcode`, `post_type`, `content`, `likes_count`, `comments_count`, `post_date` FROM `ig_post` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "ig_posts_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");
    // كتابة BOM لدعم اللغة العربية
    fwrite($output, "\xEF\xBB\xBF");

    // العناوين
    fputcsv($output, ["الكود (Shortcode)", "نوع البوست", "المحتوى", "الإعجابات", "التعليقات", "التاريخ"]);

    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;

} elseif ($tool == "Extract Likes IG") {

    $db = getDB();
    $stmt = $db->prepare("SELECT `phone`, `name`, `comment_date` FROM `ig_msg` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "ig_likes_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");
    fwrite($output, "\xEF\xBB\xBF");

    fputcsv($output, ["المعرف (ID)", "اليوزر (Username)", "التاريخ"]);

    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;

} elseif ($tool == "Extract Comments IG") {

    $db = getDB();
    $stmt = $db->prepare("SELECT `phone`, `name`, `comment`, `comment_date` FROM `ig_msg` WHERE `campaign_id` = ?");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "ig_comments_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");
    fwrite($output, "\xEF\xBB\xBF");

    fputcsv($output, ["المعرف (ID)", "اليوزر (Username)", "التعليق", "التاريخ"]);

    foreach ($result as $row) {
        fputcsv($output, $row);
    }

    fclose($output);
    exit;
}
elseif ($tool === "Extract DMs IG") {

    $db = getDB(); 

    // 💡 نسحب last_message_date بدلاً من created_at، ونرتب بالأحدث تفاعلاً
    $stmt = $db->prepare("SELECT `ig_user_id`, `username`, `full_name`, `last_message_date` FROM `ig_dms` WHERE `campaign_id` = ? ORDER BY `last_message_date` DESC");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "ig_extracted_dms_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";

    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");
    fwrite($output, "\xEF\xBB\xBF");

    // 💡 تغيير العنوان ليكون "تاريخ آخر رسالة"
    fputcsv($output, ["المعرف (IG ID)", "اسم المستخدم", "الاسم بالكامل", "تاريخ آخر رسالة"]);

    foreach ($result as $row) {
        $ig_id = "\t" . $row['ig_user_id'];
        
        fputcsv($output, [
            $ig_id,
            $row['username'],
            $row['full_name'],
            $row['last_message_date'] ?: 'غير متوفر' // 👈 طباعة التاريخ الجديد
        ]);
    }

    fclose($output);
    exit;
}
elseif ($tool === "Extract Follows IG") {

    $db = getDB(); 
    $stmt = $db->prepare("SELECT `ig_user_id`, `username`, `full_name`, `extract_type`, `created_at` FROM `ig_follow` WHERE `campaign_id` = ? ORDER BY id DESC");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) {
        die("❌ No data found for this camp_id");
    }

    $filename = "ig_follows_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";
    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=$filename");

    $output = fopen("php://output", "w");
    fwrite($output, "\xEF\xBB\xBF");
    fputcsv($output, ["المعرف (IG ID)", "اسم المستخدم", "الاسم بالكامل", "نوع الاستخراج", "تاريخ الاستخراج"]);

    foreach ($result as $row) {
        $ig_id = "\t" . $row['ig_user_id'];
        $type_text = ($row['extract_type'] === 'following') ? 'يتابعهم (Following)' : 'متابعون (Followers)';
        fputcsv($output, [$ig_id, $row['username'], $row['full_name'], $type_text, $row['created_at']]);
    }

    fclose($output);
    exit;
}
 elseif ($tool === "Search Profile IG" || $tool === "Search Bio IG") {
    // ==========================================
    // 1. تصدير حسابات إنستجرام (Profile & Bio)
    // ==========================================
    $db = getDB(); 
    $stmt = $db->prepare("SELECT * FROM `ig_search_users` WHERE `campaign_id` = ? ORDER BY id DESC");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) { die("❌ لا توجد بيانات لهذه الحملة"); }

    $prefix = ($tool === "Search Bio IG") ? "ig_search_bio_" : "ig_search_profile_";
    $filename = $prefix . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";
    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=\"$filename\"");

    $output = fopen("php://output", "w");
    fwrite($output, "\xEF\xBB\xBF"); // دعم اللغة العربية
    fputcsv($output, ["المعرف (ID)", "اسم المستخدم", "الاسم بالكامل", "كلمة البحث", "حساب خاص؟", "حساب موثق؟", "رابط الحساب", "تاريخ الاستخراج"]);

    foreach ($result as $row) {
        $is_private = ($row['is_private'] == '1') ? 'نعم' : 'لا';
        $is_verified = ($row['is_verified'] == '1') ? 'نعم' : 'لا';
        $profile_url = !empty($row['profile_url']) ? $row['profile_url'] : "https://instagram.com/" . $row['username'];
        
        fputcsv($output, [
            "\t" . $row['ig_user_id'], // \t لمنع الإكسل من تحويله لمعادلة رياضية
            $row['username'],
            $row['full_name'],
            $row['keyword'],
            $is_private,
            $is_verified,
            $profile_url,
            $row['created_at']
        ]);
    }
    fclose($output);
    exit;

} elseif ($tool === "Search Hashtag IG") {
    // ==========================================
    // 2. تصدير الهاشتاجات
    // ==========================================
    $db = getDB(); 
    $stmt = $db->prepare("SELECT * FROM `ig_search_hashtags` WHERE `campaign_id` = ? ORDER BY CAST(`media_count` AS UNSIGNED) DESC");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) { die("❌ لا توجد بيانات لهذه الحملة"); }

    $filename = "ig_search_hashtags_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";
    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=\"$filename\"");

    $output = fopen("php://output", "w");
    fwrite($output, "\xEF\xBB\xBF");
    fputcsv($output, ["المعرف (ID)", "الهاشتاج", "كلمة البحث", "عدد المنشورات", "رابط الهاشتاج", "تاريخ الاستخراج"]);

    foreach ($result as $row) {
        $hashtag_url = !empty($row['hashtag_url']) ? $row['hashtag_url'] : "https://instagram.com/explore/tags/" . $row['hashtag_name'];
        
        fputcsv($output, [
            "\t" . $row['hashtag_id'],
            "#" . $row['hashtag_name'],
            $row['keyword'],
            $row['media_count'],
            $hashtag_url,
            $row['created_at']
        ]);
    }
    fclose($output);
    exit;

} elseif ($tool === "Search Location IG") {
    // ==========================================
    // 3. تصدير الأماكن والمواقع
    // ==========================================
    $db = getDB(); 
    $stmt = $db->prepare("SELECT * FROM `ig_search_locations` WHERE `campaign_id` = ? ORDER BY id DESC");
    $stmt->execute([$camp_id]);
    $result = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!$result) { die("❌ لا توجد بيانات لهذه الحملة"); }

    $filename = "ig_search_locations_" . $camp_id . "_" . date("Y-m-d_H-i-s") . ".csv";
    header("Content-Type: text/csv; charset=UTF-8");
    header("Content-Disposition: attachment; filename=\"$filename\"");

    $output = fopen("php://output", "w");
    fwrite($output, "\xEF\xBB\xBF");
    fputcsv($output, ["المعرف (ID)", "اسم المكان", "كلمة البحث", "العنوان", "خط الطول (Lat)", "خط العرض (Lng)", "رابط خريطة جوجل", "تاريخ الاستخراج"]);

    foreach ($result as $row) {
        // تحويل الإحداثيات لرابط خريطة جوجل مباشر!
        $google_maps_url = "";
        if (!empty($row['lat']) && !empty($row['lng'])) {
            $google_maps_url = "https://www.google.com/maps/search/?api=1&query=" . $row['lat'] . "," . $row['lng'];
        }
        
        fputcsv($output, [
            "\t" . $row['location_id'],
            $row['location_name'],
            $row['keyword'],
            $row['address'],
            $row['lat'],
            $row['lng'],
            $google_maps_url,
            $row['created_at']
        ]);
    }
    fclose($output);
    exit;
}
?>
