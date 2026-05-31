<?php 
 $user_id = $_SESSION['user_id'] ; // مثال

$is_admin = getUserIsAdmin($user_id);

?>

<!-- Top Navbar -->
<nav class="top-navbar" id="topNavbar">
  <div class="nav-content">
    <button class="navToggle" id="topNavToggle" type="button" aria-label="القائمة" aria-expanded="false">
      <span></span><span></span><span></span>
    </button>
    
    <ul class="nav-links" id="topNavLinks">
      <?php $current_page = basename($_SERVER['PHP_SELF']); ?>

      <?php if (isset($_SESSION['is_viewing_as']) && $_SESSION['is_viewing_as']): ?>
      <li>
        <a href="return_to_admin.php" onclick="returnToAdmin(); return false;" style="background: linear-gradient(135deg, #f59e0b 0%, #ef4444 100%); color: white; padding: 8px 20px; border-radius: 8px; font-weight: 700;">
          <i class="fas fa-undo"></i> عودة
        </a>
      </li>
      <?php endif; ?>

      <li><a href="index.php" class="<?php echo ($current_page == 'index.php') ? 'active' : ''; ?>"><i class="fas fa-home"></i> الرئيسية</a></li>
      <li><a href="data-extraction.php" class="<?php echo ($current_page == 'data-extraction.php') ? 'active' : ''; ?>"><i class="fa-solid fa-database" style="color: #63E6BE;"></i>البيانات</a></li>
      <li><a href="sending-settings.php" class="<?php echo ($current_page == 'sending-settings.php') ? 'active' : ''; ?>"><i class="fas fa-cog"></i> الإعدادات</a></li>
      <li><a href="points.php" class="<?php echo ($current_page == 'points.php') ? 'active' : ''; ?>"><i class="fa-solid fa-gem" style="color: #74C0FC;"></i>شحن</a></li>

<?php if ($is_admin === null) { ?>
    <!-- المستخدم غير موجود -->
<?php } elseif ($is_admin == "1") { ?>
    <li>
        <a href="statistics.php" class="<?= ($current_page == 'statistics.php') ? 'active' : ''; ?>">
            <i class="fa-solid fa-unlock" style="color: #2374b6ff;"></i>
            المسؤول
        </a>
    </li>
<?php } ?>


    </ul>
    <div class="site-logo">
      <img id="site-logo-img" src="images/logo-png-WEP2.png" alt="" onerror="this.style.display='none'">
    </div>
  </div>
</nav>

<script>
(function initTopNavbarMobile(){
  const nav = document.getElementById('topNavbar');
  const btn = document.getElementById('topNavToggle');
  const links = document.getElementById('topNavLinks');
  if (!nav || !btn || !links) return;

  const setOpen = (open) => {
    nav.classList.toggle('is-open', open);
    btn.setAttribute('aria-expanded', String(open));
  };

  btn.addEventListener('click', (e) => {
    e.stopPropagation();
    setOpen(!nav.classList.contains('is-open'));
  });

  // close when clicking a link
  links.addEventListener('click', (e) => {
    const a = e.target.closest('a');
    if (!a) return;
    setOpen(false);
  });

  // close on outside click
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
