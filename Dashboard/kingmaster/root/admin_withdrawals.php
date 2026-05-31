



<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة طلبات السحب | Kingmaster";
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php'; 
$commission = getcommission_walletsById($user_id);


?>


<style>
    .admin-container {
        padding: 30px;
        max-width: 1600px;
        margin: 120px auto 0 auto;
    }
    
    .admin-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
    }
    
    .admin-title {
        font-size: 28px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
    }
    
    .stats-row {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 20px;
        margin-bottom: 30px;
    }
    
    .stat-card {
        background: var(--card-bg);
        padding: 20px;
        border-radius: 15px;
        border: 1px solid var(--border-color);
        text-align: center;
    }
    
    .stat-value {
        font-size: 32px;
        font-weight: 800;
        margin-bottom: 8px;
        font-family: 'Cairo', sans-serif;
    }
    
    .stat-label {
        font-size: 14px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
    }
    
    .admin-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        border: 1px solid var(--border-color);
        box-shadow: 0 5px 20px rgba(0,0,0,0.1);
    }
    
    .admin-table {
        width: 100%;
        border-collapse: collapse;
        font-family: 'Cairo', sans-serif;
    }
    
    .admin-table thead {
        background: linear-gradient(135deg, #667eea, #764ba2);
    }
    
    .admin-table th {
        padding: 15px 10px;
        text-align: center;
        font-weight: 700;
        color: white;
        border-bottom: 2px solid var(--border-color);
        font-size: 13px;
    }
    
    .admin-table td {
        padding: 15px 10px;
        text-align: center;
        border-bottom: 1px solid var(--border-color);
        color: var(--text-primary);
        vertical-align: middle;
        font-size: 13px;
    }
    
    .admin-table tbody tr:hover {
        background: rgba(102, 126, 234, 0.05);
    }
    
    .status-badge {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        padding: 6px 12px;
        border-radius: 15px;
        font-weight: 600;
        font-size: 12px;
    }
    
    .action-btn {
        padding: 6px 12px;
        border: none;
        border-radius: 8px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        font-size: 12px;
        display: inline-flex;
        align-items: center;
        gap: 5px;
        margin: 2px;
    }
    
    .btn-approve {
        background: linear-gradient(135deg, #10b981, #059669);
        color: white;
    }
    
    .btn-reject {
        background: linear-gradient(135deg, #ef4444, #dc2626);
        color: white;
    }
    
    .btn-view {
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
    }
    
    .action-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(0,0,0,0.3);
    }
    
    .status-pending {
        background: linear-gradient(135deg, #fef3c7, #fde68a);
        color: #92400e;
        animation: pulse 2s ease-in-out infinite;
    }
    
    .status-approved {
        background: linear-gradient(135deg, #d1fae5, #a7f3d0);
        color: #065f46;
        animation: bounce 2s ease-in-out infinite;
    }
    
    .status-rejected {
        background: linear-gradient(135deg, #fee2e2, #fecaca);
        color: #991b1b;
        animation: shake 2s ease-in-out infinite;
    }
    
    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.05); }
    }
    
    @keyframes bounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-5px); }
    }
    
    @keyframes shake {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-3px); }
        75% { transform: translateX(3px); }
    }
    
    .filter-section {
        background: var(--card-bg);
        padding: 20px;
        border-radius: 15px;
        margin-bottom: 20px;
        border: 1px solid var(--border-color);
        display: flex;
        gap: 15px;
        align-items: center;
        flex-wrap: wrap;
    }
    
    .filter-input {
        padding: 10px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: #1f2937;
        color: #e5e7eb;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
        font-size: 14px;
        transition: all 0.3s ease;
    }
    
    .filter-input option {
        background: #1f2937;
        color: #e5e7eb;
        padding: 10px;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
    }
    
    .filter-input:focus {
        outline: none;
        border-color: #667eea;
        box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }
    
    .filter-btn {
        padding: 10px 20px;
        border: none;
        border-radius: 10px;
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
        font-family: 'Cairo', sans-serif;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .filter-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
    }
</style>

