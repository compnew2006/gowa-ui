 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "تفعيل الكوبون | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>
<!-- Main Content -->
<main class="main-content">
  <div class="coupon-container">
    <div class="coupon-card">
      <div class="coupon-icon">
        <i class="fas fa-ticket-alt" style="color: #667eea; font-size: 4rem;"></i>
      </div>
      
      <h1 class="coupon-title">
        <i class="fas fa-gift" style="color: #f59e0b;"></i>
        تفعيل كوبون الخصم
      </h1>
      
      <p class="coupon-subtitle">أدخل رمز الكوبون الخاص بك للحصول على خصم فوري</p>
      
      <div class="coupon-input-group">
        <input 
          type="text" 
          id="coupon-code" 
          class="coupon-input" 
          placeholder="أدخل رمز الكوبون..." 
          maxlength="50"
          onkeypress="if(event.key === 'Enter') redeemCoupon()"
        />
        <button onclick="redeemCoupon()" class="redeem-btn">
          <i class="fas fa-check-circle"></i>
          تفعيل
        </button>
      </div>
      
      <div class="coupon-note">
        <i class="fas fa-info-circle" style="color: #3b82f6;"></i>
        <span>كل كوبون يمكن استخدامه مرة واحدة فقط لكل مستخدم</span>
      </div>
      
      <div class="coupon-examples">
        <p class="examples-title">
          <i class="fas fa-lightbulb" style="color: #f59e0b;"></i>
          جرب هذه الكوبونات:
        </p>
        <div class="examples-list">
          <span class="example-code" onclick="document.getElementById('coupon-code').value='WELCOME2024'">
            <i class="fas fa-tag"></i> WELCOME2024
          </span>
         
        </div>
      </div>
    </div>
  </div>
</main>

<!-- Confetti Canvas -->
<canvas id="confetti-canvas"></canvas>

