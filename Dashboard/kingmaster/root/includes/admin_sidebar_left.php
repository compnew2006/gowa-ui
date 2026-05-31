<!-- Left Sidebar -->
<aside class="left-sidebar">
  <div class="sidebar-title">
    <i class="fa-solid fa-hammer" style="color: #B197FC;"></i>
قائمة الادوات
  </div>
  <ul class="sidebar-menu">
    <?php $current_page = basename($_SERVER['PHP_SELF']); ?>
    <li><a href="fb-tools.php" class="<?php echo ($current_page == 'fb-tools.php') ? 'active' : ''; ?>"><i class="fa-brands fa-facebook" style="color: #276add;"></i>فيسبوك</a></li>
    <li><a href="messages.php" class="<?php echo ($current_page == 'messages.php') ? 'active' : ''; ?>"><i class="fa-brands fa-whatsapp" style="color: #58c90d;"></i>واتساب</a></li>
    <li><a href="notifications.php" class="<?php echo ($current_page == 'notifications.php') ? 'active' : ''; ?>"><i class="fa-brands fa-instagram" style="color: #cf8026;"></i> انستجرام</a></li>
    <li><a href="appointments.php" class="<?php echo ($current_page == 'appointments.php') ? 'active' : ''; ?>"><i class="fa-brands fa-telegram" style="color: #74C0FC;"></i>تليجرام</a></li>
    <li><a href="documents.php" class="<?php echo ($current_page == 'documents.php') ? 'active' : ''; ?>"><i class="fa-brands fa-google" style="color: #ccb838;"></i> جوجل</a></li>
    <li><a href="help.php" class="<?php echo ($current_page == 'help.php') ? 'active' : ''; ?>"><i class="fa-solid fa-briefcase" style="color: #23cbd7;"></i>اعمال</a></li>
    <li><a href="help.php" class="<?php echo ($current_page == 'help.php') ? 'active' : ''; ?>"><i class="fa-solid fa-envelope" style="color: #FFD43B;"></i>بريد</a></li>
    <li><a href="help.php" class="<?php echo ($current_page == 'help.php') ? 'active' : ''; ?>"><i class="fa-solid fa-comment-sms" style="color: #74C0FC;"></i>رسائل نصيه</a></li>


  </ul>
</aside>
