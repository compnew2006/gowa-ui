<?php
require_once __DIR__ . '/../config/database.php';

applyCorsHeaders('POST, OPTIONS');
requireAdminUser();

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respondError('Method not allowed', 405);
}

$data = readJsonBody();
$orderId = (int)($data['order_id'] ?? 0);
$newStatus = trim((string)($data['new_status'] ?? ''));
$notes = trim((string)($data['notes'] ?? ''));
$rejectionReason = trim((string)($data['rejection_reason'] ?? ''));
$createdBy = trim((string)($data['created_by'] ?? 'المدير'));

if ($orderId <= 0 || $newStatus === '') {
    respondError('بيانات غير مكتملة', 400);
}

$validStatuses = ['pending', 'verified', 'preparing', 'shipped', 'delivered', 'rejected'];
if (!in_array($newStatus, $validStatuses, true)) {
    respondError('حالة غير صالحة', 400);
}

try {
    $db = getDB();
    $stmt = $db->prepare('SELECT id, status, order_number FROM orders WHERE id = ? LIMIT 1');
    $stmt->execute([$orderId]);
    $order = $stmt->fetch(PDO::FETCH_ASSOC);
    if (!$order) {
        respondError('الطلب غير موجود', 404);
    }

    $db->beginTransaction();

    $historyNotes = ($newStatus === 'rejected' && $rejectionReason !== '') ? $rejectionReason : $notes;
    $updateSql = 'UPDATE orders SET status = :status, updated_at = NOW()';
    $params = [':status' => $newStatus, ':id' => $orderId];

    if ($newStatus === 'rejected' && $rejectionReason !== '') {
        $updateSql .= ', rejection_reason = :rejection_reason';
        $params[':rejection_reason'] = $rejectionReason;
    }

    $updateSql .= ' WHERE id = :id';
    $stmt = $db->prepare($updateSql);
    $stmt->execute($params);

    $stmt = $db->prepare('INSERT INTO order_status_history (order_id, status, notes, created_by) VALUES (?, ?, ?, ?)');
    $stmt->execute([$orderId, $newStatus, $historyNotes, $createdBy !== '' ? $createdBy : 'المدير']);

    $db->commit();

    $statusNames = [
        'pending' => 'بانتظار المراجعة',
        'verified' => 'تم التحقق',
        'preparing' => 'قيد التحضير',
        'shipped' => 'تم الشحن',
        'delivered' => 'تم التسليم',
        'rejected' => 'مرفوض',
    ];

    respondJson([
        'success' => true,
        'message' => 'تم تحديث حالة الطلب بنجاح',
        'order' => [
            'id' => $orderId,
            'order_number' => $order['order_number'],
            'old_status' => $order['status'],
            'old_status_name' => $statusNames[$order['status']] ?? $order['status'],
            'new_status' => $newStatus,
            'new_status_name' => $statusNames[$newStatus],
        ],
    ]);
} catch (Throwable $e) {
    if (isset($db) && $db instanceof PDO && $db->inTransaction()) {
        $db->rollBack();
    }
    error_log('update_order_status error: ' . $e->getMessage());
    respondError('فشل في تحديث الطلب', 500);
}
