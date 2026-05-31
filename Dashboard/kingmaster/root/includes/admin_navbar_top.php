<!-- Top Navbar -->
<nav class="top-navbar" id="adminTopNavbar">
  <div class="nav-content">
    <button class="navToggle" id="adminTopNavToggle" type="button" aria-label="القائمة" aria-expanded="false">
      <span></span><span></span><span></span>
    </button>
    
    <ul class="nav-links" id="adminTopNavLinks">
      <?php $current_page = basename($_SERVER['PHP_SELF']); ?>
      
      <?php if (isset($_SESSION['is_viewing_as']) && $_SESSION['is_viewing_as']): ?>
      <li>
        <a href="return_to_admin.php" onclick="returnToAdmin(); return false;" style="background: linear-gradient(135deg, #f59e0b 0%, #ef4444 100%); color: white; padding: 8px 20px; border-radius: 8px; font-weight: 700;">
          <i class="fas fa-undo"></i> عودة
        </a>
      </li>
      <?php endif; ?>
      
      <li><a href="index.php" class="<?php echo ($current_page == 'index.php') ? 'active' : ''; ?>"><i class="fas fa-home"></i>الرئيسية</a></li>
      <li><a href="manage-packages.php" class="<?php echo ($current_page == 'manage-packages.php') ? 'active' : ''; ?>"><i class="fa-solid fa-cubes" style="color: #63E6BE;"></i>الباقات</a></li>
      <li><a href="manage-users.php" class="<?php echo ($current_page == 'manage-users.php') ? 'active' : ''; ?>"><i class="fa-solid fa-elevator" style="color: #ece90aff;"></i>تفعيل</a></li>
      <li><a href="manage-points-packages.php" class="<?php echo ($current_page == 'manage-points-packages.php') ? 'active' : ''; ?>"><i class="fa-solid fa-gem" style="color: #74C0FC;"></i>النقاط</a></li>


    </ul>
    
    <script>
    (function initAdminTopNavbarMobile(){
      const nav = document.getElementById('adminTopNavbar');
      const btn = document.getElementById('adminTopNavToggle');
      const links = document.getElementById('adminTopNavLinks');
      if (!nav || !btn || !links) return;

      const setOpen = (open) => {
        nav.classList.toggle('is-open', open);
        btn.setAttribute('aria-expanded', String(open));
      };

      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        setOpen(!nav.classList.contains('is-open'));
      });

      links.addEventListener('click', (e) => {
        const a = e.target.closest('a');
        if (!a) return;
        setOpen(false);
      });

      document.addEventListener('click', (e) => {
        if (nav.contains(e.target)) return;
        setOpen(false);
      });
    })();
    </script>

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
