 

<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "المحفظة | Kingmaster";
$page_css = ['/css/account.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
   
$commission = getcommission_walletsById($user_id);


?>


<!-- Main Content -->
<main class="main-content">
  <!-- Wallet Summary -->
  <div class="wallet-summary">
    <!-- Balance Card -->
    <div class="wallet-card balance-card">
      <div class="wallet-icon">
        <i class="fas fa-money-bill-wave" style="color: #10b981;"></i>
      </div>
      <div class="wallet-info">
        <p class="wallet-label">الرصيد</p>
        <h2 class="wallet-value" id="balance">0.00</h2>
        <span class="wallet-unit">جنيه</span>
      </div>
    </div>

    <!-- Points Card -->
    <div class="wallet-card points-card">
      <div class="wallet-icon">
        <i class="fas fa-star" style=" color: #f59e0b;"></i>
      </div>
      <div class="wallet-info">
        <p class="wallet-label">النقاط</p>
        <h2 class="wallet-value" id="points">0</h2>
        <span class="wallet-unit">نقطة</span>
      </div>
    </div>



   <!-- commission Card -->
    <div class="wallet-card points-card">
      <div class="wallet-icon">
        <i class="fa-brands fa-bitcoin" style="color: #2fc624;"></i>
      </div>
      <div class="wallet-info">
        <p class="wallet-label">العمولات</p>
        <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 8px;">
          <h2 class="wallet-value" id="commission-value" style="margin: 0;">
            <span id="commission-display">••••••</span>
            <span class="commission-real" style="display: none;"><?php echo $commission['commission']; ?></span>
          </h2>
          <button onclick="toggleCommission()" class="toggle-eye-btn" title="إظهار/إخفاء">
            <i class="fas fa-eye" id="eye-icon"></i>
          </button>
          <button onclick="withdrawCommission()" class="toggle-eye-btn" style="background: rgba(16, 185, 129, 0.1); border-color: rgba(16, 185, 129, 0.3); color: #10b981;" title="سحب الرصيد">
            <i class="fas fa-money-bill-wave"></i>
          </button>
        </div>
        <span class="wallet-unit">قابل للسحب</span>
      </div>
    </div>


    <!-- Total Transactions Card -->
    <div class="wallet-card transactions-card">
      <div class="wallet-icon">
        <i class="fas fa-exchange-alt" style=" color: #667eea;"></i>
      </div>
      <div class="wallet-info">
        <p class="wallet-label">إجمالي التحويلات</p>
        <h2 class="wallet-value" id="total-transactions">0</h2>
        <span class="wallet-unit">معاملة</span>
      </div>
    </div>
  </div>

  <!-- Actions & Filters -->
  <div class="content-card" style="margin-top: 2rem;">
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem; margin-bottom: 1.5rem;">
      <h3 style="margin: 0;">
        <i class="fas fa-history" style="color: #667eea;"></i> سجل المعاملات
      </h3>
      <div style="display: flex; gap: 10px; align-items: center;">
        <div style="position: relative;">
          <button onclick="window.location.href='my_withdrawals.php'" class="withdrawals-btn">
            <i class="fas fa-file-invoice-dollar" style=""></i>
            عرض طلبات السحب
          </button>
          <span id="pending-count-badge" class="pending-badge" style="display: none;">
            <i class="fas fa-clock" style=" font-size: 10px;"></i>
            <span id="pending-count">0</span>
          </span>
        </div>
        <button onclick="openTransferModal()" class="transfer-btn">
          <i class="fas fa-paper-plane"></i>
          تحويل جديد
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters-row">
      <div class="filter-group wallet-group">
        <label><i class="fas fa-calendar-day" style="color: #3b82f6;"></i> تاريخ محدد</label>
        <input type="date" id="date-filter" class="filter-input" onchange="applyFilters()">
      </div>
      <div class="filter-group wallet-group">
        <label><i class="fas fa-calendar" style="color: #8b5cf6;"></i> شهر محدد</label>
        <input type="month" id="month-filter" class="filter-input" onchange="applyFilters()">
      </div>
      <button onclick="clearFilters()" class="clear-btn">
        <i class="fas fa-times-circle"></i> مسح الفلاتر
      </button>
    </div>
  </div>

  <!-- Transactions Table -->
  <div class="content-card" style="margin-top: 1.5rem;">
    <div class="table-responsive">
      <table class="transactions-table" id="transactions-table">
        <thead>
          <tr>
            <th><i class="fas fa-hashtag"></i> #</th>
            <th><i class="fas fa-user"></i> الشخص</th>
            <th><i class="fas fa-exchange-alt"></i> نوع المعاملة</th>
            <th><i class="fas fa-coins"></i> النوع</th>
            <th><i class="fas fa-money-bill"></i> المبلغ</th>
            <th><i class="fas fa-clock"></i> الوقت</th>
          </tr>
        </thead>
        <tbody id="transactions-body">
          <tr class="empty-row">
            <td colspan="6" style="text-align: center; padding: 3rem;">
              <i class="fas fa-inbox" style="font-size: 3rem; color: var(--text-gray); opacity: 0.3;"></i>
              <p style="color: var(--text-gray); margin-top: 1rem;">لا توجد معاملات حتى الآن</p>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</main>

<!-- Transfer Modal -->
<div class="modal-w" id="transfer-modal">
  <div class="modal-content">
    <div class="modal-header">
      <h3>
        <i class="fas fa-paper-plane" style="color: #667eea;"></i>
        تحويل جديد
      </h3>
      <button class="close-btn" onclick="closeTransferModal()">
        <i class="fas fa-times" style="color: #ef4444;"></i>
      </button>
    </div>
    <div class="modal-body">
      <!-- Email Input -->
      <div class="form-group">
        <label>
          <i class="fas fa-envelope" style="color: #3b82f6;"></i> البريد الإلكتروني للمستلم
        </label>
        <input type="email" id="to-email" class="form-input" placeholder="example@email.com" />
      </div>

      <!-- Amount Type -->
      <div class="form-group">
        <label>
          <i class="fas fa-coins" style=" color: #f59e0b;"></i> نوع المبلغ
        </label>
        <div class="radio-group">
          <label class="radio-label">
            <input type="radio" name="amount-type" value="money" checked onchange="updateMaxAmount()">
            <span><i class="fas fa-money-bill-wave" style="color: #10b981;"></i> رصيد</span>
          </label>
          <label class="radio-label">
            <input type="radio" name="amount-type" value="points" onchange="updateMaxAmount()">
            <span><i class="fas fa-star" style="color: #f59e0b;"></i> نقاط</span>
          </label>
        </div>
      </div>

      <!-- Amount Input -->
      <div class="form-group">
        <label>
          <i class="fas fa-hand-holding-usd" style="color: #8b5cf6;"></i> المبلغ
        </label>
        <input type="number" id="amount" class="form-input" placeholder="0" min="1" step="0.01" />
        <p class="helper-text">
          <i class="fas fa-info-circle"></i> الحد الأقصى: <strong id="max-amount">0</strong>
        </p>
      </div>

      <!-- Password Input -->
      <div class="form-group">
        <label>
          <i class="fas fa-lock" style=" color: #ef4444;"></i> كلمة المرور للتأكيد
        </label>
        <input type="password" id="transfer-password" class="form-input" placeholder="••••••" />
        <p class="helper-text">
          <i class="fas fa-shield-alt"></i> استخدم كلمة مرور حسابك لتأكيد التحويل
        </p>
      </div>
    </div>
    <div class="modal-footer">
      <button class="btn-cancel" onclick="closeTransferModal()">إلغاء</button>
      <button class="btn-transfer" onclick="processTransfer()">
        <i class="fas fa-paper-plane"></i>
        تحويل الآن
      </button>
    </div>
  </div>
</div>


<script>
// Load pending withdrawals count on page load
document.addEventListener('DOMContentLoaded', function() {
  loadPendingWithdrawalsCount();
});

function loadPendingWithdrawalsCount() {
  fetch('api/get_withdrawals.php')
  .then(res => res.json())
  .then(data => {
    if (data.success) {
      const pendingCount = data.withdrawals.filter(w => w.status === 'pending').length;
      if (pendingCount > 0) {
        document.getElementById('pending-count').textContent = pendingCount;
        document.getElementById('pending-count-badge').style.display = 'inline-flex';
      }
    }
  })
  .catch(error => console.error('Error loading pending count:', error));
}

// Toggle commission visibility
let commissionVisible = false;

function toggleCommission() {
  // إذا كان مخفي ونريد إظهاره، طلب OTP
  if (!commissionVisible) {
    requestOTP('view');
  } else {
    // إخفاء الرقم
    hideCommission();
  }
}

function showCommission() {
  const displayElement = document.getElementById('commission-display');
  const realElement = document.querySelector('.commission-real');
  const eyeIcon = document.getElementById('eye-icon');
  
  commissionVisible = true;
  displayElement.style.display = 'none';
  realElement.style.display = 'inline';
  eyeIcon.classList.remove('fa-eye');
  eyeIcon.classList.add('fa-eye-slash');
}

function hideCommission() {
  const displayElement = document.getElementById('commission-display');
  const realElement = document.querySelector('.commission-real');
  const eyeIcon = document.getElementById('eye-icon');
  
  commissionVisible = false;
  displayElement.style.display = 'inline';
  realElement.style.display = 'none';
  eyeIcon.classList.remove('fa-eye-slash');
  eyeIcon.classList.add('fa-eye');
}

// Request OTP
function requestOTP(action) {
  Swal.fire({
    title: 'جاري إرسال رمز التحقق...',
    allowOutsideClick: false,
    didOpen: () => {
      Swal.showLoading();
    }
  });
  
  fetch('api/wallet_otp_api.php', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({action: 'send_otp'})
  })
  .then(res => res.json())
  .then(data => {
    if (data.success) {
      verifyOTP(action, data.phone_last_digits);
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
  })
  .catch(error => {
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ في إرسال رمز التحقق',
      background: '#111827',
      color: '#e5e7eb',
      confirmButtonColor: '#667eea'
    });
  });
}

