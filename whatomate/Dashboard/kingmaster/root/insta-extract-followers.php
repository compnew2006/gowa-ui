<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; 

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

$page_title = "استخراج المتابعين | Kingmaster";
$page_css = ['https://kingmaster.info/css/f-w-i.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<div class="tools-container">
    <div class="tools-header">
        <div class="tools-title">
            <i class="fas fa-users-viewfinder fa-spin" style="--fa-animation-duration: 3s; color: #10b981;"></i>
            استخراج المتابعين (Followers / Following)
        </div>
        <button class="create-campaign-btn" onclick="openModal()">
            <i class="fas fa-plus-circle"></i>
            إنشاء حملة جديدة
        </button>
    </div>
    
    <div class="filters-section">
        <div class="filter-label"><i class="fas fa-filter fa-spin" style="--fa-animation-duration: 3s;"></i> تصفية:</div>
        <select class="filter-select" id="status-filter" onchange="applyFilters()">
            <option value="all">جميع الحالات</option>
            <option value="pending">قيد الانتظار</option>
            <option value="running">قيد التشغيل</option>
            <option value="paused">متوقف مؤقتاً</option>
            <option value="stopped">متوقف</option>
            <option value="finished">منتهي</option>
        </select>
        <input type="date" class="filter-select" id="date-filter" onchange="applyFilters()" style="padding: 10px 15px; cursor: pointer; color-scheme: dark;" placeholder="التاريخ">
        <button class="clear-filters-btn" onclick="clearFilters()"><i class="fas fa-times"></i> إعادة تعيين</button>
    </div>
    
    <div class="campaigns-grid" id="campaigns-grid">
        <div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);">جاري تحميل الحملات...</div>
    </div>
</div>

<div id="campaignModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title"><i class="fas fa-rocket"></i> إنشاء حملة جديدة</div>
            <span class="close-modal" onclick="closeModal()">&times;</span>
        </div>
        
        <form id="campaignForm">
            <div class="form-group">
                <label class="form-label"><i class="fas fa-tag"></i> اسم الحملة</label>
                <input type="text" class="form-input" id="campaign-name" placeholder="أدخل اسم الحملة" required>
            </div>
            
            <div class="form-group">
                <label class="form-label"><i class="fas fa-users"></i> الحسابات</label>
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
                <label class="form-label"><i class="fas fa-crosshairs"></i> المستهدف</label>
                <input type="text" class="form-input" id="target-user" placeholder="مثال: cristiano أو المعرف الرقمي" required>
                <small style="color: var(--text-secondary); font-size: 12px; margin-top: 5px; display: block;">* يمكنك كتابة اسم المستخدم (Username) أو رابط الحساب ليتم تحويله تلقائياً</small>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-filter"></i> نوع الاستخراج</label>
                <select class="form-select" id="extract-type" required>
                    <option value="followers">المتابِعون (Followers)</option>
                    <option value="following">الذين يتابعهم (Following)</option>
                </select>
            </div>
            
            <button type="submit" class="submit-btn" style="margin-top: 15px;"><i class="fas fa-save"></i> حفظ الحملة</button>
        </form>
    </div>
</div>

