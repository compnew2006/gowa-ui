 


<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "طرق الدفع الإلكتروني | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>
<style>
    .payment-methods-container {
        padding: 30px;
        max-width: 1000px;
        margin: 120px auto 0 auto;
    }

    .back-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 12px 25px;
        background: var(--card-bg);
        color: var(--text-primary);
        text-decoration: none;
        border-radius: 10px;
        margin-bottom: 25px;
        transition: all 0.3s ease;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        border: 1px solid var(--border-color);
    }

    .back-btn:hover {
        background: var(--primary-color);
        color: white;
        transform: translateX(5px);
    }

    .payment-header {
        text-align: center;
        margin-bottom: 40px;
    }

    .payment-header h1 {
        font-size: 32px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
    }

    .payment-header p {
        color: var(--text-secondary);
        font-size: 16px;
        font-family: 'Cairo', sans-serif;
    }

    .order-summary {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 25px;
        margin-bottom: 30px;
        border: 1px solid var(--border-color);
    }

    .summary-title {
        font-size: 20px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 20px;
        display: flex;
        align-items: center;
        gap: 10px;
        font-family: 'Cairo', sans-serif;
    }

    .summary-row {
        display: flex;
        justify-content: space-between;
        padding: 12px 0;
        border-bottom: 1px solid var(--border-color);
        font-family: 'Cairo', sans-serif;
    }

    .summary-row:last-child {
        border-bottom: none;
        font-size: 20px;
        font-weight: 800;
        color: var(--primary-color);
        padding-top: 20px;
        margin-top: 10px;
        border-top: 2px solid var(--border-color);
    }

    .payment-methods-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: 20px;
        margin-bottom: 30px;
    }

    .payment-method-card {
        background: var(--card-bg);
        border: 2px solid var(--border-color);
        border-radius: 15px;
        padding: 30px;
        cursor: pointer;
        transition: all 0.3s ease;
        text-align: center;
        position: relative;
    }

    .payment-method-card:hover {
        transform: translateY(-5px);
        border-color: var(--primary-color);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.2);
    }

    .payment-method-card.selected {
        border-color: var(--primary-color);
        background: linear-gradient(135deg, #667eea15 0%, #764ba215 100%);
    }

    .payment-icon {
        font-size: 60px;
        margin-bottom: 20px;
        animation: float 3s ease-in-out infinite;
    }

    @keyframes float {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-10px); }
    }

    .payment-icon.visa { color: #1A1F71; }
    .payment-icon.mastercard { color: #EB001B; }
    .payment-icon.paypal { color: #00457C; }
    .payment-icon.vodafone { color: #E60000; }
    .payment-icon.bank { color: #27ae60; }
    .payment-icon.crypto { color: #f7931a; }

    .payment-name {
        font-size: 20px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
    }

    .payment-desc {
        font-size: 14px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
    }

    .confirm-payment-btn {
        width: 100%;
        padding: 20px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 15px;
        font-size: 20px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
    }

    .confirm-payment-btn:hover {
        transform: translateY(-3px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
    }

    .confirm-payment-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .security-note {
        text-align: center;
        margin-top: 20px;
        padding: 15px;
        background: var(--bg-primary);
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        color: var(--text-secondary);
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
    }

    .security-note i {
        color: #27ae60;
        font-size: 20px;
    }

    @media (max-width: 768px) {
        .payment-methods-grid {
            grid-template-columns: 1fr;
        }

        .payment-header h1 {
            font-size: 26px;
        }
    }
</style>

<div class="payment-methods-container">
    <a href="javascript:history.back()" class="back-btn">
        <i class="fas fa-arrow-right"></i>
        العودة
    </a>

    <div class="payment-header">
        <h1>
            <i class="fas fa-credit-card"></i>
            اختر طريقة الدفع
        </h1>
        <p>اختر طريقة الدفع المناسبة لإتمام عملية الشراء</p>
    </div>

    <div class="order-summary" id="orderSummary">
        <div class="summary-title">
            <i class="fas fa-receipt"></i>
            ملخص الطلب
        </div>
        <div class="summary-row">
            <span>جاري التحميل...</span>
        </div>
    </div>

    <div class="payment-methods-grid">
        <div class="payment-method-card" onclick="selectPaymentMethod('visa')">
            <i class="fab fa-cc-visa payment-icon visa"></i>
            <div class="payment-name">Visa</div>
            <div class="payment-desc">الدفع ببطاقة فيزا</div>
        </div>

        <div class="payment-method-card" onclick="selectPaymentMethod('mastercard')">
            <i class="fab fa-cc-mastercard payment-icon mastercard"></i>
            <div class="payment-name">Mastercard</div>
            <div class="payment-desc">الدفع ببطاقة ماستركارد</div>
        </div>

        <div class="payment-method-card" onclick="selectPaymentMethod('paypal')">
            <i class="fab fa-cc-paypal payment-icon paypal"></i>
            <div class="payment-name">PayPal</div>
            <div class="payment-desc">الدفع عبر باي بال</div>
        </div>

        <div class="payment-method-card" onclick="selectPaymentMethod('vodafone')">
            <i class="fas fa-mobile-alt payment-icon vodafone"></i>
            <div class="payment-name">Vodafone Cash</div>
            <div class="payment-desc">الدفع عبر فودافون كاش</div>
        </div>

        <div class="payment-method-card" onclick="selectPaymentMethod('bank')">
            <i class="fas fa-university payment-icon bank"></i>
            <div class="payment-name">تحويل بنكي</div>
            <div class="payment-desc">تحويل مباشر من البنك</div>
        </div>

        <div class="payment-method-card" onclick="selectPaymentMethod('crypto')">
            <i class="fab fa-bitcoin payment-icon crypto"></i>
            <div class="payment-name">عملات رقمية</div>
            <div class="payment-desc">الدفع بالعملات المشفرة</div>
        </div>
    </div>

    <button class="confirm-payment-btn" id="confirmBtn" disabled onclick="processPayment()">
        <i class="fas fa-lock"></i>
        تأكيد الدفع
    </button>

    <div class="security-note">
        <i class="fas fa-shield-alt"></i>
        جميع المعاملات محمية بأحدث تقنيات الأمان والتشفير
    </div>
</div>

<script>
    let selectedMethod = null;
    let orderData = null;

    // تحميل بيانات الطلب من sessionStorage
    document.addEventListener('DOMContentLoaded', function() {
        const pendingOrder = sessionStorage.getItem('pendingOrder');
        
        if (!pendingOrder) {
            Swal.fire({
                icon: 'error',
                title: 'خطأ!',
                text: 'لا توجد بيانات طلب',
                confirmButtonText: 'العودة للمنتجات'
            }).then(() => {
                window.location.href = 'products.php';
            });
            return;
        }

        orderData = JSON.parse(pendingOrder);
        displayOrderSummary(orderData);
    });

    // عرض ملخص الطلب
    function displayOrderSummary(order) {
        const summaryHtml = `
            <div class="summary-title">
                <i class="fas fa-receipt"></i>
                ملخص الطلب
            </div>
            ${order.product_name ? `
                <div class="summary-row">
                    <span>المنتج</span>
                    <span style="font-weight: 700;">${order.product_name}</span>
                </div>
            ` : ''}
            <div class="summary-row">
                <span>الكمية</span>
                <span>${order.quantity}</span>
            </div>
            ${order.color ? `
                <div class="summary-row">
                    <span>اللون</span>
                    <span>${order.color}</span>
                </div>
            ` : ''}
            ${order.size ? `
                <div class="summary-row">
                    <span>المقاس</span>
                    <span>${order.size}</span>
                </div>
            ` : ''}
            ${order.phone ? `
                <div class="summary-row">
                    <span>رقم الهاتف</span>
                    <span style="direction: ltr;">${order.phone}</span>
                </div>
            ` : ''}
            ${order.address ? `
                <div class="summary-row">
                    <span>العنوان</span>
                    <span>${order.address}</span>
                </div>
            ` : ''}
            <div class="summary-row">
                <span>الإجمالي</span>
                <span>${formatPrice(order.total)}</span>
            </div>
        `;
        
        document.getElementById('orderSummary').innerHTML = summaryHtml;
    }

    // اختيار طريقة الدفع
    function selectPaymentMethod(method) {
        selectedMethod = method;
        
        // إزالة التحديد من جميع البطاقات
        document.querySelectorAll('.payment-method-card').forEach(card => {
            card.classList.remove('selected');
        });
        
        // إضافة التحديد للبطاقة المختارة
        event.currentTarget.classList.add('selected');
        
        // تفعيل زر التأكيد
        document.getElementById('confirmBtn').disabled = false;
    }

    // معالجة الدفع
    async function processPayment() {
        if (!selectedMethod) {
            Swal.fire({
                icon: 'warning',
                title: 'تنبيه',
                text: 'يرجى اختيار طريقة الدفع',
                confirmButtonText: 'حسناً'
            });
            return;
        }

        // إضافة طريقة الدفع لبيانات الطلب
        orderData.payment_method = 'electronic';
        orderData.electronic_method = selectedMethod;

        try {
            const response = await fetch('api/orders.php', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(orderData)
            });
            
            const data = await response.json();
            
            if (data.success) {
                // حذف بيانات الطلب المؤقتة
                sessionStorage.removeItem('pendingOrder');
                
                Swal.fire({
                    icon: 'success',
                    title: 'تم بنجاح!',
                    html: `
                        <p>تم إنشاء الطلب بنجاح</p>
                        <p><strong>رقم الطلب: ${data.order_id}</strong></p>
                        <p style="color: #27ae60; margin-top: 10px;">
                            <i class="fas fa-info-circle"></i>
                            سيتم التواصل معك قريباً لإتمام عملية الدفع عبر ${getMethodName(selectedMethod)}
                        </p>
                    `,
                    confirmButtonText: 'عرض طلباتي',
                    showCancelButton: true,
                    cancelButtonText: 'متابعة التسوق'
                }).then((result) => {
                    if (result.isConfirmed) {
                        window.location.href = 'orders.php';
                    } else {
                        window.location.href = 'products.php';
                    }
                });
            } else {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ!',
                    text: data.message,
                    confirmButtonText: 'حسناً'
                });
            }
        } catch (error) {
            console.error('خطأ:', error);
            Swal.fire({
                icon: 'error',
                title: 'خطأ!',
                text: 'حدث خطأ في معالجة الدفع',
                confirmButtonText: 'حسناً'
            });
        }
    }

    // الحصول على اسم طريقة الدفع
    function getMethodName(method) {
        const methods = {
            'visa': 'Visa',
            'mastercard': 'Mastercard',
            'paypal': 'PayPal',
            'vodafone': 'Vodafone Cash',
            'bank': 'التحويل البنكي',
            'crypto': 'العملات الرقمية'
        };
        return methods[method] || method;
    }

    // تنسيق السعر
    function formatPrice(price) {
        return '$' + parseFloat(price).toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        });
    }
</script>

<?php include 'includes/footer.php'; ?>
