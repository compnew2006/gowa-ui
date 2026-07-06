<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "الإحصائيات | Kingmaster";
$page_css = ['/css/toppages.css'];
$skip_chartjs = true;
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<main class="admin-container">
    <header class="admin-header">
        <h1 class="admin-title">
            <i class="fas fa-crown" aria-hidden="true"></i>
            لوحة تحكم المسؤول
        </h1>
        <p class="admin-subtitle">إحصائيات ومعلومات شاملة عن النظام</p>
        <div class="km-stats-meta" id="statsMeta"></div>
    </header>

    <div class="km-stats-error" id="statsError" role="alert" hidden>
        <i class="fas fa-exclamation-triangle" aria-hidden="true"></i>
        <span>تعذر تحميل الإحصائيات</span>
        <button type="button" id="statsRetry">إعادة المحاولة</button>
    </div>

    <section class="stats-grid" id="statsGrid" role="region" aria-label="إحصائيات النظام" aria-live="polite" aria-busy="true">
        <div class="stat-card" aria-label="إجمالي الرسائل">
            <div class="stat-icon"><i class="fas fa-envelope" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="totalMessages" role="status" aria-label="إجمالي الرسائل">&nbsp;</div>
            <div class="stat-label">إجمالي الرسائل</div>
        </div>
        <div class="stat-card" aria-label="إجمالي المستخدمين">
            <div class="stat-icon"><i class="fas fa-users" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="totalUsers" role="status" aria-label="إجمالي المستخدمين">&nbsp;</div>
            <div class="stat-label">إجمالي المستخدمين</div>
        </div>
        <div class="stat-card" aria-label="إجمالي الحسابات">
            <div class="stat-icon"><i class="fas fa-user-circle" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="totalAccounts" role="status" aria-label="إجمالي الحسابات">&nbsp;</div>
            <div class="stat-label">إجمالي الحسابات</div>
        </div>
        <div class="stat-card" aria-label="إجمالي الاستخراجات">
            <div class="stat-icon"><i class="fas fa-download" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="totalExtractions" role="status" aria-label="إجمالي الاستخراجات">&nbsp;</div>
            <div class="stat-label">إجمالي الاستخراجات</div>
        </div>
        <div class="stat-card" aria-label="المستخدمون النشطون">
            <div class="stat-icon"><i class="fas fa-user-check" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="activeUsers" role="status" aria-label="المستخدمون النشطون الآن">&nbsp;</div>
            <div class="stat-label">المستخدمون النشطون الآن</div>
        </div>
        <div class="stat-card" aria-label="المستخدمون غير النشطين">
            <div class="stat-icon"><i class="fas fa-user-slash" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="inactiveUsers" role="status" aria-label="المستخدمون غير النشطين">&nbsp;</div>
            <div class="stat-label">المستخدمون غير النشطين</div>
        </div>
        <div class="stat-card" aria-label="إجمالي الحملات">
            <div class="stat-icon"><i class="fas fa-bullhorn" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="totalCampaigns" role="status" aria-label="إجمالي الحملات">&nbsp;</div>
            <div class="stat-label">إجمالي الحملات</div>
        </div>
        <div class="stat-card" aria-label="الحملات الجارية">
            <div class="stat-icon"><i class="fas fa-play-circle" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="runningCampaigns" role="status" aria-label="الحملات الجارية الآن">&nbsp;</div>
            <div class="stat-label">الحملات الجارية الآن</div>
        </div>
        <div class="stat-card" aria-label="الحملات المتوقفة">
            <div class="stat-icon"><i class="fas fa-stop-circle" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="stoppedCampaigns" role="status" aria-label="الحملات المتوقفة">&nbsp;</div>
            <div class="stat-label">الحملات المتوقفة</div>
        </div>
        <div class="stat-card" aria-label="الحملات المنتهية">
            <div class="stat-icon"><i class="fas fa-check-circle" aria-hidden="true"></i></div>
            <div class="stat-value km-skeleton-bar" id="finishedCampaigns" role="status" aria-label="الحملات المنتهية">&nbsp;</div>
            <div class="stat-label">الحملات المنتهية</div>
        </div>
    </section>

    <section class="top-section">
        <!-- أفضل أداة -->
        <div class="top-card">
            <h2 class="top-card-title">
                <i class="fas fa-trophy" aria-hidden="true"></i>
                أفضل الأدوات استخداماً
            </h2>
            <div id="topTools">
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fas fa-tools" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">جاري التحميل...</div>
                    </div>
                    <div class="top-item-count">-</div>
                </div>
            </div>
        </div>

        <!-- أفضل منصة -->
        <div class="top-card">
            <h2 class="top-card-title">
                <i class="fas fa-star" aria-hidden="true"></i>
                أفضل المنصات استخداماً
            </h2>
            <div id="topPlatforms">
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fab fa-facebook" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">فيسبوك</div>
                    </div>
                    <div class="top-item-count">0</div>
                </div>
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fab fa-whatsapp" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">واتساب</div>
                    </div>
                    <div class="top-item-count">0</div>
                </div>
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fab fa-instagram" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">انستجرام</div>
                    </div>
                    <div class="top-item-count">0</div>
                </div>
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fab fa-telegram" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">تليجرام</div>
                    </div>
                    <div class="top-item-count">0</div>
                </div>
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fas fa-envelope" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">بريد</div>
                    </div>
                    <div class="top-item-count">0</div>
                </div>
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fas fa-map" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">جوجل ماب</div>
                    </div>
                    <div class="top-item-count">0</div>
                </div>
                <div class="top-item">
                    <div class="top-item-info">
                        <div class="top-item-icon">
                            <i class="fas fa-briefcase" aria-hidden="true"></i>
                        </div>
                        <div class="top-item-name">أعمال</div>
                    </div>
                    <div class="top-item-count">0</div>
                </div>
            </div>
        </div>
    </section>
