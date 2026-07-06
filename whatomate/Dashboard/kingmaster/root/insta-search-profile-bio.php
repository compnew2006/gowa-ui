<?php
session_start();
if (!isset($_SESSION['user_id'])) { header('Location: landing.php'); exit; }
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];
$user = getUserByUserId($user_id);
if (!empty($user['expiry_date']) && strtotime($user['expiry_date']) < time()) { header('Location: index.php'); exit; }

$page_title = "البحث عن حسابات | Kingmaster";
$page_css = ['https://kingmaster.info/css/f-w-i.css'];
include 'includes/head.php'; include 'includes/navbar_top.php'; include 'includes/navbar_actions.php'; include 'includes/navbar_extra_actions.php'; include 'includes/sidebar_right.php'; include 'includes/sidebar_left.php';
?>

<div class="tools-container">
    <div class="tools-header">
        <div class="tools-title"><i class="fas fa-user-search fa-spin" style="--fa-animation-duration: 3s; color: #3b82f6;"></i> البحث عن حسابات (Profile Search)</div>
        <button class="create-campaign-btn" onclick="openModal()"><i class="fas fa-plus-circle"></i> إنشاء حملة جديدة</button>
    </div>
    
    <div class="filters-section">
        <div class="filter-label"><i class="fas fa-filter fa-spin"></i> تصفية:</div>
        <select class="filter-select" id="status-filter" onchange="applyFilters()">
            <option value="all">جميع الحالات</option><option value="pending">قيد الانتظار</option><option value="running">قيد التشغيل</option><option value="paused">متوقف مؤقتاً</option><option value="stopped">متوقف</option><option value="finished">منتهي</option>
        </select>
        <input type="date" class="filter-select" id="date-filter" onchange="applyFilters()" style="padding: 10px 15px; cursor: pointer; color-scheme: dark;">
        <button class="clear-filters-btn" onclick="clearFilters()"><i class="fas fa-times"></i> إعادة تعيين</button>
    </div>
    <div class="campaigns-grid" id="campaigns-grid"><div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);">جاري تحميل الحملات...</div></div>
</div>

<div id="campaignModal" class="modal">
    <div class="modal-content">
        <div class="modal-header"><div class="modal-title"><i class="fas fa-rocket"></i> إعدادات البحث</div><span class="close-modal" onclick="closeModal()">&times;</span></div>
        <form id="campaignForm">
            <div class="form-group"><label class="form-label"><i class="fas fa-tag"></i> اسم الحملة</label><input type="text" class="form-input" id="campaign-name" placeholder="أدخل اسم الحملة" required></div>
            <div class="form-group">
                <label class="form-label"><i class="fas fa-users"></i> الحساب المستخدم للبحث</label>
                <div class="account-selector-wrapper"><select class="form-select" id="account-select" style="flex: 1;"><option value="">اختر حساب</option></select><button type="button" class="add-account-btn" onclick="addAccount()"><i class="fas fa-plus"></i> إضافة</button></div>
                <div class="selected-accounts" id="selected-accounts"></div>
            </div>
            <div class="form-group"><label class="form-label"><i class="fas fa-keyboard"></i> كلمة البحث (Keyword)</label><input type="text" class="form-input" id="keyword" placeholder="مثال: marketing, real estate" required></div>
            <button type="submit" class="submit-btn"><i class="fas fa-save"></i> حفظ الحملة</button>
        </form>
    </div>
</div>

