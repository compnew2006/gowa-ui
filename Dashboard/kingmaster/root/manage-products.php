 


<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة المنتجات | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>
<style>
    .products-container {
        padding: 30px;
        max-width: 1800px;
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
    }

    .add-product-btn {
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

    .add-product-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
    }

    /* Search and Filter */
    .filters-section {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 20px;
        margin-bottom: 30px;
        border: 2px solid var(--border-color);
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
    }

    .search-box {
        flex: 1;
        min-width: 250px;
        position: relative;
    }

    .search-box input {
        width: 100%;
        padding: 12px 45px 12px 20px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: var(--bg-secondary);
        color: var(--text-primary);
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .search-box input:focus {
        outline: none;
        border-color: #667eea;
    }

    .search-box i {
        position: absolute;
        right: 15px;
        top: 50%;
        transform: translateY(-50%);
        color: var(--text-secondary);
    }

    .filter-select {
        padding: 12px 20px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: var(--bg-secondary);
        color: var(--text-primary);
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        min-width: 150px;
    }

    /* Products Table */
    .products-table-container {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 20px;
        border: 2px solid var(--border-color);
        overflow-x: auto;
    }

    .products-table {
        width: 100%;
        border-collapse: collapse;
    }

    .products-table thead {
        background: rgba(102, 126, 234, 0.1);
    }

    .products-table th {
        padding: 15px;
        text-align: right;
        font-weight: 700;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        font-size: 15px;
        border-bottom: 2px solid var(--border-color);
    }

    .products-table td {
        padding: 15px;
        text-align: right;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        border-bottom: 1px solid var(--border-color);
    }

    .products-table tbody tr {
        transition: all 0.3s ease;
    }

    .products-table tbody tr:hover {
        background: rgba(102, 126, 234, 0.05);
    }

    .product-image {
        width: 60px;
        height: 60px;
        border-radius: 10px;
        object-fit: cover;
        border: 2px solid var(--border-color);
    }

    .product-name {
        font-weight: 600;
        color: var(--text-primary);
    }

    .product-price {
        font-weight: 700;
        color: #667eea;
        font-size: 16px;
    }

    .old-price {
        text-decoration: line-through;
        color: var(--text-secondary);
        font-size: 14px;
        margin-left: 8px;
    }

    .discount-badge {
        background: linear-gradient(135deg, #f43f5e 0%, #e11d48 100%);
        color: white;
        padding: 3px 8px;
        border-radius: 6px;
        font-size: 12px;
        font-weight: 600;
        margin-right: 8px;
    }

    .status-badge {
        padding: 6px 12px;
        border-radius: 8px;
        font-size: 13px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
    }

    .status-active {
        background: rgba(34, 197, 94, 0.1);
        color: #16a34a;
    }

    .status-inactive {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
    }

    .status-out {
        background: rgba(251, 191, 36, 0.1);
        color: #d97706;
    }

    .badge {
        padding: 4px 10px;
        border-radius: 6px;
        font-size: 12px;
        font-weight: 600;
        margin: 0 3px;
    }

    .badge-digital {
        background: rgba(102, 126, 234, 0.1);
        color: #667eea;
    }

    .badge-new {
        background: rgba(34, 197, 94, 0.1);
        color: #16a34a;
    }

    .badge-featured {
        background: rgba(251, 191, 36, 0.1);
        color: #d97706;
    }

    .action-buttons {
        display: flex;
        gap: 8px;
        justify-content: center;
    }

    .btn-action {
        width: 35px;
        height: 35px;
        border-radius: 8px;
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.3s ease;
        font-size: 14px;
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

    /* Modal Styles */
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

    .form-input,
    .form-textarea,
    .form-select {
        width: 100%;
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: var(--bg-secondary);
        color: var(--text-primary);
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .form-input:focus,
    .form-textarea:focus,
    .form-select:focus {
        outline: none;
        border-color: #667eea;
    }

    .form-textarea {
        min-height: 100px;
        resize: vertical;
    }

    .form-checkbox-group {
        display: flex;
        gap: 20px;
        flex-wrap: wrap;
    }

    .checkbox-item {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .checkbox-item input[type="checkbox"] {
        width: 20px;
        height: 20px;
        cursor: pointer;
    }

    .checkbox-item label {
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        color: var(--text-primary);
    }

    .form-row {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 15px;
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

    /* Light Theme */
    body.light-theme .filters-section,
    body.light-theme .products-table-container,
    body.light-theme .modal-content {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .search-box input,
    body.light-theme .filter-select,
    body.light-theme .form-input,
    body.light-theme .form-textarea,
    body.light-theme .form-select {
        background: #f8f9fa;
        border-color: rgba(0, 0, 0, 0.1);
    }

    /* Responsive */
    @media (max-width: 768px) {
        .products-container {
            padding: 20px;
            margin-top: 100px;
        }

        .page-header {
            flex-direction: column;
            align-items: stretch;
        }

        .form-row {
            grid-template-columns: 1fr;
        }

        .products-table {
            font-size: 14px;
        }

        .products-table th,
        .products-table td {
            padding: 10px;
        }
    }
</style>

<div class="products-container">
    <div class="page-header">
        <div class="page-title">
            <i class="fas fa-box-open"></i>
            إدارة المنتجات
        </div>
        <button class="add-product-btn" onclick="openAddModal()">
            <i class="fas fa-plus"></i>
            إضافة منتج جديد
        </button>
    </div>

    <!-- Filters Section -->
    <div class="filters-section">
        <div class="search-box">
            <input type="text" id="searchInput" placeholder="ابحث عن منتج..." onkeyup="filterProducts()">
            <i class="fas fa-search"></i>
        </div>
        <select class="filter-select" id="categoryFilter" onchange="filterProducts()">
            <option value="">جميع الفئات</option>
            <option value="courses">دورات</option>
            <option value="clothing">ملابس</option>
            <option value="ebooks">كتب إلكترونية</option>
            <option value="shoes">أحذية</option>
        </select>
        <select class="filter-select" id="statusFilter" onchange="filterProducts()">
            <option value="">جميع الحالات</option>
            <option value="active">نشط</option>
            <option value="inactive">غير نشط</option>
            <option value="out_of_stock">نفذت الكمية</option>
        </select>
    </div>

    <!-- Products Table -->
    <div class="products-table-container">
        <table class="products-table" id="productsTable">
            <thead>
                <tr>
                    <th>الصورة</th>
                    <th>اسم المنتج</th>
                    <th>الفئة</th>
                    <th>السعر</th>
                    <th>الكمية</th>
                    <th>التقييم</th>
                    <th>الحالة</th>
                    <th>الخصائص</th>
                    <th>الإجراءات</th>
                </tr>
            </thead>
            <tbody id="productsTableBody">
                <!-- سيتم ملؤها ديناميكياً -->
            </tbody>
        </table>
    </div>
</div>

<!-- Add/Edit Product Modal -->
<div class="modal" id="productModal">
    <div class="modal-content">
        <div class="modal-header">
            <h2 class="modal-title" id="modalTitle">إضافة منتج جديد</h2>
            <button class="close-modal" onclick="closeModal()">&times;</button>
        </div>
        <form id="productForm" onsubmit="saveProduct(event)">
            <input type="hidden" id="productId" name="id">
            
            <div class="form-group">
                <label class="form-label">اسم المنتج *</label>
                <input type="text" class="form-input" id="productName" name="name" required>
            </div>

            <div class="form-group">
                <label class="form-label">الوصف</label>
                <textarea class="form-textarea" id="productDescription" name="description"></textarea>
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label class="form-label">السعر *</label>
                    <input type="number" step="0.01" class="form-input" id="productPrice" name="price" required>
                </div>
                <div class="form-group">
                    <label class="form-label">نسبة الخصم %</label>
                    <input type="number" step="0.01" class="form-input" id="productDiscount" name="discount_percentage" value="0">
                </div>
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label class="form-label">الكمية المتاحة</label>
                    <input type="number" class="form-input" id="productStock" name="stock_quantity" value="0">
                </div>
                <div class="form-group">
                    <label class="form-label">الفئة</label>
                    <select class="form-select" id="productCategory" name="category">
                        <option value="courses">دورات</option>
                        <option value="clothing">ملابس</option>
                        <option value="ebooks">كتب إلكترونية</option>
                        <option value="shoes">أحذية</option>
                        <option value="other">أخرى</option>
                    </select>
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">رابط الصورة</label>
                <input type="url" class="form-input" id="productImage" name="image_url" placeholder="https://example.com/image.jpg">
            </div>

            <div class="form-group">
                <label class="form-label">الحالة</label>
                <select class="form-select" id="productStatus" name="status">
                    <option value="active">نشط</option>
                    <option value="inactive">غير نشط</option>
                    <option value="out_of_stock">نفذت الكمية</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label">الخصائص</label>
                <div class="form-checkbox-group">
                    <div class="checkbox-item">
                        <input type="checkbox" id="isDigital" name="is_digital">
                        <label for="isDigital">منتج رقمي</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="isNew" name="is_new">
                        <label for="isNew">جديد</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="isFeatured" name="is_featured">
                        <label for="isFeatured">مميز</label>
                    </div>
                </div>
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
// تحميل المنتجات عند تحميل الصفحة
document.addEventListener('DOMContentLoaded', function() {
    loadProducts();
});

// تحميل جميع المنتجات
function loadProducts() {
    fetch('api/get_products.php?admin=true')
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            displayProducts(data.products);
        } else {
            console.error('Error loading products:', data.message);
        }
    })
    .catch(error => {
        console.error('Error:', error);
    });
}

// عرض المنتجات في الجدول
function displayProducts(products) {
    const tbody = document.getElementById('productsTableBody');
    
    if (!products || products.length === 0) {
        tbody.innerHTML = '<tr><td colspan="9" style="text-align: center; padding: 40px;">لا توجد منتجات</td></tr>';
        return;
    }

    tbody.innerHTML = products.map(product => `
        <tr>
            <td><img src="${product.image_url || 'https://via.placeholder.com/60'}" alt="${product.name}" class="product-image"></td>
            <td class="product-name">${product.name}</td>
            <td>${getCategoryName(product.category)}</td>
            <td>
                ${product.discount_percentage > 0 ? `<span class="discount-badge">-${product.discount_percentage}%</span>` : ''}
                <span class="product-price">${product.final_price} ريال</span>
                ${product.discount_percentage > 0 ? `<span class="old-price">${product.price} ريال</span>` : ''}
            </td>
            <td>${product.stock_quantity}</td>
            <td>
                <i class="fas fa-star" style="color: #fbbf24;"></i>
                ${product.rating} (${product.reviews_count})
            </td>
            <td><span class="status-badge status-${product.status}">${getStatusName(product.status)}</span></td>
            <td>
                ${product.is_digital ? '<span class="badge badge-digital">رقمي</span>' : ''}
                ${product.is_new ? '<span class="badge badge-new">جديد</span>' : ''}
                ${product.is_featured ? '<span class="badge badge-featured">مميز</span>' : ''}
            </td>
            <td>
                <div class="action-buttons">
                    <button class="btn-action btn-edit" onclick='editProduct(${JSON.stringify(product)})' title="تعديل">
                        <i class="fas fa-edit"></i>
                    </button>
                    <button class="btn-action btn-delete" onclick="deleteProduct(${product.id})" title="حذف">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </td>
        </tr>
    `).join('');
}

// الحصول على اسم الفئة بالعربية
function getCategoryName(category) {
    const categories = {
        'courses': 'دورات',
        'clothing': 'ملابس',
        'ebooks': 'كتب إلكترونية',
        'shoes': 'أحذية',
        'other': 'أخرى'
    };
    return categories[category] || category;
}

// الحصول على اسم الحالة بالعربية
function getStatusName(status) {
    const statuses = {
        'active': 'نشط',
        'inactive': 'غير نشط',
        'out_of_stock': 'نفذت الكمية'
    };
    return statuses[status] || status;
}

// فتح مودال الإضافة
function openAddModal() {
    document.getElementById('modalTitle').textContent = 'إضافة منتج جديد';
    document.getElementById('productForm').reset();
    document.getElementById('productId').value = '';
    document.getElementById('productModal').classList.add('active');
}

// فتح مودال التعديل
function editProduct(product) {
    document.getElementById('modalTitle').textContent = 'تعديل المنتج';
    document.getElementById('productId').value = product.id;
    document.getElementById('productName').value = product.name;
    document.getElementById('productDescription').value = product.description || '';
    document.getElementById('productPrice').value = product.price;
    document.getElementById('productDiscount').value = product.discount_percentage || 0;
    document.getElementById('productStock').value = product.stock_quantity || 0;
    document.getElementById('productCategory').value = product.category || 'other';
    document.getElementById('productImage').value = product.image_url || '';
    document.getElementById('productStatus').value = product.status || 'active';
    document.getElementById('isDigital').checked = product.is_digital == 1;
    document.getElementById('isNew').checked = product.is_new == 1;
    document.getElementById('isFeatured').checked = product.is_featured == 1;
    document.getElementById('productModal').classList.add('active');
}

// إغلاق المودال
function closeModal() {
    document.getElementById('productModal').classList.remove('active');
}

// حفظ المنتج (إضافة أو تعديل)
function saveProduct(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    const productId = document.getElementById('productId').value;
    
    // تحويل checkboxes إلى قيم 0 أو 1
    formData.set('is_digital', document.getElementById('isDigital').checked ? 1 : 0);
    formData.set('is_new', document.getElementById('isNew').checked ? 1 : 0);
    formData.set('is_featured', document.getElementById('isFeatured').checked ? 1 : 0);
    
    const url = productId ? 'api/update_product.php' : 'api/add_product.php';
    
    fetch(url, {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            closeModal();
            loadProducts();
            showNotification('تم حفظ المنتج بنجاح!', 'success');
        } else {
            showNotification('حدث خطأ: ' + data.message, 'error');
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showNotification('حدث خطأ في الاتصال', 'error');
    });
}

// حذف منتج
function deleteProduct(productId) {
    if (!confirm('هل أنت متأكد من حذف هذا المنتج؟')) {
        return;
    }
    
    fetch('api/delete_product.php', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ id: productId })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            loadProducts();
            showNotification('تم حذف المنتج بنجاح!', 'success');
        } else {
            showNotification('حدث خطأ: ' + data.message, 'error');
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showNotification('حدث خطأ في الاتصال', 'error');
    });
}

// فلترة المنتجات
function filterProducts() {
    const searchTerm = document.getElementById('searchInput').value.toLowerCase();
    const categoryFilter = document.getElementById('categoryFilter').value;
    const statusFilter = document.getElementById('statusFilter').value;
    const rows = document.querySelectorAll('#productsTableBody tr');
    
    rows.forEach(row => {
        const name = row.cells[1]?.textContent.toLowerCase() || '';
        const category = row.cells[2]?.textContent || '';
        const status = row.querySelector('.status-badge')?.className || '';
        
        const matchesSearch = name.includes(searchTerm);
        const matchesCategory = !categoryFilter || category === getCategoryName(categoryFilter);
        const matchesStatus = !statusFilter || status.includes(`status-${statusFilter}`);
        
        row.style.display = (matchesSearch && matchesCategory && matchesStatus) ? '' : 'none';
    });
}

// إظهار إشعار
function showNotification(message, type) {
    // يمكنك استخدام مكتبة للإشعارات مثل toastr أو إنشاء إشعار مخصص
    alert(message);
}
</script>

<?php include 'includes/footer.php'; ?>
