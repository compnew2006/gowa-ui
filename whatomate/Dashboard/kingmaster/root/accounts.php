
<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "الحسابات | Kingmaster";
$page_css = ['/css/account.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>


<div class="accounts-container">
    <div class="accounts-header">
        <div class="header-top">
            <div style="display: flex; align-items: center; gap: 20px; flex-wrap: wrap;">
                <div class="accounts-title">
                    <i class="fas fa-users"></i>
                    <span data-i18n="maneg_account">إدارة الحسابات</span>
                </div>
                <div id="accountLimitInfo"></div>
            </div>
            <button class="add-account-btn" onclick="openPlatformSelector()">
                <i class="fas fa-plus"></i>
                إضافة حساب
            </button>
        </div>
        
        <div class="filters-section">
            <div class="filter-group">
                <span class="filter-label"><i class="fas fa-layer-group"></i> المنصة:</span>
                <select class="filter-select" id="platformFilter" onchange="applyFilters()">
                    <option value="all">الكل</option>
                    <option value="facebook">Facebook</option>
                    <option value="whatsapp">WhatsApp</option>
                    <option value="instagram">Instagram</option>
                    <option value="telegram">Telegram</option>
                    <option value="email">Email</option>
                </select>
            </div>
            
            <div class="filter-group">
                <span class="filter-label"><i class="fas fa-toggle-on"></i> الحالة:</span>
                <select class="filter-select" id="statusFilter" onchange="applyFilters()">
                    <option value="all">الكل</option>
                    <option value="active">نشط</option>
                    <option value="inactive">غير نشط</option>
                </select>
            </div>
            
            <div class="filter-group">
                <span class="filter-label"><i class="fas fa-search"></i> بحث:</span>
                <input type="text" class="search-input" id="searchInput" placeholder="ابحث بالاسم..." oninput="applyFilters()">
            </div>
        </div>
    </div>

    <div id="accountsGrid" class="accounts-grid">
        <div class="empty-state">
            <i class="fas fa-user-plus"></i>
            <h3>لا توجد حسابات</h3>
            <p>ابدأ بإضافة حساباتك على المنصات المختلفة</p>
        </div>
    </div>
</div>

<!-- Platform Selector Modal -->
<div id="platformModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">اختر المنصة</div>
            <span class="close-modal" onclick="closeModal('platformModal')">&times;</span>
        </div>
        <div class="platform-selection">
            <div class="platform-btn" onclick="selectPlatform('facebook')">
                <i class="fab fa-facebook platform-facebook"></i>
                <span>Facebook</span>
            </div>
            <div class="platform-btn" onclick="selectPlatform('whatsapp')">
                <i class="fab fa-whatsapp platform-whatsapp"></i>
                <span>WhatsApp</span>
            </div>
            <div class="platform-btn" onclick="selectPlatform('instagram')">
                <i class="fab fa-instagram platform-instagram"></i>
                <span>Instagram</span>
            </div>
            <div class="platform-btn" onclick="selectPlatform('telegram')">
                <i class="fab fa-telegram platform-telegram"></i>
                <span>Telegram</span>
            </div>
            <div class="platform-btn" onclick="selectPlatform('email')">
                <i class="fas fa-envelope platform-email"></i>
                <span>Email</span>
            </div>
        </div>
    </div>
</div>

<!-- Facebook Modals -->
<div id="facebookMethodModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title"><i class="fab fa-facebook"></i> Facebook - طريقة التسجيل</div>
            <span class="close-modal" onclick="closeModal('facebookMethodModal')">&times;</span>
        </div>
        <div class="method-selection">
            <div class="method-btn" style="display:none;"  onclick="selectFacebookMethod('credentials')">
                <i class="fas fa-user-lock"></i><br>
                عبر البيانات
            </div>
            <div class="method-btn" onclick="selectFacebookMethod('cookies')">
                <i class="fas fa-cookie"></i><br>
                عبر الكوكيز
            </div>
        </div>
    </div>
</div>

<div id="facebookCredentialsModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">تسجيل عبر البيانات</div>
            <span class="close-modal" onclick="closeModal('facebookCredentialsModal')">&times;</span>
        </div>
        <form onsubmit="event.preventDefault(); submitFacebookCredentials();">
            <div class="form-group">
                <label class="form-label">المعرف / البريد</label>
                <input type="text" class="form-input" id="fbUsername" required>
            </div>
            <div class="form-group">
                <label class="form-label">كلمة السر</label>
                <input type="password" class="form-input" id="fbPassword" required>
            </div>
            <div class="form-group">
                <label class="form-label">رمز المصادقة (اختياري)</label>
                <input type="text" class="form-input" id="fbTwoFa">
            </div>
            <button type="submit" class="submit-btn">
                <i class="fas fa-check"></i> تسجيل الدخول
            </button>
        </form>
    </div>
</div>

<div id="facebookCookiesTypeModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">نوع الكوكيز</div>
            <span class="close-modal" onclick="closeModal('facebookCookiesTypeModal')">&times;</span>
        </div>
        <div class="method-selection">
            <div class="method-btn" onclick="selectFacebookCookiesType('single')">
                <i class="fas fa-user"></i><br>
                حساب محدد
            </div>
            <div class="method-btn" style="display:none;" onclick="selectFacebookCookiesType('multiple')">
                <i class="fas fa-users"></i><br>
                حسابات متعددة
            </div>
        </div>
    </div>
</div>

<div id="facebookSingleCookieModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">إدخال الكوكيز</div>
            <span class="close-modal" onclick="closeModal('facebookSingleCookieModal')">&times;</span>
        </div>
        <form onsubmit="event.preventDefault(); submitFacebookSingleCookie();">
            <div class="form-group">
                <label class="form-label">لصق الكوكيز هنا</label>
                <textarea class="form-textarea" id="fbSingleCookie" required></textarea>
            </div>
            <button type="submit" class="submit-btn">
                <i class="fas fa-plus"></i> إضافة الحساب
            </button>
        </form>
    </div>
</div>

<div id="facebookMultipleCookiesModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">رفع ملف حسابات</div>
            <span class="close-modal" onclick="closeModal('facebookMultipleCookiesModal')">&times;</span>
        </div>
        <div class="file-upload-area" onclick="document.getElementById('fbMultipleCookiesFile').click()">
            <i class="fas fa-cloud-upload-alt"></i>
            <p>اضغط لاختيار ملف TXT</p>
            <small>يجب فصل الحسابات بعلامة |</small>
        </div>
        <input type="file" id="fbMultipleCookiesFile" accept=".txt" style="display: none;" onchange="handleFacebookMultipleCookies()">
    </div>
</div>

<!-- WhatsApp Modal -->
<div id="whatsappModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title"><i class="fab fa-whatsapp"></i> WhatsApp - مسح QR</div>
            <span class="close-modal" onclick="closeModal('whatsappModal')">&times;</span>
        </div>
        <div class="qr-section" id="whatsappQRSection">
            <div class="qr-code">
                <img src="https://ipaymu.com/wp-content/themes/ipaymu_v2/assets/new-assets/image/wa.png" alt="QR Code">
            </div>
            <div class="qr-timer" id="whatsappTimer">15 ثانية</div>
            <p>امسح رمز QR باستخدام WhatsApp</p>
        </div>
        <div class="qr-section" id="whatsappReloadSection" style="display: none;">
            <i class="fas fa-clock" style="font-size: 60px; color: #f39c12; margin-bottom: 20px;"></i>
            <p>انتهى وقت رمز QR</p>
            <button class="reload-qr-btn" onclick="reloadWhatsAppQR()">
                <i class="fas fa-redo"></i> إعادة التحميل
            </button>
        </div>
    </div>
</div>

<!-- Instagram Modal -->
<div id="instagramModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title"><i class="fab fa-instagram"></i> Instagram</div>
            <span class="close-modal" onclick="closeModal('instagramModal')">&times;</span>
        </div>
        <form onsubmit="event.preventDefault(); submitInstagram();">
			<div class="form-group">
                <label class="form-label">لصق الكوكيز هنا</label>
                <textarea class="form-textarea" id="instaSingleCookie" required></textarea>
            </div>
           <!-- <div class="form-group">
                <label class="form-label">اسم المستخدم</label>
                <input type="text" class="form-input" id="instaUsername" required>
            </div>
            <div class="form-group">
                <label class="form-label">كلمة السر</label>
                <input type="password" class="form-input" id="instaPassword" required>
            </div>
            -->
            <button type="submit" class="submit-btn">
                <i class="fas fa-sign-in-alt"></i> تسجيل الدخول
            </button>
        </form>
    </div>
</div>

<!-- Telegram Modal -->
<div id="telegramModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title"><i class="fab fa-telegram"></i> Telegram</div>
            <span class="close-modal" onclick="closeModal('telegramModal')">&times;</span>
        </div>
        <form onsubmit="event.preventDefault(); loginTelegram();">
            <div class="form-group">
                <label class="form-label">المعرف</label>
                <input type="text" class="form-input" id="telegramUsername" placeholder="@username">
            </div>
            <div class="form-group">
                <label class="form-label">الرمز</label>
                <input type="text" class="form-input" id="telegramCode" placeholder="رمز التطبيق">
            </div>
            <div class="form-group">
                <label class="form-label">رقم الهاتف</label>
                <input type="tel" class="form-input" id="telegramPhone" placeholder="+201234567890" required>
            </div>
            
            <!-- قسم رمز التحقق مخفي بالبداية -->
            <div id="telegramVerificationSection" style="display: none;">
                <div class="form-group">
                    <label class="form-label">الكود</label>
                    <div style="display: flex; gap: 10px;">
                        <input type="text" class="form-input" id="telegramVerificationCode" placeholder="أدخل الكود" style="flex: 1;">
                        <button type="button" class="submit-btn" onclick="sendTelegramVerificationCode()" style="width: auto; padding: 12px 20px; white-space: nowrap;">
                            <i class="fas fa-paper-plane"></i> إرسال الرمز
                        </button>
                    </div>
                </div>
            </div>
            
            <button type="submit" class="submit-btn">
                <i class="fas fa-sign-in-alt"></i> تسجيل الدخول
            </button>
        </form>
    </div>
</div>

<!-- Email Modal -->
<div id="emailModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title"><i class="fas fa-envelope"></i> Email</div>
            <span class="close-modal" onclick="closeModal('emailModal')">&times;</span>
        </div>
        <form onsubmit="event.preventDefault(); submitEmail();">
            <div class="form-group">
                <label class="form-label">عنوان البريد الإلكتروني</label>
                <input type="email" class="form-input" id="emailAddress" required>
            </div>
            <div class="form-group">
                <label class="form-label">كلمة السر</label>
                <input type="password" class="form-input" id="emailPassword" required>
            </div>
            <button type="submit" class="submit-btn">
                <i class="fas fa-sign-in-alt"></i> تسجيل الدخول
            </button>
        </form>
    </div>
</div>
<script src="https://code.jquery.com/jquery-3.7.1.min.js"></script>

<script src="js/accounts.js?v=20260210_714"></script>

<?php include 'includes/footer.php'; ?>