<div class="admin-container">
    <div class="admin-header">
        <div class="admin-title">
            <i class="fas fa-tasks fa-fade" style="--fa-animation-duration: 3s; color: #667eea;"></i>
            إدارة طلبات السحب
        </div>
    </div>
    
    <div class="filter-section">
        <label style="color: var(--text-primary); font-weight: 700;">
            <i class="fas fa-filter fa-fade" style="--fa-animation-duration: 3s;"></i> تصفية:
        </label>
        
        <select id="filter-status" class="filter-input" style="cursor: pointer;">
            <option value="all">جميع الحالات</option>
            <option value="pending">قيد المعالجة</option>
            <option value="approved">مقبول</option>
            <option value="rejected">مرفوض</option>
        </select>
        
        <input type="date" id="filter-from" class="filter-input" placeholder="من تاريخ">
        <input type="date" id="filter-to" class="filter-input" placeholder="إلى تاريخ">
        
        <button class="filter-btn" onclick="applyFilters()">
            <i class="fas fa-search"></i> بحث
        </button>
        <button class="filter-btn" onclick="clearFilter()" style="background: linear-gradient(135deg, #6b7280, #4b5563);">
            <i class="fas fa-times"></i> إعادة تعيين
        </button>
    </div>
    
    <div class="stats-row">
        <div class="stat-card">
            <div class="stat-value" style="color: #f59e0b;" id="pending-count">0</div>
            <div class="stat-label">
                <i class="fas fa-clock fa-spin"></i> قيد المعالجة
            </div>
            <div style="margin-top: 10px; font-size: 16px; font-weight: 700; color: #f59e0b;" id="pending-amount">0 جنيه</div>
        </div>
        <div class="stat-card">
            <div class="stat-value" style="color: #10b981;" id="approved-count">0</div>
            <div class="stat-label">
                <i class="fas fa-check-circle fa-beat"></i> مقبول
            </div>
            <div style="margin-top: 10px; font-size: 16px; font-weight: 700; color: #10b981;" id="approved-amount">0 جنيه</div>
        </div>
        <div class="stat-card">
            <div class="stat-value" style="color: #ef4444;" id="rejected-count">0</div>
            <div class="stat-label">
                <i class="fas fa-times-circle fa-shake"></i> مرفوض
            </div>
            <div style="margin-top: 10px; font-size: 16px; font-weight: 700; color: #ef4444;" id="rejected-amount">0 جنيه</div>
        </div>
        <div class="stat-card">
            <div class="stat-value" style="color: #667eea;" id="total-amount">0</div>
            <div class="stat-label">
                <i class="fas fa-money-bill-wave fa-fade"></i> إجمالي المبالغ
            </div>
        </div>
    </div>
    
    <div class="admin-card">
        <div class="table-responsive">
            <table class="admin-table">
                <thead>
                    <tr>
                        <th>#</th>
                        <th>المستخدم</th>
                        <th>البريد/الهاتف</th>
                        <th>المبلغ</th>
                        <th>الطريقة</th>
                        <th>الحالة</th>
                        <th>التاريخ</th>
                        <th>الإجراءات</th>
                    </tr>
                </thead>
                <tbody id="withdrawals-body">
                    <tr><td colspan="8" style="padding: 40px; text-align: center;">جاري التحميل...</td></tr>
                </tbody>
            </table>
        </div>
    </div>
</div>

<script>
let allWithdrawals = [];

document.addEventListener('DOMContentLoaded', function() {
    loadWithdrawals();
});

function loadWithdrawals() {
    fetch('api/get_all_withdrawals.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            allWithdrawals = data.withdrawals;
            renderWithdrawals(allWithdrawals);
            updateStats(allWithdrawals);
        }
    })
    .catch(error => console.error('Error:', error));
}

function applyFilters() {
    const status = document.getElementById('filter-status').value;
    const fromDate = document.getElementById('filter-from').value;
    const toDate = document.getElementById('filter-to').value;
    
    let filtered = allWithdrawals;
    
    // Filter by status
    if (status !== 'all') {
        filtered = filtered.filter(w => w.status === status);
    }
    
    // Filter by date
    if (fromDate || toDate) {
        filtered = filtered.filter(w => {
            const wDate = new Date(w.created_at);
            const from = fromDate ? new Date(fromDate) : new Date('1900-01-01');
            const to = toDate ? new Date(toDate) : new Date('2100-12-31');
            
            // Set time to end of day for 'to' date
            to.setHours(23, 59, 59, 999);
            
            return wDate >= from && wDate <= to;
        });
    }
    
    renderWithdrawals(filtered);
    updateStats(filtered);
    
    Swal.fire({
        icon: 'success',
        title: 'تم!',
        text: `تم العثور على ${filtered.length} طلب`,
        timer: 1500,
        showConfirmButton: false,
        background: '#111827',
        color: '#e5e7eb'
    });
}

