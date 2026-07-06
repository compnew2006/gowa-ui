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

<div class="users-container">
    <div class="page-header">
        <div class="page-title">
            <i class="fas fa-users"></i>
            إدارة المستخدمين
        </div>
        <div class="search-box">
            <input type="text" class="search-input" id="searchInput" placeholder="بحث عن مستخدم...">
        </div>
    </div>

    <div class="users-table-container">
        <table class="users-table">
            <thead>
                <tr>
                    <th>الاسم</th>
                    <th>البريد الإلكتروني</th>
                    <th>الهاتف</th>
                    <th>النوع</th>
                    <th>الحالة</th>
                    <th>الباقة</th>
                    <th>تاريخ الانتهاء</th>
                    <th>الإجراءات</th>
                </tr>
            </thead>
            <tbody id="usersTableBody">
                <tr>
                    <td colspan="8" class="loading">
                        <i class="fas fa-spinner fa-spin"></i>
                        جاري التحميل...
                    </td>
                </tr>
            </tbody>
        </table>

        <div class="pagination" id="pagination"></div>
    </div>
</div>

<!-- Edit User Modal -->
<div class="modal manageuser-modal" id="editUserModal">
    <div class="modal-content">
        <div class="modal-header">
            <h2 class="modal-title">تفعيل المستخدم</h2>
            <button class="close-modal" onclick="closeModal()">&times;</button>
        </div>
        <form id="editUserForm" onsubmit="saveUser(event)">
            <input type="hidden" id="userId" name="user_id">
            
            <div class="form-group">
                <label class="form-label">اسم المستخدم</label>
                <input type="text" class="form-input" id="userName" readonly>
            </div>

            <div class="form-group">
                <label class="form-label">الصلاحية</label>
                <label class="toggle-switch">
                    <input type="checkbox" id="isAdmin" name="is_admin">
                    <span class="toggle-slider"></span>
                </label>
                <span id="adminLabel" style="margin-right: 10px; font-weight: 600; color: var(--text-primary);">مستخدم</span>
            </div>

            <div class="form-group">
                <label class="form-label">الباقة</label>
                <select class="form-select" id="packageSelect" name="package" required>
                    <option value="">اختر الباقة</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label">تاريخ الانتهاء</label>
                <input type="datetime-local" class="form-input" id="expiryDate" name="expiry_date" required>
            </div>

     <div class="form-group">
                <label class="form-label">عدد النقط</label>
                <input type="text" class="form-input" id="pints" name="pints">
            </div>
            
            <div class="modal-footer">
                <button type="button" class="btn-cancel" onclick="closeModal()">إلغاء</button>
                <button type="submit" class="btn-submit">
                    <i class="fas fa-save"></i> حفظ التعديلات
                </button>
            </div>
        </form>
    </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/sweetalert2@11"></script>
<script>
let currentPage = 1;
let totalPages = 1;
let searchQuery = '';
let packages = [];

// تحميل المستخدمين عند فتح الصفحة
document.addEventListener('DOMContentLoaded', function() {
    loadPackages();
    loadUsers();
    
    // البحث
    document.getElementById('searchInput').addEventListener('input', function(e) {
        searchQuery = e.target.value;
        currentPage = 1;
        loadUsers();
    });
    
    // Toggle Admin
    document.getElementById('isAdmin').addEventListener('change', function() {
        document.getElementById('adminLabel').textContent = this.checked ? 'مسؤول' : 'مستخدم';
    });
});

// تحميل الباقات
function loadPackages() {
    fetch('get_packages.php')
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            packages = data.data || data.packages || [];
            const select = document.getElementById('packageSelect');
            // إضافة الباقة التجريبية (0) يدوياً
            select.innerHTML = '<option value="">اختر الباقة</option><option value="0">باقة تجريبية</option>';
            // إضافة باقي الباقات من قاعدة البيانات
            packages.forEach(pkg => {
                select.innerHTML += `<option value="${pkg.id}">${pkg.name}</option>`;
            });
        } else {
            // في حالة فشل تحميل الباقات، عرض الباقة التجريبية على الأقل
            const select = document.getElementById('packageSelect');
            select.innerHTML = '<option value="">اختر الباقة</option><option value="0">باقة تجريبية</option>';
        }
    })
    .catch(error => {
        console.error('Error loading packages:', error);
        // عرض الباقة التجريبية عند حدوث خطأ
        const select = document.getElementById('packageSelect');
        select.innerHTML = '<option value="">اختر الباقة</option><option value="0">باقة تجريبية</option>';
    });
}

