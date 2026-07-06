

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
    <h2 style="margin-bottom: 2rem;"><i class="fa-brands fa-instagram fa-fade" style="color: #cf8026;"></i>أدوات الانستجرام</h2>
    
    <!-- Accordion Container -->
    <div class="accordion-container">
      
  
      <!-- البحث - Insta Tools -->
      <div class="accordion-item">
        <div class="accordion-header" onclick="toggleAccordion('accordion2')">
          <div style="display: flex; align-items: center; gap: 1rem;">
            <i class="fa-brands fa-searchengin fa-beat" style="font-size: 1.5rem; color: #cf8026;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات البحث</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion2"></i>
        </div>
        <div class="accordion-content" id="accordion2">
          <div class="tools-grid">
            
            <!-- البحث عن بروفايل -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-magnifying-glass fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
              </div>
              <h4 class="tool-title">البحث عن بروفايل</h4>
              <p class="tool-description"> يمكنك من خلال هذه الاداه البحث عن بروفايل على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-profile.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="#" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
            <!-- البحث عن موقع -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-location-dot fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
              </div>
              <h4 class="tool-title">البحث عن موقع</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه البحث عن موقع على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-location.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="#" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
       <!-- البحث عن هاشتاج -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-hashtag fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
              </div>
              <h4 class="tool-title">البحث عن هاشتاج</h4>
              <p class="tool-description"> يمكنك من خلال هذه الاداه البحث عن هاشتاج على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-hashtag.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="#" class="tool-btn secondary">
                  <i class="fas fa-video"></i>
                  شاهد الشرح
                </a>
              </div>
            </div>
              <!-- البحث عن بروفايل -->
            <div class="tool-card">
              <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                <i class="fa-solid fa-magnifying-glass fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
              </div>
              <h4 class="tool-title">البحث عن بروفايل البحث بالبايو</h4>
              <p class="tool-description">يمكنك من خلال هذه الاداه البحث عن بروفايل البحث بالبايو على الانستجرام</p>
              <div class="tool-buttons">
                <a href="insta-search-profile%20-%20Bio.php" class="tool-btn primary">
                  <i class="fas fa-play"></i>
                  استخدم الأداة
                </a>
                <a href="#" class="tool-btn secondary">
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
            <i class="fa-brands fa-stack-exchange fa-beat" style="font-size: 1.5rem; color: #cf8026;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الأستخراج</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion3"></i>
        </div>
        <div class="accordion-content" id="accordion3">
          <div class="tools-grid">
            
                                <!-- استخراج الراسلئ -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-image fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">استخراج منشورات بروفايل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع منشورات بروفايل</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-posts.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>


                    <!-- استخراج جهات الاتصا -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-message fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">استخراج تعليقات المنشور</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع تعليقات على منشور علي الانستجرام</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-comments.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>


                    <!-- استخراج مجموعاتي -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-heart fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">استخراج اعجابات المنشور</h4>
                                <p class="tool-description">استخراج قائمة المعجبين من أي منشور أو صفحة</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-likes" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>



                    <!-- استخراج اعضاء -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-user-group fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">استخراج المتابعين والمتابعين</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج قائمة بالمتابعين</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-followers.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>
								<!-- استخراج مشاهدي الاستوري -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-clock fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">استخراج مشاهدي الاستوري</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع مشاهدي الاستوري</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-viewers.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>
								<!-- استخراج جميع مستخدمي الرسائل -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-envelope fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">استخراج جميع مستخدمي الرسائل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه استخراج جميع مستخدمي الرسائل</p>
                                <div class="tool-buttons">
                                    <a href="insta-extract-messages.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
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
            <i class="fa-solid fa-square-plus fa-beat" style="font-size: 1.5rem; color: #cf8026;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات التفاعل</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion4"></i>
        </div>
        <div class="accordion-content" id="accordion4">
          <div class="tools-grid">
            
                                <!-- استخراج الانضمام -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-user-plus fa-beat" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">أداة المتابعة</h4>
                                <p class="tool-description">يمكنك من خلال هذه الادأه لمتابعة مجموعة</p>
                                <div class="tool-buttons">
                                    <a href="#" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>


                    <!-- اضافه اعضاء -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-at fa-beat" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">أداة المنشن</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداة لعمل منشن</p>
                                <div class="tool-buttons">
                                    <a href="wa-add.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
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
            <i class="fa-regular fa-paper-plane fa-beat" style="font-size: 1.5rem; color: #cf8026;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات النشر</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion5"></i>
        </div>
        <div class="accordion-content" id="accordion5">
          <div class="tools-grid">
            
                                <!-- استخراج الانضمام -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-play fa-beat" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">نشر ستوري تلقائي</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداه لمشاركة نشر الاستوري </p>
                                <div class="tool-buttons">
                                    <a href="insta-auto-story.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
                                    <i class="fas fa-video"></i>
                                    شاهد الشرح
                                    </a>
                                </div>
                                </div>


                    <!-- اضافه اعضاء -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                    <i class="fa-solid fa-image fa-beat" style="font-size: 2rem; color: #25D366;"></i>
                                </div>
                                <h4 class="tool-title">نشر بوست تلقائي</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداة لنشر البوست تلقائياً</p>
                                <div class="tool-buttons">
                                    <a href="insta-auto-post.php" class="tool-btn primary">
                                    <i class="fas fa-image"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
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
            <i class="fa-solid fa-handshake-angle fa-beat" style="font-size: 1.5rem; color: #cf8026;"></i>
            <span style="font-size: 1.1rem; font-weight: 600;">أدوات الرسائل</span>
          </div>
          <i class="fas fa-chevron-down accordion-icon" id="icon-accordion6"></i>
        </div>
        <div class="accordion-content" id="accordion6">
          <div class="tools-grid">
            
                                <!-- الفلتر -->
                                <div class="tool-card">
                                <div class="tool-icon" style="background: rgba(37, 211, 102, 0.1);">
                                  <!--  <i class="fa-solid fa-envelope /*fa-beat*/" style="font-size: 2rem; color: #cf8026;"></i>-->
								 <i class="fa-solid fa-envelope" style="font-size: 2rem; color: #cf8026;"></i>
                                </div>
                                <h4 class="tool-title">ارسال الرسائل</h4>
                                <p class="tool-description">يمكنك من خلال هذه الاداه لارسال الرسائل</p>
                                <div class="tool-buttons">
                                    <a href="filter-wa.php" class="tool-btn primary">
                                    <i class="fas fa-play"></i>
                                    استخدم الأداة
                                    </a>
                                    <a href="#" class="tool-btn secondary">
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