function clearFilter() {
    document.getElementById('filter-status').value = 'all';
    document.getElementById('filter-from').value = '';
    document.getElementById('filter-to').value = '';
    renderWithdrawals(allWithdrawals);
    updateStats(allWithdrawals);
    
    Swal.fire({
        icon: 'success',
        title: 'تم!',
        text: 'تم إعادة تعيين الفلتر',
        timer: 1500,
        showConfirmButton: false,
        background: '#111827',
        color: '#e5e7eb'
    });
}

function updateStats(withdrawals) {
    const pending = withdrawals.filter(w => w.status === 'pending').length;
    const approved = withdrawals.filter(w => w.status === 'approved').length;
    const rejected = withdrawals.filter(w => w.status === 'rejected').length;
    
    const pendingAmount = withdrawals.filter(w => w.status === 'pending').reduce((sum, w) => sum + parseFloat(w.amount), 0);
    const approvedAmount = withdrawals.filter(w => w.status === 'approved').reduce((sum, w) => sum + parseFloat(w.amount), 0);
    const rejectedAmount = withdrawals.filter(w => w.status === 'rejected').reduce((sum, w) => sum + parseFloat(w.amount), 0);
    const totalAmount = withdrawals.reduce((sum, w) => sum + parseFloat(w.amount), 0);
    
    document.getElementById('pending-count').textContent = pending;
    document.getElementById('approved-count').textContent = approved;
    document.getElementById('rejected-count').textContent = rejected;
    document.getElementById('total-amount').textContent = totalAmount.toFixed(2);
    
    document.getElementById('pending-amount').textContent = pendingAmount.toFixed(2) + ' جنيه';
    document.getElementById('approved-amount').textContent = approvedAmount.toFixed(2) + ' جنيه';
    document.getElementById('rejected-amount').textContent = rejectedAmount.toFixed(2) + ' جنيه';
}

function renderWithdrawals(withdrawals) {
    const tbody = document.getElementById('withdrawals-body');
    
    if (withdrawals.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" style="padding: 40px;">لا توجد طلبات</td></tr>';
        return;
    }
    
    tbody.innerHTML = withdrawals.map((w, i) => {
        const statusClass = {
            'pending': 'status-pending',
            'approved': 'status-approved',
            'rejected': 'status-rejected'
        }[w.status];
        
        const statusIcon = {
            'pending': 'fa-clock fa-spin',
            'approved': 'fa-check-circle fa-beat',
            'rejected': 'fa-times-circle fa-shake'
        }[w.status];
        
        const statusText = {
            'pending': 'جاري المعالجة',
            'approved': 'مقبول',
            'rejected': 'مرفوض'
        }[w.status];
        
        let actions = '';
        if (w.status === 'pending') {
            actions = `
                <button class="action-btn btn-approve" onclick="updateStatus(${w.id}, 'approved')">
                    <i class="fas fa-check"></i> قبول
                </button>
                <button class="action-btn btn-reject" onclick="updateStatus(${w.id}, 'rejected')">
                    <i class="fas fa-times"></i> رفض
                </button>
            `;
        } else if (w.status === 'rejected') {
            actions = `
                <button class="action-btn btn-approve" onclick="updateStatus(${w.id}, 'approved')">
                    <i class="fas fa-check"></i> قبول
                </button>
                <button class="action-btn btn-reject" onclick="updateStatus(${w.id}, 'pending')" style="background: linear-gradient(135deg, #f59e0b, #d97706);">
                    <i class="fas fa-undo"></i> إعادة للمعالجة
                </button>
            `;
        } else if (w.status === 'approved') {
            actions = '<span style="color: var(--text-secondary); font-size: 12px;">تم القبول ✅</span>';
        }
        
        return `
            <tr>
                <td><strong>${i + 1}</strong></td>
                <td>${w.first_name || w.user_id}</td>
                <td style="font-size: 11px;">
                    ${w.email || ''}<br>
                    ${w.phone || ''}
                </td>
                <td style="color: #10b981; font-weight: 700;">${parseFloat(w.amount).toFixed(2)}</td>
                <td>${w.withdrawal_type}</td>
                <td>
                    <span class="status-badge ${statusClass}">
                        <i class="fas ${statusIcon}"></i>
                        ${statusText}
                    </span>
                </td>
                <td style="font-size: 11px;">${new Date(w.created_at).toLocaleDateString('ar-EG')}</td>
                <td>
                    <button class="action-btn btn-view" onclick="viewDetails(${i})">
                        <i class="fas fa-eye"></i> عرض
                    </button>
                    ${actions}
                </td>
            </tr>
        `;
    }).join('');
    
    window.withdrawalsData = withdrawals;
}