// Verify OTP
function verifyOTP(action, phoneDigits) {
  Swal.fire({
    title: 'رمز التحقق',
    html: `
      <div style="text-align: center; margin: 20px 0;">
        <i class="fas fa-mobile-alt fa-3x" style="color: #667eea; margin-bottom: 15px;"></i>
        <p style="font-size: 16px; margin-bottom: 10px; color: #e5e7eb;">
          تم إرسال رمز التحقق إلى رقم هاتفك
        </p>
        <p style="font-size: 14px; color: #9ca3af; margin-bottom: 20px;">
          <i class="fas fa-phone" style="color: #10b981;"></i>
          ينتهي بـ ****${phoneDigits}
        </p>
      </div>
      <div style="text-align: center;">
        <input type="text" id="otp-input" class="swal2-input" 
          placeholder="أدخل الرمز المكون من 6 أرقام" 
          maxlength="6"
          style="width: 80%; font-size: 24px; letter-spacing: 8px; text-align: center; background: #1f2937; color: #e5e7eb; border: 2px solid #374151;">
      </div>
    `,
    showCancelButton: true,
    confirmButtonText: '<i class="fas fa-check"></i> تحقق',
    cancelButtonText: '<i class="fas fa-times"></i> إلغاء',
    confirmButtonColor: '#10b981',
    cancelButtonColor: '#6b7280',
    background: '#111827',
    color: '#e5e7eb',
    preConfirm: () => {
      const otp = document.getElementById('otp-input').value;
      if (!otp || otp.length !== 6) {
        Swal.showValidationMessage('يرجى إدخال رمز مكون من 6 أرقام');
        return false;
      }
      return otp;
    }
  }).then((result) => {
    if (result.isConfirmed) {
      // التحقق من OTP
      Swal.fire({
        title: 'جاري التحقق...',
        allowOutsideClick: false,
        didOpen: () => {
          Swal.showLoading();
        }
      });
      
      fetch('api/wallet_otp_api.php', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({
          action: 'verify_otp',
          otp: result.value
        })
      })
      .then(res => res.json())
      .then(data => {
        if (data.success) {
          // OTP صحيح
          if (action === 'view') {
            Swal.close();
            showCommission();
          } else if (action === 'withdraw') {
            Swal.close();
            showWithdrawModal();
          }
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
      })
      .catch(error => {
        Swal.fire({
          icon: 'error',
          title: 'خطأ',
          text: 'حدث خطأ في التحقق',
          background: '#111827',
          color: '#e5e7eb',
          confirmButtonColor: '#667eea'
        });
      });
    }
  });
}

