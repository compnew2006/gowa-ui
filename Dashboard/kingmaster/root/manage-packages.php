<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة الكوبونات | Kingmaster";
$page_css = ['https://kingmaster.info/css/internal.css'];
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php';
?>





<div class="packages-container">
    <div class="page-header">
        <div class="page-title">
            <i class="fas fa-box-open"></i>
            إدارة الباقات
        </div>
        <button class="add-package-btn" onclick="openAddModal()">
            <i class="fas fa-plus"></i>
            إضافة باقة جديدة
        </button>
    </div>

    <div class="packages-grid" id="packagesGrid">
        <div class="empty-state">
            <i class="fas fa-box-open"></i>
            <h3>جاري تحميل الباقات...</h3>
        </div>
    </div>
</div>

<!-- Add/Edit Package Modal -->
<div class="modal" id="packageModal">
    <div class="modal-content">
        <div class="modal-header">
            <h2 class="modal-title" id="modalTitle">إضافة باقة جديدة</h2>
            <button class="close-modal" onclick="closeModal()">&times;</button>
        </div>
        <form id="packageForm" onsubmit="savePackage(event)">
            <input type="hidden" id="packageId" name="id">
            
            <div class="form-group">
                <label class="form-label">اسم الباقة *</label>
                <input type="text" class="form-input" id="packageName" name="name" required placeholder="مثال: الباقة الذهبية">
            </div>

            <div class="form-group">
                <label class="form-label">وصف الباقة</label>
                <textarea class="form-textarea" id="packageDescription" name="description" placeholder="وصف مختصر عن الباقة"></textarea>
            </div>

            <div class="form-group">
                <label class="form-label">مميزات الباقة *</label>
                <div class="features-input-container">
                    <input type="text" class="form-input" id="featureInput" placeholder="أدخل ميزة واضغط إضافة">
                    <button type="button" class="add-feature-btn" onclick="addFeature()">
                        <i class="fas fa-plus"></i> إضافة
                    </button>
                </div>
                <div class="features-list" id="featuresList"></div>
                <input type="hidden" id="featuresData" name="features">
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label class="form-label">السعر الحالي *</label>
                    <input type="number" class="form-input" id="packagePrice" name="price" required placeholder="0.00" step="0.01" min="0">
                </div>

                <div class="form-group">
                    <label class="form-label">السعر الأصلي (قبل الخصم)</label>
                    <input type="number" class="form-input" id="originalPrice" name="original_price" placeholder="0.00" step="0.01" min="0">
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">العملة *</label>
                <select class="form-select" id="packageCurrency" name="currency" required>
                    <option value="EGP">🇪🇬 جنيه مصري (EGP)</option>
                    <option value="USD">🇺🇸 دولار أمريكي (USD)</option>
                    <option value="SAR">🇸🇦 ريال سعودي (SAR)</option>
                    <option value="AED">🇦🇪 درهم إماراتي (AED)</option>
                    <option value="KWD">🇰🇼 دينار كويتي (KWD)</option>
                    <option value="QAR">🇶🇦 ريال قطري (QAR)</option>
                    <option value="EUR">🇪🇺 يورو (EUR)</option>
                    <option value="GBP">🇬🇧 جنيه إسترليني (GBP)</option>
                </select>
            </div>

            <div class="form-group">
                <div class="checkbox-item">
                    <input type="checkbox" id="hasDiscount" name="has_discount" value="1">
                    <label for="hasDiscount">هل الباقة عليها خصم؟</label>
                </div>
            </div>

            <div class="form-group">
                <div class="checkbox-item">
                    <input type="checkbox" id="isPopular" name="is_popular" value="1">
                    <label for="isPopular">هل الباقة شعبية؟ (الأكثر مبيعاً)</label>
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">المنصات المتاحة *</label>
                <div class="checkbox-group">
                    <div class="checkbox-item">
                        <input type="checkbox" id="platform_facebook" name="platforms[]" value="facebook">
                        <label for="platform_facebook"><i class="fab fa-facebook"></i> فيسبوك</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="platform_whatsapp" name="platforms[]" value="whatsapp">
                        <label for="platform_whatsapp"><i class="fab fa-whatsapp"></i> واتساب</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="platform_telegram" name="platforms[]" value="telegram">
                        <label for="platform_telegram"><i class="fab fa-telegram"></i> تليجرام</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="platform_instagram" name="platforms[]" value="instagram">
                        <label for="platform_instagram"><i class="fab fa-instagram"></i> إنستجرام</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="platform_email" name="platforms[]" value="email">
                        <label for="platform_email"><i class="fas fa-envelope"></i> بريد</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="platform_business" name="platforms[]" value="business">
                        <label for="platform_business"><i class="fas fa-briefcase"></i> أعمال</label>
                    </div>
                </div>
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label class="form-label">عدد الحسابات *</label>
                    <input type="number" class="form-input" id="accountsCount" name="accounts_count" required placeholder="0" min="0">
                </div>

                <div class="form-group">
                    <label class="form-label">عدد الرسائل *</label>
                    <input type="number" class="form-input" id="messagesCount" name="messages_count" required placeholder="0" min="0">
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">عدد النقاط *</label>
                <input type="number" class="form-input" id="points" name="points" required placeholder="0" min="0">
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
let featuresArray = [];