<div id="contactsModal" class="modal">
  <div class="modal-content" style="max-width: 900px; background: #1e293b; border-radius: 16px; border: 1px solid #334155; padding: 24px; box-shadow: 0 10px 25px rgba(0,0,0,0.5);">
    <div class="modal-header" style="border-bottom: 1px solid #334155; padding-bottom: 15px;">
      <div class="modal-title" style="font-size: 1.25rem; font-weight: bold; color: #f8fafc;"><i class="fas fa-address-book" style="color: #3b82f6;"></i> المستخدمين المستخرجين</div>
      <span class="close-modal" onclick="closeContactsModal()" style="cursor: pointer; font-size: 1.5rem; color: #94a3b8;">&times;</span>
    </div>

    <div class="filters-section" style="display: flex; gap: 15px; margin-bottom: 20px; border:none; padding:0; background:transparent;">
      <div style="flex: 1; position: relative;">
         <i class="fas fa-search" style="position: absolute; right: 15px; top: 50%; transform: translateY(-50%); color: #94a3b8;"></i>
         <input type="text" id="contactsSearch" class="form-input" placeholder="بحث بالمعرف أو الاسم..." oninput="debouncedSearchContacts()" style="width: 100%; padding: 10px 40px 10px 15px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff; outline: none;">
      </div>
      <select id="contactsPerPage" class="form-select" onchange="reloadContactsPage(1)" style="width: 130px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff;">
        <option value="25">25 نتيجة</option><option value="50">50 نتيجة</option><option value="100">100 نتيجة</option>
      </select>
      <button class="action-btn-small btn-save" onclick="openAddContactsModal()" style="white-space:nowrap; padding: 10px 15px; border-radius: 8px; background: linear-gradient(135deg, #10b981, #059669);"><i class="fas fa-user-plus"></i> حفظ في قائمة</button>
    </div>

    <div style="overflow-x: auto; background: #0f172a; border-radius: 12px; border: 1px solid #334155;">
      <table style="width: 100%; border-collapse: collapse; text-align: right; color: #e2e8f0; table-layout: fixed;">
        <thead>
          <tr style="background: #1e293b; border-bottom: 2px solid #334155;">
            <th style="width: 10%; padding: 15px; text-align: center;"><input type="checkbox" id="contactsSelectAll" onchange="toggleSelectAllContacts(this)"></th>
            <th style="width: 30%; padding: 15px; font-weight: 600; color: #94a3b8;">اسم المستخدم</th>
            <th style="width: 30%; padding: 15px; font-weight: 600; color: #94a3b8;">الاسم بالكامل</th>
            <th style="width: 30%; padding: 15px; font-weight: 600; color: #94a3b8; text-align: center;">المعرف (IG ID)</th>
          </tr>
        </thead>
        <tbody id="contactsTableBody"><tr><td colspan="4" style="padding:30px; text-align:center; color:#94a3b8;">جاري التحميل...</td></tr></tbody>
      </table>
    </div>

    <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 20px;">
      <div id="contactsInfo" style="color: #94a3b8; font-size: 0.95rem;">—</div>
      <div id="contactsPagination" style="display: flex; gap: 6px; flex-wrap: wrap;"></div>
    </div>
  </div>
</div>

<div id="addContactsModal" class="modal">
  <div class="modal-content" style="max-width: 400px; background: #1e293b; border: 1px solid #334155; border-radius: 12px;">
    <div class="modal-header" style="border-bottom: 1px solid #334155;"><div class="modal-title" style="color: #fff;"><i class="fas fa-list"></i> حفظ في جهات الاتصال</div><span class="close-modal" onclick="closeAddContactsModal()" style="color: #94a3b8; cursor: pointer;">&times;</span></div>
    <div class="form-group" style="margin-top: 15px;"><label class="form-label" style="color: #cbd5e1;"><i class="fas fa-tag"></i> اسم القائمة</label><input type="text" id="contactsListName" class="form-input" style="background: #0f172a; border: 1px solid #334155; color: #fff;" placeholder="مثال: قائمة المتابعين" /></div>
    <div class="hint" id="addContactsCountHint" style="color: #10b981; margin-bottom: 15px;">—</div>
    <button class="submit-btn" style="width: 100%; background: #3b82f6;" onclick="submitAddContactsList()"><i class="fas fa-save"></i> حفظ القائمة</button>
  </div>
</div>

<script>
let selectedAccounts = [];
let allAccounts = [];
let allCampaigns = [];
let editingCampaignId = null;

const TOOL_NAME = 'Extract Follows IG';

document.addEventListener('DOMContentLoaded', function() {
    loadAccounts();
    loadCampaigns(TOOL_NAME);
});

function openModal() { document.getElementById('campaignModal').style.display = 'block'; }
function closeModal() {
    document.getElementById('campaignModal').style.display = 'none';
    document.getElementById('campaignForm').reset();
    selectedAccounts = [];
    document.getElementById('selected-accounts').innerHTML = '';
    editingCampaignId = null;
    document.querySelector('.modal-title').innerHTML = '<i class="fas fa-rocket"></i> إنشاء حملة جديدة';
}

window.onclick = function(event) {
    if (event.target == document.getElementById('campaignModal')) closeModal();
    if (event.target == document.getElementById('contactsModal')) closeContactsModal();
    if (event.target == document.getElementById('addContactsModal')) closeAddContactsModal();
}

function loadAccounts() {
    fetch('api/get_accounts_ig.php').then(res => res.json()).then(data => {
        if (data.success) {
            data.accounts.forEach(account => {
                document.getElementById('account-select').innerHTML += `<option value="${account.account_uid}" data-name="${account.name}">${account.name}</option>`;
            });
        }
    });
}

