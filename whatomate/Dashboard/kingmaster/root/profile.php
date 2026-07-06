<?php
 
 
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "الملف الشخصي | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
 

$user = getUserData($user_id);
$isExpired = (strtotime($user['expiry_date']) < time());

 $activity_log = getActivityLog($user_id);

 
?>

<style>
    .profile-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }

    .profile-header {
        margin-bottom: 30px;
    }

    .profile-title {
        font-size: 28px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
        margin-bottom: 10px;
    }

    .profile-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .profile-grid {
        display: grid;
        grid-template-columns: 1fr 1.5fr;
        gap: 25px;
        margin-bottom: 25px;
    }

    .left-column {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .stats-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 15px;
        flex: 1;
    }

    /* بطاقة المستخدم */
    .user-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 25px;
        border: 1px solid var(--border-color);
        text-align: center;
        transition: all 0.3s ease;
    }

    .user-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.2);
    }

    .user-avatar {
        width: 120px;
        height: 120px;
        border-radius: 50%;
        margin: 0 auto 15px;
        border: 5px solid #667eea;
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3);
        transition: all 0.3s ease;
    }

    .user-avatar:hover {
        transform: scale(1.05);
        box-shadow: 0 15px 40px rgba(102, 126, 234, 0.5);
    }

    .user-name {
        font-size: 22px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 6px;
        font-family: 'Cairo', sans-serif;
    }

    .user-email {
        font-size: 14px;
        color: var(--text-secondary);
        margin-bottom: 15px;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
    }

    .edit-profile-btn {
        padding: 10px 25px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 10px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
    }

    .edit-profile-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
    }

    /* بطاقات النقاط والإحالات */
    .stat-card {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 20px;
        border: 1px solid var(--border-color);
        text-align: center;
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        height: 100%;
    }

    .stat-card::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 4px;
        background: linear-gradient(90deg, #667eea, #764ba2);
    }

    .stat-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 25px rgba(102, 126, 234, 0.2);
    }

    .stat-icon {
        font-size: 40px;
        margin-bottom: 15px;
        animation: statIconFloat 3s ease-in-out infinite;
    }

    @keyframes statIconFloat {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-5px); }
    }

    .stat-value {
        font-size: 36px;
        font-weight: 800;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        margin-bottom: 8px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .stat-label {
        font-size: 15px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
    }

    .points-icon {
        color: #f59e0b;
    }

    .referrals-icon {
        color: #10b981;
    }

    /* بطاقة الباقة */
    .plan-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        border: 1px solid var(--border-color);
        transition: all 0.3s ease;
    }

    .plan-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(245, 158, 11, 0.2);
    }

    .plan-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 20px;
        padding-bottom: 20px;
        border-bottom: 2px solid var(--border-color);
    }

    .plan-name {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 24px;
        font-weight: 700;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }

    .plan-icon {
        font-size: 32px;
        animation: iconFloat 3s ease-in-out infinite;
    }

    @keyframes iconFloat {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-10px); }
    }

    .plan-badge {
        padding: 8px 16px;
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);
        color: white;
        border-radius: 20px;
        font-size: 13px;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
    }

    .plan-features {
        margin-bottom: 25px;
    }

    .feature-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 0;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        border-bottom: 1px solid var(--border-color);
    }

    .feature-item:last-child {
        border-bottom: none;
    }

    .feature-item i {
        color: #10b981;
        font-size: 18px;
    }

    .plan-dates {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 15px;
    }

    .date-box {
        background: var(--bg-primary);
        padding: 15px;
        border-radius: 12px;
        border: 2px solid var(--border-color);
    }

    .date-label {
        font-size: 12px;
        color: var(--text-secondary);
        margin-bottom: 5px;
        font-family: 'Cairo', sans-serif;
    }

    .date-value {
        font-size: 16px;
        font-weight: 700;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }

    .days-remaining {
        margin-top: 15px;
        padding: 15px;
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
        border-radius: 12px;
        text-align: center;
    }

    .days-remaining-text {
        font-size: 14px;
        color: var(--text-secondary);
        margin-bottom: 5px;
    }

    .days-remaining-count {
        font-size: 28px;
        font-weight: 800;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        font-family: 'Cairo', sans-serif;
    }

    /* Activity Log */
    .activity-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        border: 1px solid var(--border-color);
    }

    .activity-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 25px;
        padding-bottom: 15px;
        border-bottom: 2px solid var(--border-color);
    }

    .activity-title {
        font-size: 22px;
        font-weight: 700;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }

    .activity-list {
        max-height: 500px;
        overflow-y: auto;
    }

    .activity-item {
        display: flex;
        align-items: center;
        gap: 15px;
        padding: 15px;
        background: var(--bg-primary);
        border-radius: 12px;
        margin-bottom: 12px;
        border: 1px solid var(--border-color);
        transition: all 0.3s ease;
    }

    .activity-item:hover {
        transform: translateX(-5px);
        box-shadow: 0 5px 15px rgba(0, 0, 0, 0.1);
    }

    .activity-icon {
        width: 45px;
        height: 45px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 18px;
        color: white;
        flex-shrink: 0;
    }

    .activity-content {
        flex: 1;
    }

    .activity-action {
        font-size: 15px;
        font-weight: 600;
        color: var(--text-primary);
        margin-bottom: 3px;
        font-family: 'Cairo', sans-serif;
    }

    .activity-time {
        font-size: 13px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
    }

    /* Light Theme Support */
    body.light-theme .user-card,
    body.light-theme .plan-card,
    body.light-theme .activity-card,
    body.light-theme .stat-card {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .date-box,
    body.light-theme .activity-item {
        background: #f8f9fa;
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .user-name,
    body.light-theme .plan-name,
    body.light-theme .activity-title,
    body.light-theme .date-value,
    body.light-theme .feature-item,
    body.light-theme .activity-action,
    body.light-theme .stat-value {
        color: #2d3436;
    }

    body.light-theme .user-email,
    body.light-theme .date-label,
    body.light-theme .days-remaining-text,
    body.light-theme .activity-time,
    body.light-theme .stat-label {
        color: #636e72;
    }

    /* Scrollbar */
    .activity-list::-webkit-scrollbar {
        width: 8px;
    }

    .activity-list::-webkit-scrollbar-track {
        background: var(--bg-primary);
        border-radius: 10px;
    }

    .activity-list::-webkit-scrollbar-thumb {
        background: #667eea;
        border-radius: 10px;
    }

    /* Responsive */
    @media (max-width: 1024px) {
        .profile-grid {
            grid-template-columns: 1fr;
        }
    }

    @media (max-width: 768px) {
        .profile-container {
            padding: 20px;
            margin-top: 100px;
        }

        .plan-dates {
            grid-template-columns: 1fr;
        }
    }
</style>

<div class="profile-container">
    <div class="profile-header">
        <div class="profile-title">
            <i class="fas fa-user-circle"></i>
            الملف الشخصي
        </div>
    </div>



    <div class="profile-grid">
        <!-- العمود الأيسر -->
        <div class="left-column">
            <!-- بطاقة المستخدم -->
            <div class="user-card">
                <img src="<?php echo $user['avatar']; ?>" alt="Avatar" class="user-avatar">
                <div class="user-name"><?php echo $user['name']; ?></div>
                <div class="user-email">
                    <i class="fas fa-envelope"></i>
                    <?php echo $user['email']; ?>
                </div>
                <div class="user-email" style="margin-top: 8px; color: #667eea; font-weight: 600;">
                    <i class="fas fa-link"></i>
                    رمز الإحالة: <?php echo $_SESSION['user_id']; ?>
                </div>
                <button class="edit-profile-btn" onclick="window.location.href='settings.php'">
                        <i class="fas fa-edit"></i>
                        تعديل الملف الشخصي
                </button>

            </div>

            <!-- كروت النقاط والإحالات -->
            <div class="stats-grid">
                <!-- كارت النقاط -->
                <div class="stat-card">
                    <div class="stat-icon points-icon">
                        <i class="fas fa-star"></i>
                    </div>
                    <div class="stat-value"><?php echo number_format($user['points']); ?></div>
                    <div class="stat-label">نقطة</div>
                </div>

                <!-- كارت الإحالات -->
                <div class="stat-card">
                    <div class="stat-icon referrals-icon">
                        <i class="fas fa-user-friends"></i>
                    </div>
                    <div class="stat-value"><?php echo $user['referrals']; ?></div>
                    <div class="stat-label">إحالة</div>
                </div>
            </div>
        </div>

        <!-- بطاقة الباقة -->
        <div class="plan-card">
            <div class="plan-header">
                <div class="plan-name">
                    <i class="<?php echo $user['plan_icon']; ?> plan-icon" style="color: <?php echo $user['plan_color']; ?>"></i>
                    <?php echo $user['plan']; ?>
                </div>

<div class="plan-badge" onclick="window.location.href='packages.php'" style="
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-weight: bold;
    padding: 6px 10px;
    border-radius: 8px;
    color: #fff;
    cursor: pointer;
    transition: all 0.3s ease;
    <?php echo $isExpired ? 'background-color:#dc2626;' : 'background-color:#16a34a;'; ?>
" onmouseover="this.style.transform='scale(1.05)'; this.style.boxShadow='0 5px 15px rgba(0,0,0,0.3)';" onmouseout="this.style.transform='scale(1)'; this.style.boxShadow='none';">
    <i class="fas <?php echo $isExpired ? 'fa-times-circle' : 'fa-check-circle'; ?>"></i>
    <?php echo $isExpired ? 'منتهي' : 'نشط'; ?>
</div>


            </div>

            <div class="plan-features">
                <?php foreach($user['features'] as $feature): ?>
                <div class="feature-item">
                    <i class="fas fa-check-circle"></i>
                    <span><?php echo $feature; ?></span>
                </div>
                <?php endforeach; ?>
            </div>

            <div class="plan-dates">
                <div class="date-box">
                    <div class="date-label">تاريخ الاشتراك</div>
                    <div class="date-value">
                        <i class="fas fa-calendar-check"></i>
                        <?php echo date('Y/m/d', strtotime($user['subscription_date'])); ?>
                    </div>
                </div>
                <div class="date-box">
                    <div class="date-label">تاريخ الانتهاء</div>
                    <div class="date-value">
                        <i class="fas fa-calendar-times"></i>
                        <?php echo date('Y/m/d', strtotime($user['expiry_date'])); ?>
                    </div>
                </div>
            </div>

            <div class="days-remaining">
                <div class="days-remaining-text">الأيام المتبقية</div>
                <div class="days-remaining-count">
                    <?php echo $user['days_remaining']; ?> يوم
                </div>
            </div>
        </div>
    </div>

    <!-- Activity Log -->
    <div class="activity-card">
        <div class="activity-header">
            <i class="fas fa-history" style="color: #667eea; font-size: 24px;"></i>
            <span class="activity-title">سجل النشاطات</span>
        </div>

        <div class="activity-list">
            <?php foreach($activity_log as $log): ?>
            <div class="activity-item">
                <div class="activity-icon" style="background: <?php echo $log['color']; ?>">
                    <i class="fas <?php echo $log['icon']; ?>"></i>
                </div>
                <div class="activity-content">
                    <div class="activity-action"><?php echo $log['action']; ?></div>
                    <div class="activity-time">
                        <i class="fas fa-clock"></i>
                        <?php echo $log['time']; ?>
                    </div>
                </div>
            </div>
            <?php endforeach; ?>
        </div>
    </div>
</div>

<script>
function editProfile() {
    Swal.fire({
        title: 'تعديل الملف الشخصي',
        html: `
            <div style="text-align: right;">
                <div style="margin-bottom: 15px;">
                    <label style="display: block; margin-bottom: 5px; font-weight: 600;">الاسم</label>
                    <input type="text" id="edit-name" class="swal2-input" value="<?php echo $user['name']; ?>" style="width: 90%;">
                </div>
                <div style="margin-bottom: 15px;">
                    <label style="display: block; margin-bottom: 5px; font-weight: 600;">البريد الإلكتروني</label>
                    <input type="email" id="edit-email" class="swal2-input" value="<?php echo $user['email']; ?>" style="width: 90%;">
                </div>
                <div style="margin-bottom: 15px;">
                    <label style="display: block; margin-bottom: 5px; font-weight: 600;">كلمة المرور الجديدة (اختياري)</label>
                    <input type="password" id="edit-password" class="swal2-input" placeholder="اتركه فارغاً للإبقاء على كلمة المرور الحالية" style="width: 90%;">
                </div>
            </div>
        `,
        showCancelButton: true,
        confirmButtonText: 'حفظ التعديلات',
        cancelButtonText: 'إلغاء',
        preConfirm: () => {
            return {
                name: document.getElementById('edit-name').value,
                email: document.getElementById('edit-email').value,
                password: document.getElementById('edit-password').value
            }
        }
    }).then((result) => {
        if (result.isConfirmed) {
            // هنا يتم إرسال البيانات للسيرفر
            Swal.fire({
                icon: 'success',
                title: 'تم التحديث!',
                text: 'تم تحديث معلومات الملف الشخصي بنجاح',
                timer: 2000,
                showConfirmButton: false
            });
        }
    });
}
</script>

<?php include 'includes/footer.php'; ?>
