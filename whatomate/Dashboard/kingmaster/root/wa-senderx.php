




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



<style>
    .tools-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }
    
    .tools-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
    }
    
    .tools-title {
        font-size: 32px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 15px;
        font-family: 'Cairo', sans-serif;
    }
    
    .create-campaign-btn {
        padding: 12px 25px;
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
        border: none;
        border-radius: 12px;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 16px;
    }
    
    .create-campaign-btn:hover {
        transform: translateY(-3px);
        box-shadow: 0 10px 25px rgba(102, 126, 234, 0.4);
    }
    
    .campaigns-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
        gap: 25px;
        margin-top: 30px;
    }
    
    .campaign-card {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 25px;
        border: 2px solid var(--border-color);
        transition: all 0.3s ease;
    }
    
    .campaign-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        border-color: #667eea;
    }
    
    .campaign-name {
        font-size: 20px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 15px;
        font-family: 'Cairo', sans-serif;
    }
    
    .campaign-info {
        display: flex;
        flex-direction: column;
        gap: 10px;
        margin-bottom: 15px;
    }
    
    .campaign-info-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
    }
    
    .campaign-info-label {
        color: var(--text-secondary);
        font-weight: 600;
    }
    
    .campaign-info-value {
        color: var(--text-primary);
        font-weight: 700;
    }
    
    .status-badge {
        padding: 5px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 700;
        text-align: center;
    }
    
    .status-pending { background: linear-gradient(135deg, #fef3c7, #fde68a); color: #92400e; }
    .status-running { background: linear-gradient(135deg, #dbeafe, #bfdbfe); color: #1e40af; }
    .status-paused { background: linear-gradient(135deg, #fde68a, #fcd34d); color: #78350f; }
    .status-stopped { background: linear-gradient(135deg, #fee2e2, #fecaca); color: #991b1b; }
    .status-finished { background: linear-gradient(135deg, #d1fae5, #a7f3d0); color: #065f46; }
    
    .campaign-actions-btn {
        width: 100%;
        padding: 12px;
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
        border: none;
        border-radius: 10px;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        transition: all 0.3s ease;
        margin-top: 10px;
    }
    
    .campaign-actions-btn:hover {
        background: linear-gradient(135deg, #764ba2, #667eea);
    }
    
    .accordion-content {
        max-height: 0;
        overflow: hidden;
        transition: max-height 0.3s ease;
    }
    
    .accordion-content.active {
        max-height: 500px;
    }
    
    .accordion-section {
        margin-top: 15px;
        padding: 15px;
        background: rgba(102, 126, 234, 0.1);
        border-radius: 10px;
    }
    
    .accordion-section-title {
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
    }
    
    .action-btns {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }
    
    .action-btn-full {
        width: 100%;
        padding: 8px 15px;
        border: none;
        border-radius: 8px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        transition: all 0.3s ease;
        font-size: 12px;
        color: white;
        margin-top: 5px;
    }
    
    .action-btn-small {
        padding: 8px 15px;
        border: none;
        border-radius: 8px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        transition: all 0.3s ease;
        font-size: 12px;
        color: white;
    }
    
    .btn-stop { background: linear-gradient(135deg, #ef4444, #dc2626); }
    .btn-pause { background: linear-gradient(135deg, #f59e0b, #d97706); }
    .btn-edit { background: linear-gradient(135deg, #3b82f6, #2563eb); }
    .btn-save { background: linear-gradient(135deg, #10b981, #059669); }
    .btn-delete { background: linear-gradient(135deg, #991b1b, #7f1d1d); }
    .btn-data { background: linear-gradient(135deg, #10b981, #059669); }
    .btn-send { background: linear-gradient(135deg, #8b5cf6, #7c3aed); }
    .btn-manage { background: linear-gradient(135deg, #06b6d4, #0891b2); }
    
    .action-btn-small:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(0,0,0,0.3);
    }
    
    .action-btn-full:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(0,0,0,0.3);
    }
    
    /* Modal Styles */
    .modal {
        display: none;
        position: fixed;
        z-index: 1000;
        left: 0;
        top: 0;
        width: 100%;
        height: 100%;
        background: rgba(0,0,0,0.8);
        backdrop-filter: blur(5px);
    }
    
    .modal-content {
        background: var(--card-bg);
        margin: 5% auto;
        padding: 30px;
        border-radius: 20px;
        width: 90%;
        max-width: 600px;
        max-height: 80vh;
        overflow-y: auto;
        border: 2px solid var(--border-color);
        box-shadow: 0 24px 48px rgba(0,0,0,0.5);
        overscroll-behavior: contain;
        -webkit-overflow-scrolling: touch;
        scrollbar-width: thin; /* Firefox */
        scrollbar-color: rgba(102,126,234,.6) transparent; /* Firefox */
    }
    /* Custom scrollbar (WebKit) */
    .modal-content::-webkit-scrollbar { width: 10px; }
    .modal-content::-webkit-scrollbar-track { background: transparent; }
    .modal-content:hover::-webkit-scrollbar-track { background: rgba(255,255,255,.06); border-radius: 10px; }
    .modal-content::-webkit-scrollbar-thumb { background: linear-gradient(180deg, #667eea, #764ba2); border-radius: 10px; border: 2px solid rgba(0,0,0,0); }
    .modal-content:hover::-webkit-scrollbar-thumb { background: linear-gradient(180deg, #7c3aed, #667eea); }
    
    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 25px;
    }
    
    .modal-title {
        font-size: 24px;
        font-weight: 800;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }
    
    .close-modal {
        font-size: 28px;
        font-weight: bold;
        color: var(--text-secondary);
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .close-modal:hover {
        color: #ef4444;
        transform: rotate(90deg);
    }
    
    .form-group {
        margin-bottom: 20px;
    }
    
    .form-label {
        display: block;
        margin-bottom: 8px;
        font-weight: 700;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
    }
    
    .form-input, .form-select {
        width: 100%;
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: #1f2937;
        color: #e5e7eb;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
        font-size: 14px;
        transition: all 0.3s ease;
    }
    
.form-input:focus, .form-select:focus {
        outline: none;
        border-color: #667eea;
        box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }

    /* Speed cards UI */
    .speed-cards { display:grid; grid-template-columns: 1fr 1fr; gap:12px; }
    .speed-card { position:relative; display:flex; align-items:center; gap:12px; padding:14px; border:2px solid var(--border-color); border-radius:14px; background: rgba(31,41,55,0.9); cursor:pointer; transition:.25s; }
    .speed-card:hover { transform: translateY(-2px); box-shadow: 0 8px 22px rgba(0,0,0,.25); }
    .speed-card.selected { border-color:#667eea; box-shadow: 0 0 0 3px rgba(102,126,234,.15) inset; }
    .speed-card.disabled { opacity:.55; cursor:not-allowed; filter: grayscale(0.3); }
    .speed-card-icon i { font-size:22px; color:#fff; }
    .speed-card-title { font-weight:800; color:#fff; }
    .speed-card-desc { font-size:12px; color:#9ca3af; }
    .badge { position:absolute; top:10px; left:10px; font-size:11px; padding:4px 8px; border-radius:999px; border:1px solid rgba(255,255,255,.12); background:rgba(255,255,255,.06); color:#e5e7eb; }
    .badge-green { border-color: rgba(16,185,129,.35); background: rgba(16,185,129,.1); color:#10b981; }
    .badge-blue { border-color: rgba(59,130,246,.35); background: rgba(59,130,246,.1); color:#60a5fa; }
    .lock-badge { position:absolute; inset:auto 10px 10px auto; font-size:12px; padding:6px 10px; border-radius:10px; background: rgba(239,68,68,.12); color:#ef4444; border:1px solid rgba(239,68,68,.25); display:inline-flex; align-items:center; gap:6px; }

    /* Option cards (message types) */
    .option-cards { display:grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap:10px; }
    .option-card { display:flex; align-items:center; gap:10px; padding:12px; border:2px solid var(--border-color); border-radius:14px; background: rgba(31,41,55,0.9); cursor:pointer; transition:.25s; }
    .option-card:hover { transform: translateY(-2px); box-shadow: 0 8px 22px rgba(0,0,0,.2); }
    .option-card.selected { border-color:#667eea; box-shadow: 0 0 0 3px rgba(102,126,234,.15) inset; }
    .option-card i { font-size:18px; color:#e5e7eb; }
    .option-title { font-weight:800; color:#fff; }
    .option-desc { font-size:12px; color:#9ca3af; }

    /* Segmented controls */
    .segmented { display:inline-flex; background: rgba(31,41,55,0.7); border:2px solid var(--border-color); border-radius:12px; overflow:hidden; }
    .segmented button { appearance:none; background:transparent; color:#e5e7eb; border:0; padding:10px 14px; cursor:pointer; font-weight:700; transition:.2s; }
    .segmented button:hover { background: rgba(255,255,255,.06); }
    .segmented button.selected { background: linear-gradient(135deg, #667eea, #764ba2); color:#fff; }

    /* Build List button */
    .btn-list-builder {
        padding: 12px 16px;
        background: linear-gradient(135deg, #8b5cf6, #7c3aed);
        color: #fff;
        border: 0;
        border-radius: 12px;
        font-weight: 800;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        transition: all .25s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 100%;
        gap: 8px;
        box-shadow: 0 8px 24px rgba(124,58,237,.35);
    }
    .btn-list-builder:hover {
        transform: translateY(-2px);
        box-shadow: 0 12px 30px rgba(124,58,237,.5);
    }
    .btn-list-builder i { animation: float3d 8s ease-in-out infinite, pulse 2.2s ease-in-out infinite; }

    .account-selector-wrapper {
        display: flex;
        gap: 10px;
        margin-bottom: 10px;
    }
    
    .add-account-btn {
        padding: 12px 20px;
        background: linear-gradient(135deg, #10b981, #059669);
        color: white;
        border: none;
        border-radius: 10px;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        transition: all 0.3s ease;
        white-space: nowrap;
    }
    
    .add-account-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(16, 185, 129, 0.4);
    }
    
    .selected-accounts {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        margin-top: 10px;
    }
    
    .account-tag {
        padding: 8px 15px;
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
        border-radius: 20px;
        font-size: 13px;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 8px;
    }
    
    .remove-account {
        cursor: pointer;
        font-weight: bold;
        transition: all 0.3s ease;
    }
    
    .remove-account:hover {
        color: #ef4444;
        transform: scale(1.2);
    }
    
    .range-slider-wrapper {
        margin-top: 10px;
    }
    
    .range-value {
        display: inline-block;
        padding: 5px 15px;
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
        border-radius: 15px;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        margin-bottom: 10px;
    }
    
    .range-slider {
        width: 100%;
        height: 8px;
        border-radius: 5px;
        background: #d1d5db;
        outline: none;
        -webkit-appearance: none;
    }
    
    .range-slider::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background: linear-gradient(135deg, #667eea, #764ba2);
        cursor: pointer;
    }
    
    .range-slider::-moz-range-thumb {
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background: linear-gradient(135deg, #667eea, #764ba2);
        cursor: pointer;
        border: none;
    }
    
    .submit-btn {
        width: 100%;
        padding: 15px;
        background: linear-gradient(135deg, #10b981, #059669);
        color: white;
        border: none;
        border-radius: 12px;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        font-size: 16px;
        cursor: pointer;
        transition: all 0.3s ease;
        margin-top: 10px;
    }
    
    .submit-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 25px rgba(16, 185, 129, 0.4);
    }
    
    .filters-section {
        background: var(--card-bg);
        padding: 20px;
        border-radius: 15px;
        margin-bottom: 25px;
        border: 2px solid var(--border-color);
        display: flex;
        gap: 15px;
        align-items: center;
        flex-wrap: wrap;
    }
    
    .filter-label {
        color: var(--text-primary);
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 8px;
    }
    
    .filter-select {
        padding: 10px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: #1f2937;
        color: #e5e7eb;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
        font-size: 14px;
        cursor: pointer;
        transition: all 0.3s ease;
        min-width: 150px;
    }
    
    .filter-select:focus {
        outline: none;
        border-color: #667eea;
        box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
    }
    
    .clear-filters-btn {
        padding: 10px 20px;
        background: linear-gradient(135deg, #6b7280, #4b5563);
        color: white;
        border: none;
        border-radius: 10px;
        font-weight: 700;
        font-family: 'Cairo', sans-serif;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        align-items: center;
        gap: 8px;
    }
    
    .clear-filters-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(107, 114, 128, 0.4);
    }
    
    /* Toggle Switch */
    .toggle-switch {
        position: relative;
        display: inline-block;
        width: 60px;
        height: 30px;
    }
    
    .toggle-switch input {
        opacity: 0;
        width: 0;
        height: 0;
    }
    
    .toggle-slider {
        position: absolute;
        cursor: pointer;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: linear-gradient(135deg, #ef4444, #dc2626);
        transition: 0.4s;
        border-radius: 30px;
    }
    
    .toggle-slider:before {
        position: absolute;
        content: "";
        height: 22px;
        width: 22px;
        left: 4px;
        bottom: 4px;
        background-color: white;
        transition: 0.4s;
        border-radius: 50%;
    }
    
    .toggle-switch input:checked + .toggle-slider {
        background: linear-gradient(135deg, #10b981, #059669);
    }
    
    .toggle-switch input:checked + .toggle-slider:before {
        transform: translateX(30px);
    }
    
    .toggle-slider:hover {
        box-shadow: 0 0 15px rgba(102, 126, 234, 0.5);
    }
</style>

<div class="tools-container">
    <div class="tools-header">
        <div class="tools-title">
            <i class="fas fa-tools fa-spin" style="--fa-animation-duration: 3s; color: #667eea;"></i>
            الأدوات
        </div>
        <button class="create-campaign-btn" onclick="openModal()">
            <i class="fas fa-plus-circle"></i>
            إنشاء حملة جديدة
        </button>
    </div>
    
    <div class="filters-section">
        <div class="filter-label">
            <i class="fas fa-filter fa-spin" style="--fa-animation-duration: 3s;"></i>
            تصفية:
        </div>
        
   
        <select class="filter-select" id="status-filter" onchange="applyFilters()">
            <option value="all">جميع الحالات</option>
            <option value="pending">قيد الانتظار</option>
            <option value="running">قيد التشغيل</option>
            <option value="paused">متوقف مؤقتاً</option>
            <option value="stopped">متوقف</option>
            <option value="finished">منتهي</option>
        </select>
        
        <input type="date" class="filter-select" id="date-filter" onchange="applyFilters()" 
               style="padding: 10px 15px; cursor: pointer; color-scheme: dark;"
               placeholder="التاريخ">
        
        <button class="clear-filters-btn" onclick="clearFilters()">
            <i class="fas fa-times"></i>
            إعادة تعيين
        </button>
    </div>
    
    <div class="campaigns-grid" id="campaigns-grid">
        <div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);">
            جاري تحميل الحملات...
        </div>
    </div>
</div>

<!-- Modal Create Campaign -->
<div id="campaignModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">
                <i class="fas fa-rocket"></i> إنشاء حملة جديدة
            </div>
            <span class="close-modal" onclick="closeModal()">&times;</span>
        </div>
        
        <form id="campaignForm">
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-tag"></i> اسم الحملة
                </label>
                <input type="text" class="form-input" id="campaign-name" placeholder="أدخل اسم الحملة" required>
            </div>
            
            <div class="form-group">
                <label class="form-label"><i class="fas fa-gauge-high"></i> سرعة التنفيذ</label>
                <div class="speed-cards" id="speedCards">
                    <div class="speed-card selected" data-value="slow" id="speedCardSlow">
                        <div class="speed-card-icon"><i class="fas fa-gauge-low fa-beat"></i></div>
                        <div class="speed-card-body">
                            <div class="speed-card-title">بطيء</div>
                            <div class="speed-card-desc">آمن ولا يستهلك نقاط</div>
                        </div>
                     </div>
                    <div class="speed-card" data-value="fast" id="speedCardFast">
                        <div class="speed-card-icon"><i class="fas fa-rocket fa-fade"></i></div>
                        <div class="speed-card-body">
                            <div class="speed-card-title">سريع</div>
                            <div class="speed-card-desc">أسرع تنفيذ — قد يستهلك نقاط</div>
                        </div>
                    </div>
                </div>
                <input type="hidden" id="speed-mode" value="slow">
                <div id="speedHint" class="muted" style="margin-top:6px;"></div>
            </div>

            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-users"></i> الحسابات
                </label>
                <div class="account-selector-wrapper">
                    <select class="form-select" id="account-select" style="flex: 1;">
                        <option value="">اختر حساب</option>
                    </select>
                    <button type="button" class="add-account-btn" onclick="addAccount()">
                        <i class="fas fa-plus"></i> إضافة
                    </button>
                </div>
                <div class="selected-accounts" id="selected-accounts"></div>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-address-book"></i> اختر جهة الاتصال (قائمة محفوظة)</label>
                <select class="form-select" id="contact-list-select">
                    <option value="">— اختر —</option>
                </select>
                <div id="contactListHint" style="margin-top:6px; font-size:12px; color:var(--text-secondary);"></div>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-file-alt"></i> المحتوى</label>
                <select class="form-select" id="sender-content-select">
                    <option value="">— اختر المحتوى —</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-clock"></i> إعدادات الفواصل الزمنية (واتساب)</label>
                <select class="form-select" id="sender-interval-select">
                    <option value="">— اختر الإعداد —</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-message"></i> نوع الرسالة</label>
                <div class="option-cards" id="msgTypeCards">
                  <div class="option-card selected" data-value="text"><i class="fas fa-align-left"></i><div><div class="option-title">نص</div><div class="option-desc">رسالة نصية فقط</div></div></div>
                  <div class="option-card" data-value="text_image"><i class="fas fa-image"></i><div><div class="option-title">نص وصورة</div><div class="option-desc">أرفق صورة</div></div></div>
                  <div class="option-card" data-value="text_video"><i class="fas fa-video"></i><div><div class="option-title">نص وفيديو</div><div class="option-desc">أرفق فيديو</div></div></div>
                  <div class="option-card" data-value="text_file"><i class="fas fa-paperclip"></i><div><div class="option-title">نص وملف</div><div class="option-desc">أرفق ملف</div></div></div>
                  <div class="option-card" data-value="poll"><i class="fas fa-square-poll-horizontal"></i><div><div class="option-title">استطلاع رأي</div><div class="option-desc">سؤال بخيارات</div></div></div>
                  <div class="option-card" data-value="list"><i class="fas fa-list"></i><div><div class="option-title">قائمة</div><div class="option-desc">قائمة تفاعلية</div></div></div>
                </div>
                <input type="hidden" id="msg-type" value="text">
                <button type="button" id="openListBuilderBtn" class="btn-list-builder" style="display:none; margin-top:10px;"><i class="fas fa-layer-group"></i> بناء قائمة تفاعلية</button>
            </div>

            <div class="form-group" id="sender-file-wrap" style="display:none;">
                <label class="form-label"><i class="fas fa-paperclip"></i> اختر ملف</label>
                <select class="form-select" id="sender-file-select">
                    <option value="">— اختر ملف —</option>
                </select>
                <div class="muted" style="margin-top:6px;">سيظهر هذا الخيار فقط لأنواع الرسائل التي تدعم ملفًا مرفقًا.</div>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-users-gear"></i> جهة الإرسال والهوية</label>
                <div style="display:grid; gap:14px;">
                  <div>
                    <div class="muted" style="margin-bottom:6px;">الهدف</div>
                    <div class="segmented" id="targetSegment">
                      <button type="button" data-value="group">مجموعة</button>
                      <button type="button" data-value="person" class="selected">أشخاص</button>
                    </div>
                    <input type="hidden" id="target-type" value="person">
                  </div>
                  <div>
                    <div class="muted" style="margin-bottom:6px;">نوع الهوية</div>
                    <div class="segmented" id="idModeSegment">
                      <button type="button" data-value="numbers" class="selected">أرقام</button>
                      <button type="button" data-value="identifiers">معرفات</button>
                    </div>
                    <input type="hidden" id="id-mode" value="numbers">
                  </div>
                </div>
            </div>

            <textarea id="list-config" style="display:none;"></textarea>
            <textarea id="poll-config" style="display:none;"></textarea>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-calendar"></i> وقت الإرسال</label>
                <div class="segmented" id="scheduleSegment">
                  <button type="button" data-value="now" class="selected">الآن</button>
                  <button type="button" data-value="schedule">جدولة</button>
                </div>
                <input type="hidden" id="schedule-mode" value="now">
                <div id="scheduleFields" style="display:none; margin-top:10px;">
                  <div style="display:grid; grid-template-columns: 1fr 1fr; gap:10px;">
                    <input type="date" id="schedule-date" class="form-input" />
                    <input type="time" id="schedule-time" class="form-input" />
                  </div>
                  <div id="schedule-time-hint" class="hint" style="margin-top:6px; color: var(--text-secondary);">—</div>
                </div>
            </div>

            <button type="submit" class="submit-btn">
                <i class="fas fa-save"></i> حفظ الحملة
            </button>
        </form>
    </div>
</div>

<!-- Modal Manage Comments -->
<div id="manageCommentsModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">
                <i class="fas fa-comments"></i> إدارة التعليقات
            </div>
            <span class="close-modal" onclick="closeCommentsModal()">&times;</span>
        </div>
        
        <form id="commentsForm">
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-tag"></i> اسم الحملة
                </label>
                <input type="text" class="form-input" id="comments-campaign-name" placeholder="أدخل اسم الحملة" required>
            </div>
            
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-users"></i> الحسابات
                </label>
                <div class="account-selector-wrapper">
                    <select class="form-select" id="comments-account-select" style="flex: 1;">
                        <option value="">اختر حساب</option>
                    </select>
                    <button type="button" class="add-account-btn" onclick="addCommentAccount()">
                        <i class="fas fa-plus"></i> إضافة
                    </button>
                </div>
                <div class="selected-accounts" id="comments-selected-accounts"></div>
            </div>
            
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-file-alt"></i> المحتوى
                </label>
                <select class="form-select" id="content-select" required>
                    <option value="">اختر المحتوى</option>
                </select>
            </div>
            
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-clock"></i> إعدادات الفواصل الزمنية
                </label>
                <select class="form-select" id="comments-interval-select" required>
                    <option value="">اختر إعدادات الفاصل الزمني</option>
                </select>
            </div>
            
            <div class="form-group">
                <label class="form-label" style="display: flex; align-items: center; justify-content: space-between;">
                    <span>
                        <i class="fas fa-thumbs-up fa-beat" style="--fa-animation-duration: 2s; color: #3b82f6;"></i> الإعجاب بالتعليقات
                    </span>
                    <label class="toggle-switch">
                        <input type="checkbox" id="can-like-toggle" checked>
                        <span class="toggle-slider"></span>
                    </label>
                </label>
            </div>
            
            <button type="submit" class="submit-btn">
                <i class="fas fa-save"></i> حفظ الحملة
            </button>
        </form>
</div>
</div>

<!-- Contacts Modal -->
<div id="contactsModal" class="modal">
  <div class="modal-content" style="max-width: 900px;">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-address-book"></i> جهات الاتصال</div>
      <span class="close-modal" onclick="closeContactsModal()">&times;</span>
    </div>
    <div class="filters-section" style="margin-top:0; gap:10px; align-items:center;">
      <input type="text" id="contactsSearch" class="form-input" placeholder="بحث بالاسم أو الرقم" oninput="debouncedSearchContacts()" style="flex:1;">
      <select id="contactsPerPage" class="form-select" onchange="reloadContactsPage(1)" style="max-width:160px;">
        <option value="25">25</option>
        <option value="50">50</option>
        <option value="100">100</option>
      </select>
      <button class="action-btn-small btn-save" onclick="openAddContactsModal()" style="white-space:nowrap;">
        <i class="fas fa-user-plus"></i> إضافة جهات اتصال
      </button>
    </div>
    <div style="overflow:auto; border:1px solid var(--border-color); border-radius:12px;">
      <table style="width:100%; border-collapse:collapse;">
        <thead>
          <tr style="background:#0f172a;">
            <th style="padding:10px; border-bottom:1px solid var(--border-color); text-align:center;"><input type="checkbox" id="contactsSelectAll" onchange="toggleSelectAllContacts(this)"></th>
            <th style="padding:10px; border-bottom:1px solid var(--border-color); text-align:right;">الرقم</th>
            <th style="padding:10px; border-bottom:1px solid var(--border-color); text-align:right;">الحالة</th>
          </tr>
        </thead>
        <tbody id="contactsTableBody">
          <tr><td colspan="4" style="padding:20px; text-align:center; color:var(--text-secondary);">جاري التحميل...</td></tr>
        </tbody>
      </table>
    </div>
    <div style="display:flex; justify-content:space-between; align-items:center; margin-top:12px;">
      <div id="contactsInfo" class="hint">—</div>
      <div id="contactsPagination" style="display:flex; gap:8px;"></div>
    </div>
</div>
</div>

<!-- List Builder Modal -->
<div id="listBuilderModal" class="modal">
  <div class="modal-content" style="max-width:720px;">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-list"></i> إنشاء قائمة تفاعلية</div>
      <span class="close-modal" onclick="closeListBuilder()">&times;</span>
    </div>
    <div class="form-group">
      <label class="form-label"><i class="fas fa-hand-pointer"></i> نص الزر</label>
      <input type="text" id="lb-buttonText" class="form-input" placeholder="مثال: فتح القائمة" value="Open list"/>
    </div>
    <div id="lb-sections"></div>
    <button class="action-btn-small btn-edit" type="button" onclick="lbAddSection()"><i class="fas fa-plus"></i> إضافة قسم</button>
    <div style="margin-top:14px; display:flex; gap:10px; justify-content:flex-end;">
      <button class="action-btn-small btn-delete" type="button" onclick="closeListBuilder()"><i class="fas fa-times"></i> إلغاء</button>
      <button class="action-btn-small btn-save" type="button" onclick="lbSave()"><i class="fas fa-save"></i> حفظ</button>
    </div>
  </div>
</div>

<!-- Poll Builder Modal -->
<div id="pollBuilderModal" class="modal">
  <div class="modal-content" style="max-width:720px;">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-square-poll-horizontal"></i> إنشاء استطلاع رأي</div>
      <span class="close-modal" onclick="closePollBuilder()">&times;</span>
    </div>
    <div class="form-group">
      <label class="form-label"><i class="fas fa-list"></i> الاختيارات</label>
      <div id="pb-choices"></div>
      <button class="action-btn-small btn-edit" type="button" style="margin-top:8px;" onclick="pbAddChoice()"><i class="fas fa-plus"></i> إضافة اختيار</button>
    </div>
    <div style="margin-top:14px; display:flex; gap:10px; justify-content:flex-end;">
      <button class="action-btn-small btn-delete" type="button" onclick="closePollBuilder()"><i class="fas fa-times"></i> إلغاء</button>
      <button class="action-btn-small btn-save" type="button" onclick="pbSave()"><i class="fas fa-save"></i> حفظ</button>
    </div>
  </div>
</div>

<!-- Add Contacts Modal -->
<div id="addContactsModal" class="modal">
  <div class="modal-content" style="max-width: 520px;">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-list"></i> إضافة جهات اتصال</div>
      <span class="close-modal" onclick="closeAddContactsModal()">&times;</span>
    </div>
    <div class="form-group">
      <label class="form-label"><i class="fas fa-tag"></i> اسم جهة الاتصال</label>
      <input type="text" id="contactsListName" class="form-input" placeholder="أدخل الاسم" />
    </div>
    <div class="hint" id="addContactsCountHint">—</div>
    <button class="submit-btn" onclick="submitAddContactsList()"><i class="fas fa-save"></i> حفظ</button>
  </div>
</div>

<script>
let selectedAccounts = [];
let allAccounts = [];

let commentsSelectedAccounts = [];

document.addEventListener('DOMContentLoaded', function() {
    loadAccounts();
    loadIntervalsWA();
    loadCampaigns('Wa Sender');
    loadContentSender();
    loadContactLists();
    setupMsgTypeToggle();
    loadFilesList();
    // Bind segmented toggles for target and id mode
    setupSegmented('targetSegment','target-type');
    setupSegmented('idModeSegment','id-mode');
    setupSegmented('scheduleSegment','schedule-mode');
    setupScheduleToggle();
    updateScheduleTimeHint();
});

let userPoints = null;
async function fetchUserPoints(){
    try {
        const res = await fetch('api/user_points.php', { credentials: 'same-origin' });
        const j = await res.json();
        if (j && j.success) { userPoints = parseInt(j.points||0,10); }
        else { userPoints = 0; }
    } catch(e){ userPoints = 0; }
}

function openModal() {
    document.getElementById('campaignModal').style.display = 'block';
    // lock body scroll while modal open
    try { document.body.style.overflow = 'hidden'; } catch(e){}
    // reset speed selection
    const speedHidden = document.getElementById('speed-mode');
    const slowCard = document.getElementById('speedCardSlow');
    const fastCard = document.getElementById('speedCardFast');
    const fastLock = document.getElementById('speedFastLock');
    const hint = document.getElementById('speedHint');
    if (speedHidden) speedHidden.value = 'slow';
    if (slowCard) slowCard.classList.add('selected');
    if (fastCard) fastCard.classList.remove('selected');
    if (hint) hint.textContent = '';

    // reset message type UI to default (text)
    const msgHidden = document.getElementById('msg-type');
    const msgCards = document.getElementById('msgTypeCards');
    const fileWrap = document.getElementById('sender-file-wrap');
    const listBtn = document.getElementById('openListBuilderBtn');
    if (msgHidden) msgHidden.value = 'text';
    if (msgCards) {
        msgCards.querySelectorAll('.option-card').forEach(el=>el.classList.remove('selected'));
        const textCard = msgCards.querySelector('.option-card[data-value="text"]');
        if (textCard) textCard.classList.add('selected');
    }
    if (fileWrap) fileWrap.style.display = 'none';
    if (listBtn) listBtn.style.display = 'none';

    // reset schedule to default (now)
    const schH = document.getElementById('schedule-mode');
    const schSeg = document.getElementById('scheduleSegment');
    const schFields = document.getElementById('scheduleFields');
    const sdate = document.getElementById('schedule-date');
    const stime = document.getElementById('schedule-time');
    if (schH) schH.value = 'now';
    if (schSeg) {
      schSeg.querySelectorAll('button').forEach(b=>b.classList.remove('selected'));
      const nowBtn = schSeg.querySelector('button[data-value="now"]');
      if (nowBtn) nowBtn.classList.add('selected');
    }
    if (schFields) schFields.style.display='none';
    if (sdate) sdate.value='';
    if (stime) stime.value='';
    updateScheduleTimeHint();

    // fetch points and toggle fast availability
    fetchUserPoints().then(()=>{
        const pointsText = (typeof userPoints === 'number') ? `رصيدك الحالي: ${userPoints} نقطة` : '';
        if (typeof userPoints === 'number' && userPoints <= 0) {
            if (fastCard) fastCard.classList.add('disabled');
            if (fastLock) fastLock.style.display = 'inline-flex';
            if (hint) hint.textContent = (pointsText ? pointsText + ' — ' : '') + 'لا يوجد نقاط كافية للوضع السريع.';
        } else {
            if (fastCard) fastCard.classList.remove('disabled');
            if (fastLock) fastLock.style.display = 'none';
            if (hint) hint.textContent = pointsText;
        }
    });
}

// Speed cards selection handler
(function(){
    const wrap = document.getElementById('speedCards');
    if (!wrap) return;
    wrap.addEventListener('click', async (e)=>{
        const card = e.target.closest('.speed-card');
        if (!card) return;
        const value = card.getAttribute('data-value');
        if (value === 'fast') {
            if (userPoints === null) await fetchUserPoints();
            if (typeof userPoints === 'number' && userPoints <= 0) {
                if (typeof Swal !== 'undefined') {
                    Swal.fire({ icon:'warning', title:'تنبيه', text:'لا يوجد نقاط كافية', background:'#111827', color:'#e5e7eb', confirmButtonColor:'#667eea' });
                } else { alert('لا يوجد نقاط كافية'); }
                return; // keep selection unchanged
            }
        }
        document.querySelectorAll('#speedCards .speed-card').forEach(el=>el.classList.remove('selected'));
        card.classList.add('selected');
        const hidden = document.getElementById('speed-mode');
        if (hidden) hidden.value = value || 'slow';
    });
})();

function closeModal() {
    document.getElementById('campaignModal').style.display = 'none';
    // restore body scroll
    try { document.body.style.overflow = ''; } catch(e){}
    document.getElementById('campaignForm').reset();
    selectedAccounts = [];
    document.getElementById('selected-accounts').innerHTML = '';
    editingCampaignId = null;
    //
    

    // Reset modal title and button
    document.querySelector('.modal-title').innerHTML = '<i class="fas fa-rocket"></i> إنشاء حملة جديدة';
    document.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ الحملة';
    
    // Show page url and range fields
}

window.onclick = function(event) {
    const modal = document.getElementById('campaignModal');
    const commentsModal = document.getElementById('manageCommentsModal');
    if (event.target == modal) {
        closeModal();
    }
    if (event.target == commentsModal) {
        closeCommentsModal();
    }
}

function loadAccounts() {
    fetch('api/get_accounts_wa.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            allAccounts = data.accounts;
            const select = document.getElementById('account-select');
            const commentsSelect = document.getElementById('comments-account-select');
            data.accounts.forEach(account => {
                const option1 = document.createElement('option');
                option1.value = account.account_uid;
                option1.textContent = account.name;
                option1.dataset.name = account.name;
                select.appendChild(option1);
                
                const option2 = document.createElement('option');
                option2.value = account.account_uid;
                option2.textContent = account.name;
                option2.dataset.name = account.name;
                commentsSelect.appendChild(option2);
            });
        }
    });
}



function loadContactLists(){
  const hint = document.getElementById('contactListHint');
  hint.textContent = 'جاري تحميل القوائم...';
  fetch('api/contacts_lists_wa.php', { credentials: 'same-origin' })
    .then(r=>r.json())
    .then(j=>{
      const sel = document.getElementById('contact-list-select');
      sel.innerHTML = '<option value="">— اختر —</option>';
      if(j.success && Array.isArray(j.lists)){
        contactLists = j.lists;
        j.lists.forEach(row=>{
          const o = document.createElement('option');
          o.value = row.id;
          o.textContent = row.name;
          sel.appendChild(o);
        });
        hint.textContent = j.lists.length ? `تم تحميل ${j.lists.length} قائمة` : 'لا توجد قوائم جهات اتصال محفوظة';
      } else {
        hint.textContent = 'تعذر تحميل القوائم';
      }
    })
    .catch(()=>{ hint.textContent = 'تعذر تحميل القوائم'; });
}


function loadContentSender() {
    fetch('api/get_content.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const sel1 = document.getElementById('sender-content-select');
            if (sel1) {
                sel1.innerHTML = '<option value="">— اختر المحتوى —</option>';
                data.content.forEach(c => {
                    const o = document.createElement('option');
                    o.value = c.id; o.textContent = c.name; sel1.appendChild(o);
                });
            }
            // keep old content-select if exists (comments modal)
            const sel2 = document.getElementById('content-select');
            if (sel2) {
                sel2.innerHTML = '<option value="">اختر المحتوى</option>';
                data.content.forEach(c => {
                    const o = document.createElement('option'); o.value = c.id; o.textContent = c.name; sel2.appendChild(o);
                });
            }
        }
    });
}

function loadIntervalsWA() {
    fetch('api/get_intervals_wa.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const sel = document.getElementById('sender-interval-select');
            if (sel) {
                sel.innerHTML = '<option value="">— اختر الإعداد —</option>';
                data.intervals.forEach(it => {
                    const o = document.createElement('option'); o.value = it.id; o.textContent = it.settings_name; sel.appendChild(o);
                });
            }
        }
    });
}

function loadFilesList(){
    fetch('api/files_api.php?action=get_all', { credentials: 'same-origin' })
      .then(r=>r.json()).then(j=>{
        const sel = document.getElementById('sender-file-select');
        if (!sel) return;
        sel.innerHTML = '<option value="">— اختر ملف —</option>';
        if (j && j.success && Array.isArray(j.files)){
          j.files.forEach(f=>{
            const o = document.createElement('option');
            o.value = f.id; o.textContent = f.name || f.original_name || ('ملف #' + f.id);
            sel.appendChild(o);
          });
        }
      }).catch(()=>{});
}

function setupMsgTypeToggle(){
  const wrap = document.getElementById('sender-file-wrap');
  const listBtn = document.getElementById('openListBuilderBtn');
  const cards = document.getElementById('msgTypeCards');
  const hidden = document.getElementById('msg-type');
  const applyVis = (v)=>{
    if (v==='text_image' || v==='text_video' || v==='text_file') {
      if (wrap) wrap.style.display = '';
      if (listBtn) listBtn.style.display = 'none';
    } else if (v==='list') {
      if (wrap) wrap.style.display = 'none';
      if (listBtn) listBtn.style.display = '';
    } else if (v==='poll') {
      if (wrap) wrap.style.display = 'none';
      if (listBtn) listBtn.style.display = 'none';
      openPollBuilder();
    } else {
      if (wrap) wrap.style.display = 'none';
      if (listBtn) listBtn.style.display = 'none';
    }
  };
  if (cards) {
    cards.addEventListener('click', (e)=>{
      const card = e.target.closest('.option-card');
      if (!card) return;
      const v = card.getAttribute('data-value');
      document.querySelectorAll('#msgTypeCards .option-card').forEach(el=>el.classList.remove('selected'));
      card.classList.add('selected');
      if (hidden) hidden.value = v;
      applyVis(v);
    });
  }
  if (listBtn) listBtn.addEventListener('click', openListBuilder);
  // init
  applyVis(hidden ? hidden.value : 'text');
}

// List builder state
let lbState = { buttonText: 'Open list', sections: [] };

// Poll builder state
let pbState = { choices: [] };

// Segmented helpers
function setSelectValueWhenReady(selectId, value, tries=30, delay=120){
  const sel = document.getElementById(selectId);
  if (!sel || value==null || value==='') return;
  const ok = [...sel.options].some(o=>String(o.value)===String(value));
  if (ok) { sel.value = String(value); return; }
  if (tries<=0) return;
  setTimeout(()=>setSelectValueWhenReady(selectId, value, tries-1, delay), delay);
}

// Segmented helpers
function setupSegmented(segId, hiddenId){
  const seg = document.getElementById(segId); const hid = document.getElementById(hiddenId);
  if (!seg || !hid) return;
  seg.addEventListener('click', (e)=>{
    const btn = e.target.closest('button[data-value]'); if (!btn) return;
    seg.querySelectorAll('button').forEach(b=>b.classList.remove('selected'));
    btn.classList.add('selected');
    hid.value = btn.getAttribute('data-value');
  });
}
function setupScheduleToggle(){
  const seg = document.getElementById('scheduleSegment');
  const hid = document.getElementById('schedule-mode');
  const fld = document.getElementById('scheduleFields');
  const tEl = document.getElementById('schedule-time');
  if (!seg || !hid || !fld) return;
  const apply = ()=>{ fld.style.display = (hid.value==='schedule') ? '' : 'none'; };
  seg.addEventListener('click', ()=>{ setTimeout(apply, 0); });
  if (tEl) tEl.addEventListener('input', updateScheduleTimeHint);
  apply();
}
function formatArabicTime12h(t){
  if (!t) return '';
  const parts = t.split(':');
  const h = parseInt(parts[0]||'0',10);
  const m = parseInt(parts[1]||'0',10);
  const isPM = h >= 12;
  let h12 = h % 12; if (h12 === 0) h12 = 12;
  const hh = String(h12).padStart(2,'0');
  const mm = String(m).padStart(2,'0');
  const suffix = isPM ? 'مساءً' : 'صباحاً';
  return `${hh}:${mm} ${suffix}`;
}
function updateScheduleTimeHint(){
  const t = (document.getElementById('schedule-time')||{}).value || '';
  const hint = document.getElementById('schedule-time-hint');
  if (!hint) return;
  hint.textContent = t ? `الوقت المختار: ${formatArabicTime12h(t)}` : '—';
}
function openListBuilder(){
  document.getElementById('lb-buttonText').value = lbState.buttonText || 'Open list';
  renderLb();
  document.getElementById('listBuilderModal').style.display='block';
}
function closeListBuilder(){ document.getElementById('listBuilderModal').style.display='none'; }
function lbAddSection(){ lbState.sections.push({ title: 'قسم', rows: [] }); renderLb(); }
function lbRemoveSection(idx){ lbState.sections.splice(idx,1); renderLb(); }
function lbAddRow(idx){ lbState.sections[idx].rows.push({ rowId: 'opt_'+(lbState.sections[idx].rows.length+1), title:'', description:'' }); renderLb(); }
function lbRemoveRow(s,r){ lbState.sections[s].rows.splice(r,1); renderLb(); }
function renderLb(){
  const cont = document.getElementById('lb-sections');
  cont.innerHTML = lbState.sections.map((sec,si)=>{
    const rows = (sec.rows||[]).map((row,ri)=>`
      <div style="display:grid; grid-template-columns: 1fr 1fr auto; gap:8px; align-items:center; margin-top:6px;">
        <input class="form-input" placeholder="عنوان الخيار" value="${escapeHtml(row.title||'')}" oninput="lbState.sections[${si}].rows[${ri}].title=this.value"/>
        <input class="form-input" placeholder="الوصف" value="${escapeHtml(row.description||'')}" oninput="lbState.sections[${si}].rows[${ri}].description=this.value"/>
        <button class="action-btn-small btn-delete" type="button" onclick="lbRemoveRow(${si},${ri})"><i class='fas fa-trash'></i></button>
      </div>
    `).join('');
    return `
      <div style="border:1px solid var(--border-color); border-radius:12px; padding:12px; margin-bottom:10px;">
        <div style="display:flex; gap:8px; align-items:center; justify-content:space-between;">
          <div style="flex:1;">
            <label class="form-label"><i class="fas fa-folder"></i> اسم القسم</label>
            <input class="form-input" value="${escapeHtml(sec.title||'')}" oninput="lbState.sections[${si}].title=this.value"/>
          </div>
          <button class="action-btn-small btn-delete" type="button" onclick="lbRemoveSection(${si})"><i class='fas fa-trash'></i></button>
        </div>
        <div style="margin-top:8px;">
          <label class="form-label"><i class="fas fa-list"></i> العناصر</label>
          ${rows || ''}
          <button class="action-btn-small btn-edit" type="button" style="margin-top:8px;" onclick="lbAddRow(${si})"><i class='fas fa-plus'></i> إضافة عنصر</button>
        </div>
      </div>`;
  }).join('');
}
function lbSave(){
  lbState.buttonText = document.getElementById('lb-buttonText').value || 'Open list';
  document.getElementById('list-config').value = JSON.stringify(lbState);
  closeListBuilder();
}

// Poll builder
function openPollBuilder(){
  // Prefill from hidden if exists
  try { const raw = document.getElementById('poll-config').value; if (raw) pbState = JSON.parse(raw); } catch(e){}
  renderPoll();
  document.getElementById('pollBuilderModal').style.display='block';
}
function closePollBuilder(){ document.getElementById('pollBuilderModal').style.display='none'; }
function pbAddChoice(){ pbState.choices.push(''); renderPoll(); }
function pbRemoveChoice(idx){ pbState.choices.splice(idx,1); renderPoll(); }
function renderPoll(){
  const cont = document.getElementById('pb-choices');
  cont.innerHTML = (pbState.choices||[]).map((ch,i)=>`
    <div style="display:grid; grid-template-columns: 1fr auto; gap:8px; align-items:center; margin-top:6px;">
      <input class="form-input" placeholder="اختيار ${i+1}" value="${escapeHtml(ch||'')}" oninput="pbState.choices[${i}]=this.value"/>
      <button class="action-btn-small btn-delete" type="button" onclick="pbRemoveChoice(${i})"><i class='fas fa-trash'></i></button>
    </div>
  `).join('');
}
function pbSave(){
  // filter empty choices
  pbState.choices = (pbState.choices||[]).map(c=>String(c||'').trim()).filter(Boolean);
  document.getElementById('poll-config').value = JSON.stringify(pbState);
  closePollBuilder();
}

function addAccount() {
    const select = document.getElementById('account-select');
    const selectedOption = select.options[select.selectedIndex];
    
    if (!select.value) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'يرجى اختيار حساب',
            background: '#111827',
            color: '#e5e7eb',
            confirmButtonColor: '#667eea'
        });
        return;
    }
    
    // Check if already added
    if (selectedAccounts.some(acc => acc.account_uid === select.value)) {
        Swal.fire({
            icon: 'info',
            title: 'تنبيه',
            text: 'هذا الحساب مضاف بالفعل',
            background: '#111827',
            color: '#e5e7eb',
            confirmButtonColor: '#667eea'
        });
        return;
    }
    
    const account = {
        account_uid: select.value,
        name: selectedOption.dataset.name
    };
    
    selectedAccounts.push(account);
    renderSelectedAccounts();
    select.value = '';
}

function removeAccount(account_uid) {
    selectedAccounts = selectedAccounts.filter(acc => acc.account_uid !== account_uid);
    renderSelectedAccounts();
}

function renderSelectedAccounts() {
    const container = document.getElementById('selected-accounts');
    container.innerHTML = selectedAccounts.map(acc => `
        <div class="account-tag">
            <i class="fas fa-user-circle"></i>
            ${acc.name}
            <span class="remove-account" onclick="removeAccount('${acc.account_uid}')">✕</span>
        </div>
    `).join('');
}

function updateRangeValue(value) {
    document.getElementById('range-value').textContent = value + ' منشورات';
}

document.getElementById('campaignForm').addEventListener('submit', function(e) {
    e.preventDefault();
    
    if (selectedAccounts.length === 0) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'يرجى إضافة حساب واحد على الأقل',
            background: '#111827',
            color: '#e5e7eb',
            confirmButtonColor: '#667eea'
        });
        return;
    }
    

  


    // Check if we're editing or creating
    if (editingCampaignId) {
        // Update existing campaign (also persist pram1/id_action, contact, interval, speed)
        const contact = document.getElementById('contact-list-select').value || null;
        const speed = (document.getElementById('speed-mode')||{}).value || 'slow';
        const content_id = document.getElementById('sender-content-select').value || null;
        const interval_id = document.getElementById('sender-interval-select').value || null;
        const msg_type = (document.getElementById('msg-type')||{}).value || 'text';
        const file_id = document.getElementById('sender-file-select').value || null;
        const target_type = (document.getElementById('target-type')||{}).value || 'person';
        const id_mode = (document.getElementById('id-mode')||{}).value || 'numbers';
        const schedule_mode = (document.getElementById('schedule-mode')||{}).value || 'now';
        const schedule_date = (document.getElementById('schedule-date')||{}).value || null;
        const schedule_time = (document.getElementById('schedule-time')||{}).value || null;
        const schedule_time12h = (schedule_mode==='schedule' && schedule_time) ? formatArabicTime12h(schedule_time) : null;
        const schedule_datetime = (schedule_mode==='schedule' && schedule_date && schedule_time) ? `${schedule_date}T${schedule_time}:00` : null;
        const schedule_tz = (Intl && Intl.DateTimeFormat && Intl.DateTimeFormat().resolvedOptions) ? (Intl.DateTimeFormat().resolvedOptions().timeZone||null) : null;
      const list_cfg_raw = document.getElementById('list-config').value || '';
      let list_cfg = null; try { list_cfg = list_cfg_raw ? JSON.parse(list_cfg_raw) : null; } catch(e){ list_cfg = null; }
      const poll_cfg_raw = document.getElementById('poll-config').value || '';
      let poll_config = null; try { poll_config = poll_cfg_raw ? JSON.parse(poll_cfg_raw) : null; } catch(e){ poll_config = null; }
      const config = { content_id, interval_id, msg_type, file_id, target_type, id_mode, list_config: list_cfg, poll_config, schedule: { mode: schedule_mode, date: schedule_mode==='schedule' ? schedule_date : null, time: schedule_mode==='schedule' ? schedule_time : null, time12h: schedule_time12h, datetime: schedule_datetime, timezone: schedule_tz } };

      // Prepare update payload
      const data = {
            action: 'update',
            campaign_id: editingCampaignId,
            name: document.getElementById('campaign-name').value,
            accounts: selectedAccounts,
            interval_id: interval_id ? parseInt(interval_id,10) : null,
            id_action: JSON.stringify(config),
            contact: contact,
            speed: speed,
        };
        
        Swal.fire({
            title: 'جاري الحفظ...',
            allowOutsideClick: false,
            didOpen: () => Swal.showLoading()
        });
        
        fetch('api/manage_campaign.php', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        })
        .then(res => res.json())
        .then(data => {
            if (data.success) {
                Swal.fire({
                    icon: 'success',
                    title: 'تم!',
                    text: data.message,
                    timer: 2000,
                    showConfirmButton: false,
                    background: '#111827',
                    color: '#e5e7eb'
                });
                closeModal();
                loadCampaigns('Wa Sender');
            } else {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ',
                    text: data.message,
                    background: '#111827',
                    color: '#e5e7eb',
                    confirmButtonColor: '#667eea'
                });
            }
        });
    } else {
      let contact = document.getElementById("contact-list-select").value;
      const speed = (document.getElementById('speed-mode')||{}).value || 'slow';
      const content_id = document.getElementById('sender-content-select').value || null;
      const interval_id = document.getElementById('sender-interval-select').value || null;
      const msg_type = (document.getElementById('msg-type')||{}).value || 'text';
      const file_id = document.getElementById('sender-file-select').value || null;
      const target_type = (document.getElementById('target-type')||{}).value || 'person';
      const id_mode = (document.getElementById('id-mode')||{}).value || 'numbers';
      const schedule_mode = (document.getElementById('schedule-mode')||{}).value || 'now';
      const schedule_date = (document.getElementById('schedule-date')||{}).value || null;
      const schedule_time = (document.getElementById('schedule-time')||{}).value || null;
      const schedule_time12h = (schedule_mode==='schedule' && schedule_time) ? formatArabicTime12h(schedule_time) : null;
      const schedule_datetime = (schedule_mode==='schedule' && schedule_date && schedule_time) ? `${schedule_date}T${schedule_time}:00` : null;
      const schedule_tz = (Intl && Intl.DateTimeFormat && Intl.DateTimeFormat().resolvedOptions) ? (Intl.DateTimeFormat().resolvedOptions().timeZone||null) : null;
      const list_cfg_raw = document.getElementById('list-config').value || '';
      let list_cfg = null; try { list_cfg = list_cfg_raw ? JSON.parse(list_cfg_raw) : null; } catch(e){ list_cfg = null; }
      const poll_cfg_raw = document.getElementById('poll-config').value || '';
      let poll_config = null; try { poll_config = poll_cfg_raw ? JSON.parse(poll_cfg_raw) : null; } catch(e){ poll_config = null; }

      const config = { content_id, interval_id, msg_type, file_id, target_type, id_mode, list_config: list_cfg, poll_config, schedule: { mode: schedule_mode, date: schedule_mode==='schedule' ? schedule_date : null, time: schedule_mode==='schedule' ? schedule_time : null, time12h: schedule_time12h, datetime: schedule_datetime, timezone: schedule_tz } };

      // Create new campaign
      const data = {
            name: document.getElementById('campaign-name').value,
            accounts: selectedAccounts,
            tools: "Wa Sender",
            paltform: "WhatsApp",
            contact: contact,
            speed: speed,
            interval_id: interval_id ? parseInt(interval_id,10) : null,
            id_action: JSON.stringify(config)
        };
        
        Swal.fire({
            title: 'جاري الحفظ...',
            allowOutsideClick: false,
            didOpen: () => Swal.showLoading()
        });
        
        fetch('api/create_campaign.php', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(data)
        })
        .then(res => res.json())
        .then(data => {
            if (data.success) {
                Swal.fire({
                    icon: 'success',
                    title: 'تم!',
                    text: data.message,
                    timer: 2000,
                    showConfirmButton: false,
                    background: '#111827',
                    color: '#e5e7eb'
                });
                closeModal();
                loadCampaigns('Wa Sender');
            } else {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ',
                    text: data.message,
                    background: '#111827',
                    color: '#e5e7eb',
                    confirmButtonColor: '#667eea'
                });
            }
        });
    }
});

let allCampaigns = [];

function loadCampaigns(tool) {
    fetch('api/get_campaigns.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tool: tool })
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            allCampaigns = data.campaigns;
            applyFilters();
        } else {
            console.error('حدث خطأ:', data.message);
        }
    })
    .catch(err => console.error('خطأ في الاتصال:', err));
}


function applyFilters() {
    const statusFilter = document.getElementById('status-filter').value;
    const dateFilter = document.getElementById('date-filter').value;
    
    let filtered = allCampaigns;
    

    // Filter by status
    if (statusFilter !== 'all') {
        filtered = filtered.filter(c => c.status === statusFilter);
    }
    
    // Filter by date
    if (dateFilter) {
        filtered = filtered.filter(c => {
            const campaignDate = new Date(c.created_at).toISOString().split('T')[0];
            return campaignDate === dateFilter;
        });
    }
    
    renderCampaigns(filtered);
}

function clearFilters() {
    document.getElementById('status-filter').value = 'all';
    document.getElementById('date-filter').value = '';
    renderCampaigns(allCampaigns);
}

async function getTotalCount(contactId) {
    const res = await fetch("api/get_count_contacts.php?id=" + contactId);
    const text = await res.text(); 
    return text.trim(); 
}



async function renderCampaigns(campaigns) {
    const grid = document.getElementById('campaigns-grid');

    if (campaigns.length === 0) {
        grid.innerHTML = `
            <div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);">
                <i class="fas fa-inbox fa-3x" style="margin-bottom: 20px;"></i><br>
                لا توجد حملات حتى الآن
            </div>
        `;
        return;
    }

    // اجلب total لكل حملة أولًا
    const campaignsWithTotal = [];
    for (let campaign of campaigns) {
        const total = await getTotalCount(campaign.contact);
        campaignsWithTotal.push({ ...campaign, total });
    }

    // استخدم campaignsWithTotal في map
    grid.innerHTML = campaignsWithTotal.map(c => `
        <div class="campaign-card">
            <div class="campaign-name">
                <i class="fas fa-bullhorn"></i> ${c.name}
            </div>
            
            <div class="campaign-info">
                <div class="campaign-info-item">
                    <span class="campaign-info-label">
                        <i class="fas fa-check-double fa-beat" style="--fa-animation-duration: 2s; color: #10b981;"></i> العدد:
                    </span>
                    <span class="campaign-info-value">${c.true_count} - ${c.total}</span>
                </div>



                <div class="campaign-info-item">
                    <span class="campaign-info-label">
                        <i class="fa-solid fa-check fa-beat" style="color: #63E6BE;"></i> الناجح:
                    </span>
                    <span class="campaign-info-value">${c.true_count}</span>
                </div>


                <div class="campaign-info-item">
                    <span class="campaign-info-label">
                      <i class="fa-solid fa-xmark fa-beat" style="color: #e70d0d;"></i> الفاشل:
                    </span>
                    <span class="campaign-info-value">${c.false_count}</span>
                </div>



                <div class="campaign-info-item">
                    <span class="campaign-info-label">
                        <i class="fa-brands fa-whatsapp fa-beat" style="--fa-animation-duration: 2s; color: #1dc717ff;"></i> المنصة:
                    </span>
                    <span class="campaign-info-value">${c.paltform}</span>
                </div>



                <div class="campaign-info-item">
                    <span class="campaign-info-label">${getStatusIcon(c.status)} الحالة:</span>
                    <span class="status-badge status-${c.status}">
                        ${getStatusText(c.status)}
                    </span>
                </div>
            </div>

            <button class="campaign-actions-btn" onclick="toggleAccordion(${c.id})">
                <i class="fas fa-cog"></i> الإجراءات
            </button>

            <div class="accordion-content" id="accordion-${c.id}">
                <div class="accordion-section">
                    <div class="accordion-section-title">
                        <i class="fas fa-sliders-h"></i> الإعدادات
                    </div>
                    <div class="action-btns" id="actions-${c.id}">
                        ${getActionButtons(c)}
                    </div>
                </div>

                <div class="accordion-section">
                    <div class="accordion-section-title">
                        <i class="fas fa-tasks"></i> الإجراء
                    </div>
                    <div class="action-btns">
                        <button class="action-btn-full btn-send" onclick="sendMessage(${c.campaign_id})">
                            <i class="fa-solid fa-table-cells fa-flip" style="--fa-animation-duration: 3s;"></i> عرض في جدول
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `).join('');
}

function getStatusText(status) {
    const statusMap = {
        'pending': 'قيد الانتظار',
        'running': 'قيد التشغيل',
        'paused': 'متوقف مؤقتاً',
        'stopped': 'متوقف',
        'finished': 'منتهي'
    };
    return statusMap[status] || status;
}

function getStatusIcon(status) {
    const iconMap = {
        'pending': '<i class="fas fa-hourglass-half fa-spin" style="--fa-animation-duration: 3s; color: #f59e0b;"></i>',
        'running': '<i class="fas fa-spinner fa-spin" style="--fa-animation-duration: 1s; color: #3b82f6;"></i>',
        'paused': '<i class="fas fa-pause-circle fa-beat" style="--fa-animation-duration: 2s; color: #f59e0b;"></i>',
        'stopped': '<i class="fas fa-stop-circle fa-fade" style="--fa-animation-duration: 2s; color: #ef4444;"></i>',
        'finished': '<i class="fas fa-check-circle fa-bounce" style="--fa-animation-duration: 2s; color: #10b981;"></i>'
    };
    return iconMap[status] || '<i class="fas fa-info-circle"></i>';
}

function getActionButtons(campaign) {
    const status = campaign.status;
    const isReplyComments = campaign.tool === 'Reply Comments FB' && campaign.type_tools === 'Reply';
    let buttons = '';
    
    // For all campaigns (including Reply Comments), show stop/pause buttons based on status
    if (status === 'paused' || status === 'stopped') {
        buttons += `
            <button class="action-btn-small" style="background: linear-gradient(135deg, #10b981, #059669);" onclick="changeStatus(${campaign.id}, 'pending')">
                <i class="fas fa-play fa-beat" style="--fa-animation-duration: 1.5s; color: #fff;"></i> تشغيل
            </button>
        `;
    } else {
        buttons += `
            <button class="action-btn-small btn-stop" onclick="changeStatus(${campaign.id}, 'stopped')">
                <i class="fas fa-stop fa-fade" style="--fa-animation-duration: 2s;"></i> إيقاف
            </button>
            <button class="action-btn-small btn-pause" onclick="changeStatus(${campaign.id}, 'paused')">
                <i class="fas fa-pause fa-flip" style="--fa-animation-duration: 2s;"></i> إيقاف مؤقت
            </button>
        `;
    }
    
    buttons += `
        <button class="action-btn-small btn-edit" onclick="editCampaign(${campaign.id})">
            <i class="fas fa-edit fa-bounce" style="--fa-animation-duration: 2s;"></i> تعديل
        </button>
        <button class="action-btn-small btn-save" onclick="saveCampaign(${campaign.campaign_id})">
            <i class="fas fa-save fa-beat" style="--fa-animation-duration: 2s;"></i> حفظ
        </button>
    `;
    
    // Add delete button as full width
    buttons += `
        <button class="action-btn-full btn-delete" onclick="deleteCampaign(${campaign.id})">
            <i class="fas fa-trash fa-shake" style="--fa-animation-duration: 3s;"></i> حذف الحملة
        </button>
    `;



    
    return buttons;
}

function toggleAccordion(id) {
    const accordion = document.getElementById(`accordion-${id}`);
    accordion.classList.toggle('active');
}

// Campaign management functions
function changeStatus(id, newStatus) {
    const statusText = {
        'pending': 'تشغيل',
        'paused': 'إيقاف مؤقت',
        'stopped': 'إيقاف'
    }[newStatus] || newStatus;
    
    Swal.fire({
        title: 'تأكيد ' + statusText,
        text: 'هل أنت متأكد؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، ' + statusText,
        cancelButtonText: 'إلغاء',
        background: '#111827',
        color: '#e5e7eb',
        confirmButtonColor: newStatus === 'stopped' ? '#ef4444' : '#f59e0b'
    }).then((result) => {
        if (result.isConfirmed) {
            Swal.fire({
                title: 'جاري المعالجة...',
                allowOutsideClick: false,
                didOpen: () => Swal.showLoading()
            });
            
            fetch('api/manage_campaign.php', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    action: 'change_status',
                    campaign_id: id,
                    status: newStatus
                })
            })
            .then(res => res.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم!',
                        text: data.message,
                        timer: 2000,
                        showConfirmButton: false,
                        background: '#111827',
                        color: '#e5e7eb'
                    });
                    loadCampaigns('Wa Sender');
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        background: '#111827',
                        color: '#e5e7eb',
                        confirmButtonColor: '#667eea'
                    });
                }
            });
        }
    });
}

let editingCampaignId = null;

function editCampaign(id) {
    const campaign = allCampaigns.find(c => Number(c.id) === Number(id));
    if (!campaign) return;
    
    editingCampaignId = id;
    
    // Populate form with campaign data
    document.getElementById('campaign-name').value = campaign.name;
    
    // Parse and set accounts
    selectedAccounts = JSON.parse(campaign.token || '[]');
    renderSelectedAccounts();

    // Change modal title and button
    document.querySelector('.modal-title').innerHTML = '<i class="fas fa-edit"></i> تعديل الحملة';
    document.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ التعديلات';
    
    openModal();

    // Prefill from pram1 (config)
    try {
        const confRaw = campaign.pram1 || campaign.id_action || '';
        if (confRaw) {
            const conf = typeof confRaw === 'string' ? JSON.parse(confRaw) : confRaw;
            // contact list
            setSelectValueWhenReady('contact-list-select', campaign.contact);
            // content / interval (async selects)
            setSelectValueWhenReady('sender-content-select', conf.content_id);
            // interval can be in pram1.interval_id OR campaigns.interval_id OR campaigns.interval
            (function(){
                const ival = (conf && (conf.interval_id||conf.interval)) || campaign.interval_id || campaign.interval;
                setSelectValueWhenReady('sender-interval-select', ival);
            })();
            // msg type cards + file select visibility
            const mtHidden = document.getElementById('msg-type'); if (mtHidden && conf.msg_type) mtHidden.value = conf.msg_type;
            const cardsWrap = document.getElementById('msgTypeCards');
            if (cardsWrap && conf.msg_type) {
                cardsWrap.querySelectorAll('.option-card').forEach(el=>{
                    if (el.getAttribute('data-value') === String(conf.msg_type)) el.classList.add('selected'); else el.classList.remove('selected');
                });
            }
            (function(v){
                v = String(v||'').toLowerCase();
                const apply = ()=>{
                    const wrap = document.getElementById('sender-file-wrap');
                    const listBtn = document.getElementById('openListBuilderBtn');
                    if (v==='text_image' || v==='text_video' || v==='text_file') { if (wrap) wrap.style.display=''; if (listBtn) listBtn.style.display='none'; }
                    else if (v==='list') { if (wrap) wrap.style.display='none'; if (listBtn) listBtn.style.display=''; }
                    else { if (wrap) wrap.style.display='none'; if (listBtn) listBtn.style.display='none'; }
                };
                // apply now and twice shortly to avoid race with layout
                apply();
                setTimeout(apply, 120);
                setTimeout(apply, 300);
            })(conf.msg_type||'text');
            // file id (async)
            setSelectValueWhenReady('sender-file-select', conf.file_id);
            // target segmented
            const tgtH = document.getElementById('target-type'); if (tgtH && conf.target_type) tgtH.value = conf.target_type;
            const tgtSeg = document.getElementById('targetSegment');
            if (tgtSeg && conf.target_type) {
                tgtSeg.querySelectorAll('button').forEach(b=>{ b.classList.toggle('selected', b.getAttribute('data-value')===String(conf.target_type)); });
            }
            // id mode segmented
            const idH = document.getElementById('id-mode'); if (idH && conf.id_mode) idH.value = conf.id_mode;
            const idSeg = document.getElementById('idModeSegment');
            if (idSeg && conf.id_mode) {
                idSeg.querySelectorAll('button').forEach(b=>{ b.classList.toggle('selected', b.getAttribute('data-value')===String(conf.id_mode)); });
            }
            // list builder state
            if (conf.list_config) {
                try { lbState = conf.list_config; } catch(e) { /* ignore */ }
                const lc = document.getElementById('list-config'); if (lc) lc.value = JSON.stringify(lbState);
            }
            // poll builder state
            if (conf.poll_config) {
                try { pbState = conf.poll_config; } catch(e) { /* ignore */ }
                const pc = document.getElementById('poll-config'); if (pc) pc.value = JSON.stringify(pbState);
            }
            // schedule prefill
            (function(){
              const sch = conf.schedule || {};
              const schH = document.getElementById('schedule-mode');
              const schSeg = document.getElementById('scheduleSegment');
              const schFields = document.getElementById('scheduleFields');
              const sdate = document.getElementById('schedule-date');
              const stime = document.getElementById('schedule-time');
              if (schH) schH.value = sch.mode || 'now';
              if (schSeg && schH) {
                schSeg.querySelectorAll('button').forEach(b=>{ b.classList.toggle('selected', b.getAttribute('data-value')===String(schH.value)); });
              }
              if (sdate) sdate.value = sch.date || '';
              if (stime) stime.value = sch.time || '';
              if (schFields && schH) schFields.style.display = (schH.value==='schedule') ? '' : 'none';
              updateScheduleTimeHint();
            })();
        }
        // speed (from column speed)
        if (campaign.speed) {
            const sm = document.getElementById('speed-mode'); if (sm) sm.value = campaign.speed;
            const scs = document.getElementById('speedCardSlow'); const scf = document.getElementById('speedCardFast');
            if (scs && scf) {
                scs.classList.toggle('selected', campaign.speed==='slow');
                scf.classList.toggle('selected', campaign.speed==='fast');
            }
        }
    } catch (e) { /* ignore parse errors */ }
}


function saveCampaign(id) {
   
     
     window.location.href = "api/save_data.php?camp_id=" + id+"&tool=Extract-Messages-WA-send";
}

 

function deleteCampaign(id) {
    Swal.fire({
        title: 'حذف الحملة',
        text: 'هل أنت متأكد من حذف هذه الحملة؟ لا يمكن التراجع عن هذا الإجراء',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، حذف',
        cancelButtonText: 'إلغاء',
        background: '#111827',
        color: '#e5e7eb',
        confirmButtonColor: '#991b1b'
    }).then((result) => {
        if (result.isConfirmed) {
            Swal.fire({
                title: 'جاري الحذف...',
                allowOutsideClick: false,
                didOpen: () => Swal.showLoading()
            });
            
            fetch('api/manage_campaign.php', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    action: 'delete',
                    campaign_id: id
                })
            })
            .then(res => res.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم!',
                        text: data.message,
                        timer: 2000,
                        showConfirmButton: false,
                        background: '#111827',
                        color: '#e5e7eb'
                    });
                    loadCampaigns('Wa Sender');
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        background: '#111827',
                        color: '#e5e7eb',
                        confirmButtonColor: '#667eea'
                    });
                }
            });
        }
    });
}

function viewData(id) {
    Swal.fire({
        title: 'البيانات',
        text: 'سيتم عرض البيانات قريباً',
        icon: 'info',
        background: '#111827',
        color: '#e5e7eb',
        confirmButtonColor: '#10b981'
    });
}

// ===== Contacts Table (modal) =====
let contactsState = { campaign_id: null, page: 1, per_page: 25, q: '', data: [], total: 0, total_pages: 0, selected: new Set(), selectedData: {} };
let contactsSearchTimer = null;

function sendMessage(campaignId) {
  contactsState.campaign_id = campaignId;
  contactsState.page = 1;
  contactsState.q = '';
  contactsState.selected = new Set();
  contactsState.selectedData = {};
  document.getElementById('contactsSearch').value = '';
  document.getElementById('contactsPerPage').value = '25';
  document.getElementById('contactsModal').style.display = 'block';
  loadContacts();
}

function closeContactsModal() {
  document.getElementById('contactsModal').style.display = 'none';
}

function debouncedSearchContacts() {
  clearTimeout(contactsSearchTimer);
  contactsSearchTimer = setTimeout(() => {
    contactsState.q = document.getElementById('contactsSearch').value.trim();
    reloadContactsPage(1);
  }, 300);
}

function reloadContactsPage(page) {
  contactsState.page = page;
  contactsState.per_page = parseInt(document.getElementById('contactsPerPage').value, 10) || 25;
  loadContacts();
}

function loadContacts() {
  const params = new URLSearchParams({
    campaign_id: contactsState.campaign_id,
    page: contactsState.page,
    per_page: contactsState.per_page,
    q: contactsState.q
  });
  fetch('api/wa_rep.php?' + params.toString())
    .then(r=>r.json())
    .then(j=>{
      if (!j.success) throw new Error(j.message || 'failed');
      contactsState.data = j.data || [];
      contactsState.total = j.total || 0;
      contactsState.total_pages = j.total_pages || 0;
      renderContactsTable();
      renderContactsPagination();
      document.getElementById('contactsInfo').textContent = `الصفحة ${contactsState.page} من ${contactsState.total_pages} — إجمالي ${contactsState.total}`;
    })
    .catch(e=>{
      const tb = document.getElementById('contactsTableBody');
      tb.innerHTML = `<tr><td colspan="4" style="padding:20px; text-align:center; color:#ef4444;">خطأ: ${e.message}</td></tr>`;
    });
}

function renderContactsTable() {
  const tb = document.getElementById('contactsTableBody');
  if (!contactsState.data.length) {
    tb.innerHTML = '<tr><td colspan="4" style="padding:20px; text-align:center; color:var(--text-secondary);">لا توجد بيانات</td></tr>';
    return;
  }
  tb.innerHTML = contactsState.data.map((row, idx) => {
    const id = `${contactsState.page}-${idx}-${row.phone}`;
      const st = `${contactsState.page}-${idx}-${row.st}`;
    
    const checked = contactsState.selected.has(id) ? 'checked' : '';
    return `
      <tr>
        <td style="padding:8px; text-align:center;"><input type="checkbox" data-id="${id}" data-phone="${escapeHtml(row.phone||'')}" data-name="${escapeHtml((row.st||row.st||'').toString())}" onchange="toggleSelectContact(this)" ${checked}></td>
        <td style="padding:8px; direction:ltr;">${escapeHtml(row.phone || '')}</td>
        <td style="padding:8px;">${escapeHtml(row.st || '')}</td>
      </tr>`;
  }).join('');
  // reset header selectAll
  const selAll = document.getElementById('contactsSelectAll');
  if (selAll) selAll.checked = false;
}

function renderContactsPagination() {
  const el = document.getElementById('contactsPagination');
  const total = contactsState.total_pages;
  const cur = contactsState.page;
  if (!total || total <= 1) { el.innerHTML = ''; return; }
  let html = '';
  const btn = (p, t, dis=false) => `<button class="action-btn-small" style="background: linear-gradient(135deg, #3b82f6, #2563eb); ${dis?'opacity:.5; cursor:not-allowed;':''}" ${dis?'disabled':''} onclick="reloadContactsPage(${p})">${t}</button>`;
  html += btn(1, 'الأولى', cur===1);
  html += btn(Math.max(1, cur-1), 'السابق', cur===1);
  // window of pages
  const start = Math.max(1, cur-2);
  const end = Math.min(total, cur+2);
  for (let p=start; p<=end; p++) {
    html += `<button class="action-btn-small" style="${p===cur?'background: linear-gradient(135deg, #10b981, #059669);':'background: linear-gradient(135deg, #6b7280, #4b5563);'}" onclick="reloadContactsPage(${p})">${p}</button>`;
  }
  html += btn(Math.min(total, cur+1), 'التالي', cur===total);
  html += btn(total, 'الأخيرة', cur===total);
  el.innerHTML = html;
}

function toggleSelectAllContacts(chk) {
  const inputs = document.querySelectorAll('#contactsTableBody input[type=checkbox]');
  inputs.forEach(i=>{
    i.checked = chk.checked;
    const id = i.getAttribute('data-id');
    const phone = i.getAttribute('data-phone')||'';
    const name = i.getAttribute('data-name')||'';
    if (!id) return;
    if (chk.checked) {
      contactsState.selected.add(id);
      if (phone) contactsState.selectedData[phone] = { identifier: phone, name: name };
    } else {
      contactsState.selected.delete(id);
      if (phone && contactsState.selectedData[phone]) delete contactsState.selectedData[phone];
    }
  });
}

function toggleSelectContact(input) {
  const id = input.getAttribute('data-id');
  const phone = input.getAttribute('data-phone')||'';
  const name = input.getAttribute('data-name')||'';
  if (!id) return;
  if (input.checked) {
    contactsState.selected.add(id);
    if (phone) contactsState.selectedData[phone] = { identifier: phone, name: name };
  } else {
    contactsState.selected.delete(id);
    if (phone && contactsState.selectedData[phone]) delete contactsState.selectedData[phone];
  }
}

function openAddContactsModal(){
  const count = Object.keys(contactsState.selectedData).length;
  if (count === 0) {
    Swal.fire({ icon:'warning', title:'تنبيه', text:'يرجى تحديد صف واحد على الأقل', background:'#111827', color:'#e5e7eb', confirmButtonColor:'#667eea' });
    return;
  }
  const defName = 'Contacts ' + new Date().toLocaleString('en-GB');
  document.getElementById('contactsListName').value = defName;
  document.getElementById('addContactsCountHint').textContent = `سيتم إضافة ${count} جهة اتصال`;
  document.getElementById('addContactsModal').style.display = 'block';
}

function closeAddContactsModal(){
  document.getElementById('addContactsModal').style.display = 'none';
}

async function submitAddContactsList(){
  const name = document.getElementById('contactsListName').value.trim();
  const count = Object.keys(contactsState.selectedData).length;
  if (!name) {
    Swal.fire({ icon:'warning', title:'تنبيه', text:'يرجى إدخال اسم جهة الاتصال', background:'#111827', color:'#e5e7eb', confirmButtonColor:'#667eea' });
    return;
  }
  const dataArr = Object.values(contactsState.selectedData);
  try {
    const res = await fetch('api/contacts_add.php', {
      method:'POST',
      headers:{'Content-Type':'application/json'},
      credentials: 'same-origin',
      body: JSON.stringify({ name, platform:'whatsapp', type:'csv', count, data: dataArr })
    });
    const j = await res.json();
    if (!j.success) throw new Error(j.message||'failed');
    Swal.fire({ icon:'success', title:'تم!', text:'تم إضافة جهات الاتصال', timer:1800, showConfirmButton:false, background:'#111827', color:'#e5e7eb' });
    closeAddContactsModal();
  } catch(e) {
    Swal.fire({ icon:'error', title:'خطأ', text:e.message, background:'#111827', color:'#e5e7eb', confirmButtonColor:'#667eea' });
  }
}

function escapeHtml(s){
  return String(s).replace(/[&<>\"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'}[c]));
}

function manageComments(id) {
    // Open manage comments modal
    document.getElementById('manageCommentsModal').style.display = 'block';
}

function closeCommentsModal() {
    document.getElementById('manageCommentsModal').style.display = 'none';
    document.getElementById('commentsForm').reset();
    commentsSelectedAccounts = [];
    document.getElementById('comments-selected-accounts').innerHTML = '';
}

function addCommentAccount() {
    const select = document.getElementById('comments-account-select');
    const selectedOption = select.options[select.selectedIndex];
    
    if (!select.value) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'يرجى اختيار حساب',
            background: '#111827',
            color: '#e5e7eb',
            confirmButtonColor: '#667eea'
        });
        return;
    }
    
    if (commentsSelectedAccounts.some(acc => acc.account_uid === select.value)) {
        Swal.fire({
            icon: 'info',
            title: 'تنبيه',
            text: 'هذا الحساب مضاف بالفعل',
            background: '#111827',
            color: '#e5e7eb',
            confirmButtonColor: '#667eea'
        });
        return;
    }
    
    const account = {
        account_uid: select.value,
        name: selectedOption.dataset.name
    };
    
    commentsSelectedAccounts.push(account);
    renderCommentsSelectedAccounts();
    select.value = '';
}

function removeCommentAccount(account_uid) {
    commentsSelectedAccounts = commentsSelectedAccounts.filter(acc => acc.account_uid !== account_uid);
    renderCommentsSelectedAccounts();
}

function renderCommentsSelectedAccounts() {
    const container = document.getElementById('comments-selected-accounts');
    container.innerHTML = commentsSelectedAccounts.map(acc => `
        <div class="account-tag">
            <i class="fas fa-user-circle"></i>
            ${acc.name}
            <span class="remove-account" onclick="removeCommentAccount('${acc.account_uid}')">✕</span>
        </div>
    `).join('');
}

// Handle comments form submission
document.getElementById('commentsForm').addEventListener('submit', function(e) {
    e.preventDefault();
    
    if (commentsSelectedAccounts.length === 0) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'يرجى إضافة حساب واحد على الأقل',
            background: '#111827',
            color: '#e5e7eb',
            confirmButtonColor: '#667eea'
        });
        return;
    }
    
    const data = {
        name: document.getElementById('comments-campaign-name').value,
        accounts: commentsSelectedAccounts,
    };
    
    Swal.fire({
        title: 'جاري الحفظ...',
        allowOutsideClick: false,
        didOpen: () => Swal.showLoading()
    });
    
    fetch('api/create_comments_campaign.php', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(data)
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تم!',
                text: data.message,
                timer: 2000,
                showConfirmButton: false,
                background: '#111827',
                color: '#e5e7eb'
            });
            closeCommentsModal();
            loadCampaigns('Wa Sender');
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message,
                background: '#111827',
                color: '#e5e7eb',
                confirmButtonColor: '#667eea'
            });
        }
    });
  setupSegmented('targetSegment','target-type');
  setupSegmented('idModeSegment','id-mode');
});
</script>

<?php include 'includes/footer.php'; ?>
