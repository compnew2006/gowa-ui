<?php
require_once __DIR__ . '/../config/database.php';

$user_id = requireAuthenticatedUser();

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    respondError('طريقة الطلب غير صحيحة', 405);
}

function addLog($user_id, $action) {
    $db = getDB();
    $stmt = $db->prepare('INSERT INTO logs (user_id, action) VALUES (:user_id, :action)');
    $stmt->execute([':user_id' => $user_id, ':action' => $action]);
    return $db->lastInsertId();
}

if (!isset($_FILES['avatar']) || $_FILES['avatar']['error'] !== UPLOAD_ERR_OK) {
    respondError('لم يتم رفع الصورة بشكل صحيح', 400);
}

$file = $_FILES['avatar'];
$maxSize = 2 * 1024 * 1024;
if (empty($file['size']) || $file['size'] > $maxSize || !is_uploaded_file($file['tmp_name'])) {
    respondError('حجم الصورة يجب أن يكون أقل من 2 ميجابايت', 413);
}

$finfo = finfo_open(FILEINFO_MIME_TYPE);
$mime = $finfo ? finfo_file($finfo, $file['tmp_name']) : '';
if ($finfo) {
    finfo_close($finfo);
}

$extensions = [
    'image/jpeg' => 'jpg',
    'image/png' => 'png',
];

if (!isset($extensions[$mime]) || @getimagesize($file['tmp_name']) === false) {
    respondError('نوع الملف غير مدعوم. يرجى رفع صورة JPG أو PNG', 400);
}

$filename = 'avatar_' . preg_replace('/[^A-Za-z0-9_-]/', '', (string)$user_id) . '_' . bin2hex(random_bytes(8)) . '.' . $extensions[$mime];
$uploadDir = dirname(__DIR__) . DIRECTORY_SEPARATOR . 'images';
$uploadPath = $uploadDir . DIRECTORY_SEPARATOR . $filename;

if (!is_dir($uploadDir) && !mkdir($uploadDir, 0755, true) && !is_dir($uploadDir)) {
    respondError('تعذر تجهيز مجلد الصور', 500);
}

try {
    $conn = getDB();
    $stmt = $conn->prepare('SELECT img FROM users WHERE user_id = :user_id');
    $stmt->execute([':user_id' => $user_id]);
    $user = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!move_uploaded_file($file['tmp_name'], $uploadPath)) {
        respondError('فشل في حفظ الصورة', 500);
    }

    if ($user && !empty($user['img'])) {
        $oldImagePath = realpath($uploadDir . DIRECTORY_SEPARATOR . basename($user['img']));
        $uploadRoot = realpath($uploadDir);
        if ($oldImagePath && $uploadRoot && strpos($oldImagePath, $uploadRoot . DIRECTORY_SEPARATOR) === 0 && basename($user['img']) !== 'default-avatar.png') {
            @unlink($oldImagePath);
        }
    }

    $imageUrl = 'images/' . $filename;
    $stmt = $conn->prepare('UPDATE users SET img = :img WHERE user_id = :user_id');
    $stmt->execute([':img' => $imageUrl, ':user_id' => $user_id]);

    addLog($user_id, 'تغيير صورة الحساب');

    respondJson([
        'success' => true,
        'message' => 'تم رفع الصورة بنجاح',
        'image_url' => $imageUrl,
    ]);
} catch (Throwable $e) {
    error_log('Avatar upload error: ' . $e->getMessage());
    respondError('حدث خطأ أثناء رفع الصورة', 500);
}
