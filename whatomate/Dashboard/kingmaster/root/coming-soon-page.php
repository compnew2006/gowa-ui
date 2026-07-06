<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}

require_once 'includes/functions.php';

$user_id = $_SESSION['user_id'];
$page_title = "قريباً | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<main class="main-content">
  <section class="content-card km-empty km-page-empty">
    <div class="km-empty-icon">
      <i class="fa-solid fa-hourglass-half"></i>
    </div>
    <h1 class="km-empty-heading">قريباً</h1>
    <p class="km-empty-text">نعمل حالياً على تجهيز هذه الصفحة وإضافة المزيد من المحتوى والخدمات قريباً.</p>
    <span class="km-chip">الصفحة تحت التجهيز</span>
    <div class="km-empty-cta">
      <a class="btn btn-primary" href="index.php">
        <i class="fa-solid fa-arrow-right"></i>
        <span>العودة للرئيسية</span>
      </a>
    </div>
  </section>
</main>

</body>
</html>