function addAccount() {
    const select = document.getElementById('account-select');
    if (!select.value) return Swal.fire({ icon: 'warning', text: 'يرجى اختيار حساب', background: '#111827', color: '#e5e7eb' });
    if (selectedAccounts.length >= 1) return Swal.fire({ icon: 'info', text: 'يمكنك اختيار حساب واحد فقط للاستخراج', background: '#111827', color: '#e5e7eb' });
    selectedAccounts.push({ account_uid: select.value, name: select.options[select.selectedIndex].dataset.name });
    renderSelectedAccounts();
    select.value = '';
}

function removeAccount(uid) { selectedAccounts = selectedAccounts.filter(acc => acc.account_uid !== uid); renderSelectedAccounts(); }
function renderSelectedAccounts() {
    document.getElementById('selected-accounts').innerHTML = selectedAccounts.map(acc => `
        <div class="account-tag"><i class="fa-brands fa-instagram"></i> ${acc.name} <span class="remove-account" onclick="removeAccount('${acc.account_uid}')">✕</span></div>
    `).join('');
}

document.getElementById('campaignForm').addEventListener('submit', function(e) {
    e.preventDefault();
    if (selectedAccounts.length === 0) return Swal.fire({ icon: 'warning', text: 'يرجى إضافة حساب', background: '#111827', color: '#e5e7eb' });
    
    const target = document.getElementById('target-user').value.trim();
    const type = document.getElementById('extract-type').value;
    const combinedAction = target + "::" + type; 

    const data = {
        name: document.getElementById('campaign-name').value,
        accounts: selectedAccounts,
        tools: TOOL_NAME,
        paltform: "Instagram",
        id_action: combinedAction 
    };
    
    if (editingCampaignId) { data.action = 'update'; data.campaign_id = editingCampaignId; }
    
    Swal.fire({ title: 'جاري الحفظ...', didOpen: () => Swal.showLoading() });
    fetch(editingCampaignId ? 'api/manage_campaign.php' : 'api/create_campaign.php', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data) })
    .then(res => res.json()).then(data => {
        if (data.success) { Swal.fire({ icon: 'success', title: 'تم!', timer: 1500, showConfirmButton: false, background: '#111827', color: '#e5e7eb' }); closeModal(); loadCampaigns(TOOL_NAME); }
        else { Swal.fire({ icon: 'error', title: 'خطأ', text: data.message, background: '#111827', color: '#e5e7eb' }); }
    });
});

function loadCampaigns(tool) { fetch('api/get_campaigns.php', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ tool: tool }) }).then(res => res.json()).then(data => { if (data.success) { allCampaigns = data.campaigns; applyFilters(); } }); }
function applyFilters() { const st = document.getElementById('status-filter').value; renderCampaigns(st === 'all' ? allCampaigns : allCampaigns.filter(c => c.status === st)); }
function clearFilters() { document.getElementById('status-filter').value = 'all'; document.getElementById('date-filter').value = ''; renderCampaigns(allCampaigns); }

// ========================================================
// التصميم المتطابق تماماً مع صفحة اللايكات
// ========================================================
function renderCampaigns(campaigns) {
    const grid = document.getElementById('campaigns-grid');
    
    if (campaigns.length === 0) {
        grid.innerHTML = '<div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);"><i class="fas fa-inbox fa-3x" style="margin-bottom: 20px;"></i><br>لا توجد حملات حتى الآن</div>';
        return;
    }
    
    grid.innerHTML = campaigns.map(c => {
        // فك دمج المستهدف والنوع
        const rawTarget = c.pram1 || c.id_action || "";
        let targetUser = rawTarget;
        let extractType = "المتابعون";
        
        if (rawTarget.includes("::")) {
            const parts = rawTarget.split("::");
            targetUser = parts[0];
            extractType = parts[1] === "following" ? "الذين يتابعهم" : "المتابعون";
        }

        return `
        <div class="campaign-card">
            <div class="campaign-name">
                <i class="fas fa-bullhorn"></i> ${c.name}
            </div>
            
            <div class="campaign-info">
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fas fa-check-double fa-beat" style="--fa-animation-duration: 2s; color: #10b981;"></i> العدد:</span>
                    <span class="campaign-info-value">${c.true_count || 0}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fa-brands fa-instagram fa-beat" style="--fa-animation-duration: 2s; color: #1dc717ff;"></i> المنصة:</span>
                    <span class="campaign-info-value">${c.paltform || 'Instagram'}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fas fa-crosshairs fa-beat" style="--fa-animation-duration: 2s; color: #f59e0b;"></i> المستهدف:</span>
                    <span class="campaign-info-value" style="direction:ltr; font-weight:bold;">${targetUser}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fas fa-filter fa-beat" style="--fa-animation-duration: 2s; color: #3b82f6;"></i> النوع:</span>
                    <span class="campaign-info-value">${extractType}</span>
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
                         <button class="action-btn-full btn-send" onclick="viewResults('${c.campaign_id || c.id}')">
                            <i class="fa-solid fa-table-cells fa-flip" style="--fa-animation-duration: 3s;"></i> عرض في جدول
                        </button>
                    </div>
                </div>
            </div>
        </div>
        `;
    }).join('');
}