// Withdraw commission
function withdrawCommission() {
  // طلب OTP للسحب
  requestOTP('withdraw');
}

function showWithdrawModal() {
  // إظهار الرقم إذا كان مخفي
  if (!commissionVisible) {
    showCommission();
  }
  
  const commissionAmount = parseFloat(document.querySelector('.commission-real').textContent);
  
  if (commissionAmount <= 0) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'لا يوجد رصيد للسحب',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      background: '#111827',
      color: '#e5e7eb'
    });
    return;
  }
  
  // عرض شاشة اختيار طريقة السحب
  Swal.fire({
    title: 'سحب العمولات',
    html: `
      <div style="text-align: center; margin: 15px 0;">
        <i class="fas fa-money-bill-wave fa-2x" style="color: #10b981; margin-bottom: 12px;"></i>
        <p style="font-size: 15px; font-weight: 600; margin-bottom: 8px; color: #e5e7eb;">
          <i class="fas fa-wallet" style="color: #667eea; margin-left: 5px;"></i>
          الرصيد المتاح للسحب
        </p>
        <h2 style="font-size: 26px; color: #10b981; font-weight: 800; margin: 0;">${commissionAmount.toFixed(2)} جنيه</h2>
      </div>
      <div style="text-align: center; margin-top: 15px;">
        <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-size: 14px; font-family: 'Cairo', sans-serif;">
          <i class="fas fa-hand-pointer" style="color: #667eea; margin-left: 5px;"></i>
          اختر طريقة السحب
        </label>
        <select id="withdraw-method" class="swal2-select" style="
          width: 80%; 
          background: #1f2937; 
          color: #e5e7eb; 
          border: 2px solid #374151; 
          padding: 10px 12px; 
          border-radius: 10px;
          font-size: 14px;
          font-family: 'Cairo', sans-serif;
          font-weight: 600;
          cursor: pointer;
          transition: all 0.3s ease;
          outline: none;
          margin: 0 auto;
        ">
          <option value="paypal" style="background: #1f2937; color: #e5e7eb; font-family: 'Cairo', sans-serif;">PayPal</option>
          <option value="vodafone" style="background: #1f2937; color: #e5e7eb; font-family: 'Cairo', sans-serif;">فودافون كاش</option>
          <option value="bank" style="background: #1f2937; color: #e5e7eb; font-family: 'Cairo', sans-serif;">تحويل بنكي</option>
        </select>
      </div>
    `,
    showCancelButton: true,
    confirmButtonText: '<i class="fas fa-arrow-left"></i> متابعة',
    cancelButtonText: '<i class="fas fa-times"></i> إلغاء',
    confirmButtonColor: '#10b981',
    cancelButtonColor: '#6b7280',
    background: '#111827',
    color: '#e5e7eb',
    width: '450px',
    preConfirm: () => {
      return document.getElementById('withdraw-method').value;
    }
  }).then((result) => {
    if (result.isConfirmed) {
      const method = result.value;
      if (method === 'paypal') {
        showPayPalForm(commissionAmount);
      } else if (method === 'vodafone') {
        showVodafoneForm(commissionAmount);
      } else if (method === 'bank') {
        showBankForm(commissionAmount);
      }
    }
  });
}

