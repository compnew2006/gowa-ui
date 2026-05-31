<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];

$page_title = "المطورين | Kingmaster";
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
        <div class="km-page-kicker">منطقة المطورين</div>
        <h1 class="km-page-title">أهداف التطوير والتكامل</h1>
        <p class="km-page-copy">
          هذه الصفحة تجمع المسارات التشغيلية المرتبطة بالتكاملات، البيانات، والأدوات التي يعتمد عليها فريق العمل.
        </p>
      </div>
      <a class="btn btn-primary" href="help_center.php">
        <i class="fas fa-book-open"></i>
        الوثائق
      </a>
    </section>

    <section class="km-feature-grid">
      <article class="km-feature-card">
        <h3><i class="fas fa-database"></i> جودة البيانات</h3>
        <p>تأكد من أن الاستخراجات محفوظة ومنظمة قبل نقلها إلى الحملات أو قوائم العملاء.</p>
        <ul>
          <li>مراجعة نتائج الاستخراج</li>
          <li>تنظيف القوائم قبل الإرسال</li>
          <li>تجنب التكرار في جهات الاتصال</li>
        </ul>
        <div class="km-feature-actions">
          <a class="btn btn-secondary" href="data-extraction.php">استخراج البيانات</a>
          <a class="btn btn-secondary" href="files.php">الملفات</a>
        </div>
      </article>

      <article class="km-feature-card">
        <h3><i class="fas fa-plug"></i> تكاملات المنصات</h3>
        <p>تابع حالة الحسابات ونقاط الاتصال قبل تشغيل الأدوات التي تعتمد على منصات خارجية.</p>
        <ul>
          <li>اتصال واتساب</li>
          <li>حسابات فيسبوك وانستجرام</li>
          <li>إعدادات الإرسال</li>
        </ul>
        <div class="km-feature-actions">
          <a class="btn btn-secondary" href="accounts.php">الحسابات</a>
          <a class="btn btn-secondary" href="sending-settings.php">الإعدادات</a>
        </div>
      </article>

      <article class="km-feature-card">
        <h3><i class="fas fa-chart-line"></i> مراقبة التشغيل</h3>
        <p>تابع العمليات المالية، النقاط، وحالة النظام من الصفحات المركزية قبل اتخاذ قرارات تشغيلية.</p>
        <ul>
          <li>المحفظة والتحويلات</li>
          <li>شحن النقاط</li>
          <li>إحصائيات النظام</li>
        </ul>
        <div class="km-feature-actions">
          <a class="btn btn-secondary" href="wallet.php">المحفظة</a>
          <a class="btn btn-secondary" href="points.php">النقاط</a>
        </div>
      </article>
    </section>
  </div>
</main>

<?php include 'includes/footer.php'; ?>
