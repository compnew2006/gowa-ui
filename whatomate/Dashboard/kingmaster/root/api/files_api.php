<?php
require_once __DIR__ . '/../config/database.php';

$user_id = requireAuthenticatedUser();
$pdo = getDB();

$action = $_POST['action'] ?? ($_GET['action'] ?? '');

switch ($action) {
    case 'get_all':
        getAllFiles($pdo, $user_id);
        break;
    case 'upload':
        uploadFile($pdo, $user_id);
        break;
    case 'update':
        updateFile($pdo, $user_id);
        break;
    case 'delete':
        deleteFile($pdo, $user_id);
        break;
    case 'get_storage':
        getStorageInfo($pdo, $user_id);
        break;
    default:
        respondError('Invalid action', 400);
}

function getAllFiles($pdo, $user_id) {
    try {
        $stmt = $pdo->prepare('SELECT * FROM files WHERE user_id = ? ORDER BY created_at DESC');
        $stmt->execute([$user_id]);
        respondJson(['success' => true, 'files' => $stmt->fetchAll(PDO::FETCH_ASSOC)]);
    } catch (Throwable $e) {
        error_log('files_api get_all error: ' . $e->getMessage());
        respondError('خطأ في جلب البيانات', 500);
    }
}

function uploadFile($pdo, $user_id) {
    try {
        $name = trim((string)($_POST['name'] ?? ''));
        if ($name === '') {
            respondError('اسم الملف مطلوب', 400);
        }

        if (!isset($_FILES['file']) || $_FILES['file']['error'] !== UPLOAD_ERR_OK || !is_uploaded_file($_FILES['file']['tmp_name'])) {
            respondError('لم يتم رفع الملف بشكل صحيح', 400);
        }

        $file = $_FILES['file'];
        $fileSize = (int)$file['size'];
        if ($fileSize <= 0) {
            respondError('الملف فارغ', 400);
        }

        $finfo = finfo_open(FILEINFO_MIME_TYPE);
        $mimeType = $finfo ? finfo_file($finfo, $file['tmp_name']) : '';
        if ($finfo) {
            finfo_close($finfo);
        }

        $allowed = [
            'image/jpeg' => ['image', 'jpg'],
            'image/png' => ['image', 'png'],
            'image/gif' => ['image', 'gif'],
            'image/webp' => ['image', 'webp'],
            'video/mp4' => ['video', 'mp4'],
            'video/webm' => ['video', 'webm'],
            'application/pdf' => ['pdf', 'pdf'],
            'text/plain' => ['document', 'txt'],
            'text/csv' => ['document', 'csv'],
            'application/vnd.ms-excel' => ['document', 'xls'],
            'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' => ['document', 'xlsx'],
            'application/msword' => ['document', 'doc'],
            'application/vnd.openxmlformats-officedocument.wordprocessingml.document' => ['document', 'docx'],
        ];

        if (!isset($allowed[$mimeType])) {
            respondError('نوع الملف غير مسموح', 400);
        }

        $stmt = $pdo->prepare('SELECT COALESCE(SUM(file_size), 0) as total_size FROM files WHERE user_id = ?');
        $stmt->execute([$user_id]);
        $usedStorage = (int)$stmt->fetchColumn();
        $maxStorage = (int)configValue('USER_FILE_STORAGE_BYTES', (string)(100 * 1024 * 1024));

        if (($usedStorage + $fileSize) > $maxStorage) {
            respondError('لقد تجاوزت الحد المسموح به من التخزين', 413);
        }

        $uploadDir = dirname(__DIR__) . DIRECTORY_SEPARATOR . 'uploads' . DIRECTORY_SEPARATOR . safeBaseName($user_id);
        if (!is_dir($uploadDir) && !mkdir($uploadDir, 0755, true) && !is_dir($uploadDir)) {
            respondError('تعذر تجهيز مجلد الرفع', 500);
        }

        $fileType = $allowed[$mimeType][0];
        $extension = $allowed[$mimeType][1];
        $uniqueName = bin2hex(random_bytes(16)) . '.' . $extension;
        $filePath = $uploadDir . DIRECTORY_SEPARATOR . $uniqueName;

        if (!move_uploaded_file($file['tmp_name'], $filePath)) {
            respondError('فشل حفظ الملف', 500);
        }

        $relativePath = 'uploads/' . safeBaseName($user_id) . '/' . $uniqueName;
        $stmt = $pdo->prepare('INSERT INTO files (user_id, name, original_name, file_path, file_type, mime_type, file_size) VALUES (?, ?, ?, ?, ?, ?, ?)');
        $stmt->execute([$user_id, $name, safeBaseName($file['name'], 'upload.' . $extension), $relativePath, $fileType, $mimeType, $fileSize]);

        respondJson([
            'success' => true,
            'message' => 'تم رفع الملف بنجاح',
            'file_id' => $pdo->lastInsertId(),
            'file_path' => $relativePath,
        ]);
    } catch (Throwable $e) {
        error_log('files_api upload error: ' . $e->getMessage());
        respondError('خطأ في رفع الملف', 500);
    }
}

