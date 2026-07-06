<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة الكوبونات | Kingmaster";
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php';
?>


 


<style>
    .coupons-container {
        padding: 30px;
        max-width: 1600px;
        margin: 120px auto 0 auto;
    }

    .page-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
        flex-wrap: wrap;
        gap: 20px;
    }

    .page-title {
        font-size: 32px;
        font-weight: 800;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .page-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: bounce 2s ease-in-out infinite;
    }

    @keyframes bounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-10px); }
    }

    .add-coupon-btn {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        padding: 12px 30px;
        border-radius: 12px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
    }

    .add-coupon-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
    }

    /* Coupons Grid */
    .coupons-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 25px;
        margin-bottom: 30px;
    }

    .coupon-card {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 25px;
        border: 2px solid var(--border-color);
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
    }

    .coupon-card::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 4px;
        background: linear-gradient(90deg, #667eea, #764ba2, #f093fb);
    }

    .coupon-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 15px 35px rgba(102, 126, 234, 0.3);
        border-color: #667eea;
    }

    .coupon-code {
        font-size: 24px;
        font-weight: 800;
        color: #667eea;
        font-family: 'Courier New', monospace;
        margin-bottom: 15px;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .coupon-code i {
        color: #fbbf24;
        animation: sparkle 2s ease-in-out infinite;
    }

    @keyframes sparkle {
        0%, 100% { transform: scale(1) rotate(0deg); }
        50% { transform: scale(1.2) rotate(180deg); }
    }

    .coupon-type {
        display: inline-block;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 13px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
        margin-bottom: 15px;
    }

    .type-points {
        background: rgba(59, 130, 246, 0.1);
        color: #3b82f6;
    }

    .type-duration {
        background: rgba(16, 185, 129, 0.1);
        color: #10b981;
    }

    .type-discount {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
    }

    .coupon-info {
        margin: 15px 0;
    }

    .info-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
    }

    .info-label {
        color: var(--text-secondary);
        font-weight: 600;
    }

    .info-value {
        color: var(--text-primary);
        font-weight: 700;
    }

    .usage-bar {
        width: 100%;
        height: 8px;
        background: var(--bg-secondary);
        border-radius: 10px;
        overflow: hidden;
        margin: 10px 0;
    }

    .usage-progress {
        height: 100%;
        background: linear-gradient(90deg, #667eea, #764ba2);
        transition: width 0.3s ease;
    }

    .coupon-actions {
        display: flex;
        gap: 10px;
        margin-top: 15px;
        padding-top: 15px;
        border-top: 2px solid var(--border-color);
    }

    .btn-action {
        flex: 1;
        padding: 10px;
        border: none;
        border-radius: 8px;
        cursor: pointer;
        font-weight: 600;
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        transition: all 0.3s ease;
    }

    .btn-edit {
        background: rgba(34, 197, 94, 0.1);
        color: #16a34a;
    }

    .btn-edit:hover {
        background: #16a34a;
        color: white;
        transform: translateY(-2px);
    }

    .btn-delete {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
    }

    .btn-delete:hover {
        background: #dc2626;
        color: white;
        transform: translateY(-2px);
    }

    /* Modal */
    .modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        z-index: 9999;
        justify-content: center;
        align-items: center;
        backdrop-filter: blur(5px);
    }

    .modal.active {
        display: flex;
    }

    .modal-content {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        max-width: 600px;
        width: 90%;
        max-height: 90vh;
        overflow-y: auto;
        border: 2px solid var(--border-color);
        animation: modalSlideIn 0.3s ease;
    }

    @keyframes modalSlideIn {
        from {
            opacity: 0;
            transform: translateY(-50px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 25px;
        padding-bottom: 15px;
        border-bottom: 2px solid var(--border-color);
    }

    .modal-title {
        font-size: 24px;
        font-weight: 800;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }

    .modal-title::before {
        content: '🎟️';
        margin-left: 10px;
    }

    .close-modal {
        background: none;
        border: none;
        font-size: 28px;
        color: var(--text-secondary);
        cursor: pointer;
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 50%;
        transition: all 0.3s ease;
    }

    .close-modal:hover {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
        transform: rotate(90deg);
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-label {
        display: block;
        margin-bottom: 8px;
        font-weight: 600;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        font-size: 15px;
    }

    .input-with-button {
        display: flex;
        gap: 10px;
    }

    .input-with-button input {
        flex: 1;
    }

    .generate-btn {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        color: white;
        border: none;
        padding: 12px 20px;
        border-radius: 10px;
        cursor: pointer;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
        white-space: nowrap;
    }

    .generate-btn:hover {
        transform: scale(1.05);
        box-shadow: 0 5px 15px rgba(16, 185, 129, 0.4);
    }

    .form-input,
    .form-select {
        width: 100%;
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: #1e293b;
        color: #f1f5f9;
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .form-input:focus,
    .form-select:focus {
        outline: none;
        border-color: #667eea;
    }

    .form-select option {
        background: #1e293b;
        color: #f1f5f9;
    }

    .duration-inputs {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 10px;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 25px;
        padding-top: 20px;
        border-top: 2px solid var(--border-color);
    }

    .btn-submit {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        padding: 12px 30px;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .btn-submit:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
    }

    .btn-cancel {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
        border: none;
        padding: 12px 30px;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .btn-cancel:hover {
        background: #dc2626;
        color: white;
    }

    .expired-badge {
        position: absolute;
        top: 15px;
        left: 15px;
        background: rgba(239, 68, 68, 0.9);
        color: white;
        padding: 5px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
    }

    /* Light Theme */
    body.light-theme .form-input,
    body.light-theme .form-select {
        background: #ffffff;
        color: #2d3436;
    }

    body.light-theme .form-select option {
        background: #ffffff;
        color: #2d3436;
    }

    body.light-theme .coupon-card,
    body.light-theme .modal-content {
        background: rgba(255, 255, 255, 0.95);
    }

    /* Empty State */
    .empty-state {
        text-align: center;
        padding: 80px 20px;
    }

    .empty-state i {
        font-size: 120px;
        background: linear-gradient(135deg, #667eea, #764ba2, #f093fb);
        background-size: 400% 400%;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: gradient-move 3s ease infinite;
        margin-bottom: 20px;
    }

    @keyframes gradient-move {
        0% { background-position: 0% 50%; }
        50% { background-position: 100% 50%; }
        100% { background-position: 0% 50%; }
    }

    .empty-state h3 {
        font-size: 24px;
        color: var(--text-primary);
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
    }

    .empty-state p {
        font-size: 16px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
    }

    /* Responsive */
    @media (max-width: 768px) {
        .coupons-container {
            padding: 20px;
            margin-top: 100px;
        }

        .coupons-grid {
            grid-template-columns: 1fr;
        }

        .duration-inputs {
            grid-template-columns: 1fr;
        }
    }
</style>

<div class="coupons-container">
    <div class="page-header">
        <div class="page-title">
            <i class="fas fa-ticket-alt"></i>
            إدارة الكوبونات
        </div>
        <button class="add-coupon-btn" onclick="openAddModal()">
            <i class="fas fa-plus"></i>
            إضافة كوبون جديد
        </button>
    </div>

    <div class="coupons-grid" id="couponsGrid">
        <div class="empty-state">
            <i class="fas fa-ticket-alt"></i>
            <h3>جاري تحميل الكوبونات...</h3>
        </div>
    </div>
</div>

<!-- Add/Edit Coupon Modal -->
<div class="modal" id="couponModal">
    <div class="modal-content">
        <div class="modal-header">
            <h2 class="modal-title" id="modalTitle">إضافة كوبون جديد</h2>
            <button class="close-modal" onclick="closeModal()">&times;</button>
        </div>
        <form id="couponForm" onsubmit="saveCoupon(event)">
            <input type="hidden" id="couponId" name="id">
            
            <div class="form-group">
                <label class="form-label">كود الكوبون *</label>
                <div class="input-with-button">
                    <input type="text" class="form-input" id="couponCode" name="code" required placeholder="مثال: SUMMER2024">
                    <button type="button" class="generate-btn" onclick="generateCode()">
                        <i class="fas fa-random"></i> توليد
                    </button>
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">نوع الكوبون *</label>
                <select class="form-select" id="discountType" name="discount_type" onchange="updateValueInput()" required>
                    <option value="">اختر النوع</option>
                    <option value="points">نقاط</option>
                    <option value="duration">مدة</option>
                    <option value="discount">خصم</option>
                </select>
            </div>

            <div class="form-group" id="valueInputContainer">
                <label class="form-label">قيمة الكوبون *</label>
                <input type="number" class="form-input" id="discountValue" name="discount_value" placeholder="0" min="0.01" step="0.01">
            </div>

            <div class="form-group" id="durationInputContainer" style="display: none;">
                <label class="form-label">المدة (بالأيام) *</label>
                <input type="number" class="form-input" id="durationDays" placeholder="عدد الأيام" min="1">
            </div>

            <div class="form-group">
                <label class="form-label">عدد الاستخدامات المتاحة *</label>
                <input type="number" class="form-input" id="usesLimit" name="uses_limit" required placeholder="0" min="1">
            </div>

            <div class="form-group">
                <label class="form-label">تاريخ انتهاء الكوبون</label>
                <input type="datetime-local" class="form-input" id="expiresAt" name="expires_at">
            </div>

            <div class="modal-footer">
                <button type="button" class="btn-cancel" onclick="closeModal()">إلغاء</button>
                <button type="submit" class="btn-submit">
                    <i class="fas fa-save"></i> حفظ
                </button>
            </div>
        </form>
    </div>
</div>

<script>
document.addEventListener('DOMContentLoaded', function() {
    loadCoupons();
});

// تحميل جميع الكوبونات
function loadCoupons() {
    fetch('api/get_coupons.php')
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            displayCoupons(data.coupons);
        } else {
            console.error('Error loading coupons:', data.message);
        }
    })
    .catch(error => {
        console.error('Error:', error);
    });
}

// عرض الكوبونات
function displayCoupons(coupons) {
    const grid = document.getElementById('couponsGrid');
    
    if (!coupons || coupons.length === 0) {
        grid.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-ticket-alt"></i>
                <h3>لا توجد كوبونات</h3>
                <p>ابدأ بإضافة كوبون جديد</p>
            </div>
        `;
        return;
    }

    grid.innerHTML = coupons.map(coupon => {
        const usagePercent = (coupon.used_count / coupon.uses_limit) * 100;
        const isExpired = coupon.expires_at && new Date(coupon.expires_at) < new Date();
        const typeLabels = {
            'points': 'نقاط',
            'duration': 'مدة',
            'discount': 'خصم'
        };
        const typeClasses = {
            'points': 'type-points',
            'duration': 'type-duration',
            'discount': 'type-discount'
        };

        return `
            <div class="coupon-card">
                ${isExpired ? '<div class="expired-badge">منتهي</div>' : ''}
                <div class="coupon-code">
                    <i class="fas fa-gift"></i>
                    ${coupon.code}
                </div>
                <span class="coupon-type ${typeClasses[coupon.discount_type] || 'type-points'}">
                    ${typeLabels[coupon.discount_type] || coupon.discount_type}
                </span>
                
                <div class="coupon-info">
                    <div class="info-row">
                        <span class="info-label">النوع:</span>
                        <span class="info-value">${typeLabels[coupon.discount_type] || coupon.discount_type}</span>
                    </div>
                    <div class="info-row">
                        <span class="info-label">القيمة:</span>
                        <span class="info-value">${coupon.discount_value}${coupon.discount_type === 'duration' ? ' يوم' : ''}</span>
                    </div>
                    <div class="info-row">
                        <span class="info-label">الاستخدامات:</span>
                        <span class="info-value">${coupon.used_count} / ${coupon.uses_limit}</span>
                    </div>
                    <div class="usage-bar">
                        <div class="usage-progress" style="width: ${usagePercent}%"></div>
                    </div>
                    ${coupon.expires_at ? `
                    <div class="info-row">
                        <span class="info-label">ينتهي في:</span>
                        <span class="info-value">${new Date(coupon.expires_at).toLocaleDateString('ar-EG')}</span>
                    </div>
                    ` : ''}
                </div>

                <div class="coupon-actions">
                    <button class="btn-action btn-edit" onclick='editCoupon(${JSON.stringify(coupon).replace(/'/g, "&apos;")})'>
                        <i class="fas fa-edit"></i> تعديل
                    </button>
                    <button class="btn-action btn-delete" onclick="deleteCoupon(${coupon.id})">
                        <i class="fas fa-trash"></i> حذف
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

// فتح مودال الإضافة
function openAddModal() {
    document.getElementById('modalTitle').textContent = 'إضافة كوبون جديد';
    document.getElementById('couponForm').reset();
    document.getElementById('couponId').value = '';
    document.getElementById('valueInputContainer').style.display = 'block';
    document.getElementById('durationInputContainer').style.display = 'none';
    document.getElementById('couponModal').classList.add('active');
}

// فتح مودال التعديل
function editCoupon(coupon) {
    document.getElementById('modalTitle').textContent = 'تعديل الكوبون';
    document.getElementById('couponId').value = coupon.id;
    document.getElementById('couponCode').value = coupon.code;
    document.getElementById('discountType').value = coupon.discount_type;
    document.getElementById('usesLimit').value = coupon.uses_limit;
    
    if (coupon.expires_at) {
        const date = new Date(coupon.expires_at);
        const formatted = date.toISOString().slice(0, 16);
        document.getElementById('expiresAt').value = formatted;
    }
    
    updateValueInput();
    
    if (coupon.discount_type === 'duration') {
        document.getElementById('durationDays').value = coupon.discount_value;
    } else {
        document.getElementById('discountValue').value = coupon.discount_value;
    }
    
    document.getElementById('couponModal').classList.add('active');
}

// إغلاق المودال
function closeModal() {
    document.getElementById('couponModal').classList.remove('active');
}

// توليد كود عشوائي
function generateCode() {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    let code = '';
    for (let i = 0; i < 8; i++) {
        code += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    document.getElementById('couponCode').value = code;
}

// تحديث حقل القيمة حسب النوع
function updateValueInput() {
    const type = document.getElementById('discountType').value;
    const valueContainer = document.getElementById('valueInputContainer');
    const durationContainer = document.getElementById('durationInputContainer');
    
    if (type === 'duration') {
        valueContainer.style.display = 'none';
        durationContainer.style.display = 'block';
    } else {
        valueContainer.style.display = 'block';
        durationContainer.style.display = 'none';
    }
}

// حفظ الكوبون
function saveCoupon(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    const type = document.getElementById('discountType').value;
    
    // التحقق من نوع الكوبون
    if (!type) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'الرجاء اختيار نوع الكوبون',
            confirmButtonText: 'حسناً'
        });
        return;
    }
    
    // حساب قيمة المدة بالأيام
    if (type === 'duration') {
        const days = parseInt(document.getElementById('durationDays').value) || 0;
        
        if (days <= 0) {
            Swal.fire({
                icon: 'warning',
                title: 'تنبيه',
                text: 'الرجاء إدخال عدد أيام صالح',
                confirmButtonText: 'حسناً'
            });
            return;
        }
        
        formData.set('discount_value', days);
    } else {
        // التحقق من قيمة الخصم أو النقاط
        const value = parseFloat(document.getElementById('discountValue').value);
        if (!value || value <= 0) {
            Swal.fire({
                icon: 'warning',
                title: 'تنبيه',
                text: 'الرجاء إدخال قيمة صالحة للكوبون',
                confirmButtonText: 'حسناً'
            });
            return;
        }
    }
    
    const couponId = document.getElementById('couponId').value;
    const url = couponId ? 'api/update_coupon.php' : 'api/add_coupon.php';
    
    fetch(url, {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            closeModal();
            loadCoupons();
            Swal.fire({
                icon: 'success',
                title: 'نجاح',
                text: 'تم حفظ الكوبون بنجاح!',
                timer: 2000,
                showConfirmButton: false
            });
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message,
                confirmButtonText: 'حسناً'
            });
        }
    })
    .catch(error => {
        console.error('Error:', error);
        Swal.fire({
            icon: 'error',
            title: 'خطك في الاتصال',
            text: 'حدث خطأ أثناء الاتصال بالخادم',
            confirmButtonText: 'حسناً'
        });
    });
}

// حذف كوبون
function deleteCoupon(couponId) {
    Swal.fire({
        title: 'هل أنت متأكد؟',
        text: 'هل تريد حذف هذا الكوبون؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#dc2626',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (!result.isConfirmed) return;
    
        fetch('api/delete_coupon.php', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ id: couponId })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                loadCoupons();
                Swal.fire({
                    icon: 'success',
                    title: 'تم الحذف',
                    text: 'تم حذف الكوبون بنجاح',
                    timer: 2000,
                    showConfirmButton: false
                });
            } else {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ',
                    text: data.message,
                    confirmButtonText: 'حسناً'
                });
            }
        })
        .catch(error => {
            console.error('Error:', error);
            Swal.fire({
                icon: 'error',
                title: 'خطأ في الاتصال',
                text: 'حدث خطأ أثناء الاتصال بالخادم',
                confirmButtonText: 'حسناً'
            });
        });
    });
}
</script>

<?php include 'includes/admin_footer.php'; ?>