function updateStatus(id, status) {
    const statusText = status === 'approved' ? 'قبول' : 'رفض';
    
    Swal.fire({
        title: 'تأكيد ' + statusText,
        text: 'هل أنت متأكد من ' + statusText + ' هذا الطلب؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، ' + statusText,
        cancelButtonText: 'إلغاء',
        confirmButtonColor: status === 'approved' ? '#10b981' : '#ef4444',
        background: '#111827',
        color: '#e5e7eb'
    }).then((result) => {
        if (result.isConfirmed) {
            Swal.fire({
                title: 'جاري المعالجة...',
                allowOutsideClick: false,
                didOpen: () => Swal.showLoading()
            });
            
            fetch('api/manage_withdrawal.php', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    action: 'update_status',
                    withdrawal_id: id,
                    status: status
                })
            })
            .then(res => res.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم!',
                        text: data.message,
                        timer: 2000,
                        showConfirmButton: false,
                        background: '#111827',
                        color: '#e5e7eb'
                    });
                    loadWithdrawals();
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        background: '#111827',
                        color: '#e5e7eb',
                        confirmButtonColor: '#667eea'
                    });
                }
            });
        }
    });
}

function viewDetails(index) {
    const w = window.withdrawalsData[index];
    
    Swal.fire({
        title: 'تفاصيل الطلب',
        html: `
            <div style="text-align: right; font-family: 'Cairo', sans-serif;">
                <div style="margin-bottom: 15px; padding: 15px; background: #1f2937; border-radius: 10px;">
                    <p style="color: #9ca3af; font-size: 12px; margin-bottom: 5px;">المستخدم</p>
                    <p style="color: #e5e7eb; font-weight: 600; margin: 0;">${w.first_name || w.user_id}</p>
                    <p style="color: #9ca3af; font-size: 11px; margin: 0;">${w.email || ''}</p>
                </div>
                
                <div style="margin-bottom: 15px; padding: 15px; background: #1f2937; border-radius: 10px;">
                    <p style="color: #9ca3af; font-size: 12px; margin-bottom: 5px;">المبلغ</p>
                    <p style="color: #10b981; font-weight: 700; font-size: 20px; margin: 0;">${parseFloat(w.amount).toFixed(2)} جنيه</p>
                </div>
                
                <div style="margin-bottom: 15px; padding: 15px; background: #1f2937; border-radius: 10px;">
                    <p style="color: #9ca3af; font-size: 12px; margin-bottom: 5px;">طريقة السحب</p>
                    <p style="color: #e5e7eb; font-weight: 600; margin: 0;">${w.withdrawal_type}</p>
                </div>
                
                <div style="padding: 15px; background: #1f2937; border-radius: 10px;">
                    <p style="color: #9ca3af; font-size: 12px; margin-bottom: 5px;">بيانات التحويل</p>
                    <p style="color: #e5e7eb; font-size: 13px; margin: 0; white-space: pre-line;">${w.withdrawal_details}</p>
                </div>
            </div>
        `,
        confirmButtonText: 'إغلاق',
        confirmButtonColor: '#667eea',
        background: '#111827',
        color: '#e5e7eb'
    });
}
</script>

<?php include 'includes/admin_footer.php'; ?>