</main>

<script>
// --- State ---
var kmFirstLoad = true;
var kmRefreshTimer = null;
var kmReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

// --- Helpers ---
function kmFormatNum(n) {
    return (typeof n === 'number') ? n.toLocaleString() : '0';
}

function kmFormatTime(date) {
    return date.toLocaleTimeString('ar-EG', { hour: '2-digit', minute: '2-digit' });
}

function kmSetMeta(html) {
    var el = document.getElementById('statsMeta');
    if (el) el.innerHTML = html;
}

function kmShowError(show) {
    var el = document.getElementById('statsError');
    if (el) el.hidden = !show;
}

function kmSetBusy(busy) {
    var el = document.getElementById('statsGrid');
    if (el) el.setAttribute('aria-busy', busy ? 'true' : 'false');
}

function kmRemoveSkeleton(id) {
    var el = document.getElementById(id);
    if (el) el.classList.remove('km-skeleton-bar');
}

// --- Number animation (rAF-based, respects prefers-reduced-motion) ---
function kmAnimateValue(element, end) {
    if (kmReducedMotion || !element) {
        element.textContent = kmFormatNum(end);
        return;
    }
    var start = parseInt(element.textContent.replace(/[^\d-]/g, ''), 10) || 0;
    if (start === end) { element.textContent = kmFormatNum(end); return; }

    var duration = 600;
    var startTime = null;
    function step(ts) {
        if (!startTime) startTime = ts;
        var progress = Math.min((ts - startTime) / duration, 1);
        var eased = 1 - Math.pow(1 - progress, 4);
        var current = Math.round(start + (end - start) * eased);
        element.textContent = kmFormatNum(current);
        if (progress < 1) requestAnimationFrame(step);
    }
    requestAnimationFrame(step);
}