// PayPal Form
function showPayPalForm(maxAmount) {
  Swal.fire({
    title: 'PayPal',
    html: `
      <div style="text-align: right;">
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-dollar-sign" style="color: #10b981;"></i> المبلغ
          </label>
          <input type="number" id="paypal-amount" class="swal2-input" 
            placeholder="المبلغ" 
            max="${maxAmount}" 
            min="1" 
            step="0.01"
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
          <small style="color: #9ca3af; font-size: 12px;">الحد الأقصى: ${maxAmount.toFixed(2)} جنيه</small>
        </div>
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-envelope" style="color: #667eea;"></i> بريد PayPal
          </label>
          <input type="email" id="paypal-email" class="swal2-input" 
            placeholder="example@paypal.com"
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
        </div>
      </div>
    `,
    showCancelButton: true,
    confirmButtonText: '<i class="fas fa-paper-plane"></i> سحب',
    cancelButtonText: '<i class="fas fa-arrow-right"></i> رجوع',
    confirmButtonColor: '#10b981',
    cancelButtonColor: '#6b7280',
    background: '#111827',
    color: '#e5e7eb',
    preConfirm: () => {
      const amount = parseFloat(document.getElementById('paypal-amount').value);
      const email = document.getElementById('paypal-email').value;
      
      if (!amount || amount <= 0) {
        Swal.showValidationMessage('يرجى إدخال مبلغ صحيح');
        return false;
      }
      if (amount > maxAmount) {
        Swal.showValidationMessage('المبلغ أكبر من رصيدك المتاح');
        return false;
      }
      if (!email || !email.includes('@')) {
        Swal.showValidationMessage('يرجى إدخال بريد إلكتروني صحيح');
        return false;
      }
      
      return {amount, email};
    }
  }).then((result) => {
    if (result.isConfirmed) {
      processWithdrawal('PayPal', result.value.amount, `بريد PayPal: ${result.value.email}`);
    }
  });
}

