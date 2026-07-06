<?php
require_once __DIR__ . '/../config/database.php';

requireAuthenticatedUser();

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    respondError('Method not allowed', 405);
}

if (!isset($_FILES['image']) || $_FILES['image']['error'] !== UPLOAD_ERR_OK) {
    respondError('لم يتم رفع الملف', 400);
}

$file = $_FILES['image'];
$maxSize = 5 * 1024 * 1024;
if (empty($file['size']) || $file['size'] > $maxSize || !is_uploaded_file($file['tmp_name'])) {
    respondError('حجم الملف غير صالح أو كبير جداً', 413);
}

$finfo = finfo_open(FILEINFO_MIME_TYPE);
$mime = $finfo ? finfo_file($finfo, $file['tmp_name']) : '';
if ($finfo) {
    finfo_close($finfo);
}

$extensions = [
    'image/jpeg' => 'jpg',
    'image/png' => 'png',
    'image/gif' => 'gif',
    'image/webp' => 'webp',
];

if (!isset($extensions[$mime]) || @getimagesize($file['tmp_name']) === false) {
    respondError('نوع الملف غير مدعوم', 400);
}

$uploadDir = dirname(__DIR__) . DIRECTORY_SEPARATOR . 'uploads' . DIRECTORY_SEPARATOR . 'messages';
if (!is_dir($uploadDir) && !mkdir($uploadDir, 0755, true) && !is_dir($uploadDir)) {
    respondError('تعذر تجهيز مجلد الرفع', 500);
}

$filename = bin2hex(random_bytes(16)) . '.' . $extensions[$mime];
$filepath = $uploadDir . DIRECTORY_SEPARATOR . $filename;

if (!move_uploaded_file($file['tmp_name'], $filepath)) {
    respondError('فشل رفع الملف', 500);
}

respondJson([
    'success' => true,
    'message' => 'تم رفع الصورة بنجاح',
    'image_path' => 'uploads/messages/' . $filename,
]);
