<?php
require_once __DIR__ . '/../config/database.php';

$user_id = requireAuthenticatedUser();
enforceRateLimit('contacts_api:' . $user_id, 120, 60);
$pdo = getDB();
$action = $_POST['action'] ?? ($_GET['action'] ?? '');

switch ($action) {
    case 'get_all':
        getAllContacts($pdo, $user_id);
        break;
    case 'add':
        verifyCsrfToken();
        addContact($pdo, $user_id);
        break;
    case 'update':
        verifyCsrfToken();
        updateContact($pdo, $user_id);
        break;
    case 'delete':
        verifyCsrfToken();
        deleteContact($pdo, $user_id);
        break;
    case 'get_sending_status':
        getSendingStatus($pdo, $user_id);
        break;
    case 'update_sending_progress':
        verifyCsrfToken();
        updateSendingProgress($pdo, $user_id);
        break;
    case 'reset_sending':
        verifyCsrfToken();
        resetSending($pdo, $user_id);
        break;
    default:
        respondError('Invalid action', 400);
}

function normalizeContactData($raw) {
    $decoded = is_string($raw) ? json_decode($raw, true) : $raw;
    if (!is_array($decoded)) {
        respondError('بيانات جهات الاتصال غير صالحة', 400);
    }
    if (count($decoded) > 5000) {
        respondError('عدد جهات الاتصال كبير جداً', 413);
    }
    return json_encode($decoded, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
}

function getAllContacts($pdo, $user_id) {
    try {
        $stmt = $pdo->prepare('SELECT * FROM contacts WHERE user_id = ? ORDER BY created_at DESC LIMIT 500');
        $stmt->execute([$user_id]);
        $contacts = $stmt->fetchAll(PDO::FETCH_ASSOC);
        foreach ($contacts as &$contact) {
            $contact['data'] = json_decode($contact['data'], true);
        }
        respondJson(['success' => true, 'contacts' => $contacts]);
    } catch (Throwable $e) {
        error_log('contacts get_all error: ' . $e->getMessage());
        respondError('خطأ في جلب البيانات', 500);
    }
}

function addContact($pdo, $user_id) {
    try {
        $name = cleanText($_POST['name'] ?? '', 120);
        $platform = cleanText($_POST['platform'] ?? '', 30);
        $type = cleanText($_POST['type'] ?? '', 30);
        $count = max(0, min((int)($_POST['count'] ?? 0), 100000));
        $data = normalizeContactData($_POST['data'] ?? '[]');

        if ($name === '' || $platform === '' || $type === '') {
            respondError('البيانات المطلوبة مفقودة', 400);
        }

        $allowedPlatforms = ['facebook', 'instagram', 'whatsapp', 'telegram', 'other'];
        if (!in_array(strtolower($platform), $allowedPlatforms, true)) {
            respondError('المنصة غير صالحة', 400);
        }

        $stmt = $pdo->prepare('INSERT INTO contacts (user_id, name, platform, type, count, data) VALUES (?, ?, ?, ?, ?, ?)');
        $stmt->execute([$user_id, $name, strtolower($platform), $type, $count, $data]);
        respondJson(['success' => true, 'message' => 'تم إضافة جهة الاتصال بنجاح', 'contact_id' => $pdo->lastInsertId()]);
    } catch (Throwable $e) {
        error_log('contacts add error: ' . $e->getMessage());
        respondError('خطأ في الإضافة', 500);
    }
}

function updateContact($pdo, $user_id) {
    try {
        $id = (int)($_POST['id'] ?? 0);
        $name = cleanText($_POST['name'] ?? '', 120);
        $platform = cleanText($_POST['platform'] ?? '', 30);
        $count = max(0, min((int)($_POST['count'] ?? 0), 100000));
        $data = normalizeContactData($_POST['data'] ?? '[]');

        if ($id <= 0 || $name === '' || $platform === '') {
            respondError('البيانات المطلوبة مفقودة', 400);
        }

        $stmt = $pdo->prepare('UPDATE contacts SET name = ?, platform = ?, count = ?, data = ? WHERE id = ? AND user_id = ?');
        $stmt->execute([$name, strtolower($platform), $count, $data, $id, $user_id]);
        if ($stmt->rowCount() === 0) {
            respondError('غير مصرح لك بتعديل هذه البيانات', 403);
        }
        respondJson(['success' => true, 'message' => 'تم التحديث بنجاح']);
    } catch (Throwable $e) {
        error_log('contacts update error: ' . $e->getMessage());
        respondError('خطأ في التحديث', 500);
    }
}

function deleteContact($pdo, $user_id) {
    try {
        $id = (int)($_POST['id'] ?? 0);
        if ($id <= 0) {
            respondError('معرّف جهة الاتصال مفقود', 400);
        }

        $stmt = $pdo->prepare('DELETE FROM contacts WHERE id = ? AND user_id = ?');
        $stmt->execute([$id, $user_id]);
        if ($stmt->rowCount() === 0) {
            respondError('غير مصرح لك بحذف هذه البيانات', 403);
        }
        respondJson(['success' => true, 'message' => 'تم الحذف بنجاح']);
    } catch (Throwable $e) {
        error_log('contacts delete error: ' . $e->getMessage());
        respondError('خطأ في الحذف', 500);
    }
}

function getSendingStatus($pdo, $user_id) {
    try {
        $id = (int)($_GET['id'] ?? 0);
        if ($id <= 0) {
            respondError('معرّف جهة الاتصال مفقود', 400);
        }
        $stmt = $pdo->prepare('SELECT id, name, count, last_sent_index, sending_status, sent_count, data FROM contacts WHERE id = ? AND user_id = ?');
        $stmt->execute([$id, $user_id]);
        $contact = $stmt->fetch(PDO::FETCH_ASSOC);
        if (!$contact) {
            respondError('جهة الاتصال غير موجودة', 404);
        }
        $contact['data'] = json_decode($contact['data'], true);
        respondJson(['success' => true, 'contact' => $contact]);
    } catch (Throwable $e) {
        error_log('contacts status error: ' . $e->getMessage());
        respondError('خطأ في جلب البيانات', 500);
    }
}

function updateSendingProgress($pdo, $user_id) {
    try {
        $id = (int)($_POST['id'] ?? 0);
        $lastSent = max(0, (int)($_POST['last_sent_index'] ?? 0));
        $status = cleanText($_POST['sending_status'] ?? 'idle', 20);
        $sentCount = max(0, (int)($_POST['sent_count'] ?? 0));
        if ($id <= 0 || !in_array($status, ['idle', 'sending', 'paused', 'completed'], true)) {
            respondError('بيانات حالة الإرسال غير صالحة', 400);
        }
        $stmt = $pdo->prepare('UPDATE contacts SET last_sent_index = ?, sending_status = ?, sent_count = ? WHERE id = ? AND user_id = ?');
        $stmt->execute([$lastSent, $status, $sentCount, $id, $user_id]);
        if ($stmt->rowCount() === 0) {
            respondError('غير مصرح لك بتحديث هذه البيانات', 403);
        }
        respondJson(['success' => true, 'message' => 'تم تحديث حالة الإرسال']);
    } catch (Throwable $e) {
        error_log('contacts progress error: ' . $e->getMessage());
        respondError('خطأ في التحديث', 500);
    }
}

function resetSending($pdo, $user_id) {
    try {
        $id = (int)($_POST['id'] ?? 0);
        if ($id <= 0) {
            respondError('معرّف جهة الاتصال مفقود', 400);
        }
        $stmt = $pdo->prepare("UPDATE contacts SET last_sent_index = 0, sending_status = 'idle', sent_count = 0 WHERE id = ? AND user_id = ?");
        $stmt->execute([$id, $user_id]);
        if ($stmt->rowCount() === 0) {
            respondError('غير مصرح لك بتحديث هذه البيانات', 403);
        }
        respondJson(['success' => true, 'message' => 'تم إعادة تعيين حالة الإرسال']);
    } catch (Throwable $e) {
        error_log('contacts reset error: ' . $e->getMessage());
        respondError('خطأ في إعادة التعيين', 500);
    }
}