// Vodafone Form
function showVodafoneForm(maxAmount) {
  Swal.fire({
    title: 'فودافون كاش',
    html: `
      <div style="text-align: right;">
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-dollar-sign" style="color: #10b981;"></i> المبلغ
          </label>
          <input type="number" id="vodafone-amount" class="swal2-input" 
            placeholder="المبلغ" 
            max="${maxAmount}" 
            min="1" 
            step="0.01"
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
          <small style="color: #9ca3af; font-size: 12px;">الحد الأقصى: ${maxAmount.toFixed(2)} جنيه</small>
        </div>
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-phone" style="color: #667eea;"></i> رقم فودافون كاش
          </label>
          <input type="tel" id="vodafone-phone" class="swal2-input" 
            placeholder="01XXXXXXXXX"
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
        </div>
      </div>
    `,
    showCancelButton: true,
    confirmButtonText: '<i class="fas fa-paper-plane"></i> سحب',
    cancelButtonText: '<i class="fas fa-arrow-right"></i> رجوع',
    confirmButtonColor: '#10b981',
    cancelButtonColor: '#6b7280',
    background: '#111827',
    color: '#e5e7eb',
    preConfirm: () => {
      const amount = parseFloat(document.getElementById('vodafone-amount').value);
      const phone = document.getElementById('vodafone-phone').value;
      
      if (!amount || amount <= 0) {
        Swal.showValidationMessage('يرجى إدخال مبلغ صحيح');
        return false;
      }
      if (amount > maxAmount) {
        Swal.showValidationMessage('المبلغ أكبر من رصيدك المتاح');
        return false;
      }
      if (!phone || phone.length < 11) {
        Swal.showValidationMessage('يرجى إدخال رقم صحيح');
        return false;
      }
      
      return {amount, phone};
    }
  }).then((result) => {
    if (result.isConfirmed) {
      processWithdrawal('فودافون كاش', result.value.amount, `رقم فودافون كاش: ${result.value.phone}`);
    }
  });
}