function updateFile($pdo, $user_id) {
    try {
        $id = (int)($_POST['id'] ?? 0);
        $name = trim((string)($_POST['name'] ?? ''));
        if ($id <= 0 || $name === '') {
            respondError('البيانات المطلوبة مفقودة', 400);
        }

        $stmt = $pdo->prepare('UPDATE files SET name = ? WHERE id = ? AND user_id = ?');
        $stmt->execute([$name, $id, $user_id]);
        if ($stmt->rowCount() === 0) {
            respondError('غير مصرح لك بتعديل هذا الملف', 403);
        }

        respondJson(['success' => true, 'message' => 'تم التحديث بنجاح']);
    } catch (Throwable $e) {
        error_log('files_api update error: ' . $e->getMessage());
        respondError('خطأ في التحديث', 500);
    }
}

function deleteFile($pdo, $user_id) {
    try {
        $id = (int)($_POST['id'] ?? 0);
        if ($id <= 0) {
            respondError('معرّف الملف مفقود', 400);
        }

        $stmt = $pdo->prepare('SELECT file_path FROM files WHERE id = ? AND user_id = ?');
        $stmt->execute([$id, $user_id]);
        $file = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$file) {
            respondError('غير مصرح لك بحذف هذا الملف', 403);
        }

        $stmt = $pdo->prepare('DELETE FROM files WHERE id = ? AND user_id = ?');
        $stmt->execute([$id, $user_id]);

        $uploadRoot = realpath(dirname(__DIR__) . DIRECTORY_SEPARATOR . 'uploads' . DIRECTORY_SEPARATOR . safeBaseName($user_id));
        $filePath = realpath(dirname(__DIR__) . DIRECTORY_SEPARATOR . $file['file_path']);
        if ($uploadRoot && $filePath && strpos($filePath, $uploadRoot . DIRECTORY_SEPARATOR) === 0 && is_file($filePath)) {
            @unlink($filePath);
        }

        respondJson(['success' => true, 'message' => 'تم الحذف بنجاح']);
    } catch (Throwable $e) {
        error_log('files_api delete error: ' . $e->getMessage());
        respondError('خطأ في الحذف', 500);
    }
}

function getStorageInfo($pdo, $user_id) {
    try {
        $stmt = $pdo->prepare('SELECT COALESCE(SUM(file_size), 0) as total_size, COUNT(*) as file_count FROM files WHERE user_id = ?');
        $stmt->execute([$user_id]);
        $result = $stmt->fetch(PDO::FETCH_ASSOC);
        $usedStorage = (int)($result['total_size'] ?? 0);
        $fileCount = (int)($result['file_count'] ?? 0);
        $maxStorage = (int)configValue('USER_FILE_STORAGE_BYTES', (string)(100 * 1024 * 1024));

        respondJson([
            'success' => true,
            'used_storage' => $usedStorage,
            'remaining_storage' => max(0, $maxStorage - $usedStorage),
            'max_storage' => $maxStorage,
            'file_count' => $fileCount,
            'used_mb' => round($usedStorage / (1024 * 1024), 2),
            'remaining_mb' => round(max(0, $maxStorage - $usedStorage) / (1024 * 1024), 2),
        ]);
    } catch (Throwable $e) {
        error_log('files_api storage error: ' . $e->getMessage());
        respondError('خطأ في جلب معلومات التخزين', 500);
    }
}
