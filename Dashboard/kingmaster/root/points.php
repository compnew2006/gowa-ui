 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "شحن النقاط | Kingmaster";
$page_css = ['/css/toppages.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>


<div class="points-container">
    <div class="points-header">
        <div class="points-title">
            <i class="fas fa-gem"></i>
            شحن النقاط
        </div>
        <p class="points-subtitle">اختر الباقة المناسبة لك وابدأ في استخدام خدماتنا</p>
    </div>

    <div id="packagesGrid" class="packages-grid">
        <!-- سيتم تحميل الباقات هنا -->
    </div>
</div>

<!-- Payment Modal -->
<div id="paymentModal" class="modal">
    <div class="modal-content modal-point">
        <span class="close-modal" onclick="closePaymentModal()">&times;</span>
        <div class="modal-headerp">
            <div class="modal-title">
                <i class="fas fa-credit-card"></i>
                اختر طريقة الدفع
            </div>
            <p class="modal-subtitle">الباقة المختارة: <strong id="selectedPackageInfo"></strong></p>
        </div>

        <div class="payment-methods">
            <div class="payment-method" onclick="selectPaymentMethod('balance')">
                <div class="payment-icon">
                    <i class="fas fa-wallet"></i>
                </div>
                <div class="payment-info">
                    <div class="payment-title">الدفع من الرصيد</div>
                    <div class="payment-desc">استخدم رصيدك الحالي لشراء النقاط</div>
                </div>
                <i class="fas fa-chevron-left" style="color: var(--text-secondary);"></i>
            </div>

            <div class="payment-method" onclick="selectPaymentMethod('online')">
                <div class="payment-icon">
                    <i class="fas fa-credit-card"></i>
                </div>
                <div class="payment-info">
                    <div class="payment-title">الدفع الإلكتروني</div>
                    <div class="payment-desc">الدفع عبر البطاقة الائتمانية أو المحفظة الإلكترونية</div>
                </div>
                <i class="fas fa-chevron-left" style="color: var(--text-secondary);"></i>
            </div>

            <div class="payment-method" onclick="selectPaymentMethod('agent')">
                <div class="payment-icon">
                    <i class="fas fa-user-tie"></i>
                </div>
                <div class="payment-info">
                    <div class="payment-title">الدفع عن طريق المندوب</div>
                    <div class="payment-desc">سيتواصل معك أحد مندوبينا لإتمام الدفع</div>
                </div>
                <i class="fas fa-chevron-left" style="color: var(--text-secondary);"></i>
            </div>
        </div>
    </div>
</div>

<script>
let selectedPackageId = null;
let selectedPackageData = null;

function loadPackages() {
    fetch('api/get_points_packages.php')
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            renderPackages(data.packages);
        }
    })
    .catch(error => {
        console.error('Error loading packages:', error);
    });
}

function renderPackages(packages) {
    const grid = document.getElementById('packagesGrid');
    
    if (packages.length === 0) {
        grid.innerHTML = `
            <div style="grid-column: 1/-1; text-align: center; padding: 60px 20px;">
                <i class="fas fa-box-open" style="font-size: 80px; color: #667eea; margin-bottom: 20px; opacity: 0.5;"></i>
                <h3 style="color: var(--text-primary); font-family: 'Cairo', sans-serif;">لا توجد باقات متاحة</h3>
            </div>
        `;
        return;
    }
    
    grid.innerHTML = packages.map(pkg => `
        <div class="package-card">
            <div class="package-icon">
                <i class="fas fa-gem"></i>
            </div>
            <div class="package-points">
                ${pkg.points_count.toLocaleString()} نقطة
            </div>
            <div class="package-price">
                ${parseFloat(pkg.price).toLocaleString()} <span>جنيه</span>
            </div>
            <button class="charge-btn" onclick="openPaymentModal(${pkg.id}, ${pkg.points_count}, ${pkg.price})">
                <i class="fas fa-bolt"></i>
                اشحن الآن
            </button>
        </div>
    `).join('');
}

function openPaymentModal(packageId, points, price) {
    selectedPackageId = packageId;
    selectedPackageData = { points, price };
    
    document.getElementById('selectedPackageInfo').textContent = 
        `${points.toLocaleString()} نقطة - ${parseFloat(price).toLocaleString()} جنيه`;
    
    document.getElementById('paymentModal').style.display = 'block';
}

function closePaymentModal() {
    document.getElementById('paymentModal').style.display = 'none';
}

function selectPaymentMethod(method) {
    const methodNames = {
        'balance': 'الدفع من الرصيد',
        'online': 'الدفع الإلكتروني',
        'agent': 'الدفع عن طريق المندوب'
    };
    
    Swal.fire({
        title: 'تأكيد الشراء',
        html: `
            <p>هل تريد شراء <strong>${selectedPackageData.points.toLocaleString()} نقطة</strong></p>
            <p>بسعر <strong>${parseFloat(selectedPackageData.price).toLocaleString()} جنيه</strong></p>
            <p>عن طريق: <strong>${methodNames[method]}</strong></p>
        `,
        icon: 'question',
        showCancelButton: true,
        confirmButtonText: 'تأكيد الشراء',
        cancelButtonText: 'إلغاء',
        confirmButtonColor: '#667eea'
    }).then((result) => {
        if (result.isConfirmed) {
            processPayment(method);
        }
    });
}

function processPayment(method) {
    // إرسال طلب الشراء للخادم
    fetch('api/purchase_points.php', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            package_id: selectedPackageId,
            payment_method: method
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تمت عملية الشراء بنجاح!',
                html: `
                    <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                        <p style="font-size: 18px; margin: 10px 0;">تم إضافة <strong style="color: #667eea;">${data.data.points_added.toLocaleString()}</strong> نقطة</p>
                        <p style="font-size: 16px; margin: 10px 0; color: #666;">رصيد النقاط الجديد: <strong>${data.data.new_points.toLocaleString()}</strong> نقطة</p>
                        <p style="font-size: 14px; margin: 10px 0; color: #999;">رصيدك المالي: <strong>${data.data.new_balance.toLocaleString()}</strong> جنيه</p>
                    </div>
                `,
                confirmButtonText: 'حسناً',
                confirmButtonColor: '#667eea'
            });
            closePaymentModal();
        } else {
            // في حالة عدم كفاية الرصيد
            if (data.shortage) {
                Swal.fire({
                    icon: 'error',
                    title: 'رصيد غير كافٍ',
                    html: `
                        <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                            <p style="font-size: 16px; margin: 10px 0;">رصيدك الحالي: <strong>${data.current_balance.toLocaleString()}</strong> جنيه</p>
                            <p style="font-size: 16px; margin: 10px 0;">المبلغ المطلوب: <strong>${data.required.toLocaleString()}</strong> جنيه</p>
                            <p style="font-size: 18px; margin: 15px 0; color: #ef4444;">تحتاج إلى: <strong>${data.shortage.toLocaleString()}</strong> جنيه إضافي</p>
                        </div>
                    `,
                    confirmButtonText: 'حسناً',
                    confirmButtonColor: '#667eea'
                });
            } else {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ',
                    text: data.message,
                    confirmButtonText: 'حسناً'
                });
            }
        }
    })
    .catch(error => {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ في الاتصال',
            confirmButtonText: 'حسناً'
        });
    });
}

// Close modal when clicking outside
window.onclick = function(event) {
    if (event.target.id === 'paymentModal') {
        closePaymentModal();
    }
}

// Load packages on page load
loadPackages();
</script>

<?php include 'includes/footer.php'; ?>