// Bank Form
function showBankForm(maxAmount) {
  Swal.fire({
    title: 'تحويل بنكي',
    html: `
      <div style="text-align: center;">
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 12px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif; font-size: 14px;">
            <i class="fas fa-university" style="color: #667eea; margin-left: 5px;"></i> نوع التحويل
          </label>
          <select id="bank-type" class="swal2-select" style="width: 80%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; padding: 10px 12px; border-radius: 10px; font-family: 'Cairo', sans-serif; font-weight: 600; margin: 0 auto; outline: none;">
            <option value="bank_account">حساب بنكي</option>
            <option value="instapay">إنستا باي</option>
          </select>
        </div>
      </div>
    `,
    showCancelButton: true,
    confirmButtonText: '<i class="fas fa-arrow-left"></i> متابعة',
    cancelButtonText: '<i class="fas fa-arrow-right"></i> رجوع',
    confirmButtonColor: '#10b981',
    cancelButtonColor: '#6b7280',
    background: '#111827',
    color: '#e5e7eb',
    width: '450px',
    preConfirm: () => {
      return document.getElementById('bank-type').value;
    }
  }).then((result) => {
    if (result.isConfirmed) {
      if (result.value === 'bank_account') {
        showBankAccountForm(maxAmount);
      } else {
        showInstaPayForm(maxAmount);
      }
    }
  });
}

// Bank Account Form
function showBankAccountForm(maxAmount) {
  Swal.fire({
    title: 'حساب بنكي',
    html: `
      <div style="text-align: right;">
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-dollar-sign" style="color: #10b981;"></i> المبلغ
          </label>
          <input type="number" id="bank-amount" class="swal2-input" 
            max="${maxAmount}" min="1" step="0.01"
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
          <small style="color: #9ca3af; font-size: 12px;">الحد الأقصى: ${maxAmount.toFixed(2)}</small>
        </div>
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-building" style="color: #667eea;"></i> اسم البنك
          </label>
          <input type="text" id="bank-name" class="swal2-input" 
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
        </div>
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-hashtag" style="color: #f59e0b;"></i> رقم الحساب
          </label>
          <input type="text" id="bank-account" class="swal2-input" 
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
        </div>
      </div>
    `,
    showCancelButton: true,
    confirmButtonText: '<i class="fas fa-paper-plane"></i> سحب',
    cancelButtonText: '<i class="fas fa-arrow-right"></i> رجوع',
    confirmButtonColor: '#10b981',
    cancelButtonColor: '#6b7280',
    background: '#111827',
    color: '#e5e7eb',
    preConfirm: () => {
      const amount = parseFloat(document.getElementById('bank-amount').value);
      const bankName = document.getElementById('bank-name').value;
      const accountNumber = document.getElementById('bank-account').value;
      
      if (!amount || amount <= 0 || amount > maxAmount) {
        Swal.showValidationMessage('يرجى إدخال مبلغ صحيح');
        return false;
      }
      if (!bankName || !accountNumber) {
        Swal.showValidationMessage('يرجى إدخال جميع البيانات');
        return false;
      }
      
      return {amount, bankName, accountNumber};
    }
  }).then((result) => {
    if (result.isConfirmed) {
      processWithdrawal('تحويل بنكي', result.value.amount, `البنك: ${result.value.bankName}\nرقم الحساب: ${result.value.accountNumber}`);
    }
  });
}

