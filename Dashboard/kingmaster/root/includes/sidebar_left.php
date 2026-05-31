<!-- Left Sidebar -->

<!-- Mobile toggle for left sidebar -->
<button class="leftSidebarFab" id="leftSidebarFab" type="button" aria-label="فتح قائمة الأدوات" aria-controls="leftSidebar" aria-expanded="false">
  <i class="fa-solid fa-hammer"></i>
</button>
<div class="sidebarOverlay" id="leftSidebarOverlay"></div>

<aside class="left-sidebar" id="leftSidebar" aria-label="قائمة الأدوات">
  <button class="sidebarClose" id="leftSidebarClose" type="button" aria-label="إغلاق">
    <i class="fa-solid fa-xmark"></i>
  </button>

  <div class="sidebar-title">
    <i class="fa-solid fa-hammer" style="color: #B197FC;"></i>
قائمة الادوات
  </div>
  <ul class="sidebar-menu">
    <?php $current_page = basename($_SERVER['PHP_SELF']); ?>
    <li><a href="fb-tools.php" class="<?php echo ($current_page == 'fb-tools.php') ? 'active' : ''; ?>"><i class="fa-brands fa-facebook" style="color: #276add;"></i>فيسبوك</a></li>
    <li><a href="wa-tools.php" class="<?php echo ($current_page == 'wa-tools.php') ? 'active' : ''; ?>"><i class="fa-brands fa-whatsapp" style="color: #58c90d;"></i>واتساب</a></li>
    <li><a href="insta-tools.php" class="<?php echo ($current_page == 'insta-tools.php') ? 'active' : ''; ?>"><i class="fa-brands fa-instagram" style="color: #cf8026;"></i> انستجرام</a></li>
    <!--notifications.php-->
    <li><a href="coming-soon-page.php" class="<?php echo ($current_page == 'coming-soon-page.php') ? 'active' : ''; ?>"><i class="fa-brands fa-telegram" style="color: #74C0FC;"></i>تليجرام</a></li>
    <!--appointments.php-->
    <li><a href="coming-soon-page.php" class="<?php echo ($current_page == 'coming-soon-page.php') ? 'active' : ''; ?>"><i class="fa-brands fa-google" style="color: #ccb838;"></i> جوجل</a></li>
    <!--documents.php-->
    <li><a href="coming-soon-page.php" class="<?php echo ($current_page == 'coming-soon-page.php') ? 'active' : ''; ?>"><i class="fa-solid fa-briefcase" style="color: #23cbd7;"></i>اعمال</a></li>
    <!--help.php-->
    <li><a href="coming-soon-page.php" class="<?php echo ($current_page == 'coming-soon-page.php') ? 'active' : ''; ?>"><i class="fa-solid fa-envelope" style="color: #FFD43B;"></i>بريد</a></li>
    <!--help.php-->
    <li><a href="coming-soon-page.php" class="<?php echo ($current_page == 'coming-soon-page.php') ? 'active' : ''; ?>"><i class="fa-solid fa-comment-sms" style="color: #74C0FC;"></i>رسائل نصيه</a></li>
    <!--help.php-->


  </ul>
</aside>

<script>
(function initLeftSidebarMobile(){
  const sidebar = document.getElementById('leftSidebar');
  const openBtn = document.getElementById('leftSidebarFab');
  const closeBtn = document.getElementById('leftSidebarClose');
  const overlay = document.getElementById('leftSidebarOverlay');
  if (!sidebar || !openBtn || !overlay) return;

  const setOpen = (open) => {
    // ensure only one drawer is open
    if (open) document.body.classList.remove('rightSidebarOpen');
    document.body.classList.toggle('leftSidebarOpen', open);
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
