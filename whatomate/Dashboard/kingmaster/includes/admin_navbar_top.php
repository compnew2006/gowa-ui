<!-- Top Navbar -->
<nav class="top-navbar">
  <div class="nav-content">
    <div></div>
    
    <ul class="nav-links">
      <?php $current_page = basename($_SERVER['PHP_SELF']); ?>
      
      <?php if (isset($_SESSION['is_viewing_as']) && $_SESSION['is_viewing_as']): ?>
      <li>
        <a href="#" onclick="returnToAdmin(); return false;" style="background: linear-gradient(135deg, #f59e0b 0%, #ef4444 100%); color: white; padding: 8px 20px; border-radius: 8px; font-weight: 700; animation: returnPulse 2s ease-in-out infinite;">
          <i class="fas fa-undo fa-spin"></i> عودة
        </a>
      </li>
      <?php endif; ?>
      
      <li><a href="index.php" class="<?php echo ($current_page == 'index.php') ? 'active' : ''; ?>"><i class="fas fa-home fa-fade"></i>الرئيسية</a></li>
      <li><a href="manage-packages.php" class="<?php echo ($current_page == 'manage-packages.php') ? 'active' : ''; ?>"><i class="fa-solid fa-cubes fa-beat-fade" style="color: #63E6BE;"></i>الباقات</a></li>
      <li><a href="manage-users.php" class="<?php echo ($current_page == 'manage-users.php') ? 'active' : ''; ?>"><i class="fa-solid fa-elevator fa-fade" style="color: #ece90aff;"></i>تفعيل</a></li>
      <li><a href="manage-points-packages.php" class="<?php echo ($current_page == 'manage-points-packages.php') ? 'active' : ''; ?>"><i class="fa-solid fa-gem fa-beat-fade" style="color: #74C0FC;"></i>النقاط</a></li>


    </ul>
    
    <style>
    @keyframes returnPulse {
        0%, 100% {
            box-shadow: 0 0 10px rgba(245, 158, 11, 0.5);
            transform: scale(1);
        }
        50% {
            box-shadow: 0 0 25px rgba(245, 158, 11, 0.8);
            transform: scale(1.05);
        }
    }
    </style>
    
    <script>
    function returnToAdmin() {
        Swal.fire({
            title: 'هل أنت متأكد؟',
            text: 'سيتم العودة إلى حسابك الأصلي',
            icon: 'question',
            showCancelButton: true,
            confirmButtonColor: '#f59e0b',
            cancelButtonColor: '#6b7280',
            confirmButtonText: 'نعم، عودة',
            cancelButtonText: 'إلغاء'
        }).then((result) => {
            if (result.isConfirmed) {
                fetch('return_to_admin.php', {
                    method: 'POST'
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        Swal.fire({
                            icon: 'success',
                            title: 'تم العودة!',
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
                        text: 'حدث خطأ أثناء العودة',
                        confirmButtonText: 'حسناً'
                    });
                });
            }
        });
    }
    </script>
    <div class="site-logo">
      <img id="site-logo-img" src="images/logo-png-WEP2.png" alt="Logo">
    </div>
  </div>
</nav>