// InstaPay Form
function showInstaPayForm(maxAmount) {
  Swal.fire({
    title: 'إنستا باي',
    html: `
      <div style="text-align: right;">
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-dollar-sign" style="color: #10b981;"></i> المبلغ
          </label>
          <input type="number" id="instapay-amount" class="swal2-input" 
            max="${maxAmount}" min="1" step="0.01"
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
          <small style="color: #9ca3af; font-size: 12px;">الحد الأقصى: ${maxAmount.toFixed(2)}</small>
        </div>
        <div style="margin-bottom: 20px;">
          <label style="display: block; margin-bottom: 8px; font-weight: 600; color: #e5e7eb; font-family: 'Cairo', sans-serif;">
            <i class="fas fa-user" style="color: #667eea;"></i> اسم المستخدم
          </label>
          <input type="text" id="instapay-username" class="swal2-input" 
            placeholder="@username"
            style="width: 90%; background: #1f2937; color: #e5e7eb; border: 2px solid #374151; font-family: 'Cairo', sans-serif;">
        </div>
      </div>
    `,
    showCancelButton: true,
    confirmButtonText: '<i class="fas fa-paper-plane"></i> سحب',
    cancelButtonText: '<i class="fas fa-arrow-right"></i> رجوع',
    confirmButtonColor: '#10b981',
    cancelButtonColor: '#6b7280',
    background: '#111827',
    color: '#e5e7eb',
    preConfirm: () => {
      const amount = parseFloat(document.getElementById('instapay-amount').value);
      const username = document.getElementById('instapay-username').value;
      
      if (!amount || amount <= 0 || amount > maxAmount) {
        Swal.showValidationMessage('يرجى إدخال مبلغ صحيح');
        return false;
      }
      if (!username) {
        Swal.showValidationMessage('يرجى إدخال اسم المستخدم');
        return false;
      }
      
      return {amount, username};
    }
  }).then((result) => {
    if (result.isConfirmed) {
      processWithdrawal('إنستا باي', result.value.amount, `اسم المستخدم: ${result.value.username}`);
    }
  });
}

// Process Withdrawal
function processWithdrawal(type, amount, details) {
  Swal.fire({
    title: 'جاري المعالجة...',
    allowOutsideClick: false,
    didOpen: () => {
      Swal.showLoading();
    }
  });
  
  fetch('api/process_withdrawal.php', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({type, amount, details})
  })
  .then(res => res.json())
  .then(data => {
    if (data.success) {
      Swal.fire({
        icon: 'success',
        title: 'تم إرسال طلب السحب!',
        html: `
          <p style="color: #e5e7eb;">
            <i class="fas fa-check-circle" style="color: #10b981; margin-left: 5px;"></i>
            تم إرسال طلبك بنجاح
          </p>
          <p style="color: #9ca3af;">
            <i class="fas fa-money-bill-wave" style="color: #10b981; margin-left: 5px;"></i>
            المبلغ: <strong style="color: #10b981;">${amount.toFixed(2)} جنيه</strong>
          </p>
          <p style="color: #9ca3af;">
            <i class="fas fa-exchange-alt" style=" color: #667eea; margin-left: 5px;"></i>
            الطريقة: <strong style="color: #e5e7eb;">${type}</strong>
          </p>
          <p style="margin-top: 15px; font-size: 14px; color: #9ca3af;">
            <i class="fas fa-clock" style=" color: #f59e0b; margin-left: 5px;"></i>
            سيتم مراجعة طلبك والتحويل خلال 24 ساعة
          </p>
        `,
        timer: 3000,
        showConfirmButton: false,
        background: '#111827',
        color: '#e5e7eb'
      });
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
  })
  .catch(error => {
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ في معالجة الطلب',
      background: '#111827',
      color: '#e5e7eb',
      confirmButtonColor: '#667eea'
    });
  });
}
</script>

<script src="js/wallet.js"></script>

<?php include 'includes/footer.php'; ?>
