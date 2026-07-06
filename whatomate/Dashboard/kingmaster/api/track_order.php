<?php
require_once __DIR__ . '/../config/database.php';

applyCorsHeaders('POST, OPTIONS');

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respondError('Method not allowed', 405);
}

$data = readJsonBody();
$search = trim((string)($data['search'] ?? ''));
if ($search === '' || mb_strlen($search) > 120) {
    respondError('يجب إدخال رقم الطلب أو كود التتبع', 400);
}

try {
    $db = getDB();
    $stmt = $db->prepare("
        SELECT id, order_number, customer_name, customer_email, customer_phone, customer_address,
               total_amount, status, rejection_reason, tracking_code, notes, created_at, updated_at
        FROM orders
        WHERE order_number = ? OR tracking_code = ?
        LIMIT 1
    ");
    $stmt->execute([$search, $search]);
    $order = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!$order) {
        respondError('لم يتم العثور على الطلب', 404);
    }

    $historyStmt = $db->prepare('SELECT status, notes, created_by, created_at FROM order_status_history WHERE order_id = ? ORDER BY created_at ASC');
    $historyStmt->execute([(int)$order['id']]);
    $history = $historyStmt->fetchAll(PDO::FETCH_ASSOC);

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
        'message' => 'تم العثور على الطلب',
        'order' => [
            'id' => $order['id'],
            'order_number' => $order['order_number'],
            'customer_name' => $order['customer_name'],
            'customer_email' => $order['customer_email'],
            'customer_phone' => $order['customer_phone'],
            'customer_address' => $order['customer_address'],
            'total_amount' => $order['total_amount'],
            'status' => $order['status'],
            'status_name' => $statusNames[$order['status']] ?? $order['status'],
            'rejection_reason' => $order['rejection_reason'],
            'tracking_code' => $order['tracking_code'],
            'notes' => $order['notes'],
            'created_at' => $order['created_at'],
            'updated_at' => $order['updated_at'],
        ],
        'history' => $history,
        'status_names' => $statusNames,
    ]);
} catch (Throwable $e) {
    error_log('track_order error: ' . $e->getMessage());
    respondError('فشل في تتبع الطلب', 500);
}
