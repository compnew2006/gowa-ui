

<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$user = getUserByUserId($user_id);
$expiry_date = $user['expiry_date'];
$date_only = explode(' ', $expiry_date)[0];

 
if (!empty($expiry_date)) {
    $expiry_timestamp = strtotime($expiry_date);
    $now_timestamp = time();
    
    if ($expiry_timestamp < $now_timestamp) {
       header('Location: index.php');
    exit;
    }
}

$page_title = "أدوات الانستجرام | Kingmaster";
$page_css = ['/css/wa-tools.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';


?>

<!-- Main Content -->
<main class="main-content">
  <div class="content-card">
    <h2 style="margin-bottom: 2rem;" ><i class="fa-brands fa-instagram insta-icon" ></i>أدوات الانستجرام</h2>
    
    <!-- Accordion Container -->
    <div class="accordion-container">
      
  
      <!-- البحث - Insta Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion2')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-brands fa-searchengin" style="font-size: 1.5rem; color: #f09f87;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات البحث</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion2"></i>
        </div>
        <div class="accordion-content" id="accordion2">
          <div class="tools-grid">
            
            <!-- البحث عن بروفايل -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-magnifying-glass" style="font-size: 2rem; color: #f09f87;"></i>
              </div>
              <h4 class="tool-title">البحث عن بروفايل</h4>
              <p class="tool-description"> يمكنك من خلال هذه الاداه البحث عن بروفايل على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-profile.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
            <!-- البحث عن موقع -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-location-dot" style="font-size: 2rem; color: #f09f87;"></i>
              </div>
              <h4 class="tool-title">البحث عن موقع</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه البحث عن موقع على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-location.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
       <!-- البحث عن هاشتاج -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-hashtag" style="font-size: 2rem; color: #f09f87;"></i>
              </div>
              <h4 class="tool-title">البحث عن هاشتاج</h4>
              <p class="tool-description"> يمكنك من خلال هذه الاداه البحث عن هاشتاج على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-hashtag.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
              <!-- البحث عن بروفايل -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-magnifying-glass" style="font-size: 2rem; color: #f09f87;"></i>
              </div>
              <h4 class="tool-title">البحث عن بروفايل البحث بالبايو</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه البحث عن بروفايل البحث بالبايو على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-profile-bio.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
          </div>
        </div>
      </div>

 



           <!--  الاستخراج - insta Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion3')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-brands fa-stack-exchange" style="font-size: 1.5rem; color: #f09f87;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الأستخراج</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion3"></i>
        </div>
        <div class="accordion-content" id="accordion3">
          <div class="tools-grid">
            
                                <!-- استخراج الراسلئ -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-image" style="font-size: 2rem; color:#f09f87;"></i>
                                </div>
                                <h4 class="tool-title">استخراج منشورات بروفايل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع منشورات بروفايل</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-posts.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>


                    <!-- استخراج جهات الاتصا -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-message" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">استخراج تعليقات المنشور</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع تعليقات على منشور علي الانستجرام</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-comments.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>


                    <!-- استخراج مجموعاتي -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-heart" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">استخراج اعجابات المنشور</h4>
                                <p class="tool-description">استخراج قائمة المعجبين من أي منشور أو صفحة</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-likes.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>



                    <!-- استخراج اعضاء -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-user-group" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">استخراج المتابعين والمتابعين</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج قائمة بالمتابعين</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-followers.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>
								<!-- استخراج مشاهدي الاستوري
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-clock" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">استخراج مشاهدي الاستوري</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع مشاهدي الاستوري</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-viewers.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>
                                -->
								<!-- استخراج جميع مستخدمي الرسائل -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-envelope" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">استخراج جميع مستخدمي الرسائل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع مستخدمي الرسائل</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-messages.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>







       

          </div>
        </div>
      </div>



      <!--  التفاعل - insta Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion4')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-solid fa-square-plus" style="font-size: 1.5rem; color: #f09f87;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات التفاعل</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion4"></i>
        </div>
        <div class="accordion-content" id="accordion4">
          <div class="tools-grid">
            
                                <!-- استخراج الانضمام -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-user-plus" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">أداة المتابعة</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه لمتابعة الأشخاص</p>
                                <div class="tool-buttons">
                                    <a href="insta-follow-tool.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>
								<!-- الغاء المتابعة -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-user-minus" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">أداة المتابعة</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه لالغاء المتابعة</p>
                                <div class="tool-buttons">
                                    <a href="insta-unfollow-tool.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>

                    <!-- اضافه اعضاء -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-at" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">أداة المنشن</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداة لعمل منشن</p>
                                <div class="tool-buttons">
                                    <a href="insta-mention-tool.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>




             




       

          </div>
        </div>
      </div>


 <!--  الاضافه - WhatsApp Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion5')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-regular fa-paper-plane" style="font-size: 1.5rem; color: #f09f87;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات النشر</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion5"></i>
        </div>
        <div class="accordion-content" id="accordion5">
          <div class="tools-grid">
            
                                <!-- استخراج الانضمام -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-play" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">نشر ستوري تلقائي</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداه لمشاركة نشر الاستوري </p>
                                <div class="tool-buttons">
                                    <a href="insta-auto-story.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>


                    <!-- اضافه اعضاء -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-image" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">نشر بوست تلقائي</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداة لنشر البوست تلقائياً</p>
                                <div class="tool-buttons">
                                    <a href="insta-auto-post.php" class="tool-btn primary">
                                    <i class="fas fa-image"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>




             




       

          </div>
        </div>
      </div>


 <!--  المساعدة - WhatsApp Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion6')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-solid fa-handshake-angle" style="font-size: 1.5rem; color: #f09f87;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الرسائل</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion6"></i>
        </div>
        <div class="accordion-content" id="accordion6">
          <div class="tools-grid">
            
                                <!-- الفلتر -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                  <!--  <i class="fa-solid fa-envelope " style="font-size: 2rem; color: #cf8026;"></i>-->
								 <i class="fa-solid fa-envelope" style="font-size: 2rem; color: #f09f87;"></i>
                                </div>
                                <h4 class="tool-title">ارسال الرسائل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداه لارسال الرسائل</p>
                                <div class="tool-buttons">
                                    <a href="insta-send-message.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="tut.php" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>

 


             




       

          </div>
        </div>
      </div>





    </div>
  </div>
</main>




<script>
function toggleAccordion(id) {
  // Get all accordion contents and icons
  const allContents = document.querySelectorAll('.accordion-content');
  const allIcons = document.querySelectorAll('.accordion-icon');
  const targetContent = document.getElementById(id);
  const targetIcon = document.getElementById('icon-' + id);

  if (!targetContent) return;
  
  // Close all accordions
  allContents.forEach(content => {
    if (content.id !== id) {
      content.classList.remove('active');
    }
  });
  
  // Reset all icons
  allIcons.forEach(icon => {
    if (icon.id !== 'icon-' + id) {
      icon.classList.remove('rotate');
    }
  });
  
  // Toggle target accordion
  targetContent.classList.toggle('active');
  if (targetIcon) targetIcon.classList.toggle('rotate');
}
</script>

<?php include 'includes/footer.php'; ?>
