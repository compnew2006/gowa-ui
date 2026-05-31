 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "طلباتي | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>
<style>
    .orders-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }

    .orders-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
        flex-wrap: wrap;
        gap: 20px;
    }

    .orders-header h1 {
        font-size: 32px;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 15px;
    }

    .orders-header h1 i {
        color: var(--primary-color);
        font-size: 36px;
    }

    .back-to-products {
        padding: 12px 25px;
        background: var(--primary-color);
        color: white;
        text-decoration: none;
        border-radius: 10px;
        display: flex;
        align-items: center;
        gap: 10px;
        font-weight: 600;
        transition: all 0.3s ease;
    }

    .back-to-products:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
    }

    .orders-grid {
        display: grid;
        gap: 20px;
    }

    .order-card {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 25px;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
        transition: all 0.3s ease;
        border: 2px solid var(--border-color);
    }

    .order-card:hover {
        transform: translateY(-3px);
        box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        border-color: var(--primary-color);
    }

    .order-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 20px;
        padding-bottom: 15px;
        border-bottom: 2px solid var(--border-color);
    }

    .order-id {
        font-size: 20px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .order-id i {
        color: var(--primary-color);
    }

    .order-status {
        padding: 8px 16px;
        border-radius: 20px;
        font-weight: 700;
        font-size: 13px;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .status-pending {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        color: white;
    }

    .status-approved {
        background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        color: white;
    }

    .status-preparing {
        background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
        color: white;
    }

    .status-shipping {
        background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
        color: #333;
    }

    .status-delivered {
        background: linear-gradient(135deg, #13ce66 0%, #00b894 100%);
        color: white;
    }

    .status-completed {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
    }

    .status-rejected {
        background: linear-gradient(135deg, #ff6b6b 0%, #ee5a6f 100%);
        color: white;
    }

    .status-cancelled {
        background: linear-gradient(135deg, #b8b8b8 0%, #8e8e8e 100%);
        color: white;
    }

    .order-body {
        display: grid;
        grid-template-columns: 2fr 1fr 1fr;
        gap: 20px;
        margin-bottom: 15px;
    }

    .order-product {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .product-name {
        font-size: 18px;
        font-weight: 700;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .product-details {
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
    }

    .detail-badge {
        padding: 5px 12px;
        background: var(--bg-primary);
        border-radius: 10px;
        font-size: 13px;
        color: var(--text-secondary);
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .order-pricing {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .price-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 14px;
    }

    .price-label {
        color: var(--text-secondary);
    }

    .price-value {
        font-weight: 700;
        color: var(--text-primary);
    }

    .total-price {
        font-size: 22px;
        font-weight: 800;
        color: var(--primary-color);
        margin-top: 8px;
        padding-top: 8px;
        border-top: 2px solid var(--border-color);
    }

    .order-info {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .info-item {
        display: flex;
        align-items: flex-start;
        gap: 8px;
        font-size: 13px;
        color: var(--text-secondary);
    }

    .info-item i {
        color: var(--primary-color);
        margin-top: 2px;
    }

    .order-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-top: 15px;
        padding-top: 15px;
        border-top: 2px solid var(--border-color);
    }

    .order-date {
        font-size: 13px;
        color: var(--text-secondary);
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .product-type-badge {
        padding: 5px 12px;
        border-radius: 15px;
        font-size: 12px;
        font-weight: 600;
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }

    .badge-digital {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
    }

    .badge-physical {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        color: white;
    }

    .empty-state {
        text-align: center;
        padding: 80px 20px;
        background: var(--card-bg);
        border-radius: 15px;
    }

    .empty-state i {
        font-size: 120px;
        color: var(--text-secondary);
        margin-bottom: 20px;
        opacity: 0.3;
    }

    .empty-state h3 {
        font-size: 24px;
        color: var(--text-primary);
        margin-bottom: 15px;
    }

    .empty-state p {
        font-size: 16px;
        color: var(--text-secondary);
        margin-bottom: 25px;
    }

    .shop-now-btn {
        padding: 15px 35px;
        background: var(--primary-color);
        color: white;
        text-decoration: none;
        border-radius: 10px;
        display: inline-flex;
        align-items: center;
        gap: 10px;
        font-weight: 600;
        transition: all 0.3s ease;
    }

    .shop-now-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
    }

    .loading {
        text-align: center;
        padding: 60px 20px;
    }

    .spinner {
        border: 4px solid var(--border-color);
        border-top: 4px solid var(--primary-color);
        border-radius: 50%;
        width: 50px;
        height: 50px;
        animation: spin 1s linear infinite;
        margin: 0 auto 20px;
    }

    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }

    /* Light Theme Fixes */
    body.light-theme .order-card {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
    }

    body.light-theme .order-card:hover {
        box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
    }

    body.light-theme .empty-state {
        background: rgba(255, 255, 255, 0.95);
        border: 1px solid rgba(0, 0, 0, 0.1);
    }

    body.light-theme .order-id,
    body.light-theme .product-name,
    body.light-theme .price-value,
    body.light-theme .empty-state h3 {
        color: #2d3436;
    }

    body.light-theme .price-label,
    body.light-theme .order-date,
    body.light-theme .info-item,
    body.light-theme .empty-state p {
        color: #636e72;
    }

    body.light-theme .detail-badge {
        background: #f5f6fa;
        color: #636e72;
    }

    /* أيقونات ملونة ومتحركة */
    .orders-header h1 i {
        animation: bounce 2s ease-in-out infinite;
    }

    @keyframes bounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-10px); }
    }

    .back-to-products i {
        color: white;
        animation: bounce-small 1.5s ease-in-out infinite;
    }

    @keyframes bounce-small {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-5px); }
    }

    .order-id i {
        animation: pulse 2s ease-in-out infinite;
    }

    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }

    .order-status i {
        animation: heartbeat 1.5s ease-in-out infinite;
    }

    @keyframes heartbeat {
        0%, 100% { transform: scale(1); }
        10%, 30% { transform: scale(1.1); }
        20%, 40% { transform: scale(0.9); }
    }

    .product-name i {
        margin-left: 5px;
    }

    .product-name i.fa-download {
        color: #8b5cf6;
        animation: float 2s ease-in-out infinite;
    }

    .product-name i.fa-box {
        color: #f59e0b;
        animation: shake 2s ease-in-out infinite;
    }

    @keyframes float {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-5px); }
    }

    @keyframes shake {
        0%, 100% { transform: translateX(0); }
        10%, 30%, 50%, 70%, 90% { transform: translateX(-3px); }
        20%, 40%, 60%, 80% { transform: translateX(3px); }
    }

    .detail-badge i {
        margin-left: 3px;
    }

    .detail-badge i.fa-cubes {
        color: #3b82f6;
    }

    .detail-badge i.fa-palette {
        color: #f59e0b;
        animation: colorChange 3s ease-in-out infinite;
    }

    @keyframes colorChange {
        0% { color: #f59e0b; }
        33% { color: #ef4444; }
        66% { color: #8b5cf6; }
        100% { color: #f59e0b; }
    }

    .detail-badge i.fa-ruler {
        color: #06b6d4;
    }

    .info-item i {
        animation: tada 2s ease-in-out infinite;
    }

    @keyframes tada {
        0%, 100% { transform: scale(1) rotate(0deg); }
        10%, 20% { transform: scale(0.9) rotate(-3deg); }
        30%, 50%, 70%, 90% { transform: scale(1.1) rotate(3deg); }
        40%, 60%, 80% { transform: scale(1.1) rotate(-3deg); }
    }

    .info-item i.fa-map-marker-alt {
        color: #ef4444;
    }

    .info-item i.fa-phone {
        color: #10b981;
    }

    .order-date i {
        color: #8b5cf6;
        animation: swing 2s ease-in-out infinite;
    }

    @keyframes swing {
        0%, 100% { transform: rotate(0deg); }
        25% { transform: rotate(-10deg); }
        75% { transform: rotate(10deg); }
    }

    .product-type-badge i {
        animation: heartbeat 1.5s ease-in-out infinite;
    }

    .shop-now-btn i {
        animation: bounce-small 1.5s ease-in-out infinite;
    }

    .empty-state i {
        animation: float-large 3s ease-in-out infinite;
    }

    @keyframes float-large {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-20px); }
    }

    @media (max-width: 968px) {
        .order-body {
            grid-template-columns: 1fr;
        }

        .orders-container {
            padding: 20px 15px;
        }

        .orders-header h1 {
            font-size: 26px;
        }
    }
</style>

<div class="orders-container">
 
    <div id="ordersGrid" class="orders-grid">
        <div class="loading">
            <div class="spinner"></div>
            <p>جاري تحميل الطلبات...</p>
        </div>
    </div>
</div>

<script>
    // تحميل الطلبات عند فتح الصفحة
    document.addEventListener('DOMContentLoaded', loadOrders);

    async function loadOrders() {
        try {
            const response = await fetch('api/orders.php');
            const data = await response.json();

            if (data.success) {
                renderOrders(data.orders);
            } else {
                showError(data.message);
            }
        } catch (error) {
            console.error('خطأ في جلب الطلبات:', error);
            showError('حدث خطأ في جلب الطلبات');
        }
    }

    function renderOrders(orders) {
        const grid = document.getElementById('ordersGrid');

        if (orders.length === 0) {
            grid.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-shopping-cart"></i>
                    <h3>لا توجد طلبات</h3>
                    <p>لم تقم بإجراء أي طلبات بعد</p>
                    <a href="products.php" class="shop-now-btn">
                        <i class="fas fa-shopping-bag"></i>
                        تسوق الآن
                    </a>
                </div>
            `;
            return;
        }

        grid.innerHTML = orders.map(order => createOrderCard(order)).join('');
    }

    function createOrderCard(order) {
        const statusInfo = getStatusInfo(order.status);
        const isDigital = parseInt(order.is_digital) === 1;

        return `
            <div class="order-card">
                <div class="order-header">
                    <div class="order-id">
                        <i class="fas fa-hashtag"></i>
                        طلب رقم ${order.id}
                    </div>
                    <div class="order-status ${statusInfo.class}">
                        <i class="${statusInfo.icon}"></i>
                        ${statusInfo.label}
                    </div>
                </div>

                <div class="order-body">
                    <div class="order-product">
                        <div class="product-name">
                            <i class="fas ${isDigital ? 'fa-download' : 'fa-box'}"></i>
                            ${order.product_name}
                        </div>
                        <div class="product-details">
                            <span class="product-type-badge ${isDigital ? 'badge-digital' : 'badge-physical'}">
                                <i class="fas ${isDigital ? 'fa-cloud-download-alt' : 'fa-shipping-fast'}"></i>
                                ${isDigital ? 'منتج رقمي' : 'منتج حقيقي'}
                            </span>
                            <span class="detail-badge">
                                <i class="fas fa-cubes"></i>
                                الكمية: ${order.quantity}
                            </span>
                            ${order.color ? `
                                <span class="detail-badge">
                                    <i class="fas fa-palette"></i>
                                    ${order.color}
                                </span>
                            ` : ''}
                            ${order.size ? `
                                <span class="detail-badge">
                                    <i class="fas fa-ruler"></i>
                                    ${order.size}
                                </span>
                            ` : ''}
                        </div>
                    </div>

                    <div class="order-pricing">
                        <div class="price-row">
                            <span class="price-label">السعر:</span>
                            <span class="price-value">${formatPrice(order.price)}</span>
                        </div>
                        <div class="price-row">
                            <span class="price-label">الكمية:</span>
                            <span class="price-value">${order.quantity}</span>
                        </div>
                        ${order.commission > 0 ? `
                            <div class="price-row">
                                <span class="price-label" style="color: #27ae60;">عمولة:</span>
                                <span class="price-value" style="color: #27ae60;">${formatPrice(order.commission)}</span>
                            </div>
                        ` : ''}
                        <div class="total-price">
                            الإجمالي: ${formatPrice(order.total)}
                        </div>
                    </div>

                    <div class="order-info">
                        ${!isDigital && order.address ? `
                            <div class="info-item">
                                <i class="fas fa-map-marker-alt"></i>
                                <span>${order.address}</span>
                            </div>
                        ` : ''}
                        ${!isDigital && order.phone ? `
                            <div class="info-item">
                                <i class="fas fa-phone"></i>
                                <span>${order.phone}</span>
                            </div>
                        ` : ''}
                    </div>
                </div>

                <div class="order-footer">
                    <div class="order-date">
                        <i class="fas fa-calendar-alt"></i>
                        ${formatDate(order.created_at)}
                    </div>
                </div>
            </div>
        `;
    }

    function getStatusInfo(status) {
        const statuses = {
            'pending': { label: 'قيد الانتظار', icon: 'fas fa-clock', class: 'status-pending' },
            'approved': { label: 'تم القبول', icon: 'fas fa-check', class: 'status-approved' },
            'preparing': { label: 'قيد التحضير', icon: 'fas fa-box-open', class: 'status-preparing' },
            'shipping': { label: 'قيد الشحن', icon: 'fas fa-shipping-fast', class: 'status-shipping' },
            'delivered': { label: 'تم التوصيل', icon: 'fas fa-check-double', class: 'status-delivered' },
            'completed': { label: 'مكتمل', icon: 'fas fa-check-circle', class: 'status-completed' },
            'rejected': { label: 'مرفوض', icon: 'fas fa-times-circle', class: 'status-rejected' },
            'cancelled': { label: 'ملغي', icon: 'fas fa-ban', class: 'status-cancelled' }
        };
        return statuses[status] || statuses['pending'];
    }

    function formatPrice(price) {
        return parseFloat(price).toLocaleString('ar-EG', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        }) + ' ر.س';
    }

    function formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString('ar-EG', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    function showError(message) {
        const grid = document.getElementById('ordersGrid');
        grid.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-exclamation-triangle" style="color: #ff6b6b;"></i>
                <h3>حدث خطأ</h3>
                <p>${message}</p>
            </div>
        `;
    }
</script>

<?php include 'includes/footer.php'; ?>
