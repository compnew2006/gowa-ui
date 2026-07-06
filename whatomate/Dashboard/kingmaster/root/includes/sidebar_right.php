<!-- Right Sidebar -->

<!-- Mobile toggle for right sidebar -->
<button class="rightSidebarFab" id="rightSidebarFab" type="button" aria-label="فتح قائمة الإدارة" aria-controls="rightSidebar" aria-expanded="false">
  <i class="fa-solid fa-magnet"></i>
</button>
<div class="sidebarOverlay" id="rightSidebarOverlay"></div>

<aside class="right-sidebar" id="rightSidebar" aria-label="قائمة الإدارة">
  <button class="sidebarClose" id="rightSidebarClose" type="button" aria-label="إغلاق">
    <i class="fa-solid fa-xmark"></i>
  </button>

  <div class="sidebar-title">
    <i class="fa-solid fa-magnet" style="color: #74C0FC;"></i>
   قائمه الادارة
  </div>
  <ul class="sidebar-menu">
    <?php $current_page = basename($_SERVER['PHP_SELF']); ?>
    <li><a href="files.php" class="<?php echo ($current_page == 'files.php') ? 'active' : ''; ?>"><i class="fa-solid fa-photo-film" style="color: #74C0FC;"></i>الملفات</a></li>
    <li><a href="customers.php" class="<?php echo ($current_page == 'customers.php') ? 'active' : ''; ?>"><i class="fa-solid fa-address-book" style="color: #63E6BE;"></i>جهات الاتصال</a></li>
    <li><a href="content.php" class="<?php echo ($current_page == 'content.php') ? 'active' : ''; ?>"><i class="fa-regular fa-message" style="color: #FFD43B;"></i>المحتوي</a></li>
    <li><a href="accounts.php" class="<?php echo ($current_page == 'accounts.php') ? 'active' : ''; ?>"><i class="fa-solid fa-user" style="color: #6547bd;"></i>الحسابات</a></li>
    <li><a href="wallet.php" class="<?php echo ($current_page == 'wallet.php') ? 'active' : ''; ?>"><i class="fas fa-wallet"></i> المحفظة</a></li>
    <li><a href="products.php" class="<?php echo ($current_page == 'products.php') ? 'active' : ''; ?>"><i class="fa-brands fa-product-hunt" style="color: #5abb25;"></i>المنتجات</a></li>
    <li><a href="coupons.php" class="<?php echo ($current_page == 'coupons.php') ? 'active' : ''; ?>"><i class="fas fa-gift"></i> الكوبونات</a></li>
    <li><a href="goals.php" class="<?php echo ($current_page == 'goals.php') ? 'active' : ''; ?>"><i class="fa-solid fa-code" style="color: #cbe21d;"></i>المطورين</a></li>
  </ul>
</aside>

<script>
(function initRightSidebarMobile(){
  const sidebar = document.getElementById('rightSidebar');
  const openBtn = document.getElementById('rightSidebarFab');
  const closeBtn = document.getElementById('rightSidebarClose');
  const overlay = document.getElementById('rightSidebarOverlay');
  if (!sidebar || !openBtn || !overlay) return;

  const setOpen = (open) => {
    // ensure only one drawer is open
    if (open) document.body.classList.remove('leftSidebarOpen');
    document.body.classList.toggle('rightSidebarOpen', open);
    openBtn.setAttribute('aria-expanded', String(open));
  };

  openBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    setOpen(true);
  });

  overlay.addEventListener('click', () => setOpen(false));
  if (closeBtn) closeBtn.addEventListener('click', () => setOpen(false));

  // Close when clicking a link inside (mobile only)
  sidebar.addEventListener('click', (e) => {
    const a = e.target.closest('a');
    if (!a) return;
    if (window.matchMedia && window.matchMedia('(max-width: 768px)').matches) {
      setOpen(false);
    }
  });

  // Close on ESC
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') setOpen(false);
  });
})();
</script>
