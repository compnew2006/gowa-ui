<?php
session_start();
error_reporting(E_ALL);
ini_set('display_errors', 1);

if (isset($_POST['access_token'])) {
    $_SESSION['fb_access_token'] = trim($_POST['access_token']);
    $_SESSION['page_id'] = trim($_POST['page_id']);
}

$access_token = $_SESSION['fb_access_token'] ?? '';
$page_id = $_SESSION['page_id'] ?? '';
$analytics_data = null;
$error_message = null;

if (!empty($access_token) && !empty($page_id) && isset($_POST['fetch_data'])) {
    try {
        // معلومات الصفحة
        $page_url = "https://graph.facebook.com/v18.0/{$page_id}?fields=name,fan_count,followers_count,about,category&access_token={$access_token}";
        $page_info = @file_get_contents($page_url);
        if ($page_info === false) throw new Exception("فشل الاتصال بـ Facebook");
        $page_data = json_decode($page_info, true);
        if (isset($page_data['error'])) throw new Exception($page_data['error']['message']);

        // إحصائيات آخر 28 يوم
        $since = time() - (28 * 86400);
        $insights_url = "https://graph.facebook.com/v18.0/{$page_id}/insights?metric=page_impressions,page_engaged_users,page_post_engagements,page_fans_online_per_day&period=day&since={$since}&access_token={$access_token}";
        $insights = @file_get_contents($insights_url);
        $insights_data = json_decode($insights, true);

        // المنشورات الأخيرة مع التفاصيل
        $posts_url = "https://graph.facebook.com/v18.0/{$page_id}/posts?fields=id,message,created_time,shares,likes.summary(true),comments.summary(true),reactions.summary(true)&limit=20&access_token={$access_token}";
        $posts = @file_get_contents($posts_url);
        $posts_data = json_decode($posts, true);

        $analytics_data = [
            'page' => $page_data,
            'insights' => $insights_data['data'] ?? [],
            'posts' => $posts_data['data'] ?? []
        ];

    } catch (Exception $e) {
        $error_message = $e->getMessage();
    }
}

