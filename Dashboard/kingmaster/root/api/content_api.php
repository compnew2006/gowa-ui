<?php
require_once __DIR__ . '/../config/database.php';

$userId = requireAuthenticatedUser();
enforceRateLimit('content_api:' . $userId, 120, 60);

$pdo = getDB();
$action = $_POST['action'] ?? $_GET['action'] ?? '';

try {
    switch ($action) {
        case 'get_all':
            getAllContent($pdo, $userId);
            break;
        case 'create':
            verifyCsrfToken();
            createContent($pdo, $userId);
            break;
        case 'update':
            verifyCsrfToken();
            updateContent($pdo, $userId);
            break;
        case 'delete':
            verifyCsrfToken();
            deleteContent($pdo, $userId);
            break;
        default:
            respondError('إجراء غير صالح', 400);
    }
} catch (Throwable $e) {
    error_log('content_api error: ' . $e->getMessage());
    respondError('تعذر تنفيذ الطلب', 500);
}

function normalizeContentInput() {
    $name = cleanText($_POST['name'] ?? '', 120);
    $text = trim((string)($_POST['text'] ?? ''));
    $text = preg_replace('/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/u', '', $text);

    if ($name === '') {
        respondError('اسم المحتوى مطلوب', 422);
    }
    if ($text === '') {
        respondError('نص المحتوى مطلوب', 422);
    }
    if (mb_strlen($text, 'UTF-8') > 50000) {
        respondError('نص المحتوى طويل جداً', 413);
    }

    $wordCount = trim($text) === '' ? 0 : count(preg_split('/\s+/u', trim($text)));
    return [
        'name' => $name,
        'text' => $text,
        'char_count' => mb_strlen($text, 'UTF-8'),
        'word_count' => $wordCount,
    ];
}

function contentIdFromRequest() {
    $id = filter_var($_POST['id'] ?? 0, FILTER_VALIDATE_INT, ['options' => ['min_range' => 1]]);
    if (!$id) {
        respondError('معرّف المحتوى غير صالح', 422);
    }
    return (int)$id;
}

function getAllContent(PDO $pdo, $userId) {
    $stmt = $pdo->prepare(
        'SELECT id, name, text, char_count, word_count, created_at, updated_at
         FROM content
         WHERE user_id = ?
         ORDER BY created_at DESC
         LIMIT 500'
    );
    $stmt->execute([$userId]);
    respondJson(['success' => true, 'content' => $stmt->fetchAll(PDO::FETCH_ASSOC)]);
}

function createContent(PDO $pdo, $userId) {
    $input = normalizeContentInput();
    $stmt = $pdo->prepare(
        'INSERT INTO content (user_id, name, text, char_count, word_count)
         VALUES (?, ?, ?, ?, ?)'
    );
    $stmt->execute([$userId, $input['name'], $input['text'], $input['char_count'], $input['word_count']]);

    respondJson([
        'success' => true,
        'message' => 'تم إنشاء المحتوى بنجاح',
        'content_id' => $pdo->lastInsertId(),
    ]);
}

function updateContent(PDO $pdo, $userId) {
    $id = contentIdFromRequest();
    $input = normalizeContentInput();

    $stmt = $pdo->prepare(
        'UPDATE content
         SET name = ?, text = ?, char_count = ?, word_count = ?
         WHERE id = ? AND user_id = ?'
    );
    $stmt->execute([$input['name'], $input['text'], $input['char_count'], $input['word_count'], $id, $userId]);

    if ($stmt->rowCount() === 0) {
        respondError('المحتوى غير موجود أو غير مصرح', 404);
    }
    respondJson(['success' => true, 'message' => 'تم تحديث المحتوى بنجاح']);
}

function deleteContent(PDO $pdo, $userId) {
    $id = contentIdFromRequest();
    $stmt = $pdo->prepare('DELETE FROM content WHERE id = ? AND user_id = ?');
    $stmt->execute([$id, $userId]);

    if ($stmt->rowCount() === 0) {
        respondError('المحتوى غير موجود أو غير مصرح', 404);
    }
    respondJson(['success' => true, 'message' => 'تم حذف المحتوى بنجاح']);
}
