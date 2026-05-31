<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];

$page_title = "الشروحات | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<main class="main-content">
  <div class="km-hub">
    <section class="content-card km-page-header">
      <div>
        <div class="km-page-kicker">مركز المساعدة</div>
        <h1 class="km-page-title">شروحات Kingmaster</h1>
        <p class="km-page-copy">
          اختر المسار المناسب لعملك. هذه الصفحة تجمع روابط الشروحات والأدوات الأساسية في مكان واحد حتى لا تنتهي أزرار الشرح إلى صفحة غير موجودة.
        </p>
      </div>
      <a class="btn btn-primary" href="help_center.php">
        <i class="fas fa-circle-question"></i>
        مركز المساعدة
      </a>
    </section>

    <section class="km-feature-grid" id="facebook">
      <article class="km-feature-card">
        <h3><i class="fab fa-facebook"></i> فيسبوك</h3>
        <p>ابدأ بأدوات البحث والاستخراج، ثم راجع النتائج قبل استخدامها في الحملات.</p>
        <div class="km-feature-actions">
          <a class="btn btn-secondary" href="fb-tools.php">الأدوات</a>
          <a class="btn btn-secondary" href="data-extraction.php">استخراج البيانات</a>
        </div>
      </article>

      <article class="km-feature-card" id="whatsapp">
        <h3><i class="fab fa-whatsapp"></i> واتساب</h3>
        <p>راجع الاتصال، قوائم الإرسال، الفلاتر، ومنشئ التدفقات قبل بدء أي حملة كبيرة.</p>
        <div class="km-feature-actions">
          <a class="btn btn-secondary" href="wa-tools.php">الأدوات</a>
          <a class="btn btn-secondary" href="sending-settings.php">إعدادات الإرسال</a>
        </div>
      </article>

      <article class="km-feature-card" id="instagram">
        <h3><i class="fab fa-instagram"></i> انستجرام</h3>
        <p>استخدم أدوات البحث والاستخراج لإعداد قوائم دقيقة قبل المتابعة أو الرسائل.</p>
        <div class="km-feature-actions">
          <a class="btn btn-secondary" href="insta-tools.php">الأدوات</a>
          <a class="btn btn-secondary" href="content.php">محتوى الرسائل</a>
        </div>
      </article>
    </section>

    <section class="content-card">
      <h2><i class="fas fa-list-check"></i> خطوات التشغيل المقترحة</h2>
      <div class="km-feature-grid">
        <article class="km-feature-card">
          <h3>1. جهز الحسابات</h3>
          <p>أضف حسابات المنصات وتأكد من حالة الاتصال قبل تشغيل أي استخراج أو إرسال.</p>
          <a class="btn btn-secondary" href="accounts.php">إدارة الحسابات</a>
        </article>
        <article class="km-feature-card">
          <h3>2. نظم البيانات</h3>
          <p>احفظ الملفات وجهات الاتصال والمحتوى في أقسام واضحة لتقليل الأخطاء أثناء الحملات.</p>
          <a class="btn btn-secondary" href="customers.php">جهات الاتصال</a>
        </article>
        <article class="km-feature-card">
          <h3>3. راقب الرصيد</h3>
          <p>تابع النقاط والمحفظة قبل تنفيذ عمليات كبيرة تحتاج إلى خصم تلقائي.</p>
          <a class="btn btn-secondary" href="wallet.php">المحفظة</a>
        </article>
      </div>
    </section>
  </div>
</main>

<?php include 'includes/footer.php'; ?>