// --- Load statistics ---
function loadStatistics() {
    kmShowError(false);
    kmSetBusy(true);

    fetch('api/get_admin_statistics.php', {
        headers: window.KM_CSRF_TOKEN ? { 'X-CSRF-Token': window.KM_CSRF_TOKEN } : {}
    })
    .then(function(response) {
        if (!response.ok) throw new Error('HTTP ' + response.status);
        return response.json();
    })
    .then(function(data) {
        if (!data.success) throw new Error(data.message || 'API returned failure');
        var s = data.stats;

        var metrics = [
            ['totalMessages',    s.totalMessages],
            ['totalUsers',       s.totalUsers],
            ['totalAccounts',    s.totalAccounts],
            ['totalExtractions', s.totalExtractions],
            ['activeUsers',      s.activeUsers],
            ['inactiveUsers',    s.inactiveUsers],
            ['totalCampaigns',   s.totalCampaigns],
            ['runningCampaigns', s.runningCampaigns],
            ['stoppedCampaigns', s.stoppedCampaigns],
            ['finishedCampaigns',s.finishedCampaigns]
        ];

        metrics.forEach(function(pair) {
            var el = document.getElementById(pair[0]);
            if (el) {
                kmRemoveSkeleton(pair[0]);
                if (kmFirstLoad) {
                    el.textContent = kmFormatNum(pair[1]);
                } else {
                    kmAnimateValue(el, pair[1]);
                }
            }
        });

        kmFirstLoad = false;
        renderTopTools(s.topTools);
        renderTopPlatforms(s.topPlatforms);

        kmSetMeta('<span class="km-meta-updated">آخر تحديث: ' + kmFormatTime(new Date()) + '</span>');
        kmSetBusy(false);
        kmShowError(false);
    })
    .catch(function(error) {
        console.error('Error loading statistics:', error);
        kmSetBusy(false);
        kmShowError(true);
    });
}

function retryLoadStatistics() {
    kmShowError(false);
    loadStatistics();
}

// --- Wire retry button ---
document.getElementById('statsRetry').addEventListener('click', retryLoadStatistics);

// --- Top tools ---
function renderTopTools(tools) {
    var container = document.getElementById('topTools');
    if (!tools || tools.length === 0) {
        container.innerHTML =
            '<div class="top-item">' +
                '<div class="top-item-info">' +
                    '<div class="top-item-icon"><i class="fas fa-info-circle" aria-hidden="true"></i></div>' +
                    '<div class="top-item-name">لا توجد بيانات</div>' +
                '</div>' +
                '<div class="top-item-count">0</div>' +
            '</div>';
        return;
    }
    container.innerHTML = tools.map(function(tool) {
        return '<div class="top-item">' +
            '<div class="top-item-info">' +
                '<div class="top-item-icon"><i class="' + getToolIcon(tool.name) + '" aria-hidden="true"></i></div>' +
                '<div class="top-item-name">' + tool.name + '</div>' +
            '</div>' +
            '<div class="top-item-count">' + kmFormatNum(tool.count) + '</div>' +
        '</div>';
    }).join('');
}

// --- Top platforms ---
function renderTopPlatforms(platforms) {
    if (!platforms) return;
    var platformIcons = {
        'facebook':   'fab fa-facebook',
        'whatsapp':   'fab fa-whatsapp',
        'instagram':  'fab fa-instagram',
        'telegram':   'fab fa-telegram',
        'email':      'fas fa-envelope',
        'google_map': 'fas fa-map',
        'business':   'fas fa-briefcase'
    };
    Object.keys(platforms).forEach(function(platform) {
        var count = platforms[platform] || 0;
        var items = document.querySelectorAll('#topPlatforms .top-item');
        items.forEach(function(item) {
            var icon = item.querySelector('i');
            if (icon && icon.className === platformIcons[platform]) {
                var countEl = item.querySelector('.top-item-count');
                if (countEl) countEl.textContent = kmFormatNum(count);
            }
        });
    });
}

function getToolIcon(toolName) {
    var icons = {
        'استخراج': 'fas fa-download',
        'إرسال':   'fas fa-paper-plane',
        'تحليل':   'fas fa-chart-line',
        'default':  'fas fa-tools'
    };
    return icons[toolName] || icons['default'];
}

// --- Init ---
loadStatistics();
kmRefreshTimer = setInterval(loadStatistics, 30000);

// Stop refresh when tab is hidden, resume when visible
document.addEventListener('visibilitychange', function() {
    if (kmRefreshTimer) { clearInterval(kmRefreshTimer); kmRefreshTimer = null; }
    if (!document.hidden) {
        loadStatistics();
        kmRefreshTimer = setInterval(loadStatistics, 30000);
    }
});
</script>

<?php include 'includes/footer.php'; ?>
