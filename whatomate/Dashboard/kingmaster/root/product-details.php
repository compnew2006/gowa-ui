 
 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "تفاصيل المنتج | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';

$product_id = isset($_GET['id']) ? intval($_GET['id']) : 0;
if ($product_id <= 0) {
    header('Location: products.php');
    exit;
}

?>



<style>
    .product-details-container {
        padding: 30px;
        max-width: 1200px;
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

    .back-btn i {
        animation: bounce-small 1.5s ease-in-out infinite;
    }

    @keyframes bounce-small {
        0%, 100% { transform: translateX(0); }
        50% { transform: translateX(5px); }
    }

    .product-details-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 40px;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 40px;
        border: 1px solid var(--border-color);
    }

    .product-image-section {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        border-radius: 15px;
        padding: 60px;
        position: relative;
        min-height: 400px;
    }

    .product-image-section i {
        font-size: 150px;
        color: white;
        animation: float 3s ease-in-out infinite;
    }

    .product-image-section img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        border-radius: 0;
        animation: float 3s ease-in-out infinite;
        position: absolute;
        top: 0;
        left: 0;
    }

    @keyframes float {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-20px); }
    }

    .product-type-badge {
        position: absolute;
        top: 15px;
        right: 15px;
        padding: 6px 12px;
        border-radius: 20px;
        font-weight: 600;
        font-size: 12px;
        display: flex;
        align-items: center;
        gap: 6px;
        font-family: 'Cairo', sans-serif;
    }

    .product-type-badge i {
        font-size: 14px;
    }

    .badge-digital {
        background: rgba(255, 255, 255, 0.95);
        color: #667eea;
    }

    .badge-physical {
        background: rgba(255, 255, 255, 0.95);
        color: #f5576c;
    }

    .product-info-section {
        display: flex;
        flex-direction: column;
    }

    .product-title {
        font-size: 28px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 15px;
        font-family: 'Cairo', sans-serif;
        line-height: 1.4;
    }

    .product-category {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 8px 18px;
        background: var(--bg-primary);
        color: var(--primary-color);
        border-radius: 20px;
        font-weight: 700;
        font-size: 14px;
        margin-bottom: 20px;
        width: fit-content;
        font-family: 'Cairo', sans-serif;
    }

    .product-description {
        font-size: 15px;
        line-height: 1.8;
        color: var(--text-secondary);
        margin-bottom: 25px;
        padding: 20px;
        background: var(--bg-primary);
        border-radius: 12px;
        font-family: 'Cairo', sans-serif;
        font-weight: 500;
    }

    .product-price-section {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 20px;
        background: linear-gradient(135deg, #667eea15 0%, #764ba215 100%);
        border-radius: 12px;
        margin-bottom: 25px;
        border: 1px solid var(--border-color);
    }

    .price-value {
        font-size: 32px;
        font-weight: 800;
        color: var(--primary-color);
        display: flex;
        align-items: center;
        gap: 10px;
        font-family: 'Cairo', sans-serif;
    }

    .price-value i {
        color: #f59e0b;
        animation: coin-flip 3s ease-in-out infinite;
    }

    @keyframes coin-flip {
        0%, 100% { transform: rotateY(0deg); }
        50% { transform: rotateY(180deg); }
    }

    .commission-info {
        margin-top: 10px;
        font-size: 14px;
        color: #27ae60;
        display: flex;
        align-items: center;
        gap: 8px;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
    }

    .option-group {
        margin-bottom: 20px;
    }

    .option-label {
        font-size: 16px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 12px;
        display: flex;
        align-items: center;
        gap: 8px;
        font-family: 'Cairo', sans-serif;
    }

    .color-options,
    .size-options {
        display: flex;
        gap: 12px;
        flex-wrap: wrap;
    }

    .color-option,
    .size-option {
        padding: 10px 20px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        cursor: pointer;
        transition: all 0.3s ease;
        font-weight: 600;
        background: var(--bg-primary);
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }

    .color-option:hover,
    .size-option:hover {
        border-color: var(--primary-color);
        transform: translateY(-2px);
    }

    .color-option.selected,
    .size-option.selected {
        border-color: var(--primary-color);
        background: var(--primary-color);
        color: white;
    }

    .quantity-controls {
        display: flex;
        align-items: center;
        gap: 15px;
    }

    .quantity-btn {
        width: 45px;
        height: 45px;
        border: none;
        background: var(--primary-color);
        color: white;
        border-radius: 10px;
        font-size: 20px;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        font-family: 'Cairo', sans-serif;
        font-weight: 700;
    }

    .quantity-btn:hover {
        transform: scale(1.1);
    }

    .quantity-input {
        width: 80px;
        height: 45px;
        text-align: center;
        font-size: 18px;
        font-weight: 700;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: var(--bg-primary);
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }

    .stock-info {
        margin-top: 10px;
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
    }

    .address-input,
    .phone-input {
        width: 100%;
        padding: 14px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        font-size: 15px;
        background: #1e293b !important;
        color: #e5e7eb !important;
        margin-top: 10px;
        font-family: 'Cairo', sans-serif;
        font-weight: 500;
    }

    /* خيارات الدفع */
    .payment-section {
        margin-bottom: 20px;
    }

    .payment-options {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .payment-option {
        display: flex;
        align-items: center;
        gap: 15px;
        padding: 15px 20px;
        border: 2px solid var(--border-color);
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.3s ease;
        background: var(--bg-primary);
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
    }

    .payment-option:hover {
        border-color: var(--primary-color);
        transform: translateX(-5px);
    }

    .payment-option.selected {
        border-color: var(--primary-color);
        background: linear-gradient(135deg, #667eea15 0%, #764ba215 100%);
    }

    .payment-option input[type="radio"] {
        width: 20px;
        height: 20px;
        cursor: pointer;
    }

    .payment-option-content {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;
    }

    .payment-icon {
        font-size: 24px;
    }

    .payment-icon.balance {
        color: #10b981;
    }

    .payment-icon.cash {
        color: #f59e0b;
    }

    .payment-icon.electronic {
        color: #3b82f6;
    }

    .payment-label {
        font-size: 16px;
        color: var(--text-primary);
    }

    .payment-desc {
        font-size: 13px;
        color: var(--text-secondary);
        margin-top: 3px;
    }

    .buy-button {
        width: 100%;
        padding: 18px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 15px;
        font-size: 18px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
    }

    .buy-button:hover {
        transform: translateY(-3px);
        box-shadow: 0 10px 25px rgba(102, 126, 234, 0.4);
    }

    .buy-button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .total-section {
        background: var(--bg-primary);
        padding: 20px;
        border-radius: 12px;
        margin-bottom: 20px;
        border: 1px solid var(--border-color);
    }

    .total-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px 0;
        font-size: 18px;
        font-family: 'Cairo', sans-serif;
    }

    .total-label {
        color: var(--text-secondary);
        font-weight: 600;
    }

    .total-value {
        color: var(--primary-color);
        font-weight: 800;
        font-size: 24px;
    }

    /* Light Theme */
    body.light-theme .product-details-card,
    body.light-theme .back-btn,
    body.light-theme .payment-option,
    body.light-theme .total-section {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .address-input,
    body.light-theme .phone-input,
    body.light-theme .quantity-input,
    body.light-theme .color-option,
    body.light-theme .size-option {
        background: #ffffff !important;
        color: #2d3436 !important;
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .product-title,
    body.light-theme .option-label,
    body.light-theme .payment-label {
        color: #2d3436;
    }

    body.light-theme .product-description,
    body.light-theme .stock-info,
    body.light-theme .payment-desc {
        color: #636e72;
    }

    @media (max-width: 768px) {
        .product-details-card {
            grid-template-columns: 1fr;
            padding: 25px;
            gap: 30px;
        }

        .product-image-section {
            min-height: 300px;
            padding: 40px;
        }

        .product-image-section i {
            font-size: 100px;
        }

        .product-title {
            font-size: 24px;
        }

        .price-value {
            font-size: 28px;
        }
    }
</style>

<div class="product-details-container">
    <a href="products.php" class="back-btn">
        <i class="fas fa-arrow-right"></i>
        العودة للمنتجات
    </a>

    <div id="productDetailsCard" class="product-details-card">
        <div class="loading" style="grid-column: 1/-1; text-align: center; padding: 60px;">
            <div class="spinner" style="margin: 0 auto 20px; border: 4px solid var(--border-color); border-top: 4px solid var(--primary-color); border-radius: 50%; width: 50px; height: 50px; animation: spin 1s linear infinite;"></div>
            <p>جاري تحميل تفاصيل المنتج...</p>
        </div>
    </div>
</div>

<script>
    const productId = <?php echo $product_id; ?>;
    let currentProduct = null;
    let selectedColor = null;
    let selectedSize = null;
    let quantity = 1;
    let selectedPayment = 'balance'; // القيمة الافتراضية

    // تحميل تفاصيل المنتج
    async function loadProductDetails() {
        try {
            const response = await fetch('api/products.php', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ product_id: productId })
            });
            
            const data = await response.json();
            
            if (data.success) {
                currentProduct = data.product;
                renderProductDetails(currentProduct);
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
                text: 'حدث خطأ في تحميل تفاصيل المنتج',
                confirmButtonText: 'حسناً'
            });
        }
    }

    // عرض تفاصيل المنتج
    function renderProductDetails(product) {
        const isDigital = product.is_digital;
        const productIcon = getProductIcon(product.category, isDigital);
        
        const html = `
            <div class="product-image-section">
                ${product.image && product.image !== '' ? 
                    `<img src="uploads/products/${product.image}" alt="${product.name}" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">` +
                    `<i class="${productIcon}" style="display: none;"></i>` :
                    `<i class="${productIcon}"></i>`
                }
                <div class="product-type-badge ${isDigital ? 'badge-digital' : 'badge-physical'}">
                    <i class="fas ${isDigital ? 'fa-download' : 'fa-box'}"></i>
                    ${isDigital ? 'منتج رقمي' : 'منتج حقيقي'}
                </div>
            </div>
            
            <div class="product-info-section">
                <h1 class="product-title">${product.name}</h1>
                
                ${product.category ? `
                    <div class="product-category">
                        <i class="fas fa-tag"></i>
                        ${getCategoryName(product.category)}
                    </div>
                ` : ''}
                
                <div class="product-description">${product.description || 'لا يوجد وصف للمنتج'}</div>
                
                <div class="product-price-section">
                    <div>
                        <div class="price-label" style="font-family: 'Cairo', sans-serif; font-weight: 600;">السعر للوحدة</div>
                        ${product.commission > 0 ? `
                            <div class="commission-info" id="commissionPerUnit">
                                <i class="fas fa-hand-holding-usd"></i>
                                عمولة للوحدة: $${parseFloat(product.commission).toFixed(2)}
                            </div>
                        ` : ''}
                    </div>
                    <div class="price-value">
                        <i class="fas fa-coins"></i>
                        ${formatPrice(product.price)}
                    </div>
                </div>
                
                ${product.colors_array && product.colors_array.length > 0 ? `
                    <div class="option-group">
                        <div class="option-label">
                            <i class="fas fa-palette" style="color: #f59e0b;"></i>
                            اختر اللون
                        </div>
                        <div class="color-options">
                            ${product.colors_array.map(color => `
                                <div class="color-option" onclick="selectColor('${color}')">${color}</div>
                            `).join('')}
                        </div>
                    </div>
                ` : ''}
                
                ${product.sizes_array && product.sizes_array.length > 0 ? `
                    <div class="option-group">
                        <div class="option-label">
                            <i class="fas fa-ruler" style="color: #06b6d4;"></i>
                            اختر المقاس
                        </div>
                        <div class="size-options">
                            ${product.sizes_array.map(size => `
                                <div class="size-option" onclick="selectSize('${size}')">${size}</div>
                            `).join('')}
                        </div>
                    </div>
                ` : ''}
                
                <div class="option-group">
                    <div class="option-label">
                        <i class="fas fa-shopping-cart" style="color: #3b82f6;"></i>
                        الكمية
                    </div>
                    <div class="quantity-controls">
                        <button class="quantity-btn" onclick="decreaseQuantity()">
                            <i class="fas fa-minus"></i>
                        </button>
                        <input type="number" class="quantity-input" id="quantityInput" value="1" min="1" max="${product.stock_quantity || product.stock}" readonly>
                        <button class="quantity-btn" onclick="increaseQuantity()">
                            <i class="fas fa-plus"></i>
                        </button>
                    </div>
                    <div class="stock-info">
                        <i class="fas fa-box"></i>
                        متوفر: ${product.stock_quantity || product.stock} قطعة
                    </div>
                </div>
                
                ${!isDigital ? `
                    <div class="option-group">
                        <div class="option-label">
                            <i class="fas fa-map-marker-alt" style="color: #ef4444;"></i>
                            العنوان
                        </div>
                        <input type="text" class="address-input" id="addressInput" placeholder="أدخل عنوان التوصيل" required>
                        
                        <div class="option-label" style="margin-top: 15px;">
                            <i class="fas fa-phone" style="color: #10b981;"></i>
                            رقم الهاتف
                        </div>
                        <input type="tel" class="phone-input" id="phoneInput" placeholder="أدخل رقم الهاتف" required>
                    </div>
                ` : ''}
                
                <div class="payment-section">
                    <div class="option-label">
                        <i class="fas fa-credit-card" style="color: #8b5cf6;"></i>
                        طريقة الدفع
                    </div>
                    <div class="payment-options">
                        <label class="payment-option selected" onclick="selectPayment('balance', this)">
                            <input type="radio" name="payment" value="balance" checked>
                            <div class="payment-option-content">
                                <i class="fas fa-wallet payment-icon balance"></i>
                                <div>
                                    <div class="payment-label">الدفع من الرصيد</div>
                                    <div class="payment-desc">سيتم خصم المبلغ من رصيدك مباشرة</div>
                                </div>
                            </div>
                        </label>
                        
                        ${!isDigital ? `
                        <label class="payment-option" onclick="selectPayment('cash', this)">
                            <input type="radio" name="payment" value="cash">
                            <div class="payment-option-content">
                                <i class="fas fa-hand-holding-usd payment-icon cash"></i>
                                <div>
                                    <div class="payment-label">الدفع عند الاستلام</div>
                                    <div class="payment-desc">ادفع نقداً عند استلام الطلب</div>
                                </div>
                            </div>
                        </label>
                        ` : ''}
                        
                        <label class="payment-option" onclick="selectPayment('electronic', this)">
                            <input type="radio" name="payment" value="electronic">
                            <div class="payment-option-content">
                                <i class="fas fa-credit-card payment-icon electronic"></i>
                                <div>
                                    <div class="payment-label">الدفع الإلكتروني</div>
                                    <div class="payment-desc">ادفع عبر بطاقة ائتمان أو محفظة إلكترونية</div>
                                </div>
                            </div>
                        </label>
                    </div>
                </div>
                
                <div class="total-section">
                    <div class="total-row">
                        <span class="total-label">إجمالي السعر</span>
                        <span class="total-value" id="totalPrice">${formatPrice(product.price)}</span>
                    </div>
                    ${product.commission > 0 ? `
                    <div class="total-row" style="border-top: 1px solid var(--border-color); padding-top: 15px; margin-top: 10px;">
                        <span class="total-label" style="color: #27ae60;">
                            <i class="fas fa-hand-holding-usd"></i>
                            إجمالي العمولة
                        </span>
                        <span class="total-value" id="totalCommission" style="color: #27ae60; font-size: 20px;">${formatPrice(product.commission)}</span>
                    </div>
                    ` : ''}
                </div>
                
                <button class="buy-button" onclick="purchaseProduct()" ${parseInt(product.stock_quantity || product.stock) <= 0 ? 'disabled' : ''}>
                    <i class="fas fa-shopping-cart"></i>
                    ${parseInt(product.stock_quantity || product.stock) <= 0 ? 'نفذت الكمية' : 'تأكيد الطلب'}
                </button>
            </div>
        `;
        
        document.getElementById('productDetailsCard').innerHTML = html;
    }

    // اختيار طريقة الدفع
    function selectPayment(type, element) {
        selectedPayment = type;
        document.querySelectorAll('.payment-option').forEach(el => el.classList.remove('selected'));
        element.classList.add('selected');
    }

    // اختيار اللون
    function selectColor(color) {
        selectedColor = color;
        document.querySelectorAll('.color-option').forEach(el => el.classList.remove('selected'));
        event.target.classList.add('selected');
    }

    // اختيار المقاس
    function selectSize(size) {
        selectedSize = size;
        document.querySelectorAll('.size-option').forEach(el => el.classList.remove('selected'));
        event.target.classList.add('selected');
    }

    // زيادة الكمية
    function increaseQuantity() {
        const input = document.getElementById('quantityInput');
        const max = parseInt(input.max);
        if (quantity < max) {
            quantity++;
            input.value = quantity;
            updateTotal();
        }
    }

    // تقليل الكمية
    function decreaseQuantity() {
        if (quantity > 1) {
            quantity--;
            document.getElementById('quantityInput').value = quantity;
            updateTotal();
        }
    }

    // تحديث الإجمالي
    function updateTotal() {
        const total = currentProduct.price * quantity;
        document.getElementById('totalPrice').textContent = formatPrice(total);
        
        // تحديث إجمالي العمولة إذا كانت موجودة
        if (currentProduct.commission > 0) {
            const totalCommission = currentProduct.commission * quantity;
            const commissionElement = document.getElementById('totalCommission');
            if (commissionElement) {
                commissionElement.textContent = formatPrice(totalCommission);
            }
        }
    }

    // شراء المنتج
    async function purchaseProduct() {
        // التحقق من الخيارات
        if (currentProduct.colors_array && currentProduct.colors_array.length > 0 && !selectedColor) {
            Swal.fire({
                icon: 'warning',
                title: 'تنبيه',
                text: 'يرجى اختيار اللون',
                confirmButtonText: 'حسناً'
            });
            return;
        }
        
        if (currentProduct.sizes_array && currentProduct.sizes_array.length > 0 && !selectedSize) {
            Swal.fire({
                icon: 'warning',
                title: 'تنبيه',
                text: 'يرجى اختيار المقاس',
                confirmButtonText: 'حسناً'
            });
            return;
        }
        
        let address = null;
        let phone = null;
        
        if (!currentProduct.is_digital) {
            address = document.getElementById('addressInput')?.value.trim();
            phone = document.getElementById('phoneInput')?.value.trim();
            
            if (!address) {
                Swal.fire({
                    icon: 'warning',
                    title: 'تنبيه',
                    text: 'يرجى إدخال العنوان',
                    confirmButtonText: 'حسناً'
                });
                return;
            }
            
            if (!phone) {
                Swal.fire({
                    icon: 'warning',
                    title: 'تنبيه',
                    text: 'يرجى إدخال رقم الهاتف',
                    confirmButtonText: 'حسناً'
                });
                return;
            }
        }

        // إضا اختار الدفع الإلكتروني
        if (selectedPayment === 'electronic') {
            // حفظ بيانات الطلب في sessionStorage
            const orderData = {
                product_id: currentProduct.id,
                product_name: currentProduct.name,
                quantity: quantity,
                color: selectedColor,
                size: selectedSize,
                address: address,
                phone: phone,
                total: currentProduct.price * quantity
            };
            sessionStorage.setItem('pendingOrder', JSON.stringify(orderData));
            
            // التحويل لصفحة اختيار طريقة الدفع الإلكتروني
            window.location.href = 'payment-methods.php';
            return;
        }
        
        const orderData = {
            product_id: currentProduct.id,
            quantity: quantity,
            color: selectedColor,
            size: selectedSize,
            address: address,
            phone: phone,
            payment_method: selectedPayment
        };
        
        try {
            const response = await fetch('api/orders.php', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(orderData)
            });
            
            const data = await response.json();
            
            if (data.success) {
                Swal.fire({
                    icon: 'success',
                    title: 'تم بنجاح!',
                    html: `<p>تم إنشاء الطلب بنجاح</p><p><strong>رقم الطلب: ${data.order_id}</strong></p>`,
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
                text: 'حدث خطأ في إرسال الطلب',
                confirmButtonText: 'حسناً'
            });
        }
    }

    // دوال مساعدة
    function getProductIcon(category, isDigital) {
        const icons = {
            electronics: 'fas fa-mobile-alt',
            fashion: 'fas fa-tshirt',
            courses: 'fas fa-graduation-cap',
            books: 'fas fa-book',
            default: isDigital ? 'fas fa-download' : 'fas fa-box'
        };
        return icons[category] || icons.default;
    }

    function getCategoryName(category) {
        const names = {
            electronics: 'إلكترونيات',
            fashion: 'أزياء',
            courses: 'دورات',
            books: 'كتب'
        };
        return names[category] || category;
    }

    function formatPrice(price) {
        return '$' + parseFloat(price).toLocaleString('en-US', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        });
    }

    // تحميل التفاصيل عند فتح الصفحة
    document.addEventListener('DOMContentLoaded', loadProductDetails);
</script>

<style>
    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
</style>

<?php include 'includes/footer.php'; ?>
