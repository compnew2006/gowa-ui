<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];

$unread_announcements = getUnreadAnnouncements($user_id);

$page_title = "لوحة التحكم | Kingmaster";
$page_css = ['/css/index.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';

$user = getUserByUserId($user_id);
$expiry_date = $user['expiry_date'];
$date_only = !empty($expiry_date) ? explode(' ', $expiry_date)[0] : '';

$is_expired = false;
if (!empty($expiry_date) && strtotime($expiry_date) < time()) {
    $is_expired = true;
}

$package = $user['package'];
if ($package == "0") {
  $package = "تجريبية";
  $name_btn = "اشتري الان";
  $link_urls = "packages.php";
} else {
  $name_btn = "شاهد الشروحات";
  $link_urls = "tut.php";
  $package = getPackageName($package);
}

$lastPosts = getLastPosts();
$totalExtractTrue = getExtractTrueCount($user_id);
$totalCampaigns = getCampaignCount($user_id);
$count = getReferralCountByReferrerId($user_id);
$referrals = getReferralsByReferrerId($user_id);
?>

<!-- Main Content -->
<main class="main-content">
  <?php if (!empty($unread_announcements)): ?>
  <div class="km-banner" id="announcementBanner">
    <div class="km-banner-inner">
      <div class="km-banner-icon"><i class="fas fa-bullhorn"></i></div>
      <div class="km-banner-body">
        <strong class="km-banner-title"><?= htmlspecialchars($unread_announcements[0]['title']) ?></strong>
        <span class="km-banner-text"><?= htmlspecialchars($unread_announcements[0]['message']) ?></span>
      </div>
      <?php if (count($unread_announcements) > 1): ?>
      <span class="km-banner-count"><?= count($unread_announcements) ?> إعلانات</span>
      <?php endif; ?>
    </div>
    <button type="button" class="km-banner-dismiss" onclick="dismissBanner()" aria-label="إغلاق">
      <i class="fas fa-times"></i>
    </button>
  </div>
  <?php endif; ?>

  <!-- Row 1: Welcome + Weekly chart -->
  <div class="km-row km-row--top">
    <div class="content-card km-col km-col--8 welcome-card">
      <div class="welcome-content">
        <h2 class="welcome-title">
          مرحباً بك <?= htmlspecialchars($_SESSION['first_name']); ?>
        </h2>

        <div class="welcome-action-bar<?= $is_expired ? ' is-expired' : '' ?>">
          <div class="action-bar-info">
            <span class="km-chip">
              <i class="fa-solid fa-crown"></i>
              <?= htmlspecialchars($package ?? 'غير محدد') ?>
            </span>

            <span class="km-chip">
              <i class="fa-solid <?= $is_expired ? 'fa-triangle-exclamation' : 'fa-calendar-days' ?>"></i>
              <?php if ($is_expired): ?>
                حسابك منتهي
              <?php else: ?>
                ينتهي: <?= htmlspecialchars($date_only) ?>
              <?php endif; ?>
            </span>
          </div>

          <div class="action-bar-buttons">
            <a href="data-extraction.php" class="btn btn-primary">
              <i class="fas fa-chart-line"></i>
              <span>عرض التفاصيل</span>
            </a>
            <a href="<?= htmlspecialchars($link_urls) ?>" class="btn btn-secondary">
              <i class="fa-solid fa-rocket"></i>
              <span><?= htmlspecialchars($name_btn) ?></span>
            </a>
          </div>
        </div>
      </div>

      <div class="welcome-image-container">
        <img src="images/performanceMarketingBanner.png" alt="" class="welcome-image floating-logo" onerror="this.style.display='none'">
      </div>
      <img src="images/pattern.png" alt="" class="welcome-pattern" onerror="this.style.display='none'">
    </div>

    <div class="content-card km-col km-col--4 km-weekStatsCard">
      <h2><i class="fas fa-chart-line"></i> احصائيات رسائل الاسبوع</h2>
      <p class="km-note">إحصائيات الرسائل اليومية</p>
      <div class="km-chart">
        <canvas id="salesChart" role="img" aria-label="رسم بياني يوضح عدد الرسائل اليومية خلال الأسبوع"></canvas>
      </div>
    </div>
  </div>

  <!-- Row 2: Stats strip (primary + secondary) -->
  <div class="km-row km-row--stats">
    <div class="content-card stats-primary stats-primary--extractions">
      <p class="stats-primary-label">إجمالي الاستخراجات</p>
      <h3 class="stats-primary-value"><?= (int)$totalExtractTrue ?></h3>
      <?php if ((int)$totalExtractTrue === 0): ?>
      <a href="data-extraction.php" class="stats-primary-cta">استخرج بياناتك الأولى <i class="fas fa-arrow-left"></i></a>
      <?php endif; ?>
    </div>

    <div class="stats-secondary">
      <div class="stats-secondary-item">
        <span class="stats-secondary-value stats-secondary--info"><?= (int)$totalCampaigns ?></span>
        <span class="stats-secondary-label">حملة</span>
        <?php if ((int)$totalCampaigns === 0): ?>
        <a href="data-extraction.php" class="stat-hint">ابدأ الأولى <i class="fas fa-arrow-left"></i></a>
        <?php endif; ?>
      </div>

      <div class="stats-secondary-item">
        <span class="stats-secondary-value stats-secondary--warning"><?= (int)$count ?></span>
        <span class="stats-secondary-label">عضو فريق</span>
        <?php if ((int)$count > 0): ?>
        <span class="stats-secondary-delta">+<?= (int)$count ?></span>
        <?php else: ?>
        <a href="profile.php" class="stat-hint">ادعُ أصدقائك <i class="fas fa-arrow-left"></i></a>
        <?php endif; ?>
      </div>

      <div class="stats-secondary-item">
        <span class="stats-secondary-value stats-secondary--primary"><?= number_format((int)$user['points']) ?></span>
        <span class="stats-secondary-label">نقطة</span>
      </div>
    </div>
  </div>

  <!-- Row 3: Charts -->
  <div class="km-row km-row--halves">
    <div class="content-card km-col km-col--6">
      <h2><i class="fas fa-chart-column"></i> إحصائيات استخدام النقاط</h2>
      <p class="card-subtitle">بيانات الأداء لجميع أشهر العام</p>
      <div>
        <canvas id="monthlyChart" role="img" aria-label="رسم بياني يوضح استهلاك النقاط الشهري خلال العام"></canvas>
      </div>
    </div>

    <div class="content-card km-col km-col--6">
      <h2><i class="fas fa-chart-pie"></i> توزيع استخدام الأدوات</h2>
      <p class="card-subtitle">توزيع الاستخدام حسب المنصة</p>
      <div>
        <canvas id="platformsChart" role="img" aria-label="رسم دائري يوضح توزيع استخدام الأدوات حسب المنصة"></canvas>
      </div>
    </div>
  </div>

  <!-- Row 4: Updates & Referrals (bare sections, no card wrapper) -->
  <div class="km-row km-row--feeds">
    <div class="km-col km-col--7">
      <h2 class="km-section-heading"><i class="fas fa-bell"></i> أحدث التحديثات</h2>
      <?php if (!empty($lastPosts)): ?>
      <div class="update-feed">
        <?php foreach ($lastPosts as $post): ?>
          <?php if ($post['typs'] === "New Feature"): ?>
            <div class="update-item update-item--feature">
              <div class="update-item-icon"><i class="fas fa-star"></i></div>
              <div class="update-item-body">
                <h4 class="update-item-type">ميزة جديدة</h4>
                <p class="update-item-text"><?= htmlspecialchars($post['content']) ?></p>
                <span class="update-item-time"><i class="fas fa-clock"></i> <?= htmlspecialchars($post['created_at']) ?></span>
              </div>
            </div>
          <?php elseif ($post['typs'] === "System Update"): ?>
            <div class="update-item update-item--system">
              <div class="update-item-icon"><i class="fas fa-arrows-rotate"></i></div>
              <div class="update-item-body">
                <h4 class="update-item-type">تحديث النظام</h4>
                <p class="update-item-text"><?= htmlspecialchars($post['content']) ?></p>
                <span class="update-item-time"><i class="fas fa-clock"></i> <?= htmlspecialchars($post['created_at']) ?></span>
              </div>
            </div>
          <?php elseif ($post['typs'] === "Maintenance"): ?>
            <div class="update-item update-item--maintenance">
              <div class="update-item-icon"><i class="fas fa-wrench"></i></div>
              <div class="update-item-body">
                <h4 class="update-item-type">صيانة</h4>
                <p class="update-item-text"><?= htmlspecialchars($post['content']) ?></p>
                <span class="update-item-time"><i class="fas fa-clock"></i> <?= htmlspecialchars($post['created_at']) ?></span>
              </div>
            </div>
          <?php endif; ?>
        <?php endforeach; ?>
      </div>
      <?php else: ?>
      <div class="km-empty">
        <div class="km-empty-icon"><i class="fas fa-bell-slash"></i></div>
        <p class="km-empty-heading">لا توجد تحديثات بعد</p>
        <p class="km-empty-text">ستظهر هنا آخر أخبار المنصة والميزات الجديدة فور نشرها</p>
      </div>
      <?php endif; ?>
    </div>

    <div class="km-col km-col--5">
      <h2 class="km-section-heading"><i class="fas fa-user-plus"></i> أحدث الإحالات</h2>
      <?php if (!empty($referrals)): ?>
      <div class="referral-feed">
        <?php foreach ($referrals as $referral): ?>
          <?php
            $img = (!empty($referral['img']) && $referral['img'] != '0')
                ? $referral['img']
                : 'https://i.pravatar.cc/150?u=' . md5($referral['email']);
          ?>
          <div class="referral-item">
            <img src="<?= htmlspecialchars($img) ?>" alt="<?= htmlspecialchars($referral['full_name']) ?>" class="referral-avatar">
            <div class="referral-body">
              <h4 class="referral-name"><?= htmlspecialchars($referral['full_name']) ?></h4>
              <p class="referral-email"><?= htmlspecialchars($referral['email']) ?></p>
              <span class="referral-date"><i class="fas fa-calendar-alt"></i> انضم <?= htmlspecialchars($referral['created_at']) ?></span>
            </div>
            <div class="referral-status"><i class="fas fa-check"></i></div>
          </div>
        <?php endforeach; ?>
      </div>
      <div class="referral-view-more">
        <a href="profile.php" class="btn btn-secondary">
          <span>عرض المزيد</span>
          <i class="fas fa-arrow-left"></i>
        </a>
      </div>
      <?php else: ?>
      <div class="km-empty">
        <div class="km-empty-icon"><i class="fas fa-user-plus"></i></div>
        <p class="km-empty-heading">لم تحصل على أي إحالات بعد</p>
        <p class="km-empty-text">شارك رابط الإحالة الخاص بك مع أصدقائك لبدء بناء فريقك</p>
        <a href="profile.php" class="btn btn-primary km-empty-cta">
          <i class="fas fa-link"></i>
          <span>انسخ رابط الإحالة</span>
        </a>
      </div>
      <?php endif; ?>
    </div>
  </div>
</main>

<?php if (!empty($unread_announcements)): ?>
<script>
function dismissBanner() {
  const banner = document.getElementById('announcementBanner');
  <?= json_encode(array_map(function($a) { return $a['id']; }, $unread_announcements)) ?>.forEach(function(id) {
    fetch('mark_announcement_viewed.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ announcement_id: id })
    });
  });
  banner.style.opacity = '0';
  banner.style.transform = 'translateY(-0.5rem)';
  setTimeout(function() { banner.remove(); }, 200);
}
</script>
<?php endif; ?>

<?php include 'includes/footer.php'; ?>
