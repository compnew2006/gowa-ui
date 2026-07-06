 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "طلباتي | Kingmaster";
$page_css = ['https://kingmaster.info/css/product.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<div class="orders-container">
    <div class="orders-header">
        <h1>
            <i class="fas fa-shopping-cart"></i>
            طلباتي
        </h1>
        <a href="products.php" class="back-to-products">
            <i class="fas fa-shopping-bag"></i>
            تصفح المنتجات
        </a>
    </div>

    <!-- شريط البحث والفلاتر -->
    <div class="search-filters-section">
        <div class="search-row">
            <div class="search-input-wrapper">
                <i class="fas fa-search"></i>
                <input type="text" id="searchInput" placeholder="ابحث برقم الطلب أو رقم الهاتف...">
            </div>
            <button class="search-btn" onclick="loadOrders()">
                <i class="fas fa-search"></i>
                بحث
            </button>
        </div>

        <div class="filters-row">
            <div class="filter-group">
                <label><i class="fas fa-filter" style="color: #8b5cf6;"></i> حالة الطلب</label>
                <select id="statusFilter" onchange="loadOrders()">
                    <option value="">جميع الحالات</option>
                    <option value="pending">قيد الانتظار</option>
                    <option value="approved">تم القبول</option>
                    <option value="preparing">قيد التحضير</option>
                    <option value="shipping">قيد الشحن</option>
                    <option value="delivered">تم التوصيل</option>
                    <option value="completed">مكتمل</option>
                    <option value="rejected">مرفوض</option>
                    <option value="cancelled">ملغي</option>
                </select>
            </div>

            <div class="filter-group">
                <label><i class="fas fa-calendar-day" style="color: #10b981;"></i> التاريخ من</label>
                <input type="date" id="dateFrom" onchange="loadOrders()">
            </div>

            <div class="filter-group">
                <label><i class="fas fa-calendar-day" style="color: #ef4444;"></i> التاريخ إلى</label>
                <input type="date" id="dateTo" onchange="loadOrders()">
            </div>
        </div>
    </div>

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
        const searchQuery = document.getElementById('searchInput')?.value || '';
        const statusFilter = document.getElementById('statusFilter')?.value || '';
        const dateFrom = document.getElementById('dateFrom')?.value || '';
        const dateTo = document.getElementById('dateTo')?.value || '';

        const params = new URLSearchParams();
        if (searchQuery) params.append('search', searchQuery);
        if (statusFilter) params.append('status', statusFilter);
        if (dateFrom) params.append('date_from', dateFrom);
        if (dateTo) params.append('date_to', dateTo);

        try {
            const response = await fetch(`api/orders.php?${params}`);
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
                        ${order.payment_method ? `
                            <div class="info-item">
                                <i class="fas ${getPaymentIcon(order.payment_method)}"></i>
                                <span style="font-weight: 600;">${getPaymentName(order.payment_method)}</span>
                            </div>
                        ` : ''}
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

    function getPaymentName(method) {
        const methods = {
            'balance': 'الدفع من الرصيد',
            'cash': 'الدفع عند الاستلام',
            'electronic': 'الدفع الإلكتروني'
        };
        return methods[method] || method;
    }

    function getPaymentIcon(method) {
        const icons = {
            'balance': 'fa-wallet',
            'cash': 'fa-money-bill-wave',
            'electronic': 'fa-credit-card'
        };
        return icons[method] || 'fa-dollar-sign';
    }

    function formatPrice(price) {
        return '$' + parseFloat(price).toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        });
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