<div id="contactsModal" class="modal">
  <div class="modal-content" style="max-width: 900px; background: #1e293b; border-radius: 16px; border: 1px solid #334155; padding: 24px;">
    <div class="modal-header" style="border-bottom: 1px solid #334155; padding-bottom: 15px;"><div class="modal-title" style="font-size: 1.25rem; font-weight: bold; color: #f8fafc;"><i class="fas fa-list" style="color: #3b82f6;"></i> النتائج المستخرجة</div><span class="close-modal" onclick="closeContactsModal()" style="cursor: pointer; font-size: 1.5rem; color: #94a3b8;">&times;</span></div>
    <div class="filters-section" style="display: flex; gap: 15px; margin-bottom: 20px; border:none; padding:0; background:transparent;">
      <div style="flex: 1; position: relative;"><i class="fas fa-search" style="position: absolute; right: 15px; top: 50%; transform: translateY(-50%); color: #94a3b8;"></i><input type="text" id="contactsSearch" class="form-input" placeholder="بحث بالاسم أو اليوزر..." oninput="debouncedSearchContacts()" style="width: 100%; padding: 10px 40px 10px 15px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff;"></div>
      <select id="contactsPerPage" class="form-select" onchange="reloadContactsPage(1)" style="width: 130px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff;"><option value="25">25</option><option value="50">50</option><option value="100">100</option></select>
      <button class="action-btn-small btn-save" onclick="openAddContactsModal()" style="white-space:nowrap; padding: 10px 15px; border-radius: 8px; background: linear-gradient(135deg, #10b981, #059669);"><i class="fas fa-user-plus"></i> حفظ في قائمة</button>
    </div>
    <div style="overflow-x: auto; background: #0f172a; border-radius: 12px; border: 1px solid #334155;">
      <table style="width: 100%; border-collapse: collapse; text-align: right; color: #e2e8f0; table-layout: fixed;">
        <thead><tr style="background: #1e293b; border-bottom: 2px solid #334155;"><th style="width: 10%; padding: 15px; text-align: center;"><input type="checkbox" id="contactsSelectAll" onchange="toggleSelectAllContacts(this)"></th><th style="width: 25%; padding: 15px; color: #94a3b8;">اليوزر نيم</th><th style="width: 30%; padding: 15px; color: #94a3b8;">الاسم</th><th style="width: 15%; padding: 15px; color: #94a3b8; text-align:center;">الحالة</th><th style="width: 20%; padding: 15px; color: #94a3b8; text-align:center;">المعرف</th></tr></thead>
        <tbody id="contactsTableBody"><tr><td colspan="5" style="padding:30px; text-align:center; color:#94a3b8;">جاري التحميل...</td></tr></tbody>
      </table>
    </div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 20px;"><div id="contactsInfo" style="color: #94a3b8; font-size: 0.95rem;">—</div><div id="contactsPagination" style="display: flex; gap: 6px; flex-wrap: wrap;"></div></div>
  </div>
</div>

<div id="addContactsModal" class="modal">
  <div class="modal-content" style="max-width: 400px; background: #1e293b; border: 1px solid #334155; border-radius: 12px;">
    <div class="modal-header" style="border-bottom: 1px solid #334155;"><div class="modal-title" style="color: #fff;"><i class="fas fa-list"></i> حفظ في قائمة</div><span class="close-modal" onclick="closeAddContactsModal()" style="color: #94a3b8; cursor: pointer;">&times;</span></div>
    <div class="form-group" style="margin-top: 15px;"><label class="form-label" style="color: #cbd5e1;">اسم القائمة</label><input type="text" id="contactsListName" class="form-input" style="background: #0f172a; border: 1px solid #334155; color: #fff;" placeholder="مثال: داتا البحث" /></div>
    <button class="submit-btn" style="width: 100%; background: #3b82f6; margin-top: 15px;" onclick="submitAddContactsList()"><i class="fas fa-save"></i> حفظ</button>
  </div>
</div>

<script>
let selectedAccounts = [], allCampaigns = [], editingCampaignId = null;
const TOOL_NAME = 'Search Bio IG';
const DB_TABLE = 'ig_search_users';

document.addEventListener('DOMContentLoaded', () => { loadAccounts(); loadCampaigns(TOOL_NAME); });
function openModal() { document.getElementById('campaignModal').style.display = 'block'; }
function closeModal() { document.getElementById('campaignModal').style.display = 'none'; document.getElementById('campaignForm').reset(); selectedAccounts = []; document.getElementById('selected-accounts').innerHTML = ''; editingCampaignId = null; }
window.onclick = function(e) { if(e.target == document.getElementById('campaignModal')) closeModal(); if(e.target == document.getElementById('contactsModal')) closeContactsModal(); if(e.target == document.getElementById('addContactsModal')) closeAddContactsModal(); }

