<?php
require_once __DIR__ . '/../config/database.php';

applyCorsHeaders('GET, OPTIONS');
requireAdminUser();

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'GET') {
    respondError('Method not allowed', 405);
}

$status = trim((string)($_GET['status'] ?? 'all'));
$search = trim((string)($_GET['search'] ?? ''));
$validStatuses = ['all', 'pending', 'verified', 'preparing', 'shipped', 'delivered', 'rejected'];
if (!in_array($status, $validStatuses, true)) {
    respondError('حالة غير صالحة', 400);
}

try {
    $db = getDB();
    $where = [];
    $params = [];

    if ($status !== 'all') {
        $where[] = 'status = :status';
        $params[':status'] = $status;
    }

    if ($search !== '') {
        $where[] = '(order_number LIKE :search OR customer_name LIKE :search OR customer_email LIKE :search OR tracking_code LIKE :search)';
        $params[':search'] = '%' . $search . '%';
    }

    $sql = 'SELECT id, order_number, customer_name, customer_email, customer_phone, customer_address,
                   total_amount, status, rejection_reason, tracking_code, notes, created_at, updated_at
            FROM orders';
    if ($where) {
        $sql .= ' WHERE ' . implode(' AND ', $where);
    }
    $sql .= ' ORDER BY created_at DESC LIMIT 500';

    $stmt = $db->prepare($sql);
    $stmt->execute($params);
    $orders = $stmt->fetchAll(PDO::FETCH_ASSOC);

    $stats = [
        'total' => 0,
        'pending' => 0,
        'verified' => 0,
        'preparing' => 0,
        'shipped' => 0,
        'delivered' => 0,
        'rejected' => 0,
    ];

    $stmt = $db->query('SELECT status, COUNT(*) AS count FROM orders GROUP BY status');
    foreach ($stmt->fetchAll(PDO::FETCH_ASSOC) as $row) {
        $count = (int)$row['count'];
        $stats[$row['status']] = $count;
        $stats['total'] += $count;
    }

    $financial = $db->query("
        SELECT
            COALESCE(SUM(total_amount), 0) AS total_revenue,
            COALESCE(SUM(CASE WHEN status = 'delivered' THEN total_amount ELSE 0 END), 0) AS delivered_revenue,
            COALESCE(SUM(CASE WHEN status = 'pending' THEN total_amount ELSE 0 END), 0) AS pending_revenue,
            COALESCE(SUM(CASE WHEN status IN ('verified', 'preparing', 'shipped') THEN total_amount ELSE 0 END), 0) AS in_progress_revenue
        FROM orders
    ")->fetch(PDO::FETCH_ASSOC) ?: [];

    respondJson([
        'success' => true,
        'orders' => $orders,
        'status_names' => [
            'pending' => 'بانتظار المراجعة',
            'verified' => 'تم التحقق',
            'preparing' => 'قيد التحضير',
            'shipped' => 'تم الشحن',
            'delivered' => 'تم التسليم',
            'rejected' => 'مرفوض',
        ],
        'stats' => $stats,
        'financial_stats' => $financial,
        'count' => count($orders),
    ]);
} catch (Throwable $e) {
    error_log('get_orders error: ' . $e->getMessage());
    respondError('فشل في جلب الطلبات', 500);
}