document.addEventListener('DOMContentLoaded', function() {
    loadPackages();
});

// تحميل جميع الباقات
function loadPackages() {
    fetch('api/get_packages.php')
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            displayPackages(data.packages);
        } else {
            console.error('Error loading packages:', data.message);
        }
    })
    .catch(error => {
        console.error('Error:', error);
    });
}

// عرض الباقات
function displayPackages(packages) {
    const grid = document.getElementById('packagesGrid');
    
    if (!packages || packages.length === 0) {
        grid.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-box-open"></i>
                <h3>لا توجد باقات</h3>
                <p>ابدأ بإضافة باقة جديدة</p>
            </div>
        `;
        return;
    }

    grid.innerHTML = packages.map(pkg => {
        const features = JSON.parse(pkg.features || '[]');
        const platforms = JSON.parse(pkg.platforms || '[]');
        
        const platformIcons = {
            'facebook': '<i class="fab fa-facebook"></i>',
            'whatsapp': '<i class="fab fa-whatsapp"></i>',
            'telegram': '<i class="fab fa-telegram"></i>',
            'instagram': '<i class="fab fa-instagram"></i>',
            'email': '<i class="fas fa-envelope"></i>',
            'business': '<i class="fas fa-briefcase"></i>'
        };

        const platformNames = {
            'facebook': 'فيسبوك',
            'whatsapp': 'واتساب',
            'telegram': 'تليجرام',
            'instagram': 'إنستجرام',
            'email': 'بريد',
            'business': 'أعمال'
        };

        const discountPercent = pkg.has_discount && pkg.original_price > 0 
            ? Math.round(((pkg.original_price - pkg.price) / pkg.original_price) * 100) 
            : 0;

        const currencySymbols = {
            'EGP': 'جنيه',
            'USD': '$',
            'SAR': 'ريال',
            'AED': 'درهم',
            'KWD': 'د.ك',
            'QAR': 'ر.ق',
            'EUR': '€',
            'GBP': '£'
        };
        const currency = currencySymbols[pkg.currency] || pkg.currency || 'جنيه';

        return `
            <div class="package-card">
                <div class="package-header">
                    <div class="package-name">${pkg.name}</div>
                    ${pkg.is_popular ? '<div class="popular-badge"><i class="fas fa-star"></i> الأكثر مبيعاً</div>' : ''}
                    ${pkg.description ? `<div class="package-description">${pkg.description}</div>` : ''}
                </div>
                
                <div class="package-body">
                    <div class="package-price">
                        ${pkg.has_discount && discountPercent > 0 ? `<div class="discount-badge">خصم ${discountPercent}%</div>` : ''}
                        <div class="price-wrapper">
                            <span class="price-current">${pkg.price} ${currency}</span>
                            ${pkg.has_discount && pkg.original_price ? `<span class="price-original">${pkg.original_price} ${currency}</span>` : ''}
                        </div>
                    </div>

                    <div class="package-stats">
                        <div class="stat-item">
                            <div class="stat-icon accounts-icon">
                                <i class="fas fa-users"></i>
                            </div>
                            <div class="stat-value">${pkg.accounts_count}</div>
                            <div class="stat-label">حساب</div>
                        </div>
                        <div class="stat-item">
                            <div class="stat-icon messages-icon">
                                <i class="fas fa-envelope"></i>
                            </div>
                            <div class="stat-value">${pkg.messages_count}</div>
                            <div class="stat-label">رسالة</div>
                        </div>
                        <div class="stat-item">
                            <div class="stat-icon points-icon">
                                <i class="fas fa-star"></i>
                            </div>
                            <div class="stat-value">${pkg.points}</div>
                            <div class="stat-label">نقطة</div>
                        </div>
                    </div>

                    <div class="package-features">
                        <div class="features-title">
                            <i class="fas fa-list-check"></i>
                            <span>مميزات الباقة</span>
                        </div>
                        ${features.map(feature => `
                            <div class="feature-item">
                                <i class="fas fa-check-circle"></i>
                                <span>${feature}</span>
                            </div>
                        `).join('')}
                    </div>

                    <div class="package-platforms">
                        ${platforms.map(platform => `
                            <span class="platform-badge platform-${platform}">
                                ${platformIcons[platform] || ''} ${platformNames[platform] || platform}
                            </span>
                        `).join('')}
                    </div>

                    <div class="package-actions">
                        <button class="btn-action btn-edit" onclick='editPackage(${JSON.stringify(pkg).replace(/'/g, "&apos;")})'>
                            <i class="fas fa-edit"></i> تعديل
                        </button>
                        <button class="btn-action btn-delete" onclick="deletePackage(${pkg.id})">
                            <i class="fas fa-trash"></i> حذف
                        </button>
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

