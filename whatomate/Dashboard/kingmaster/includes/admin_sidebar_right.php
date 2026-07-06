<!-- Right Sidebar -->
<aside class="right-sidebar">
  <div class="sidebar-title">
    <i class="fa-solid fa-magnet fa-bounce" style="color: #74C0FC;"></i>
   قائمه الادارة
  </div>
  <ul class="sidebar-menu">
    <?php $current_page = basename($_SERVER['PHP_SELF']); ?>
    <li><a href="accounts.php" class="<?php echo ($current_page == 'accounts.php') ? 'active' : ''; ?>"><i class="fa-solid fa-user fa-beat-fade" style="color: #6547bd;"></i>الحسابات</a></li>
     <li><a href="admin-products.php" class="<?php echo ($current_page == 'admin-products.php') ? 'active' : ''; ?>"><i class="fa-brands fa-product-hunt fa-beat-fade" style="color: #5abb25;"></i>المنتاجات</a></li>
    <li><a href="manage-coupons.php" class="<?php echo ($current_page == 'manage-coupons.php') ? 'active' : ''; ?>"><i class="fas fa-gift fa-fade"></i> الكوبونات</a></li>
   <li><a href="posts.php" class="<?php echo ($current_page == 'posts.php') ? 'active' : ''; ?>"><i class="fa-solid fa-triangle-exclamation fa-beat-fade" style="color: #FFD43B;"></i>اشعارات النظام</a></li>
  <li><a href="syswalt_records.php" class="<?php echo ($current_page == 'syswalt_records.php') ? 'active' : ''; ?>"><i class="fa-solid fa-money-bill-1-wave fa-beat" style="color: #2cd888;"></i>المعاملات المالية</a></li>
 <li><a href="manage_announcements.php" class="<?php echo ($current_page == 'manage_announcements.php') ? 'active' : ''; ?>"><i class="fa-solid fa-bell fa-shake" style="color: #df2ad0;"></i>أداره الاشعارات</a></li>
 <li><a href="admin_withdrawals.php" class="<?php echo ($current_page == 'admin_withdrawals.php') ? 'active' : ''; ?>"><i class="fa-solid fa-money-bill-transfer fa-shake" style="color: #2a78dfff;"></i>طلبات السحب</a></li>


   </ul>
</aside>
