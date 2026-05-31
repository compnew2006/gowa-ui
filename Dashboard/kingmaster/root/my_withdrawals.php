


<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "طلبات السحب | Kingmaster";
$page_css = ['https://kingmaster.info/css/account.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
   
$commission = getcommission_walletsById($user_id);


?>




<div class="withdrawals-container">
    <div class="withdrawals-header">
        <div class="withdrawals-title">
            <i class="fas fa-history fa-spin" style="--fa-animation-duration: 3s; color: #667eea;"></i>
            طلبات السحب
        </div>
    </div>
    
    <div class="withdrawals-card">
        <div class="table-responsive">
            <table class="withdrawals-table">
                <thead>
                    <tr>
                        <th><i class="fas fa-hashtag"></i> #</th>
                        <th><i class="fas fa-calendar"></i> التاريخ</th>
                        <th><i class="fas fa-exchange-alt"></i> طريقة السحب</th>
                        <th><i class="fas fa-money-bill-wave"></i> المبلغ</th>
                        <th><i class="fas fa-info-circle"></i> الحالة</th>
                        <th><i class="fas fa-eye"></i> التفاصيل</th>
                    </tr>
                </thead>
                <tbody id="withdrawals-body">
                    <tr class="empty-row">
                        <td colspan="6" class="empty-state  my_withdrawals">
                            <i class="fas fa-inbox fa-beat-fade"></i>
                            <p style="font-weight: 600; margin-bottom: 5px;">لا توجد طلبات سحب</p>
                            <p style="font-size: 14px;">لم تقم بإنشاء أي طلبات سحب حتى الآن</p>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
</div>

<script>
// Load withdrawals on page load
document.addEventListener('DOMContentLoaded', function() {
    loadWithdrawals();
});

function loadWithdrawals() {
    fetch('api/get_withdrawals.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            renderWithdrawals(data.withdrawals);
        } else {
            console.error('Error:', data.message);
        }
    })
    .catch(error => {
        console.error('Error loading withdrawals:', error);
    });
}