function loadAccounts() { fetch('api/get_accounts_ig.php').then(r=>r.json()).then(d => { if(d.success) d.accounts.forEach(a => document.getElementById('account-select').innerHTML += `<option value="${a.account_uid}" data-name="${a.name}">${a.name}</option>`); }); }
function addAccount() { const sel = document.getElementById('account-select'); if(!sel.value) return Swal.fire({icon:'warning', text:'اختر حساب', background:'#111827', color:'#fff'}); if(selectedAccounts.length>=1) return Swal.fire({icon:'info', text:'حساب واحد فقط', background:'#111827', color:'#fff'}); selectedAccounts.push({account_uid: sel.value, name: sel.options[sel.selectedIndex].dataset.name}); document.getElementById('selected-accounts').innerHTML = selectedAccounts.map(a=>`<div class="account-tag"><i class="fa-brands fa-instagram"></i> ${a.name} <span class="remove-account" onclick="selectedAccounts=[]; document.getElementById('selected-accounts').innerHTML='';">✕</span></div>`).join(''); sel.value = ''; }

document.getElementById('campaignForm').addEventListener('submit', function(e) {
    e.preventDefault();
    if (!selectedAccounts.length) return Swal.fire({icon:'warning', text:'أضف حساباً', background:'#111827', color:'#fff'});
    const data = { name: document.getElementById('campaign-name').value, accounts: selectedAccounts, tools: TOOL_NAME, paltform: "Instagram", id_action: document.getElementById('keyword').value.trim() };
    if (editingCampaignId) { data.action = 'update'; data.campaign_id = editingCampaignId; }
    Swal.fire({title:'جاري الحفظ...', didOpen:()=>Swal.showLoading()});
    fetch(editingCampaignId ? 'api/manage_campaign.php' : 'api/create_campaign.php', { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(data) })
    .then(r=>r.json()).then(res => { if(res.success){ Swal.fire({icon:'success', title:'تم!', timer:1500, background:'#111827', color:'#fff'}); closeModal(); loadCampaigns(TOOL_NAME); } else Swal.fire({icon:'error', text:res.message, background:'#111827', color:'#fff'}); });
});

function loadCampaigns(tool) { fetch('api/get_campaigns.php', { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({tool:tool}) }).then(r=>r.json()).then(d => { if(d.success) { allCampaigns = d.campaigns; applyFilters(); } }); }
function applyFilters() { const st = document.getElementById('status-filter').value, dt = document.getElementById('date-filter').value; let f = allCampaigns; if(st !== 'all') f = f.filter(c => c.status === st); if(dt) f = f.filter(c => c.created_at.split(' ')[0] === dt); renderCampaigns(f); }
function clearFilters() { document.getElementById('status-filter').value = 'all'; document.getElementById('date-filter').value = ''; renderCampaigns(allCampaigns); }

function renderCampaigns(campaigns) {
    const grid = document.getElementById('campaigns-grid');
    if (!campaigns.length) return grid.innerHTML = '<div style="grid-column:1/-1; text-align:center; padding:40px; color:#94a3b8;"><i class="fas fa-inbox fa-3x"></i><br>لا توجد حملات</div>';
    grid.innerHTML = campaigns.map(c => `
        <div class="campaign-card">
            <div class="campaign-name"><i class="fas fa-bullhorn"></i> ${c.name}</div>
            <div class="campaign-info">
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fas fa-check-double fa-beat" style="color: #10b981;"></i> العدد:</span>
                    <span class="campaign-info-value">${c.true_count||0}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fa-brands fa-instagram fa-beat" style="color: #1dc717ff;"></i> المنصة:</span>
                    <span class="campaign-info-value">${c.paltform || 'Instagram'}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fas fa-keyboard fa-beat" style="color: #f59e0b;"></i> الكلمة:</span>
                    <span class="campaign-info-value" style="font-weight:bold;">${c.pram1 || c.id_action || ''}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label">${getStatusIcon(c.status)} الحالة:</span>
                    <span class="status-badge status-${c.status}">${getStatusText(c.status)}</span>
                </div>
            </div>
            <button class="campaign-actions-btn" onclick="document.getElementById('acc-${c.id}').classList.toggle('active')"><i class="fas fa-cog"></i> الإجراءات</button>
            <div class="accordion-content" id="acc-${c.id}">
                <div class="accordion-section">
                    <div class="accordion-section-title"><i class="fas fa-sliders-h"></i> الإعدادات</div>
                    <div class="action-btns">${getActionButtons(c)}</div>
                </div>
                <div class="accordion-section">
                    <div class="accordion-section-title"><i class="fas fa-tasks"></i> الإجراء</div>
                    <div class="action-btns"><button class="action-btn-full btn-send" onclick="viewResults('${c.campaign_id||c.id}')"><i class="fa-solid fa-table-cells fa-flip"></i> عرض في جدول</button></div>
                </div>
            </div>
        </div>
    `).join('');
}

