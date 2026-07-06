<?php
require_once __DIR__ . '/../config/database.php';

requireAuthenticatedUser();

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    respondError('Method not allowed', 405);
}

if (!isset($_FILES['file']) || $_FILES['file']['error'] !== UPLOAD_ERR_OK) {
    respondError('لم يتم رفع الملف بشكل صحيح', 400);
}

$file = $_FILES['file'];
$maxSize = 10 * 1024 * 1024;
if (empty($file['size']) || $file['size'] > $maxSize || !is_uploaded_file($file['tmp_name'])) {
    respondError('حجم الملف غير صالح أو كبير جداً', 413);
}

$finfo = finfo_open(FILEINFO_MIME_TYPE);
$mime = $finfo ? finfo_file($finfo, $file['tmp_name']) : '';
if ($finfo) {
    finfo_close($finfo);
}

$handle = fopen($file['tmp_name'], 'rb');
$magic = $handle ? fread($handle, 5) : '';
if ($handle) {
    fclose($handle);
}

if ($mime !== 'application/pdf' || $magic !== '%PDF-') {
    respondError('يسمح بملفات PDF فقط', 400);
}

$uploadDir = dirname(__DIR__) . DIRECTORY_SEPARATOR . 'uploads' . DIRECTORY_SEPARATOR . 'bot';
if (!is_dir($uploadDir) && !mkdir($uploadDir, 0755, true) && !is_dir($uploadDir)) {
    respondError('تعذر تجهيز مجلد الرفع', 500);
}

$final = bin2hex(random_bytes(16)) . '.pdf';
$dest = $uploadDir . DIRECTORY_SEPARATOR . $final;

if (!move_uploaded_file($file['tmp_name'], $dest)) {
    respondError('فشل حفظ الملف', 500);
}

respondJson(['success' => true, 'url' => 'uploads/bot/' . $final, 'name' => $final]);
