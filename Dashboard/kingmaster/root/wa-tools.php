

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

$page_title = "الأدوات | Kingmaster";
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
    <h2 style="margin-bottom: 2rem;"><i class="fa-brands fa-whatsapp" style="color: #21ca2c;"></i> أدوات الواتساب</h2>
    
    <!-- Accordion Container -->
    <div class="accordion-container">
      
  
      <!-- البحث - WhatsApp Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion2')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-brands fa-searchengin" style="font-size: 1.5rem; color: #25D366;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات البحث</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion2"></i>
        </div>
        <div class="accordion-content" id="accordion2">
          <div class="tools-grid">
            
            <!-- البحث عن مجموعات -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-magnifying-glass" style="font-size: 2rem; color: #25D366;"></i>
              </div>
              <h4 class="tool-title">أداة البحث عن مجموعات</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه البحث عن مجموعات واتساب للانضمام اليها</p>
              <div class="tool-buttons">
                <a href="search-groups.php" class="tool-btn primary">
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

 



           <!--  الاستخراج - WhatsApp Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion3')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-brands fa-stack-exchange" style="font-size: 1.5rem; color: #25D366;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الأستخراج</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion3"></i>
        </div>
        <div class="accordion-content" id="accordion3">
          <div class="tools-grid">
            
                                <!-- استخراج الراسلئ -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-message" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة استخراج الرسائل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع محادثاتك علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="wa-extract-messages.php" class="tool-btn primary">
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
                                    <i class="fa-solid fa-address-book" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة استخراج جهات الاتصال</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع جهات الاتصال علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="wa-extract-contacts.php" class="tool-btn primary">
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
                                    <i class="fa-solid fa-users-rectangle" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة استخراج المجموعات</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع المجموعات المشترك بها علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="wa-extract-groups.php" class="tool-btn primary">
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
                                    <i class="fa-solid fa-user-group" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة استخراج اعضاء المجموعه</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع اعضاء مجموعه مشترك بها علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="wa-extract-members.php" class="tool-btn primary">
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
        <div class="accordion-header" onclick="toggleAccordion('accordion4')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-solid fa-square-plus" style="font-size: 1.5rem; color: #25D366;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الأضافة</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion4"></i>
        </div>
        <div class="accordion-content" id="accordion4">
          <div class="tools-grid">
            
                                <!-- استخراج الانضمام -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-right-to-bracket" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة الانضمام الي مجموعات</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه الانضمام التلقائي في مجموعات علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="wa-add-groups.php" class="tool-btn primary">
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
                                    <i class="fa-solid fa-plus" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة اضافه الاعضاء داخل مجموعة</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداة اضافه اعضاء داخل مجموعه علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="wa-add.php" class="tool-btn primary">
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
            <i class="fa-regular fa-paper-plane" style="font-size: 1.5rem; color: #25D366;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الأرسال</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion5"></i>
        </div>
        <div class="accordion-content" id="accordion5">
          <div class="tools-grid">
            
                                <!-- استخراج الانضمام -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-paper-plane" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة أرسال الرسائل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداه ارسال الرسائل لي اشخاص او مجموعات علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="wa-sender.php" class="tool-btn primary">
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
                                    <i class="fa-solid fa-robot" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداه الرد التلقائي</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداة انشاء ردود تلقائية علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="flow-builder.php" class="tool-btn primary">
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


 <!--  المساعدة - WhatsApp Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion6')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-solid fa-handshake-angle" style="font-size: 1.5rem; color: #25D366;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات المساعدة</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion6"></i>
        </div>
        <div class="accordion-content" id="accordion6">
          <div class="tools-grid">
            
                                <!-- الفلتر -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-filter" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة التحقق من الهواتف</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداه التحقق من الهواتف المسجله علي الواتساب</p>
                                <div class="tool-buttons">
                                    <a href="filter-wa.php" class="tool-btn primary">
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
