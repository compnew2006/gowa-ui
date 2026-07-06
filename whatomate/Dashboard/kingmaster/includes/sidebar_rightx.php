<!-- Right Sidebar -->
<aside class="right-sidebar">
  <div class="sidebar-title">
    <i class="fa-solid fa-magnet fa-bounce" style="color: #74C0FC;"></i>
   قائمه الادارة
  </div>
  <ul class="sidebar-menu">
    <?php $current_page = basename($_SERVER['PHP_SELF']); ?>
    <li><a href="files.php" class="<?php echo ($current_page == 'files.php') ? 'active' : ''; ?>"><i class="fa-solid fa-photo-film fa-beat-fade" style="color: #74C0FC;"></i>الملفات</a></li>
    <li><a href="customers.php" class="<?php echo ($current_page == 'customers.php') ? 'active' : ''; ?>"><i class="fa-solid fa-address-book fa-beat-fade" style="color: #63E6BE;"></i>جهات الاتصال</a></li>
    <li><a href="content.php" class="<?php echo ($current_page == 'content.php') ? 'active' : ''; ?>"><i class="fa-regular fa-message fa-beat-fade" style="color: #FFD43B;"></i>المحتوي</a></li>
    <li><a href="accounts.php" class="<?php echo ($current_page == 'accounts.php') ? 'active' : ''; ?>"><i class="fa-solid fa-user fa-beat-fade" style="color: #6547bd;"></i>الحسابات</a></li>
    <li><a href="wallet.php" class="<?php echo ($current_page == 'wallet.php') ? 'active' : ''; ?>"><i class="fas fa-wallet fa-fade"></i> المحفظة</a></li>
    <li><a href="products.php" class="<?php echo ($current_page == 'products.php') ? 'active' : ''; ?>"><i class="fa-brands fa-product-hunt fa-beat-fade" style="color: #5abb25;"></i>المنتاجات</a></li>
    <li><a href="coupons.php" class="<?php echo ($current_page == 'coupons.php') ? 'active' : ''; ?>"><i class="fas fa-gift fa-fade"></i> الكوبونات</a></li>
    <li><a href="goals.php" class="<?php echo ($current_page == 'goals.php') ? 'active' : ''; ?>"><i class="fa-solid fa-code fa-beat" style="color: #cbe21d;"></i>المطورين</a></li>
  </ul>
</aside>
