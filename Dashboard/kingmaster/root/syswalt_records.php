<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}

require_once 'includes/functions.php';

$page_title = "سجل المعاملات المالية | Kingmaster";
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php';

// معالجة الفلاتر
$filter_date = isset($_GET['date']) ? $_GET['date'] : '';
$filter_month = isset($_GET['month']) ? $_GET['month'] : '';
$filter_year = isset($_GET['year']) ? $_GET['year'] : '';
$page = isset($_GET['page']) ? max(1, intval($_GET['page'])) : 1;
$per_page = 50;
$offset = ($page - 1) * $per_page;

// جلب البيانات باستخدام الدالة
$result = getAllSyswalt(null, $filter_date, $filter_year, $filter_month, $offset, $per_page);
$records = $result['records'];
$total_records = $result['total_records'];
$total_amount = $result['total_amount'];
$total_pages = $result['total_pages'];
?>

<style>
    .records-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }

    .records-header {
        margin-bottom: 30px;
    }

    .records-title {
        font-size: 32px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
        margin-bottom: 10px;
    }

    .records-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: iconPulse 2s ease-in-out infinite;
    }

    @keyframes iconPulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }

    .records-subtitle {
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
    }

    .filter-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 25px;
        border: 1px solid var(--border-color);
        margin-bottom: 30px;
    }

    .filter-title {
        font-size: 18px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 20px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .filter-title i {
        color: #667eea;
    }

    .filter-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 15px;
        margin-bottom: 20px;
    }

    .filter-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .filter-label {
        font-size: 13px;
        font-weight: 600;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
    }

    .filter-input {
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: var(--bg-primary) !important;
        color: var(--text-primary) !important;
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .filter-input option {
        background: #1a1a2e !important;
        color: #ffffff !important;
    }

    /* Animated Icons */
    .animated-icon {
        display: inline-block;
        animation: bounceIcon 2s ease-in-out infinite;
    }

    @keyframes bounceIcon {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-5px); }
    }

    .rotating-icon {
        display: inline-block;
        animation: rotateIcon 3s linear infinite;
    }

    @keyframes rotateIcon {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    .pulse-icon {
        display: inline-block;
        animation: pulseIcon 1.5s ease-in-out infinite;
    }

    @keyframes pulseIcon {
        0%, 100% { transform: scale(1); opacity: 1; }
        50% { transform: scale(1.15); opacity: 0.8; }
    }

    .filter-label i {
        margin-left: 5px;
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .type-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
    }

    .type-badge i {
        font-size: 14px;
    }

    .filter-input:focus {
        outline: none;
        border-color: #667eea;
        box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.15);
    }

    .filter-buttons {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
    }

    .filter-btn {
        padding: 12px 24px;
        border: none;
        border-radius: 10px;
        font-weight: 700;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .filter-btn.apply {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
    }

    .filter-btn.apply:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
    }

    .filter-btn.reset {
        background: #95a5a6;
        color: white;
    }

    .filter-btn.reset:hover {
        background: #7f8c8d;
    }

    .stats-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 20px;
        margin-bottom: 30px;
    }

    .stat-card {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 25px;
        border: 1px solid var(--border-color);
        display: flex;
        align-items: center;
        gap: 20px;
        transition: all 0.3s ease;
    }

    .stat-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.2);
    }

    .stat-icon {
        width: 60px;
        height: 60px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 24px;
        color: white;
    }

    .stat-icon.total {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    }

    .stat-icon.amount {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
    }

    .stat-info h3 {
        font-size: 28px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 5px;
        font-family: 'Cairo', sans-serif;
    }

    .stat-info p {
        font-size: 14px;
        color: var(--text-secondary);
        margin: 0;
        font-family: 'Cairo', sans-serif;
    }

    .records-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        border: 1px solid var(--border-color);
        margin-bottom: 30px;
    }

    .table-wrapper {
        overflow-x: auto;
        border-radius: 12px;
        border: 1px solid var(--border-color);
    }

    .records-table {
        width: 100%;
        border-collapse: collapse;
        font-family: 'Cairo', sans-serif;
    }

    .records-table thead {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    }

    .records-table thead th {
        color: white;
        padding: 18px 15px;
        text-align: right;
        font-weight: 700;
        font-size: 14px;
        white-space: nowrap;
    }

    .records-table tbody tr {
        border-bottom: 1px solid var(--border-color);
        transition: all 0.3s ease;
    }

    .records-table tbody tr:hover {
        background: rgba(102, 126, 234, 0.05);
    }

    .records-table tbody tr:last-child {
        border-bottom: none;
    }

    .records-table tbody td {
        padding: 18px 15px;
        color: var(--text-primary);
        font-size: 14px;
        text-align: right;
    }

    .price-cell {
        font-weight: 700;
        font-size: 15px;
        color: #10b981;
    }

    .date-cell {
        color: var(--text-secondary);
        font-size: 13px;
    }

    .empty-state {
        text-align: center;
        padding: 60px 20px;
        color: var(--text-secondary);
    }

    .empty-state i {
        font-size: 64px;
        margin-bottom: 20px;
        color: #667eea;
        opacity: 0.3;
    }

    .empty-state h3 {
        font-size: 20px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
    }

    .pagination {
        display: flex;
        justify-content: center;
        align-items: center;
        gap: 10px;
        margin-top: 30px;
        flex-wrap: wrap;
    }

    .pagination-btn {
        padding: 10px 16px;
        border: 2px solid var(--border-color);
        background: var(--card-bg);
        color: var(--text-primary);
        border-radius: 8px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
        text-decoration: none;
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .pagination-btn:hover {
        border-color: #667eea;
        background: rgba(102, 126, 234, 0.1);
    }

    .pagination-info {
        color: var(--text-secondary);
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
    }

    /* Light Theme */
    body.light-theme .records-card,
    body.light-theme .filter-card {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .stat-card {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .filter-input {
        background: #ffffff !important;
        color: #2d3436 !important;
        border-color: rgba(0, 0, 0, 0.15);
    }

    body.light-theme .filter-input option {
        background: #ffffff !important;
        color: #2d3436 !important;
    }

    body.light-theme .pagination-btn {
        background: #ffffff;
        color: #2d3436;
    }

    body.light-theme .records-table tbody tr:hover {
        background: rgba(102, 126, 234, 0.08);
    }

    /* Responsive */
    @media (max-width: 768px) {
        .records-container {
            padding: 20px;
            margin-top: 100px;
        }

        .records-title {
            font-size: 24px;
        }

        .stats-grid, .filter-grid {
            grid-template-columns: 1fr;
        }

        .table-wrapper {
            overflow-x: scroll;
        }

        .records-table {
            min-width: 600px;
        }
    }
</style>

<div class="records-container">
    <div class="records-header">
        <div class="records-title">
            <i class="fas fa-receipt animated-icon" style="background: linear-gradient(135deg, #36d1dc 0%, #5b86e5 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent;"></i>
            سجل المعاملات المالية
        </div>
        <div class="records-subtitle">عرض جميع المعاملات المالية مع إمكانية الفلترة</div>
    </div>

    <!-- فلتر التواريخ -->
    <div class="filter-card">
        <div class="filter-title">
            <i class="fas fa-filter pulse-icon" style="color:#f39c12;"></i>
            فلتر البيانات
        </div>
        <form method="GET" action="">
            <div class="filter-grid">
                <div class="filter-group">
                    <label class="filter-label"><i class="fas fa-calendar-alt"></i> تاريخ محدد</label>
                    <input type="date" name="date" class="filter-input" value="<?php echo htmlspecialchars($filter_date); ?>">
                </div>
                <div class="filter-group">
                    <label class="filter-label"><i class="fas fa-moon"></i> الشهر</label>
                    <select name="month" class="filter-input">
                        <option value="">اختر الشهر</option>
                        <?php for($m = 1; $m <= 12; $m++): ?>
                            <option value="<?php echo $m; ?>" <?php echo ($filter_month == $m) ? 'selected' : ''; ?>>
                                <?php echo date('F', mktime(0, 0, 0, $m, 1)); ?>
                            </option>
                        <?php endfor; ?>
                    </select>
                </div>
                <div class="filter-group">
                    <label class="filter-label"><i class="fas fa-calendar"></i> السنة</label>
                    <select name="year" class="filter-input">
                        <option value="">اختر السنة</option>
                        <?php 
                        $current_year = date('Y');
                        for($y = $current_year; $y >= $current_year - 5; $y--): 
                        ?>
                            <option value="<?php echo $y; ?>" <?php echo ($filter_year == $y) ? 'selected' : ''; ?>>
                                <?php echo $y; ?>
                            </option>
                        <?php endfor; ?>
                    </select>
                </div>
            </div>
            <div class="filter-buttons">
                <button type="submit" class="filter-btn apply">
                    <i class="fas fa-search rotating-icon"></i>
                    تطبيق الفلتر
                </button>
                <button type="button" class="filter-btn reset" onclick="window.location.href='syswalt_records.php'">
                    <i class="fas fa-redo animated-icon"></i>
                    إعادة تعيين
                </button>
            </div>
        </form>
    </div>

    <!-- الإحصائيات -->
    <div class="stats-grid">
        <div class="stat-card">
            <div class="stat-icon total">
                <i class="fas fa-list pulse-icon"></i>
            </div>
            <div class="stat-info">
                <h3><?php echo number_format($total_records); ?></h3>
                <p>عدد المعاملات</p>
            </div>
        </div>

        <div class="stat-card">
            <div class="stat-icon amount">
                <i class="fas fa-dollar-sign animated-icon"></i>
            </div>
            <div class="stat-info">
                <h3>$<?php echo number_format($total_amount, 2); ?></h3>
                <p>إجمالي المبلغ</p>
            </div>
        </div>
    </div>

    <!-- جدول السجلات -->
    <div class="records-card">
        <?php if (count($records) > 0): ?>
            <div class="table-wrapper">
                <table class="records-table">
                    <thead>
                        <tr>
                            <th><i class="fas fa-hashtag" style="color:#8e44ad"></i> المعرف</th>
                            <th><i class="fas fa-money-bill-wave" style="color:#16a085"></i> المبلغ</th>
                            <th><i class="fas fa-tags" style="color:#e74c3c"></i> النوع</th>
                            <th><i class="fas fa-clock" style="color:#3498db"></i> التاريخ</th>
                        </tr>
                    </thead>
                    <tbody>
                        <?php foreach ($records as $record): ?>
                            <tr>
                                <td>#<?php echo htmlspecialchars($record['id']); ?></td>
                                <td class="price-cell">
                                    <i class="fas fa-coins" style="color:#10b981"></i>
                                    $<?php echo number_format(floatval($record['price']), 2); ?>
                                </td>
                                <td>
                                    <span class="type-badge" style="background: rgba(231, 76, 60, 0.1); color: #e74c3c;">
                                        <i class="fas fa-tag"></i>
                                        <?php echo htmlspecialchars($record['typs']); ?>
                                    </span>
                                </td>
                                <td class="date-cell">
                                    <?php 
                                    $date = new DateTime($record['created_at']);
                                    echo $date->format('Y-m-d H:i:s');
                                    ?>
                                </td>
                            </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            </div>

            <!-- Pagination -->
            <?php if ($total_pages > 1): ?>
                <div class="pagination">
                    <?php if ($page > 1): ?>
                        <a href="?page=<?php echo $page - 1; ?><?php echo !empty($filter_date) ? '&date=' . $filter_date : ''; ?><?php echo !empty($filter_month) ? '&month=' . $filter_month : ''; ?><?php echo !empty($filter_year) ? '&year=' . $filter_year : ''; ?>" class="pagination-btn">
                            <i class="fas fa-chevron-right"></i> السابق
                        </a>
                    <?php endif; ?>

                    <span class="pagination-info">
                        صفحة <?php echo $page; ?> من <?php echo $total_pages; ?>
                    </span>

                    <?php if ($page < $total_pages): ?>
                        <a href="?page=<?php echo $page + 1; ?><?php echo !empty($filter_date) ? '&date=' . $filter_date : ''; ?><?php echo !empty($filter_month) ? '&month=' . $filter_month : ''; ?><?php echo !empty($filter_year) ? '&year=' . $filter_year : ''; ?>" class="pagination-btn">
                            التالي <i class="fas fa-chevron-left"></i>
                        </a>
                    <?php endif; ?>
                </div>
            <?php endif; ?>
        <?php else: ?>
            <div class="empty-state">
                <i class="fas fa-inbox animated-icon" style="color:#9b59b6"></i>
                <h3>لا توجد سجلات</h3>
                <p>لا توجد معاملات في الفترة المحددة</p>
            </div>
        <?php endif; ?>
    </div>
</div>

<?php include 'includes/admin_footer.php'; ?>