if (isset($_POST['clear_token'])) {
    unset($_SESSION['fb_access_token'], $_SESSION['page_id']);
    $access_token = $page_id = '';
    $analytics_data = null;
}
?>
<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>تحليلات Facebook | Kingmaster</title>
    <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link rel="stylesheet" href="css/styles.css">
    <link rel="stylesheet" href="css/rtl-ltr.css">
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <style>
        :root { color-scheme: dark; }
        body { font-family: 'Cairo', sans-serif; }
        .bg-gradient { position: fixed; inset: 0; z-index: -1; }
        .container { max-width: 1400px; margin: 0 auto; padding: 2rem 1rem; }
        .header-bar { display:flex; align-items:center; justify-content:space-between; gap:1rem; margin-bottom:1.5rem; flex-wrap:wrap; }
        .header-title { font-size:1.8rem; font-weight:700; color: var(--text-light); margin:0; display:flex; align-items:center; gap:.6rem; }
        
        .card { background: var(--glass-bg); border:1px solid var(--glass-border); border-radius: 16px; backdrop-filter: blur(14px); box-shadow: 0 8px 28px rgba(0,0,0,.35); margin-bottom:1.5rem; }
        .card-body { padding: 1.5rem; }
        .card-title { font-size:1.2rem; font-weight:700; color:#fff; margin:0 0 1rem; display:flex; align-items:center; gap:.6rem; }
        
        .stats-grid { display:grid; gap:1rem; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); }
        .stat { background: rgba(255,255,255,.06); border:1px solid var(--glass-border); border-radius:12px; padding: 1rem; text-align:center; }
        .stat .label { color:#9ca3af; font-size:.85rem; margin-bottom:.5rem; }
        .stat .value { color:#fff; font-size:1.8rem; font-weight:700; }
        .stat .icon { width:50px; height:50px; border-radius:10px; display:flex; align-items:center; justify-content:center; margin:0 auto 0.5rem; font-size:1.5rem; }
        .stat .icon.blue { background: rgba(59,130,246,.15); color:#60a5fa; }
        .stat .icon.green { background: rgba(34,197,94,.15); color:#4ade80; }
        .stat .icon.purple { background: rgba(168,85,247,.15); color:#c084fc; }
        .stat .icon.orange { background: rgba(251,146,60,.15); color:#fb923c; }
        
        .chart-container { position: relative; height: 350px; margin: 1rem 0; }
        
        .form-group { margin-bottom: 1rem; }
        .form-group label { display:block; margin-bottom:.5rem; color:#e5e7eb; font-weight:600; }
        .form-group input { width:100%; padding:.75rem; border:1px solid var(--glass-border); border-radius:10px; background:rgba(255,255,255,.05); color:#fff; font-family:'Cairo',sans-serif; }
        .form-group input:focus { outline:none; border-color:#60a5fa; background:rgba(255,255,255,.08); }
        
        .btn { display:inline-flex; align-items:center; gap:.5rem; padding:.75rem 1.5rem; border:none; border-radius:10px; font-family:'Cairo',sans-serif; font-weight:600; cursor:pointer; transition:all .2s; }
        .btn-primary { background:linear-gradient(135deg, #667eea, #764ba2); color:#fff; }
        .btn-primary:hover { transform:translateY(-2px); box-shadow:0 8px 20px rgba(102,126,234,.4); }
        .btn-danger { background:#ef4444; color:#fff; }
        .btn-sm { padding:.5rem 1rem; font-size:.9rem; }
        
        .alert { padding:1rem 1.25rem; border-radius:12px; margin-bottom:1rem; }
        .alert-error { background:rgba(239,68,68,.1); border:1px solid #ef4444; color:#fca5a5; }
        
        .post-item { background:rgba(255,255,255,.03); border:1px solid var(--glass-border); border-radius:12px; padding:1rem; margin-bottom:1rem; transition:all .2s; }
        .post-item:hover { background:rgba(255,255,255,.06); }
        .post-message { color:#e5e7eb; margin-bottom:.75rem; font-size:.95rem; line-height:1.6; }
        .post-stats { display:flex; gap:1.5rem; flex-wrap:wrap; }
        .post-stat { display:flex; align-items:center; gap:.4rem; color:#9ca3af; font-size:.85rem; }
        .post-stat i { color:#60a5fa; }
        .post-date { color:#9ca3af; font-size:.8rem; margin-top:.5rem; }
        
        .info-box { background:rgba(59,130,246,.1); border:1px solid #3b82f6; border-radius:12px; padding:1.25rem; margin-bottom:1.5rem; }
        .info-box h3 { color:#60a5fa; margin:0 0 .75rem; font-size:1rem; }
        .info-box ul { margin:0; padding-right:1.5rem; color:#9ca3af; }
        .info-box li { margin-bottom:.4rem; }
        
        .grid-2 { display:grid; gap:1rem; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); }
        @media(max-width:768px){ .stats-grid{ grid-template-columns:repeat(2,1fr); } }
    </style>
</head>
<body class="rtl ar">
    <div class="bg-gradient"></div>
    <div class="container">
        <div class="header-bar">
            <h1 class="header-title"><i class="fab fa-facebook"></i> تحليلات Facebook</h1>
        </div>

        <?php if ($error_message): ?>
            <div class="alert alert-error">
                <i class="fas fa-exclamation-circle"></i> <?= htmlspecialchars($error_message) ?>
            </div>
        <?php endif; ?>

        <?php if (!$analytics_data): ?>
            <div class="card">
                <div class="card-body">
                    <h2 class="card-title"><i class="fas fa-key"></i> إضافة Access Token</h2>
                    
                    <div class="info-box">
                        <h3><i class="fas fa-info-circle"></i> كيفية الحصول على Access Token:</h3>
                        <ul>
                            <li>اذهب إلى <a href="https://developers.facebook.com/tools/explorer/" target="_blank" style="color:#60a5fa;">Graph API Explorer</a></li>
                            <li>اختر صفحتك واضف الصلاحيات: pages_show_list, pages_read_engagement</li>
                            <li>اضغط "Generate Access Token" وانسخه</li>
                        </ul>
                    </div>

                    <form method="POST">
                        <div class="form-group">
                            <label><i class="fas fa-id-card"></i> Page ID</label>
                            <input type="text" name="page_id" placeholder="123456789012345" required value="<?= htmlspecialchars($page_id) ?>">
                        </div>
                        <div class="form-group">
                            <label><i class="fas fa-key"></i> Access Token</label>
                            <input type="text" name="access_token" placeholder="الصق الـ Token هنا" required value="<?= htmlspecialchars($access_token) ?>">
                        </div>
                        <button type="submit" name="fetch_data" class="btn btn-primary">
                            <i class="fas fa-chart-line"></i> عرض التحليلات
                        </button>
                    </form>
                </div>
            </div>
        <?php else: ?>
            <form method="POST" style="margin-bottom:1.5rem;">
                <button type="submit" name="clear_token" class="btn btn-danger btn-sm">
                    <i class="fas fa-times"></i> مسح البيانات
                </button>
                <button type="submit" name="fetch_data" class="btn btn-primary btn-sm">
                    <i class="fas fa-sync-alt"></i> تحديث
                </button>
            </form>

            <!-- معلومات الصفحة -->
            <div class="card">
                <div class="card-body">
                    <h2 class="card-title"><i class="fas fa-info-circle"></i> معلومات الصفحة</h2>
                    <div class="stats-grid">
                        <div class="stat">
                            <div class="icon blue"><i class="fab fa-facebook-f"></i></div>
                            <div class="label">اسم الصفحة</div>
                            <div class="value" style="font-size:1.2rem;"><?= htmlspecialchars($analytics_data['page']['name']) ?></div>
                        </div>
                        <div class="stat">
                            <div class="icon green"><i class="fas fa-users"></i></div>
                            <div class="label">المعجبين</div>
                            <div class="value"><?= number_format($analytics_data['page']['fan_count'] ?? 0) ?></div>
                        </div>
                        <div class="stat">
                            <div class="icon purple"><i class="fas fa-user-friends"></i></div>
                            <div class="label">المتابعين</div>
                            <div class="value"><?= number_format($analytics_data['page']['followers_count'] ?? 0) ?></div>
                        </div>
                        <div class="stat">
                            <div class="icon orange"><i class="fas fa-tag"></i></div>
                            <div class="label">التصنيف</div>
                            <div class="value" style="font-size:1rem;"><?= htmlspecialchars($analytics_data['page']['category'] ?? 'N/A') ?></div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- الرسوم البيانية -->
            <?php if (!empty($analytics_data['insights'])): ?>
                <div class="grid-2">
                    <div class="card">
                        <div class="card-body">
                            <h2 class="card-title"><i class="fas fa-chart-line"></i> مرات الظهور والتفاعل</h2>
                            <div class="chart-container">
                                <canvas id="mainChart"></canvas>
                            </div>
                        </div>
                    </div>

                    <div class="card">
                        <div class="card-body">
                            <h2 class="card-title"><i class="fas fa-clock"></i> أفضل وقت تواجد الجمهور</h2>
                            <div class="chart-container">
                                <canvas id="onlineChart"></canvas>
                            </div>
                        </div>
                    </div>
                </div>
            <?php endif; ?>

            <!-- إحصائيات المنشورات -->
            <?php if (!empty($analytics_data['posts'])): 
                $totalLikes = 0;
                $totalComments = 0;
                $totalShares = 0;
                foreach ($analytics_data['posts'] as $post) {
                    $totalLikes += $post['likes']['summary']['total_count'] ?? 0;
                    $totalComments += $post['comments']['summary']['total_count'] ?? 0;
                    $totalShares += $post['shares']['count'] ?? 0;
                }
            ?>
                <div class="card">
                    <div class="card-body">
                        <h2 class="card-title"><i class="fas fa-chart-bar"></i> إجمالي تفاعلات المنشورات</h2>
                        <div class="stats-grid">
                            <div class="stat">
                                <div class="icon blue"><i class="fas fa-thumbs-up"></i></div>
                                <div class="label">إجمالي الإعجابات</div>
                                <div class="value"><?= number_format($totalLikes) ?></div>
                            </div>
                            <div class="stat">
                                <div class="icon green"><i class="fas fa-comment"></i></div>
                                <div class="label">إجمالي التعليقات</div>
                                <div class="value"><?= number_format($totalComments) ?></div>
                            </div>
                            <div class="stat">
                                <div class="icon purple"><i class="fas fa-share"></i></div>
                                <div class="label">إجمالي المشاركات</div>
                                <div class="value"><?= number_format($totalShares) ?></div>
                            </div>
                            <div class="stat">
                                <div class="icon orange"><i class="fas fa-heart"></i></div>
                                <div class="label">إجمالي التفاعلات</div>
                                <div class="value"><?= number_format($totalLikes + $totalComments + $totalShares) ?></div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- أهم المنشورات -->
                <div class="card">
                    <div class="card-body">
                        <h2 class="card-title"><i class="fas fa-fire"></i> أهم المنشورات (الأكثر تفاعلاً)</h2>
                        <?php
                        usort($analytics_data['posts'], function($a, $b) {
                            $engA = ($a['likes']['summary']['total_count'] ?? 0) + ($a['comments']['summary']['total_count'] ?? 0) + ($a['shares']['count'] ?? 0);
                            $engB = ($b['likes']['summary']['total_count'] ?? 0) + ($b['comments']['summary']['total_count'] ?? 0) + ($b['shares']['count'] ?? 0);
                            return $engB - $engA;
                        });
                        foreach (array_slice($analytics_data['posts'], 0, 5) as $post): 
                            $likes = $post['likes']['summary']['total_count'] ?? 0;
                            $comments = $post['comments']['summary']['total_count'] ?? 0;
                            $shares = $post['shares']['count'] ?? 0;
                        ?>
                            <div class="post-item">
                                <?php if (!empty($post['message'])): ?>
                                    <div class="post-message"><?= nl2br(htmlspecialchars(mb_substr($post['message'], 0, 200))) ?><?= mb_strlen($post['message']) > 200 ? '...' : '' ?></div>
                                <?php endif; ?>
                                <div class="post-stats">
                                    <div class="post-stat">
                                        <i class="fas fa-thumbs-up"></i>
                                        <span><?= number_format($likes) ?> إعجاب</span>
                                    </div>
                                    <div class="post-stat">
                                        <i class="fas fa-comment"></i>
                                        <span><?= number_format($comments) ?> تعليق</span>
                                    </div>
                                    <div class="post-stat">
                                        <i class="fas fa-share"></i>
                                        <span><?= number_format($shares) ?> مشاركة</span>
                                    </div>
                                </div>
                                <div class="post-date">
                                    <i class="fas fa-clock"></i>
                                    <?= date('Y-m-d H:i', strtotime($post['created_time'])) ?>
                                </div>
                            </div>
                        <?php endforeach; ?>
                    </div>
                </div>
            <?php endif; ?>
        <?php endif; ?>
    </div>

    <?php if ($analytics_data && !empty($analytics_data['insights'])): ?>
    <script>
        Chart.defaults.color = '#9ca3af';
        Chart.defaults.borderColor = 'rgba(255,255,255,0.1)';
        
        const insights = <?= json_encode($analytics_data['insights']) ?>;
        
        // Main Chart
        const impressions = insights.find(i => i.name === 'page_impressions');
        const engaged = insights.find(i => i.name === 'page_engaged_users');
        const engagements = insights.find(i => i.name === 'page_post_engagements');
        
        if (impressions) {
            const labels = impressions.values.slice(-14).map(v => {
                const d = new Date(v.end_time);
                return d.toLocaleDateString('ar-SA', {month:'short', day:'numeric'});
            });
            
            new Chart(document.getElementById('mainChart'), {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: 'مرات الظهور',
                        data: impressions.values.slice(-14).map(v => v.value),
                        borderColor: '#60a5fa',
                        backgroundColor: 'rgba(96,165,250,0.1)',
                        tension: 0.4,
                        fill: true
                    }, {
                        label: 'المستخدمون المتفاعلون',
                        data: engaged ? engaged.values.slice(-14).map(v => v.value) : [],
                        borderColor: '#4ade80',
                        backgroundColor: 'rgba(74,222,128,0.1)',
                        tension: 0.4,
                        fill: true
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { 
                            labels: { font: { family: 'Cairo' } }
                        }
                    },
                    scales: {
                        y: { beginAtZero: true },
                        x: { grid: { display: false } }
                    }
                }
            });
        }
        
        // Online Chart
        const online = insights.find(i => i.name === 'page_fans_online_per_day');
        if (online && online.values && online.values[0] && online.values[0].value) {
            const onlineData = online.values[0].value;
            const hours = Object.keys(onlineData).sort();
            const values = hours.map(h => onlineData[h]);
            
            new Chart(document.getElementById('onlineChart'), {
                type: 'bar',
                data: {
                    labels: hours.map(h => {
                        const hour = parseInt(h);
                        return hour + ':00';
                    }),
                    datasets: [{
                        label: 'المعجبين المتصلين',
                        data: values,
                        backgroundColor: 'rgba(168,85,247,0.7)',
                        borderColor: '#a855f7',
                        borderWidth: 2,
                        borderRadius: 6
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: false }
                    },
                    scales: {
                        y: { beginAtZero: true }
                    }
                }
            });
        }
    </script>
    <?php endif; ?>
</body>
</html>
