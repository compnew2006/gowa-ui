// Wallet data
let walletData = null;
let transactions = [];

// Load data on page load
document.addEventListener('DOMContentLoaded', function() {
  loadWallet();
  loadTransactions();
});

// Load wallet info
function loadWallet() {
  fetch('api/wallet_api.php?action=get_wallet')
    .then(response => response.json())
    .then(data => {
      if(data.success) {
        walletData = data.wallet;
        document.getElementById('balance').textContent = parseFloat(data.wallet.balance).toFixed(2);
        document.getElementById('points').textContent = data.wallet.points;
        document.getElementById('total-transactions').textContent = data.total_transactions;
        updateMaxAmount();
      } else {
        console.error('Error loading wallet:', data.message);
      }
    })
    .catch(error => {
      console.error('Error loading wallet:', error);
    });
}

// Load transactions
function loadTransactions(dateFilter = '', monthFilter = '') {
  let url = 'api/wallet_api.php?action=get_transactions';
  
  if(dateFilter) {
    url += '&date=' + dateFilter;
  }
  
  if(monthFilter) {
    url += '&month=' + monthFilter;
  }
  
  fetch(url)
    .then(response => response.json())
    .then(data => {
      if(data.success) {
        transactions = data.transactions;
        renderTransactions();
      } else {
        console.error('Error loading transactions:', data.message);
      }
    })
    .catch(error => {
      console.error('Error loading transactions:', error);
    });
}

// Render transactions
// Render transactions
function renderTransactions() {
  const tbody = document.getElementById('transactions-body');
  
  // لو مفيش معاملات
  if (transactions.length === 0) {
    tbody.innerHTML = `
      <tr class="empty-row">
        <td colspan="6" style="text-align: center; padding: 3rem;">
          <i class="fas fa-inbox fa-beat-fade" style="font-size: 3rem; color: var(--text-gray); opacity: 0.3;"></i>
          <p style="color: var(--text-gray); margin-top: 1rem;">لا توجد معاملات حتى الآن</p>
        </td>
      </tr>
    `;
    return;
  }

  // تفريغ الجدول
  tbody.innerHTML = '';

  // إنشاء الصفوف
  transactions.forEach((transaction, index) => {
    const date = new Date(transaction.created_at);
    const formattedDate = date.toLocaleDateString('ar-EG', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });

    const person = transaction.transaction_type === 'send'
      ? transaction.to_email
      : transaction.from_email;

    const amount = transaction.amount_type === 'money'
      ? parseFloat(transaction.amount).toFixed(2)
      : transaction.amount;

    // شارة نوع المعاملة
    const transactionBadge =
      transaction.transaction_type === 'send'
        ? '<i class="fas fa-arrow-up" style="color: #dc2626;"></i> تحويل'
        : '<i class="fas fa-arrow-down" style="color: #059669;"></i> استقبال';

    // شارة نوع الرصيد
    const typeBadge =
      transaction.amount_type === 'money'
        ? '<i class="fas fa-money-bill-wave" style="color: #059669;"></i> رصيد'
        : '<i class="fas fa-star" style="color: #d97706;"></i> نقاط';

    // إضافة الصف للجدول
    tbody.insertAdjacentHTML('beforeend', `
      <tr>
        <td>${index + 1}</td>
        <td><i class="fas fa-user-circle" style="color: #667eea;"></i> ${person}</td>
        <td>${transactionBadge}</td>
        <td>${typeBadge}</td>
        <td><strong>${amount}</strong></td>
        <td><i class="fas fa-clock" style="color: var(--text-gray);"></i> ${formattedDate}</td>
      </tr>
    `);
  });
}

// Apply filters
function applyFilters() {
  const dateFilter = document.getElementById('date-filter').value;
  const monthFilter = document.getElementById('month-filter').value;
  
  loadTransactions(dateFilter, monthFilter);
}

// Clear filters
function clearFilters() {
  document.getElementById('date-filter').value = '';
  document.getElementById('month-filter').value = '';
  loadTransactions();
}

// Open transfer modal
function openTransferModal() {
  document.getElementById('transfer-modal').classList.add('active');
  document.getElementById('to-email').value = '';
  document.getElementById('amount').value = '';
  document.getElementById('transfer-password').value = '';
  document.querySelector('input[name="amount-type"][value="money"]').checked = true;
  updateMaxAmount();
}

// Close transfer modal
function closeTransferModal() {
  document.getElementById('transfer-modal').classList.remove('active');
}

// Update max amount based on selected type
function updateMaxAmount() {
  if(!walletData) return;
  
  const amountType = document.querySelector('input[name="amount-type"]:checked').value;
  const maxAmountEl = document.getElementById('max-amount');
  
  if(amountType === 'money') {
    maxAmountEl.textContent = parseFloat(walletData.balance).toFixed(2) + ' جنيه';
  } else {
    maxAmountEl.textContent = walletData.points + ' نقطة';
  }
}

// Process transfer
function processTransfer() {
  const toEmail = document.getElementById('to-email').value.trim();
  const amountType = document.querySelector('input[name="amount-type"]:checked').value;
  const amount = parseFloat(document.getElementById('amount').value);
  const password = document.getElementById('transfer-password').value;
  
  // Validate email
  if(!toEmail || !toEmail.includes('@')) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'يرجى إدخال بريد إلكتروني صحيح',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    return;
  }
  
  // Validate amount
  if(!amount || amount <= 0) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'يرجى إدخال مبلغ صحيح',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    return;
  }
  
  // Check balance
  if(amountType === 'money' && amount > parseFloat(walletData.balance)) {
    Swal.fire({
      icon: 'error',
      title: 'رصيد غير كافي',
      text: 'المبلغ المطلوب أكبر من رصيدك الحالي',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    return;
  }
  
  if(amountType === 'points' && amount > walletData.points) {
    Swal.fire({
      icon: 'error',
      title: 'نقاط غير كافية',
      text: 'عدد النقاط المطلوب أكبر من نقاطك الحالية',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    return;
  }
  
  // Validate password
  if(!password) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'يرجى إدخال كلمة المرور للتأكيد',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    return;
  }
  
  // Show loading
  Swal.fire({
    title: 'جاري التحويل...',
    text: 'يرجى الانتظار',
    allowOutsideClick: false,
    didOpen: () => {
      Swal.showLoading();
    },
    customClass: {
      container: 'swal-over-modal'
    }
  });
  
  // Send transfer request
  const formData = new FormData();
  formData.append('action', 'transfer');
  formData.append('to_email', toEmail);
  formData.append('amount_type', amountType);
  formData.append('amount', amount);
  formData.append('password', password);
  
  fetch('api/wallet_api.php', {
    method: 'POST',
    body: formData
  })
  .then(response => response.json())
  .then(data => {
    Swal.close();
    
    if(data.success) {
      closeTransferModal();
      loadWallet();
      loadTransactions();
      
      Swal.fire({
        icon: 'success',
        title: 'تم التحويل بنجاح!',
        text: data.message,
        timer: 2000,
        showConfirmButton: false,
        timerProgressBar: true
      });
    } else {
      Swal.fire({
        icon: 'error',
        title: 'فشل التحويل',
        text: data.message,
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea',
        customClass: {
          container: 'swal-over-modal'
        }
      });
    }
  })
  .catch(error => {
    Swal.close();
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ أثناء التحويل',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    console.error('Transfer error:', error);
  });
}

// Close modal on outside click
window.onclick = function(event) {
  const modal = document.getElementById('transfer-modal');
  if (event.target === modal) {
    closeTransferModal();
  }
}