<style>
  .main-content {
    min-height: calc(100vh - 80px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2rem;
    margin-top: 100px;
  }

  .coupon-container {
    width: 100%;
    max-width: 600px;
  }

  .coupon-card {
    background: var(--card-bg);
    border: 2px solid var(--border-color);
    border-radius: 24px;
    padding: 3rem;
    text-align: center;
    box-shadow: 0 10px 40px rgba(102, 126, 234, 0.2);
    transition: all 0.3s ease;
  }

  .coupon-card:hover {
    transform: translateY(-5px);
    box-shadow: 0 15px 50px rgba(102, 126, 234, 0.3);
  }

  .coupon-icon {
    margin-bottom: 2rem;
  }

  .coupon-title {
    font-size: 2rem;
    font-weight: 700;
    color: var(--text-light);
    margin-bottom: 1rem;
  }

  .coupon-subtitle {
    color: var(--text-gray);
    font-size: 1.1rem;
    margin-bottom: 2.5rem;
  }

  .coupon-input-group {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
  }

  .coupon-input {
    flex: 1;
    padding: 1.2rem 1.5rem;
    border: 2px solid var(--border-color);
    border-radius: 12px;
    background: rgba(102, 126, 234, 0.05);
    color: var(--text-light);
    font-size: 1.1rem;
    font-weight: 600;
    text-transform: uppercase;
    text-align: center;
    transition: all 0.3s ease;
  }

  .coupon-input:focus {
    outline: none;
    border-color: var(--primary-color);
    background: rgba(102, 126, 234, 0.1);
    box-shadow: 0 0 0 4px rgba(102, 126, 234, 0.1);
  }

  .redeem-btn {
    padding: 1.2rem 2rem;
    background: linear-gradient(135deg, #10b981, #059669);
    color: #fff;
    border: none;
    border-radius: 12px;
    font-weight: 700;
    font-size: 1.1rem;
    cursor: pointer;
    transition: all 0.3s ease;
    box-shadow: 0 4px 15px rgba(16, 185, 129, 0.4);
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .redeem-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(16, 185, 129, 0.6);
  }

  .coupon-note {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 1rem;
    background: rgba(59, 130, 246, 0.1);
    border: 1px solid rgba(59, 130, 246, 0.2);
    border-radius: 8px;
    color: var(--text-gray);
    font-size: 0.9rem;
    margin-bottom: 1.5rem;
  }

  .coupon-examples {
    background: rgba(102, 126, 234, 0.05);
    border: 1px solid rgba(102, 126, 234, 0.2);
    border-radius: 12px;
    padding: 1.5rem;
  }

  .examples-title {
    font-weight: 600;
    color: var(--text-light);
    margin-bottom: 1rem;
  }

  .examples-list {
    display: flex;
    gap: 1rem;
    justify-content: center;
    flex-wrap: wrap;
  }

  .example-code {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.8rem 1.2rem;
    background: rgba(102, 126, 234, 0.1);
    color: var(--primary-color);
    border: 1px solid rgba(102, 126, 234, 0.3);
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
  }

  .example-code:hover {
    background: rgba(102, 126, 234, 0.2);
    transform: scale(1.05);
  }

  /* Confetti Canvas */
  #confetti-canvas {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
    z-index: 9999;
  }

  /* Light Theme */
  body.light-theme .coupon-card {
    background: #ffffff;
    border: 2px solid #e5e7eb;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
  }

  body.light-theme .coupon-title {
    color: #2d3436;
  }

  /* Responsive */
  @media (max-width: 768px) {
    .coupon-card {
      padding: 2rem;
    }

    .coupon-input-group {
      flex-direction: column;
    }

    .coupon-title {
      font-size: 1.5rem;
    }
  }
</style>

<!-- Confetti Library -->
<script src="https://cdn.jsdelivr.net/npm/canvas-confetti@1.6.0/dist/confetti.browser.min.js"></script>

<script>
function redeemCoupon() {
  const code = document.getElementById('coupon-code').value.trim().toUpperCase();
  
  if(!code) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'يرجى إدخال رمز الكوبون',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    return;
  }
  
  // Show loading
  Swal.fire({
    title: 'جاري التحقق...',
    text: 'يرجى الانتظار',
    allowOutsideClick: false,
    didOpen: () => {
      Swal.showLoading();
    }
  });
  
  const formData = new FormData();
  formData.append('action', 'redeem');
  formData.append('code', code);
  
  fetch('api/coupon_api.php', {
    method: 'POST',
    body: formData
  })
  .then(response => response.json())
  .then(data => {
    Swal.close();
    
    if(data.success) {
      // Show success with confetti
      triggerConfetti();
      
      let discountText = '';
      let discountIcon = '';
      
      if(data.coupon.discount_type === 'percentage') {
        discountText = `خصم ${data.coupon.discount_value}%`;
        discountIcon = '<i class="fas fa-percent" style="color: #10b981;"></i>';
      } else if(data.coupon.discount_type === 'points') {
        discountText = `${data.coupon.discount_value} نقطة`;
        discountIcon = '<i class="fas fa-gem" style="color: #8b5cf6;"></i>';
      } else if(data.coupon.discount_type === 'money') {
        discountText = `${data.coupon.discount_value} جنيه`;
        discountIcon = '<i class="fas fa-coins" style="color: #f59e0b;"></i>';
      } else {
        discountText = `خصم ${data.coupon.discount_value} جنيه`;
        discountIcon = '<i class="fas fa-gift" style="color: #10b981;"></i>';
      }
      
      let successMessage = `
        <div style="font-size: 1.2rem; margin: 1rem 0;">
          ${discountIcon}
          <strong style="color: #10b981; font-size: 1.5rem; margin-right: 10px;">${discountText}</strong>
        </div>
        <p style="color: var(--text-gray); margin-top: 1rem;">
          تم تفعيل الكوبون <strong>${data.coupon.code}</strong> بنجاح!
        </p>
      `;
      
      if(data.coupon.discount_type === 'points') {
        successMessage += `
          <p style="color: #8b5cf6; font-size: 1.1rem; margin-top: 1rem; font-weight: 600;">
            <i class="fas fa-plus-circle"></i> تمت إضافة ${data.coupon.discount_value} نقطة إلى رصيدك!
          </p>
        `;
      }
      
      successMessage += `
        <p style="color: var(--text-gray); font-size: 0.9rem; margin-top: 0.5rem;">
          الاستخدامات المتبقية: ${data.coupon.remaining_uses}
        </p>
      `;
      
      Swal.fire({
        icon: 'success',
        title: '🎉 مبروك! 🎉',
        html: successMessage,
        confirmButtonText: 'رائع!',
        confirmButtonColor: '#10b981',
        timer: 5000,
        timerProgressBar: true
      });
      
      // Clear input
      document.getElementById('coupon-code').value = '';
      
    } else {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: data.message,
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea'
      });
    }
  })
  .catch(error => {
    Swal.close();
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ أثناء التحقق من الكوبون',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    console.error('Error:', error);
  });
}

// Confetti effect
function triggerConfetti() {
  const duration = 3000;
  const end = Date.now() + duration;

  const colors = ['#667eea', '#764ba2', '#f093fb', '#4facfe', '#00f2fe', '#43e97b', '#38f9d7'];

  (function frame() {
    confetti({
      particleCount: 7,
      angle: 60,
      spread: 55,
      origin: { x: 0 },
      colors: colors
    });
    
    confetti({
      particleCount: 7,
      angle: 120,
      spread: 55,
      origin: { x: 1 },
      colors: colors
    });

    if (Date.now() < end) {
      requestAnimationFrame(frame);
    }
  }());
  
  // Additional burst from center
  setTimeout(() => {
    confetti({
      particleCount: 100,
      spread: 70,
      origin: { y: 0.6 },
      colors: colors
    });
  }, 500);
}
</script>

<?php include 'includes/footer.php'; ?>