function getActionButtons(campaign) {
    const status = campaign.status;
    let buttons = '';
    
    // أزرار التشغيل والإيقاف
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
    
    // زري التعديل والحفظ (للإكسل)
    buttons += `
        <button class="action-btn-small btn-edit" onclick="editCampaign(${campaign.id})">
            <i class="fas fa-edit fa-bounce" style="--fa-animation-duration: 2s;"></i> تعديل
        </button>
        <button class="action-btn-small btn-save" onclick="saveCampaign('${campaign.campaign_id || campaign.id}')">
            <i class="fas fa-save fa-beat" style="--fa-animation-duration: 2s;"></i> حفظ
        </button>
    `;
    
    // زر الحذف بالعرض الكامل
    buttons += `
        <button class="action-btn-full btn-delete" style="margin-top: 10px;" onclick="deleteCampaign(${campaign.id})">
            <i class="fas fa-trash fa-shake" style="--fa-animation-duration: 3s;"></i> حذف الحملة
        </button>
    `;
    
    return buttons;
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

// دالة تحميل الإكسل المرتبطة بزر "حفظ"
function saveCampaign(id) {
    window.location.href = "api/save_data.php?camp_id=" + id + "&tool=" + encodeURIComponent(TOOL_NAME);
}

// دالة فتح وإغلاق الإجراءات
function toggleAccordion(id) { 
    document.getElementById(`accordion-${id}`).classList.toggle('active'); 
}

function getStatusText(status) { return { 'pending': 'قيد الانتظار', 'running': 'قيد التشغيل', 'paused': 'متوقف مؤقتاً', 'stopped': 'متوقف', 'finished': 'منتهي' }[status] || status; }
function changeStatus(id, st) { fetch('api/manage_campaign.php', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ action: 'change_status', campaign_id: id, status: st }) }).then(()=>loadCampaigns(TOOL_NAME)); }
function deleteCampaign(id) { Swal.fire({ title: 'تأكيد الحذف', icon: 'warning', showCancelButton: true, background: '#111827', color: '#fff', confirmButtonColor: '#ef4444' }).then(r => { if (r.isConfirmed) fetch('api/manage_campaign.php', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ action: 'delete', campaign_id: id }) }).then(()=>loadCampaigns(TOOL_NAME)); }); }

function editCampaign(id) {
    const c = allCampaigns.find(x => Number(x.id) === Number(id)); if (!c) return;
    editingCampaignId = id;
    
    const rawTarget = c.pram1 || c.id_action || "";
    let targetUser = rawTarget;
    let extractType = "followers";
    if (rawTarget.includes("::")) {
        const parts = rawTarget.split("::");
        targetUser = parts[0];
        extractType = parts[1];
    }
    
    document.getElementById('campaign-name').value = c.name;
    document.getElementById('target-user').value = targetUser;
    document.getElementById('extract-type').value = extractType;
    
    selectedAccounts = JSON.parse(c.token || '[]'); renderSelectedAccounts();
    document.querySelector('.modal-title').innerHTML = '<i class="fas fa-edit"></i> تعديل الحملة';
    openModal();
}

