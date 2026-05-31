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


<div class="manage-container">
    <div class="manage-header">
        <div class="manage-title">
            <i class="fas fa-tasks"></i>
            إدارة باقات النقاط
        </div>
        <button class="add-package-btn" onclick="openAddModal()">
            <i class="fas fa-plus"></i>
            إضافة باقة جديدة
        </button>
    </div>

    <div class="packages-table">
        <table>
            <thead>
                <tr>
                    <th>#</th>
                    <th>عدد النقاط</th>
                    <th>السعر (جنيه)</th>
                    <th>الحالة</th>
                    <th>تاريخ الإنشاء</th>
                    <th>الإجراءات</th>
                </tr>
            </thead>
            <tbody id="packagesTableBody">
                <!-- سيتم تحميل البيانات هنا -->
            </tbody>
        </table>
    </div>
</div>

<!-- Add/Edit Modal -->
<div id="packageModal" class="modal modal-points-packages">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title" id="modalTitle">
                <i class="fas fa-plus-circle"></i>
                <span id="modalTitleText">إضافة باقة جديدة</span>
            </div>
            <span class="close-modal" onclick="closeModal()">&times;</span>
        </div>
        <form id="packageForm" onsubmit="event.preventDefault(); savePackage();">
            <input type="hidden" id="packageId">
            
            <div class="form-group">
                <label class="form-label">عدد النقاط</label>
                <input type="number" class="form-input" id="pointsCount" placeholder="مثال: 1000" min="1" required>
            </div>

            <div class="form-group">
                <label class="form-label">السعر (جنيه)</label>
                <input type="number" step="0.01" class="form-input" id="price" placeholder="مثال: 85.00" min="0.01" required>
            </div>

            <button type="submit" class="submit-btn" id="submitBtn">
                <i class="fas fa-save"></i>
                <span id="submitBtnText">حفظ الباقة</span>
            </button>
        </form>
    </div>
</div>

<script>
let editMode = false;

function openAddModal() {
    editMode = false;
    document.getElementById('packageId').value = '';
    document.getElementById('pointsCount').value = '';
    document.getElementById('price').value = '';
    document.getElementById('modalTitleText').textContent = 'إضافة باقة جديدة';
    document.getElementById('submitBtnText').textContent = 'حفظ الباقة';
    document.getElementById('packageModal').style.display = 'block';
}

function openEditModal(id, points, price) {
    editMode = true;
    document.getElementById('packageId').value = id;
    document.getElementById('pointsCount').value = points;
    document.getElementById('price').value = price;
    document.getElementById('modalTitleText').textContent = 'تعديل الباقة';
    document.getElementById('submitBtnText').textContent = 'حفظ التعديلات';
    document.getElementById('packageModal').style.display = 'block';
}

function closeModal() {
    document.getElementById('packageModal').style.display = 'none';
}

function savePackage() {
    const packageId = document.getElementById('packageId').value;
    const pointsCount = document.getElementById('pointsCount').value;
    const price = document.getElementById('price').value;
    
    const url = editMode ? 'api/update_points_package.php' : 'api/add_points_package.php';
    const data = {
        points_count: pointsCount,
        price: price
    };
    
    if (editMode) {
        data.id = packageId;
    }
    
    fetch(url, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تم!',
                text: editMode ? 'تم تحديث الباقة بنجاح' : 'تم إضافة الباقة بنجاح',
                timer: 2000,
                showConfirmButton: false
            });
            closeModal();
            loadPackages();
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message
            });
        }
    })
    .catch(error => {
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'حدث خطأ في الاتصال'
        });
    });
}

function loadPackages() {
    fetch('api/get_all_points_packages.php')
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
    const tbody = document.getElementById('packagesTableBody');
    
    if (packages.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" style="padding: 40px; text-align: center; color: var(--text-secondary);">
                    <i class="fas fa-box-open" style="font-size: 50px; margin-bottom: 10px; display: block;"></i>
                    لا توجد باقات
                </td>
            </tr>
        `;
        return;
    }
    
    tbody.innerHTML = packages.map((pkg, index) => `
        <tr>
            <td><strong>${index + 1}</strong></td>
            <td><strong style="color: #667eea;">${parseInt(pkg.points_count).toLocaleString()} نقطة</strong></td>
            <td><strong>${parseFloat(pkg.price).toLocaleString()} جنيه</strong></td>
            <td>
                <span class="status-${pkg.is_active == 1 ? 'active' : 'inactive'}">
                    ${pkg.is_active == 1 ? 'نشط' : 'غير نشط'}
                </span>
            </td>
            <td>${new Date(pkg.created_at).toLocaleDateString('ar-EG')}</td>
            <td>
                <div class="action-btns">
                    <button class="btn-edit" onclick="openEditModal(${pkg.id}, ${pkg.points_count}, ${pkg.price})">
                        <i class="fas fa-edit"></i>
                        تعديل
                    </button>
                    <button class="btn-delete" onclick="deletePackage(${pkg.id})">
                        <i class="fas fa-trash"></i>
                        حذف
                    </button>
                </div>
            </td>
        </tr>
    `).join('');
}

function deletePackage(id) {
    Swal.fire({
        title: 'تأكيد الحذف',
        text: 'هل أنت متأكد من حذف هذه الباقة؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء',
        confirmButtonColor: '#ef4444'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('api/delete_points_package.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ id: id })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم الحذف!',
                        timer: 1500,
                        showConfirmButton: false
                    });
                    loadPackages();
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message
                    });
                }
            });
        }
    });
}

// Close modal when clicking outside
window.onclick = function(event) {
    if (event.target.id === 'packageModal') {
        closeModal();
    }
}

// Load packages on page load
loadPackages();
</script>

<?php include 'includes/admin_footer.php'; ?>
