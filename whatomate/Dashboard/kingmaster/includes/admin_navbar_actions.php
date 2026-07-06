<?php 
 $user_id = $_SESSION['user_id'] ; // مثال

$user = getUserByUserId($user_id);
 
$img = $user['img'];
if ($img == "0"){
  $img = "https://i.pravatar.cc/150?img=33";
 
}
?>

<div class="nav-actions-container">
  <!-- Profile Icon -->
  <div class="action-icon" onclick="toggleDropdown('profile')">
    <img src="<?php echo $img; ?>" alt="Profile">
    <div class="dropdown" id="profile-dropdown">
      <div class="dropdown-header">الملف الشخصي</div>
      <div class="dropdown-content">
        <div class="dropdown-item" onclick="window.location.href='profile.php'" style="cursor: pointer;">
          <div class="dropdown-item-title"><i class="fas fa-user fa-fade"></i> عرض الملف الشخصي</div>
        </div>
        <div class="dropdown-item" onclick="window.location.href='settings.php'" style="cursor: pointer;">
          <div class="dropdown-item-title"><i class="fas fa-cog fa-spin" style="--fa-animation-duration: 3s;"></i> الإعدادات</div>
        </div>
        <div class="dropdown-item">
          <div class="dropdown-item-title" style="color: #ef4444;"><i class="fas fa-sign-out-alt fa-fade"></i> تسجيل الخروج</div>
        </div>
      </div>
    </div>
  </div>
  
  <!-- Messages Icon -->
  <div class="action-icon" onclick="toggleDropdown('messages')">
    <i class="fas fa-envelope"></i>
    <span class="badge">3</span>
    <div class="dropdown" id="messages-dropdown">
      <div class="dropdown-header">الرسائل</div>
      <div class="dropdown-content">
        <div class="dropdown-item">
          <div class="dropdown-item-title">أحمد محمد</div>
          <div class="dropdown-item-text">مرحباً، كيف يمكنني مساعدتك؟</div>
          <div class="dropdown-item-time">منذ 5 دقائق</div>
        </div>
        <div class="dropdown-item">
          <div class="dropdown-item-title">سارة علي</div>
          <div class="dropdown-item-text">شكراً على المساعدة السريعة</div>
          <div class="dropdown-item-time">منذ 15 دقيقة</div>
        </div>
        <div class="dropdown-item">
          <div class="dropdown-item-title">محمود حسن</div>
          <div class="dropdown-item-text">هل يمكن تحديث الطلب؟</div>
          <div class="dropdown-item-time">منذ ساعة</div>
        </div>
      </div>
      <div class="dropdown-footer">
        <a href="#">قراءة المزيد <i class="fas fa-arrow-left"></i></a>
      </div>
    </div>
  </div>
  
  <!-- Notifications Icon -->
  <div class="action-icon" onclick="toggleDropdown('notifications')">
    <i class="fas fa-bell"></i>
    <span class="badge">5</span>
    <div class="dropdown" id="notifications-dropdown">
      <div class="dropdown-header">الإشعارات</div>
      <div class="dropdown-content">
        <div class="dropdown-item">
          <div class="dropdown-item-title">طلب جديد</div>
          <div class="dropdown-item-text">لديك طلب جديد من العميل #1234</div>
          <div class="dropdown-item-time">منذ دقيقتين</div>
        </div>
        <div class="dropdown-item">
          <div class="dropdown-item-title">تم إتمام الدفع</div>
          <div class="dropdown-item-text">تم استلام دفعة بقيمة 500 جنيه</div>
          <div class="dropdown-item-time">منذ 10 دقائق</div>
        </div>
        <div class="dropdown-item">
          <div class="dropdown-item-title">تنبيه المخزون</div>
          <div class="dropdown-item-text">المنتج XYZ أوشك على النفاد</div>
          <div class="dropdown-item-time">منذ 30 دقيقة</div>
        </div>
        <div class="dropdown-item">
          <div class="dropdown-item-title">تقييم جديد</div>
          <div class="dropdown-item-text">حصلت على تقييم 5 نجوم</div>
          <div class="dropdown-item-time">منذ ساعة</div>
        </div>
        <div class="dropdown-item">
          <div class="dropdown-item-title">تحديث النظام</div>
          <div class="dropdown-item-text">تحديث جديد متاح للتثبيت</div>
          <div class="dropdown-item-time">منذ ساعتين</div>
        </div>
      </div>
      <div class="dropdown-footer">
        <a href="#">قراءة المزيد <i class="fas fa-arrow-left"></i></a>
      </div>
    </div>
  </div>
</div>
