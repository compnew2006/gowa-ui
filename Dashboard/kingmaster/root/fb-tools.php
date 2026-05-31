

<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

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
    <h2 style="margin-bottom: 2rem;"><i class="fas fa-tools" style="color: #1877f2"></i> أدوات الفيسبوك</h2>
    
    <!-- Accordion Container -->
    <div class="accordion-container">
      
      <!-- Accordion 1 - Facebook Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion1')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fab fa-facebook" style="font-size: 1.5rem; color: #1877f2;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات البحث</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion1"></i>
        </div>
        <div class="accordion-content" id="accordion1">
          <div class="tools-grid">
            
   
 

            <!-- Tool Card search page -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fa-solid fa-flag" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">أداه البحث عن صفحات</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه البحث عن صفحات علي الفيسبوك</p>
              <div class="tool-buttons">
                <a href="serch_fb_page.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>


            <!-- Tool Card 7 -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fas fa-search" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">البحث عن أشخاص</h4>
              <p class="tool-description">البحث عن مستخدمين فيسبوك حسب معايير محددة</p>
              <div class="tool-buttons">
                <a href="serch_pepols.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>

    

            <!-- Tool Card 9 -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fas fa-robot" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">البحث عن مجموعات</h4>
              <p class="tool-description">يمكنك البحث عن مجموعات فيسبوك</p>
              <div class="tool-buttons">
                <a href="serch_fb_group.php" class="tool-btn primary">
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


      <!-- Accordion 1 - Facebook Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordionfac')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fab fa-facebook" style="font-size: 1.5rem; color: #1877f2;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الاستخراج</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordionfac"></i>
        </div>
        <div class="accordion-content" id="accordionfac">
          <div class="tools-grid">
            
   
 

            <!-- Tool Card search page -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fa-solid fa-flag" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">أداه استخراج اعضاء الجروب</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه استخراج اعضاء جروب علي الفيسبوك </p>
              <div class="tool-buttons">
                <a href="members_group.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>

            <!-- Tool Card search hashtag -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fa-solid fa-hashtag" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">أداه استخراج التعليقات</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه استخراج تعليقات منشور</p>
              <div class="tool-buttons">
                <a href="get_comment_post.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>



            <!-- Tool Card 5 -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fas fa-heart" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">استخراج الإعجابات</h4>
              <p class="tool-description">استخراج قائمة المعجبين من أي منشور أو صفحة</p>
              <div class="tool-buttons">
                <a href="like_post.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>

 


            <!-- Tool Card 5 -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fas fa-heart" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">استخراج مراسلين الصفحه</h4>
              <p class="tool-description">استخراج قائمة مراسليم الصفحه</p>
              <div class="tool-buttons">
                <a href="get_msg_pg.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="tut.php" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
            
            
            
              <!-- Tool Card 5 -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fas fa-heart" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">استخراج البيانات</h4>
              <p class="tool-description">استخراج قائمة حسابات</p>
              <div class="tool-buttons">
                <a href="dbs.php" class="tool-btn primary">
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


      <!-- Accordion 1 - Facebook Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordionpublsh')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fab fa-facebook" style="font-size: 1.5rem; color: #1877f2;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات النشر</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordionpublsh"></i>
        </div>
        <div class="accordion-content" id="accordionpublsh">
          <div class="tools-grid">
            
   
 


            <!-- Tool Card 8 -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fas fa-share-alt" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">مشاركة تلقائية</h4>
              <p class="tool-description">مشاركة المنشورات بشكل تلقائي في المجموعات</p>
              <div class="tool-buttons">
                <a href="send_group.php" class="tool-btn primary">
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

    <!-- Accordion 1 - Facebook Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('sends')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fab fa-facebook" style="font-size: 1.5rem; color: #1877f2;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الارسال</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-sends"></i>
        </div>
        <div class="accordion-content" id="sends">
          <div class="tools-grid">
            
   
 


            <!-- Tool Card 8 -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(24, 119, 242, 0.1);">
                <i class="fas fa-share-alt" style="font-size: 2rem; color: #1877f2;"></i>
              </div>
              <h4 class="tool-title">اعاده الاستهداف</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه اعاده استهداف مراسلين الصفحه</p>
              <div class="tool-buttons">
                <a href="send_fb.php" class="tool-btn primary">
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
  targetIcon.classList.toggle('rotate');
}
</script>

<?php include 'includes/footer.php'; ?>