function renderWithdrawals(withdrawals) {
    const tbody = document.getElementById('withdrawals-body');
    
    if (withdrawals.length === 0) {
        tbody.innerHTML = `
            <tr class="empty-row">
                <td colspan="6" class="empty-state">
                    <i class="fas fa-inbox fa-beat-fade"></i>
                    <p style="font-weight: 600; margin-bottom: 5px;">لا توجد طلبات سحب</p>
                    <p style="font-size: 14px;">لم تقم بإنشاء أي طلبات سحب حتى الآن</p>
                </td>
            </tr>
        `;
        return;
    }
    
    tbody.innerHTML = withdrawals.map((item, index) => {
        const date = new Date(item.created_at);
        const formattedDate = date.toLocaleDateString('ar-EG', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
        
        let statusBadge = '';
        let statusIcon = '';
        
        if (item.status === 'pending') {
            statusBadge = 'status-pending';
            statusIcon = 'fa-clock fa-spin';
        } else if (item.status === 'approved') {
            statusBadge = 'status-approved';
            statusIcon = 'fa-check-circle fa-beat';
        } else if (item.status === 'rejected') {
            statusBadge = 'status-rejected';
            statusIcon = 'fa-times-circle fa-shake';
        }
        
        const statusText = {
            'pending': 'جاري المعالجة',
            'approved': 'تم السحب',
            'rejected': 'مرفوض'
        }[item.status] || item.status;
        
        return `
            <tr>
                <td><strong>${index + 1}</strong></td>
                <td>
                    <i class="fas fa-calendar-alt" style="color: #667eea; margin-left: 5px;"></i>
                    ${formattedDate}
                </td>
                <td>
                    <i class="fas fa-exchange-alt" style="color: #10b981; margin-left: 5px;"></i>
                    ${item.withdrawal_type}
                </td>
                <td class="amount-cell">
                    <i class="fas fa-coins fa-fade" style="margin-left: 5px;"></i>
                    ${parseFloat(item.amount).toFixed(2)} جنيه
                </td>
                <td>
                    <span class="status-badge ${statusBadge}">
                        <i class="fas ${statusIcon}"></i>
                        ${statusText}
                    </span>
                </td>
                <td>
                    <button class="details-btn" onclick="showDetails(${index})">
                        <i class="fas fa-eye fa-fade"></i>
                        عرض
                    </button>
                </td>
            </tr>
        `;
    }).join('');
    
    // Store withdrawals globally for details view
    window.withdrawalsData = withdrawals;
}

function showDetails(index) {
    const withdrawal = window.withdrawalsData[index];
    
    const statusText = {
        'pending': 'جاري المعالجة',
        'approved': 'تم السحب',
        'rejected': 'مرفوض'
    }[withdrawal.status] || withdrawal.status;
    
    const statusColor = {
        'pending': '#f59e0b',
        'approved': '#10b981',
        'rejected': '#ef4444'
    }[withdrawal.status];
    
    Swal.fire({
        title: 'تفاصيل طلب السحب',
        html: `
            <div style="text-align: right; font-family: 'Cairo', sans-serif;">
                <div style="margin-bottom: 20px; padding: 20px; background: #1f2937; border-radius: 12px;">
                    <p style="color: #9ca3af; font-size: 14px; margin-bottom: 8px;">
                        <i class="fas fa-hashtag" style="color: #667eea;"></i> رقم الطلب
                    </p>
                    <p style="color: #e5e7eb; font-weight: 700; font-size: 18px; margin: 0;">#${withdrawal.id}</p>
                </div>
                
                <div style="margin-bottom: 20px; padding: 20px; background: #1f2937; border-radius: 12px;">
                    <p style="color: #9ca3af; font-size: 14px; margin-bottom: 8px;">
                        <i class="fas fa-money-bill-wave fa-fade" style="color: #10b981;"></i> المبلغ
                    </p>
                    <p style="color: #10b981; font-weight: 700; font-size: 24px; margin: 0;">${parseFloat(withdrawal.amount).toFixed(2)} جنيه</p>
                </div>
                
                <div style="margin-bottom: 20px; padding: 20px; background: #1f2937; border-radius: 12px;">
                    <p style="color: #9ca3af; font-size: 14px; margin-bottom: 8px;">
                        <i class="fas fa-exchange-alt" style="color: #667eea;"></i> طريقة السحب
                    </p>
                    <p style="color: #e5e7eb; font-weight: 600; font-size: 16px; margin: 0;">${withdrawal.withdrawal_type}</p>
                </div>
                
                <div style="margin-bottom: 20px; padding: 20px; background: #1f2937; border-radius: 12px;">
                    <p style="color: #9ca3af; font-size: 14px; margin-bottom: 8px;">
                        <i class="fas fa-info-circle" style="color: #667eea;"></i> بيانات التحويل
                    </p>
                    <p style="color: #e5e7eb; font-weight: 500; font-size: 14px; margin: 0; white-space: pre-line;">${withdrawal.withdrawal_details}</p>
                </div>
                
                <div style="margin-bottom: 20px; padding: 20px; background: #1f2937; border-radius: 12px;">
                    <p style="color: #9ca3af; font-size: 14px; margin-bottom: 8px;">
                        <i class="fas fa-flag" style="color: ${statusColor};"></i> الحالة
                    </p>
                    <p style="color: ${statusColor}; font-weight: 700; font-size: 18px; margin: 0;">${statusText}</p>
                </div>
                
                <div style="padding: 20px; background: #1f2937; border-radius: 12px;">
                    <p style="color: #9ca3af; font-size: 14px; margin-bottom: 8px;">
                        <i class="fas fa-calendar-alt fa-fade" style="color: #667eea;"></i> تاريخ الطلب
                    </p>
                    <p style="color: #e5e7eb; font-weight: 600; font-size: 14px; margin: 0;">${new Date(withdrawal.created_at).toLocaleString('ar-EG')}</p>
                </div>
            </div>
        `,
        confirmButtonText: '<i class="fas fa-times"></i> إغلاق',
        confirmButtonColor: '#667eea',
        background: '#111827',
        color: '#e5e7eb',
        width: '500px'
    });
}
</script>

<?php include 'includes/footer.php'; ?>