<style>
/* Accordion Styles */
.accordion-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.accordion-item {
  background: rgba(102, 126, 234, 0.05);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  overflow: hidden;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.accordion-item:hover {
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.2);
}

.accordion-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.3rem 1.8rem;
  cursor: pointer;
  background: rgba(102, 126, 234, 0.08);
  transition: all 0.3s ease;
  user-select: none;
}

.accordion-header:hover {
  background: rgba(102, 126, 234, 0.15);
}

.accordion-icon {
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  color: var(--primary-color);
  font-size: 1.2rem;
}

.accordion-icon.rotate {
  transform: rotate(180deg);
}

.accordion-content {
  max-height: 0;
  overflow: hidden;
  transition: max-height 0.5s cubic-bezier(0.4, 0, 0.2, 1);
  padding: 0 1.8rem;
}

.accordion-content.active {
  max-height: 3000px;
  padding: 1.8rem;
}

/* Tools Grid */
.tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1.8rem;
}

/* Mobile fixes */
@media (max-width: 768px){
  .accordion-header{ padding: 1rem 1.1rem; }
  .accordion-content{ padding: 0 1.1rem; }
  .accordion-content.active{ padding: 1.1rem; }
  .tools-grid{ grid-template-columns: 1fr; gap: 1rem; }
  .tool-card{ padding: 1.4rem 1.1rem; }
  .tool-description{ min-height: 0; }
  .tool-btn{ width: 100%; justify-content: center; }
}

/* Tool Card - Modern Design */
.tool-card {
  position: relative;
  background: linear-gradient(135deg, var(--card-bg), rgba(102, 126, 234, 0.03));
  border: 1px solid var(--border-color);
  border-radius: 16px;
  padding: 2rem 1.5rem;
  text-align: center;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
}

.tool-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 4px;
  background: linear-gradient(90deg, var(--primary-color), var(--secondary-color));
  transform: scaleX(0);
  transition: transform 0.4s ease;
}

.tool-card:hover::before {
  transform: scaleX(1);
}

.tool-card:hover {
  transform: translateY(-8px);
  box-shadow: 0 12px 30px rgba(102, 126, 234, 0.4);
  border-color: var(--primary-color);
}

.tool-icon {
  width: 90px;
  height: 90px;
  margin: 0 auto 1.3rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 20px;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
}

.tool-icon::after {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 20px;
  background: linear-gradient(135deg, transparent, rgba(255, 255, 255, 0.1));
  opacity: 0;
  transition: opacity 0.3s ease;
}

.tool-card:hover .tool-icon {
  transform: scale(1.15) rotate(-5deg);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.3);
}

.tool-card:hover .tool-icon::after {
  opacity: 1;
}

.tool-title {
  color: var(--text-light);
  font-size: 1.15rem;
  font-weight: 700;
  margin: 0 0 0.8rem 0;
  letter-spacing: -0.02em;
}

.tool-description {
  color: var(--text-gray);
  font-size: 0.92rem;
  line-height: 1.65;
  margin: 0 0 1.8rem 0;
  min-height: 3.3rem;
}

.tool-buttons {
  display: flex;
  gap: 0.8rem;
  justify-content: center;
  flex-wrap: wrap;
}

.tool-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.3rem;
  border-radius: 10px;
  text-decoration: none;
  font-weight: 600;
  font-size: 0.9rem;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.tool-btn::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 0;
  height: 0;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  transform: translate(-50%, -50%);
  transition: width 0.5s, height 0.5s;
}

.tool-btn:hover::before {
  width: 300px;
  height: 300px;
}

.tool-btn.primary {
  background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
  color: #fff;
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
  border: none;
}

.tool-btn.primary:hover {
  transform: translateY(-3px);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.6);
}

.tool-btn.secondary {
  background: rgba(102, 126, 234, 0.1);
  color: var(--primary-color);
  border: 2px solid rgba(102, 126, 234, 0.3);
}

.tool-btn.secondary:hover {
  background: rgba(102, 126, 234, 0.2);
  border-color: var(--primary-color);
  transform: translateY(-3px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.3);
}

/* Light Theme Support */
body.light-theme .tool-title {
  color: #2d3436;
}

body.light-theme .tool-card {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.9), rgba(102, 126, 234, 0.05));
}
</style>

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