// فتح مودال الإضافة
function openAddModal() {
    document.getElementById('modalTitle').textContent = 'إضافة باقة جديدة';
    document.getElementById('packageForm').reset();
    document.getElementById('packageId').value = '';
    featuresArray = [];
    updateFeaturesList();
    document.getElementById('packageModal').classList.add('active');
}

// فتح مودال التعديل
function editPackage(pkg) {
    document.getElementById('modalTitle').textContent = 'تعديل الباقة';
    document.getElementById('packageId').value = pkg.id;
    document.getElementById('packageName').value = pkg.name;
    document.getElementById('packageDescription').value = pkg.description || '';
    document.getElementById('packagePrice').value = pkg.price;
    document.getElementById('originalPrice').value = pkg.original_price || '';
    document.getElementById('packageCurrency').value = pkg.currency || 'EGP';
    document.getElementById('hasDiscount').checked = pkg.has_discount == 1;
    document.getElementById('isPopular').checked = pkg.is_popular == 1;
    document.getElementById('accountsCount').value = pkg.accounts_count;
    document.getElementById('messagesCount').value = pkg.messages_count;
    document.getElementById('points').value = pkg.points;
    
    // Features
    featuresArray = JSON.parse(pkg.features || '[]');
    updateFeaturesList();
    
    // Platforms
    const platforms = JSON.parse(pkg.platforms || '[]');
    document.querySelectorAll('input[name="platforms[]"]').forEach(checkbox => {
        checkbox.checked = platforms.includes(checkbox.value);
    });
    
    document.getElementById('packageModal').classList.add('active');
}

// إضافة ميزة
function addFeature() {
    const input = document.getElementById('featureInput');
    const feature = input.value.trim();
    
    if (feature && !featuresArray.includes(feature)) {
        featuresArray.push(feature);
        updateFeaturesList();
        input.value = '';
    }
}

// تحديث قائمة المميزات
function updateFeaturesList() {
    const list = document.getElementById('featuresList');
    list.innerHTML = featuresArray.map((feature, index) => `
        <div class="feature-tag">
            ${feature}
            <button type="button" onclick="removeFeature(${index})">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `).join('');
    
    document.getElementById('featuresData').value = JSON.stringify(featuresArray);
}

// حذف ميزة
function removeFeature(index) {
    featuresArray.splice(index, 1);
    updateFeaturesList();
}

// إغلاق المودال
function closeModal() {
    document.getElementById('packageModal').classList.remove('active');
}

// حفظ الباقة
function savePackage(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    
    // التحقق من المميزات
    if (featuresArray.length === 0) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'الرجاء إضافة ميزة واحدة على الأقل',
            confirmButtonText: 'حسناً'
        });
        return;
    }
    
    // التحقق من المنصات
    const platforms = formData.getAll('platforms[]');
    if (platforms.length === 0) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'الرجاء اختيار منصة واحدة على الأقل',
            confirmButtonText: 'حسناً'
        });
        return;
    }
    
    formData.set('features', JSON.stringify(featuresArray));
    formData.set('platforms', JSON.stringify(platforms));
    formData.set('has_discount', document.getElementById('hasDiscount').checked ? 1 : 0);
    formData.set('is_popular', document.getElementById('isPopular').checked ? 1 : 0);
    
    const packageId = document.getElementById('packageId').value;
    const url = packageId ? 'api/update_package.php' : 'api/add_package.php';
    
    fetch(url, {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            closeModal();
            loadPackages();
            Swal.fire({
                icon: 'success',
                title: 'نجاح',
                text: 'تم حفظ الباقة بنجاح!',
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
}

// حذف باقة
function deletePackage(packageId) {
    Swal.fire({
        title: 'هل أنت متأكد؟',
        text: 'هل تريد حذف هذه الباقة؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#dc2626',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (!result.isConfirmed) return;
    
        fetch('api/delete_package.php', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ id: packageId })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                loadPackages();
                Swal.fire({
                    icon: 'success',
                    title: 'تم الحذف',
                    text: 'تم حذف الباقة بنجاح',
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

// Enter key to add feature
document.getElementById('featureInput').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        e.preventDefault();
        addFeature();
    }
});
</script>

<?php include 'includes/admin_footer.php'; ?>