function getActionButtons(c) {
    let b = '';
    if (c.status === 'paused' || c.status === 'stopped') {
        b += `<button class="action-btn-small" style="background: linear-gradient(135deg, #10b981, #059669);" onclick="changeStatus(${c.id}, 'pending')"><i class="fas fa-play"></i> تشغيل</button>`;
    } else {
        b += `<button class="action-btn-small btn-stop" onclick="changeStatus(${c.id}, 'stopped')"><i class="fas fa-stop"></i> إيقاف</button><button class="action-btn-small btn-pause" onclick="changeStatus(${c.id}, 'paused')"><i class="fas fa-pause"></i> مؤقت</button>`;
    }
    b += `<button class="action-btn-small btn-edit" onclick="editCampaign(${c.id})"><i class="fas fa-edit"></i> تعديل</button><button class="action-btn-small btn-save" onclick="window.location.href='api/save_data.php?camp_id=${c.campaign_id||c.id}&tool='+encodeURIComponent(TOOL_NAME)"><i class="fas fa-save"></i> حفظ</button>`;
    b += `<button class="action-btn-full btn-delete" style="margin-top:10px;" onclick="deleteCampaign(${c.id})"><i class="fas fa-trash"></i> حذف الحملة</button>`;
    return b;
}

function getStatusText(status) { const m = {'pending':'قيد الانتظار','running':'قيد التشغيل','paused':'متوقف مؤقتاً','stopped':'متوقف','finished':'منتهي'}; return m[status]||status; }
function getStatusIcon(status) { const m = {'pending':'<i class="fas fa-hourglass-half fa-spin" style="color: #f59e0b;"></i>','running':'<i class="fas fa-spinner fa-spin" style="color: #3b82f6;"></i>','paused':'<i class="fas fa-pause-circle fa-beat" style="color: #f59e0b;"></i>','stopped':'<i class="fas fa-stop-circle fa-fade" style="color: #ef4444;"></i>','finished':'<i class="fas fa-check-circle fa-bounce" style="color: #10b981;"></i>'}; return m[status]||'<i class="fas fa-info-circle"></i>'; }
function changeStatus(id, st) { fetch('api/manage_campaign.php', { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({action:'change_status', campaign_id:id, status:st}) }).then(()=>loadCampaigns(TOOL_NAME)); }
function deleteCampaign(id) { Swal.fire({title:'تأكيد', icon:'warning', showCancelButton:true, background:'#111827', color:'#fff', confirmButtonColor:'#ef4444'}).then(r=>{if(r.isConfirmed) fetch('api/manage_campaign.php', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({action:'delete', campaign_id:id})}).then(()=>loadCampaigns(TOOL_NAME));}); }
function editCampaign(id) { const c = allCampaigns.find(x => Number(x.id) === Number(id)); if(!c) return; editingCampaignId = id; document.getElementById('campaign-name').value = c.name; document.getElementById('keyword').value = c.pram1 || c.id_action || ''; selectedAccounts = JSON.parse(c.token||'[]'); document.getElementById('selected-accounts').innerHTML = selectedAccounts.map(a=>`<div class="account-tag"><i class="fa-brands fa-instagram"></i> ${a.name} <span class="remove-account" onclick="selectedAccounts=[]; document.getElementById('selected-accounts').innerHTML=''">✕</span></div>`).join(''); openModal(); }

// Table Logic
let contactsState = { campaign_id: null, page: 1, per_page: 25, q: '', data: [], selected: new Set(), selectedData: {} }; let timer;
function viewResults(cid) { contactsState.campaign_id = cid; contactsState.page=1; contactsState.q=''; contactsState.selected.clear(); contactsState.selectedData={}; document.getElementById('contactsModal').style.display='block'; loadContacts(); }
function closeContactsModal() { document.getElementById('contactsModal').style.display='none'; }
function debouncedSearchContacts() { clearTimeout(timer); timer = setTimeout(()=>{ contactsState.q = document.getElementById('contactsSearch').value; reloadContactsPage(1); }, 300); }
function reloadContactsPage(p) { contactsState.page=p; contactsState.per_page = document.getElementById('contactsPerPage').value; loadContacts(); }