// Contacts Modal
let contactsState = { campaign_id: null, page: 1, per_page: 25, q: '', data: [], total: 0, total_pages: 0, selected: new Set(), selectedData: {} }; let contactsTimer;
function viewResults(cid) { contactsState.campaign_id = cid; contactsState.page=1; contactsState.q=''; contactsState.selected.clear(); contactsState.selectedData={}; if(document.getElementById('contactsSearch')) document.getElementById('contactsSearch').value=''; document.getElementById('contactsPerPage').value='25'; document.getElementById('contactsModal').style.display='block'; loadContacts(); }
function closeContactsModal() { document.getElementById('contactsModal').style.display='none'; }
function debouncedSearchContacts() { clearTimeout(contactsTimer); contactsTimer = setTimeout(()=>{ contactsState.q = document.getElementById('contactsSearch').value.trim(); reloadContactsPage(1); }, 300); }
function reloadContactsPage(p) { contactsState.page=p; contactsState.per_page = document.getElementById('contactsPerPage').value; loadContacts(); }

function loadContacts() {
  const p = new URLSearchParams({ table: 'ig_follow', campaign_id: contactsState.campaign_id, page: contactsState.page, per_page: contactsState.per_page, q: contactsState.q });
  fetch('api/ig_basic_info.php?' + p).then(r=>r.json()).then(j => {
      contactsState.data = j.data || []; contactsState.total = j.total || 0; contactsState.total_pages = j.total_pages || 0;
      const tb = document.getElementById('contactsTableBody');
      if(!j.data.length) return tb.innerHTML = '<tr><td colspan="4" style="padding:40px; text-align:center; color:#94a3b8;">لا توجد بيانات</td></tr>';
      tb.innerHTML = j.data.map(r => `
        <tr style="border-bottom: 1px solid #1e293b;">
          <td style="padding:12px; text-align:center;"><input type="checkbox" data-id="${r.ig_user_id}" data-username="${r.username}" data-fullname="${r.full_name}" onchange="toggleSelectContact(this)" ${contactsState.selected.has(r.ig_user_id)?'checked':''}></td>
          <td style="padding:12px; text-align:right;">${r.username||'--'}</td><td style="padding:12px; text-align:right;">${r.full_name||'--'}</td>
          <td style="padding:12px; text-align:center; font-family:monospace; color:#cbd5e1;">${r.ig_user_id}</td>
        </tr>`).join('');
      document.getElementById('contactsInfo').textContent = `صفحة ${j.page} من ${j.total_pages} — إجمالي ${j.total}`;
      const pg = document.getElementById('contactsPagination'); let h='';
      if(j.page>1) h+=`<button style="background:#3b82f6; color:#fff; border:none; padding:5px 10px; border-radius:5px; cursor:pointer;" onclick="reloadContactsPage(${j.page-1})">السابق</button> `;
      if(j.page<j.total_pages) h+=`<button style="background:#3b82f6; color:#fff; border:none; padding:5px 10px; border-radius:5px; cursor:pointer;" onclick="reloadContactsPage(${j.page+1})">التالي</button>`;
      pg.innerHTML = h;
  });
}
function toggleSelectContact(cb) { if(cb.checked){ contactsState.selected.add(cb.dataset.id); contactsState.selectedData[cb.dataset.id]={identifier: cb.dataset.id, name: cb.dataset.fullname || cb.dataset.username}; } else { contactsState.selected.delete(cb.dataset.id); delete contactsState.selectedData[cb.dataset.id]; } }
function toggleSelectAllContacts(main) { document.querySelectorAll('#contactsTableBody input[type=checkbox]').forEach(cb=>{cb.checked=main.checked; toggleSelectContact(cb);}); }
function openAddContactsModal(){ if(!contactsState.selected.size) return Swal.fire({icon:'warning',text:'حدد مستخدم',background:'#111827',color:'#fff'}); document.getElementById('addContactsModal').style.display='block'; }
function closeAddContactsModal(){ document.getElementById('addContactsModal').style.display='none'; }
async function submitAddContactsList(){
  const name = document.getElementById('contactsListName').value; if(!name) return;
  fetch('api/contacts_add.php', { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({name:name, platform:'instagram', type:'csv', count:contactsState.selected.size, data:Object.values(contactsState.selectedData)}) })
  .then(()=> { Swal.fire({icon:'success', title:'تم', timer:1500, background:'#111827', color:'#fff'}); closeAddContactsModal(); document.getElementById('contactsSelectAll').click(); });
}
function escapeHtml(s){ return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'}[c])); }
</script>

<?php include 'includes/footer.php'; ?>