<?php
require_once __DIR__ . '/../config/database.php';

requireAuthenticatedUser();

$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
if ($method !== 'POST') {
    respondError('Method not allowed', 405);
}

if (!class_exists('ZipArchive')) {
    respondError('ZipArchive غير متاح على الخادم', 500);
}

$payload = readJsonBody(65536);
$ids = isset($payload['ids']) && is_array($payload['ids']) ? $payload['ids'] : [];
$ids = array_values(array_unique(array_filter(array_map('intval', $ids), function ($id) {
    return $id > 0;
})));

if (!$ids) {
    respondError('لا توجد معرفات');
}
if (count($ids) > 100) {
    respondError('يمكن تنزيل 100 ملف كحد أقصى في الطلب الواحد', 413);
}

$uploadDir = realpath(__DIR__ . '/../uploads');
if ($uploadDir === false || !is_dir($uploadDir)) {
    respondError('مجلد الملفات غير متاح', 500);
}

$placeholders = implode(',', array_fill(0, count($ids), '?'));
$rows = fetchAll('SELECT id, filename, original_name FROM media_files WHERE id IN (' . $placeholders . ')', $ids);
if (!$rows) {
    respondError('لا توجد ملفات مطابقة', 404);
}

$zip = new ZipArchive();
$tmpZip = tempnam(sys_get_temp_dir(), 'km_zip_');
if ($tmpZip === false || $zip->open($tmpZip, ZipArchive::OVERWRITE) !== true) {
    respondError('تعذر إنشاء الملف المضغوط', 500);
}

$usedNames = [];
$added = 0;

foreach ($rows as $row) {
    $storedName = safeBaseName($row['filename'] ?? '');
    $path = realpath($uploadDir . DIRECTORY_SEPARATOR . $storedName);

    if ($path === false || strpos($path, $uploadDir . DIRECTORY_SEPARATOR) !== 0 || !is_file($path)) {
        continue;
    }

    $name = safeBaseName($row['original_name'] ?: $storedName, 'media_' . (int)$row['id']);
    $base = pathinfo($name, PATHINFO_FILENAME);
    $ext = pathinfo($name, PATHINFO_EXTENSION);
    $candidate = $name;
    $i = 1;

    while (isset($usedNames[$candidate])) {
        $candidate = $base . ' (' . $i++ . ')' . ($ext ? ('.' . $ext) : '');
    }

    $usedNames[$candidate] = true;
    if ($zip->addFile($path, $candidate)) {
        $added++;
    }
}

$zip->close();

if ($added === 0) {
    @unlink($tmpZip);
    respondError('لا توجد ملفات قابلة للتنزيل', 404);
}

$filename = 'media_selected_' . date('Y-m-d_H-i-s') . '.zip';
header('Content-Type: application/zip');
header('Content-Disposition: attachment; filename="' . $filename . '"');
header('Content-Length: ' . filesize($tmpZip));
header('X-Content-Type-Options: nosniff');
header('Cache-Control: no-store');
readfile($tmpZip);
@unlink($tmpZip);
exit;