function loadContacts() {
    // استخدمنا search_type للتمييز 
    fetch(`api/ig_basic_info.php?table=${DB_TABLE}&campaign_id=${contactsState.campaign_id}&page=${contactsState.page}&per_page=${contactsState.per_page}&q=${contactsState.q}`).then(r=>r.json()).then(j => {
        contactsState.data = j.data || []; const tb = document.getElementById('contactsTableBody');
        if(!j.data.length) return tb.innerHTML = '<tr><td colspan="5" style="text-align:center; padding:30px; color:#94a3b8;">لا توجد بيانات</td></tr>';
        tb.innerHTML = j.data.map(r => {
            const isVer = r.is_verified === '1' ? '<i class="fas fa-check-circle" style="color:#3b82f6;" title="موثق"></i>' : '';
            const isPriv = r.is_private === '1' ? '<i class="fas fa-lock" style="color:#ef4444;" title="حساب خاص"></i>' : '<i class="fas fa-globe" style="color:#10b981;" title="حساب عام"></i>';
            return `
            <tr style="border-bottom: 1px solid #1e293b;">
                <td style="padding:12px; text-align:center;"><input type="checkbox" data-id="${r.ig_user_id}" data-name="${r.username}" onchange="toggleSelectContact(this)" ${contactsState.selected.has(r.ig_user_id)?'checked':''}></td>
                <td style="padding:12px; font-weight:bold;"><a href="https://instagram.com/${r.username}" target="_blank" style="color:#e2e8f0; text-decoration:none;">${r.username||'--'}</a> ${isVer}</td>
                <td style="padding:12px;">${r.full_name||'--'}</td>
                <td style="padding:12px; text-align:center;">${isPriv}</td>
                <td style="padding:12px; text-align:center; font-family:monospace; color:#cbd5e1;">${r.ig_user_id}</td>
            </tr>
            `;
        }).join('');
        document.getElementById('contactsInfo').textContent = `صفحة ${j.page} من ${j.total_pages} — إجمالي ${j.total}`;
        const pg = document.getElementById('contactsPagination'); let h='';
        if(j.page>1) h+=`<button style="background:#3b82f6; color:#fff; border:none; padding:5px 10px; border-radius:5px;" onclick="reloadContactsPage(${j.page-1})">السابق</button> `;
        if(j.page<j.total_pages) h+=`<button style="background:#3b82f6; color:#fff; border:none; padding:5px 10px; border-radius:5px;" onclick="reloadContactsPage(${j.page+1})">التالي</button>`;
        pg.innerHTML = h;
    });
}
function toggleSelectContact(cb) { if(cb.checked){ contactsState.selected.add(cb.dataset.id); contactsState.selectedData[cb.dataset.id]={identifier:cb.dataset.id, name:cb.dataset.name}; } else { contactsState.selected.delete(cb.dataset.id); delete contactsState.selectedData[cb.dataset.id]; } }
function toggleSelectAllContacts(m) { document.querySelectorAll('#contactsTableBody input[type=checkbox]').forEach(cb=>{cb.checked=m.checked; toggleSelectContact(cb);}); }
function openAddContactsModal() { if(!contactsState.selected.size) return Swal.fire({icon:'warning',text:'حدد مستخدم',background:'#111827',color:'#fff'}); document.getElementById('addContactsModal').style.display='block'; }
function closeAddContactsModal() { document.getElementById('addContactsModal').style.display='none'; }
function submitAddContactsList() {
    const n = document.getElementById('contactsListName').value; if(!n) return;
    fetch('api/contacts_add.php', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ name: n, platform: 'instagram', type: 'csv', count: contactsState.selected.size, data: Object.values(contactsState.selectedData) }) }).then(()=> { Swal.fire({icon:'success', title:'تم', timer:1500, background:'#111827', color:'#fff'}); closeAddContactsModal(); document.getElementById('contactsSelectAll').click(); });
}
</script>
<?php include 'includes/footer.php'; ?>