// تحميل المستخدمين
function loadUsers() {
    const tbody = document.getElementById('usersTableBody');
    tbody.innerHTML = '<tr><td colspan="8" class="loading"><i class="fas fa-spinner fa-spin"></i> جاري التحميل...</td></tr>';
    
    fetch(`get_users.php?page=${currentPage}&search=${encodeURIComponent(searchQuery)}`)
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            displayUsers(data.users);
            totalPages = data.total_pages;
            updatePagination();
        } else {
            tbody.innerHTML = '<tr><td colspan="8" class="loading">لا يوجد مستخدمين</td></tr>';
        }
    })
    .catch(error => {
        console.error('Error:', error);
        tbody.innerHTML = '<tr><td colspan="8" class="loading">حدث خطأ أثناء التحميل</td></tr>';
    });
}

// عرض المستخدمين
function displayUsers(users) {
    const tbody = document.getElementById('usersTableBody');
    
    if (!users || users.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="loading">لا يوجد مستخدمين</td></tr>';
        return;
    }
    
    tbody.innerHTML = users.map(user => {
        const fullName = `${user.first_name} ${user.last_name}`;
        
        // تحديد نوع المستخدم (مسؤول/مستخدم) مع أيقونة
        const isAdmin = user.is_admin == 1;
        const roleClass = isAdmin ? 'badge-admin' : 'badge-user';
        const roleIcon = isAdmin ? '<i class="fas fa-user-shield status-icon icon-admin"></i>' : '<i class="fas fa-user status-icon icon-user"></i>';
        const roleText = isAdmin ? 'مسؤول' : 'مستخدم';
        
        // تحديد التفعيل (مفعّل إذا كانت package > 0، غير مفعّل إذا package = 0 أو null)
        const isActive = user.package > 0;
        const verifiedClass = isActive ? 'badge-verified' : 'badge-unverified';
        const verifiedIcon = isActive ? '<i class="fas fa-check-circle status-icon icon-active"></i>' : '<i class="fas fa-times-circle status-icon icon-inactive"></i>';
        const verifiedText = isActive ? 'مفعّل' : 'غير مفعّل';
        
        // تحديد إذا كان الاشتراك منتهي
        const now = new Date();
        const expiryDateObj = user.expiry_date ? new Date(user.expiry_date) : null;
        const isExpired = expiryDateObj && expiryDateObj < now;
        
        let expiryDisplay = '';
        if (expiryDateObj) {
            const formattedDate = expiryDateObj.toLocaleDateString('ar-EG');
            if (isExpired) {
                expiryDisplay = `<span class="badge-expired"><i class="fas fa-exclamation-triangle status-icon icon-expired"></i> منتهي (${formattedDate})</span>`;
            } else {
                expiryDisplay = formattedDate;
            }
        } else {
            expiryDisplay = 'غير محدد';
        }
        
        return `
            <tr>
                <td><strong>${fullName}</strong></td>
                <td>${user.email}</td>
                <td>${user.phone}</td>
                <td><span class="user-badge ${roleClass}">${roleIcon} ${roleText}</span></td>
                <td><span class="user-badge ${verifiedClass}">${verifiedIcon} ${verifiedText}</span></td>
                <td><strong>${user.package_name || 'باقة تجريبية'}</strong></td>
                <td>${expiryDisplay}</td>
                <td>
                    <div class="action-buttons">
                        <button class="btn-icon btn-activate" onclick='activateUser(${JSON.stringify(user).replace(/'/g, "&#39;")})' title="تفعيل">
                            <i class="fas fa-toggle-on"></i>
                        </button>
                        <button class="btn-icon btn-edit" onclick='editUser(${JSON.stringify(user).replace(/'/g, "&#39;")})' title="تعديل">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn-icon btn-view" onclick='viewUser(${user.id})' title="عرض">
                            <i class="fas fa-eye"></i>
                        </button>
                        <button class="btn-icon btn-delete" onclick='deleteUser(${user.id}, "${user.first_name} ${user.last_name}")' title="حذف">
                            <i class="fas fa-trash-alt"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

// فتح مودال التفعيل/التعديل
function activateUser(user) {
    openUserModal(user);
}

function editUser(user) {
    openSettingsModal(user);
}

function openUserModal(user) {
    document.getElementById('userId').value = user.id;
    document.getElementById('userName').value = `${user.first_name} ${user.last_name}`;
    document.getElementById('isAdmin').checked = user.is_admin == 1;
    document.getElementById('adminLabel').textContent = user.is_admin == 1 ? 'مسؤول' : 'مستخدم';
    document.getElementById('packageSelect').value = user.package || '';
    document.getElementById('pints').value = user.points || '';
    
    if (user.expiry_date) {
        const date = new Date(user.expiry_date);
        document.getElementById('expiryDate').value = date.toISOString().slice(0, 16);
    } else {
        // تعيين تاريخ افتراضي (شهر من الآن)
        const defaultDate = new Date();
        defaultDate.setMonth(defaultDate.getMonth() + 1);
        document.getElementById('expiryDate').value = defaultDate.toISOString().slice(0, 16);
    }
    
    document.getElementById('editUserModal').classList.add('active');
}

// إغلاق المودال
function closeModal() {
    document.getElementById('editUserModal').classList.remove('active');
}

// حفظ التعديلات
function saveUser(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    formData.set('is_admin', document.getElementById('isAdmin').checked ? 1 : 0);
    
    fetch('update_user.php', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'نجاح',
                text: 'تم تحديث بيانات المستخدم بنجاح',
                timer: 2000,
                showConfirmButton: false
            });
            closeModal();
            loadUsers();
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
            text: 'حدث خطأ أثناء التحديث',
            confirmButtonText: 'حسناً'
        });
    });
}

// تحديث Pagination
function updatePagination() {
    const pagination = document.getElementById('pagination');
    let html = '';
    
    // Previous button
    html += `<button onclick="changePage(${currentPage - 1})" ${currentPage === 1 ? 'disabled' : ''}>
        <i class="fas fa-chevron-right"></i> السابق
    </button>`;
    
    // Page numbers
    for (let i = 1; i <= totalPages; i++) {
        if (i === 1 || i === totalPages || (i >= currentPage - 2 && i <= currentPage + 2)) {
            html += `<button onclick="changePage(${i})" ${i === currentPage ? 'class="active"' : ''}>${i}</button>`;
        } else if (i === currentPage - 3 || i === currentPage + 3) {
            html += '<button disabled>...</button>';
        }
    }
    
    // Next button
    html += `<button onclick="changePage(${currentPage + 1})" ${currentPage === totalPages ? 'disabled' : ''}>
        التالي <i class="fas fa-chevron-left"></i>
    </button>`;
    
    pagination.innerHTML = html;
}

// تغيير الصفحة
function changePage(page) {
    if (page >= 1 && page <= totalPages) {
        currentPage = page;
        loadUsers();
    }
}

// عرض معلومات المستخدم (التبديل إلى حسابه)
function viewUser(userId) {
    Swal.fire({
        title: 'هل أنت متأكد؟',
        html: 'سيتم التبديل إلى حساب هذا المستخدم<br><span style="color: #f59e0b; font-weight: 600;">يمكنك العودة في أي وقت من زر "عودة" في القائمة</span>',
        icon: 'question',
        showCancelButton: true,
        confirmButtonColor: '#f59e0b',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، التبديل',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('switch_user.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `target_user_id=${userId}`
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم التبديل!',
                        text: data.message,
                        timer: 2000,
                        showConfirmButton: false
                    }).then(() => {
                        window.location.href = data.redirect;
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
                    text: 'حدث خطأ أثناء التبديل',
                    confirmButtonText: 'حسناً'
                });
            });
        }
    });
}

// حذف مستخدم
function deleteUser(userId, userName) {
    Swal.fire({
        title: 'هل أنت متأكد؟',
        html: `هل تريد حذف المستخدم <strong>${userName}</strong>؟<br><span style="color: #ef4444; font-weight: 600;">هـذه العمليـة لا يمكن التراجع عنها!</span>`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#ef4444',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، احـذف!',
        cancelButtonText: 'إلغاء',
        reverseButtons: true
    }).then((result) => {
        if (result.isConfirmed) {
            // إرسال طلب الحذف
            fetch('delete_user.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `user_id=${userId}`
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم الحذف!',
                        text: 'تم حذف المستخدم بنجاح',
                        timer: 2000,
                        showConfirmButton: false
                    });
                    loadUsers(); // إعادة تحميل الجدول
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message || 'فشل حذف المستخدم',
                        confirmButtonText: 'حسناً'
                    });
                }
            })
            .catch(error => {
                console.error('Error:', error);
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ في الاتصال',
                    text: 'حدث خطأ أثناء الحذف',
                    confirmButtonText: 'حسناً'
                });
            });
        }
    });
}
</script>

<?php 
include 'edit_user_settings.php';
include 'includes/admin_footer.php'; 
?>
