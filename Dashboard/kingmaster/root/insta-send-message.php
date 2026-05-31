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

$page_title = "إعادة استهداف إنستجرام | Kingmaster";
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
            <i class="fa-brands fa-instagram fa-spin" style="--fa-animation-duration: 3s; color: #E1306C;"></i>
            أدوات إنستجرام - إرسال رسائل
        </div>
        <button class="create-campaign-btn" onclick="openModal()">
            <i class="fas fa-plus-circle"></i> إنشاء حملة جديدة
        </button>
    </div>
    
    <div class="filters-section">
        <div class="filter-label">
            <i class="fas fa-filter fa-spin" style="--fa-animation-duration: 3s;"></i> تصفية:
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
               style="padding: 10px 15px; cursor: pointer; color-scheme: dark;" placeholder="التاريخ">
        
        <button class="clear-filters-btn" onclick="clearFilters()">
            <i class="fas fa-times"></i> إعادة تعيين
        </button>
    </div>
    
    <div class="campaigns-grid" id="campaigns-grid">
        <div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);">
            جاري تحميل الحملات...
        </div>
    </div>
</div>

<div id="campaignModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">
                <i class="fas fa-paper-plane"></i> إنشاء حملة إرسال إنستجرام
            </div>
            <span class="close-modal" onclick="closeModal()">&times;</span>
        </div>
        
        <form id="campaignForm">
            <div class="form-group">
                <label class="form-label"><i class="fas fa-tag"></i> اسم الحملة</label>
                <input type="text" class="form-input" id="campaign-name" placeholder="أدخل اسم الحملة" required>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-users"></i>الحسابات</label>
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
                <label class="form-label"><i class="fas fa-file-alt"></i> المحتوى (الرسالة)</label>
                <select class="form-select" id="sender-content-select">
                    <option value="">— اختر المحتوى —</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-clock"></i> إعدادات الفواصل الزمنية (إنستجرام)</label>
                <select class="form-select" id="sender-interval-select">
                    <option value="">— اختر الإعداد —</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label"><i class="fas fa-message"></i> نوع الرسالة</label>
                <div class="option-cards" id="msgTypeCards">
                  <div class="option-card selected" data-value="text"><i class="fas fa-align-left"></i><div><div class="option-title">نص</div><div class="option-desc">رسالة نصية فقط</div></div></div>
                 <!-- <div class="option-card" data-value="text_image"><i class="fas fa-image"></i><div><div class="option-title">نص وصورة</div><div class="option-desc">إرسال صورة عبر DM</div></div></div>
                </div>-->
                <input type="hidden" id="msg-type" value="text">
            </div>

            <div class="form-group" id="sender-file-wrap" style="display:none;">
                <label class="form-label"><i class="fas fa-paperclip"></i> اختر الصورة/الملف</label>
                <select class="form-select" id="sender-file-select">
                    <option value="">— اختر ملف —</option>
                </select>
            </div>

            <button type="submit" class="submit-btn">
                <i class="fas fa-save"></i> حفظ الحملة
            </button>
        </form>
    </div>
        </form>
    </div>
</div>

<div id="contactsModal" class="modal">
  <div class="modal-content" style="max-width: 800px; background: #1e293b; border-radius: 16px; border: 1px solid #334155; padding: 24px; box-shadow: 0 10px 25px rgba(0,0,0,0.5);">
    
    <div class="modal-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; border-bottom: 1px solid #334155; padding-bottom: 15px;">
      <div class="modal-title" style="font-size: 1.25rem; font-weight: bold; color: #f8fafc;">
        <i class="fas fa-address-book" style="color: #3b82f6; margin-left: 8px;"></i> جهات الاتصال
      </div>
      <span class="close-modal" onclick="closeContactsModal()" style="cursor: pointer; font-size: 1.5rem; color: #94a3b8; transition: 0.3s;" onmouseover="this.style.color='#ef4444'" onmouseout="this.style.color='#94a3b8'">&times;</span>
    </div>

    <div class="filters-section" style="display: flex; gap: 15px; margin-bottom: 20px; align-items: center;">
      <div style="flex: 1; position: relative;">
         <i class="fas fa-search" style="position: absolute; right: 15px; top: 50%; transform: translateY(-50%); color: #94a3b8;"></i>
         <input type="text" id="contactsSearch" class="form-input" placeholder="بحث بالمعرف..." oninput="debouncedSearchContacts()" style="width: 100%; padding: 10px 40px 10px 15px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff; outline: none;">
      </div>
      <select id="contactsPerPage" class="form-select" onchange="reloadContactsPage(1)" style="width: 130px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff; padding: 10px; cursor: pointer; outline: none;">
        <option value="25">25 نتيجة</option>
        <option value="50">50 نتيجة</option>
        <option value="100">100 نتيجة</option>
      </select>
    </div>

    <div style="overflow-x: auto; background: #0f172a; border-radius: 12px; border: 1px solid #334155;">
     <table style="width:100%; border-collapse:collapse; table-layout: fixed;">
        <thead>
          <tr style="background:#0f172a;">
            <th style="width: 50%; padding:10px; border-bottom:1px solid var(--border-color); text-align:right;">المعرف</th>
            <th style="width: 50%; padding:10px; border-bottom:1px solid var(--border-color); text-align:center;">حالة الإرسال</th>
          </tr>
        </thead>
        <tbody id="contactsTableBody">
          <tr><td colspan="2" style="padding:20px; text-align:center; color:var(--text-secondary);">جاري التحميل...</td></tr>
        </tbody>
      </table>
    </div>

    <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 20px;">
      <div id="contactsInfo" style="color: #94a3b8; font-size: 0.95rem; font-weight: 500;">—</div>
      <div id="contactsPagination" style="display: flex; gap: 6px; flex-wrap: wrap;"></div>
    </div>

  </div>
</div>

<script>
let selectedAccounts = [];
let allAccounts = [];
let editingCampaignId = null;

document.addEventListener('DOMContentLoaded', function() {
    loadAccounts();
    loadIntervalsIG();
    loadCampaigns('Send Retarget IG'); // 👈 اسم الأداة الجديد
    loadContentSender();
    loadContactLists();
    setupMsgTypeToggle();
    loadFilesList();
});

function openModal() {
    document.getElementById('campaignModal').style.display = 'block';
    try { document.body.style.overflow = 'hidden'; } catch(e){}
    
    const msgHidden = document.getElementById('msg-type');
    const msgCards = document.getElementById('msgTypeCards');
    const fileWrap = document.getElementById('sender-file-wrap');
    if (msgHidden) msgHidden.value = 'text';
    if (msgCards) {
        msgCards.querySelectorAll('.option-card').forEach(el=>el.classList.remove('selected'));
        const textCard = msgCards.querySelector('.option-card[data-value="text"]');
        if (textCard) textCard.classList.add('selected');
    }
    if (fileWrap) fileWrap.style.display = 'none';
}

function closeModal() {
    document.getElementById('campaignModal').style.display = 'none';
    try { document.body.style.overflow = ''; } catch(e){}
    document.getElementById('campaignForm').reset();
    selectedAccounts = [];
    document.getElementById('selected-accounts').innerHTML = '';
    editingCampaignId = null;

    document.querySelector('.modal-title').innerHTML = '<i class="fas fa-paper-plane"></i> إنشاء حملة إرسال إنستجرام';
    document.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ الحملة';
}

window.onclick = function(event) {
    const modal = document.getElementById('campaignModal');
    const contactsModal = document.getElementById('contactsModal');
    if (event.target == modal) closeModal();
    if (event.target == contactsModal) closeContactsModal();
}

function loadAccounts() {
    fetch('api/get_accounts_ig.php')
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
  // 👈 مسار جلب قوائم معرفات الانستجرام
  fetch('api/contacts_lists_ig.php', { credentials: 'same-origin' }) 
    .then(r=>r.json())
    .then(j=>{
      const sel = document.getElementById('contact-list-select');
      sel.innerHTML = '<option value="">— اختر —</option>';
      if(j.success && Array.isArray(j.lists)){
        j.lists.forEach(row=>{
          const o = document.createElement('option');
          o.value = row.id;
          o.textContent = row.name;
          sel.appendChild(o);
        });
        hint.textContent = j.lists.length ? `تم تحميل ${j.lists.length} قائمة` : 'لا توجد قوائم محفوظة';
      } else {
        hint.textContent = 'تعذر تحميل القوائم';
      }
    }).catch(()=>{ hint.textContent = 'تعذر تحميل القوائم'; });
}

function loadContentSender() {
    fetch('api/get_content.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const sel = document.getElementById('sender-content-select');
            if (sel) {
                sel.innerHTML = '<option value="">— اختر المحتوى —</option>';
                data.content.forEach(c => {
                    const o = document.createElement('option');
                    o.value = c.id; o.textContent = c.name; sel.appendChild(o);
                });
            }
        }
    });
}

function loadIntervalsIG() {
    // 👈 مسار إعدادات الوقت الخاصة بالانستجرام
    fetch('api/get_intervals_ig.php') 
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const sel = document.getElementById('sender-interval-select');
            if (sel) {
                sel.innerHTML = '<option value="">— اختر الإعداد —</option>';
                data.intervals.forEach(it => {
                    const o = document.createElement('option'); 
                    o.value = it.id; o.textContent = it.settings_name; 
                    sel.appendChild(o);
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
  const cards = document.getElementById('msgTypeCards');
  const hidden = document.getElementById('msg-type');
  
  const applyVis = (v)=>{
    if (v==='text_image' || v==='text_video' || v==='text_file') {
      if (wrap) wrap.style.display = '';
    } else {
      if (wrap) wrap.style.display = 'none';
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
  applyVis(hidden ? hidden.value : 'text');
}

function addAccount() {
    const select = document.getElementById('account-select');
    const selectedOption = select.options[select.selectedIndex];
    
    if (!select.value) {
        Swal.fire({icon: 'warning', title: 'تنبيه', text: 'يرجى اختيار حساب', background: '#111827', color: '#e5e7eb'});
        return;
    }
    if (selectedAccounts.some(acc => acc.account_uid === select.value)) {
        Swal.fire({icon: 'info', title: 'تنبيه', text: 'هذا الحساب مضاف بالفعل', background: '#111827', color: '#e5e7eb'});
        return;
    }
    selectedAccounts.push({ account_uid: select.value, name: selectedOption.dataset.name });
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
            <i class="fa-brands fa-instagram"></i> ${acc.name}
            <span class="remove-account" onclick="removeAccount('${acc.account_uid}')">✕</span>
        </div>
    `).join('');
}

// ================= Form Submit (Create/Edit) =================
document.getElementById('campaignForm').addEventListener('submit', function(e) {
    e.preventDefault();
    if (selectedAccounts.length === 0) {
        Swal.fire({icon: 'warning', title: 'تنبيه', text: 'يرجى إضافة حساب واحد على الأقل', background: '#111827', color: '#e5e7eb'});
        return;
    }

    const contact = document.getElementById('contact-list-select').value || null;
    const content_id = document.getElementById('sender-content-select').value || null;
    const interval_id = document.getElementById('sender-interval-select').value || null;
    const msg_type = (document.getElementById('msg-type')||{}).value || 'text';
    const file_id = document.getElementById('sender-file-select').value || null;
    
    const config = { content_id, interval_id, msg_type, file_id };

    if (editingCampaignId) {
        // Update
        const data = {
            action: 'update',
            campaign_id: editingCampaignId,
            name: document.getElementById('campaign-name').value,
            accounts: selectedAccounts,
            interval_id: interval_id ? parseInt(interval_id,10) : null,
            id_action: JSON.stringify(config),
            contact: contact,
        };
        saveRequest('api/manage_campaign.php', data);
    } else {
        // Create
        const data = {
            name: document.getElementById('campaign-name').value,
            accounts: selectedAccounts,
            tools: "Send Retarget IG", // 👈 الأداة
            paltform: "Instagram",     // 👈 المنصة
            contact: contact,
            interval_id: interval_id ? parseInt(interval_id,10) : null,
            id_action: JSON.stringify(config)
        };
        saveRequest('api/create_campaign.php', data);
    }
});

function saveRequest(url, data) {
    Swal.fire({ title: 'جاري الحفظ...', allowOutsideClick: false, didOpen: () => Swal.showLoading() });
    fetch(url, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(data)
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            Swal.fire({icon: 'success', title: 'تم!', text: data.message, timer: 2000, showConfirmButton: false, background: '#111827', color: '#e5e7eb'});
            closeModal();
            loadCampaigns('Send Retarget IG');
        } else {
            Swal.fire({icon: 'error', title: 'خطأ', text: data.message, background: '#111827', color: '#e5e7eb'});
        }
    });
}

// ================= Load Campaigns =================
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
        }
    }).catch(err => console.error('خطأ في الاتصال:', err));
    
    
}

function applyFilters() {
    const statusFilter = document.getElementById('status-filter').value;
    const dateFilter = document.getElementById('date-filter').value;
    let filtered = allCampaigns;
    
    if (statusFilter !== 'all') filtered = filtered.filter(c => c.status === statusFilter);
    if (dateFilter) filtered = filtered.filter(c => new Date(c.created_at).toISOString().split('T')[0] === dateFilter);
    
    renderCampaigns(filtered);
}

function clearFilters() {
    document.getElementById('status-filter').value = 'all';
    document.getElementById('date-filter').value = '';
    renderCampaigns(allCampaigns);
}

async function getTotalCount(contactId) {
    // 👈 تأكد من دالة جلب إجمالي الأرقام
    const res = await fetch("api/get_count_contacts.php?id=" + contactId); 
    const text = await res.text(); 
    return text.trim(); 
}

async function renderCampaigns(campaigns) {
    const grid = document.getElementById('campaigns-grid');
    if (campaigns.length === 0) {
        grid.innerHTML = `<div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);"><i class="fas fa-inbox fa-3x" style="margin-bottom: 20px;"></i><br>لا توجد حملات حتى الآن</div>`;
        return;
    }

    const campaignsWithTotal = [];
    for (let campaign of campaigns) {
        const total = await getTotalCount(campaign.contact);
        campaignsWithTotal.push({ ...campaign, total });
    }
    grid.innerHTML = campaignsWithTotal.map(c => `
        <div class="campaign-card">
            <div class="campaign-name"><i class="fas fa-bullhorn"></i> ${c.name}</div>
            <div class="campaign-info">
                <div class="campaign-info-item"><span class="campaign-info-label"><i class="fas fa-check-double fa-beat" style="color: #10b981;"></i> العدد:</span> <span class="campaign-info-value">${c.true_count} - ${c.total}</span></div>
                <div class="campaign-info-item"><span class="campaign-info-label"><i class="fa-solid fa-check fa-beat" style="color: #63E6BE;"></i> الناجح:</span> <span class="campaign-info-value">${c.true_count}</span></div>
                <div class="campaign-info-item"><span class="campaign-info-label"><i class="fa-solid fa-xmark fa-beat" style="color: #e70d0d;"></i> الفاشل:</span> <span class="campaign-info-value">${c.false_count}</span></div>
                <div class="campaign-info-item"><span class="campaign-info-label"><i class="fa-brands fa-instagram fa-beat" style="color: #E1306C;"></i> المنصة:</span> <span class="campaign-info-value">${c.paltform}</span></div>
                <div class="campaign-info-item"><span class="campaign-info-label">${getStatusIcon(c.status)} الحالة:</span> <span class="status-badge status-${c.status}">${getStatusText(c.status)}</span></div>
            </div>
            <button class="campaign-actions-btn" onclick="toggleAccordion(${c.id})"><i class="fas fa-cog"></i> الإجراءات</button>
            <div class="accordion-content" id="accordion-${c.id}">
                <div class="accordion-section">
                    <div class="accordion-section-title"><i class="fas fa-sliders-h"></i> الإعدادات</div>
                    <div class="action-btns" id="actions-${c.id}">${getActionButtons(c)}</div>
                </div>
                <div class="accordion-section">
                    <div class="accordion-section-title"><i class="fas fa-tasks"></i> الإجراء</div>
                    <div class="action-btns">
                        <button class="action-btn-full btn-send" onclick="sendMessage(${c.campaign_id})"><i class="fa-solid fa-table-cells fa-flip"></i> عرض فى جدول</button>
                        
                        <button class="action-btn-full btn-save" onclick="downloadReport(${c.campaign_id})" style="display: none; margin-top: 10px; background: linear-gradient(135deg, #4ade80, #16a34a);"><i class="fas fa-download"></i> تحميل التقرير (Excel)</button>
                    </div>
                </div>
            </div>
        </div>
    `).join('');
}

// Helpers
function getStatusText(status) { const map = {'pending': 'قيد الانتظار','running': 'قيد التشغيل','paused': 'متوقف مؤقتاً','stopped': 'متوقف','finished': 'منتهي'}; return map[status] || status; }
function getStatusIcon(status) { const map = {'pending': '<i class="fas fa-hourglass-half fa-spin" style="color: #f59e0b;"></i>','running': '<i class="fas fa-spinner fa-spin" style="color: #3b82f6;"></i>','paused': '<i class="fas fa-pause-circle fa-beat" style="color: #f59e0b;"></i>','stopped': '<i class="fas fa-stop-circle fa-fade" style="color: #ef4444;"></i>','finished': '<i class="fas fa-check-circle fa-bounce" style="color: #10b981;"></i>'}; return map[status] || '<i class="fas fa-info-circle"></i>'; }
function toggleAccordion(id) { document.getElementById(`accordion-${id}`).classList.toggle('active'); }

function getActionButtons(campaign) {
    const status = campaign.status;
    let buttons = '';
    if (status === 'paused' || status === 'stopped') {
        buttons += `<button class="action-btn-small" style="background: linear-gradient(135deg, #10b981, #059669);" onclick="changeStatus(${campaign.id}, 'pending')"><i class="fas fa-play"></i> تشغيل</button>`;
    } else {
        buttons += `<button class="action-btn-small btn-stop" onclick="changeStatus(${campaign.id}, 'stopped')"><i class="fas fa-stop"></i> إيقاف</button>
                    <button class="action-btn-small btn-pause" onclick="changeStatus(${campaign.id}, 'paused')"><i class="fas fa-pause"></i> إيقاف مؤقت</button>`;
    }
    buttons += `<button class="action-btn-small btn-edit" onclick="editCampaign(${campaign.id})"><i class="fas fa-edit"></i> تعديل</button>
                <button class="action-btn-full btn-delete" style="margin-top:10px;" onclick="deleteCampaign(${campaign.id})"><i class="fas fa-trash"></i> حذف الحملة</button>`;
    return buttons;
}

function changeStatus(id, newStatus) {
    Swal.fire({ title: 'هل أنت متأكد؟', icon: 'warning', showCancelButton: true, background: '#111827', color: '#e5e7eb'}).then((result) => {
        if (result.isConfirmed) {
            fetch('api/manage_campaign.php', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ action: 'change_status', campaign_id: id, status: newStatus }) })
            .then(res => res.json())
            .then(data => { if(data.success) loadCampaigns('Send Retarget IG'); });
        }
    });
}

function deleteCampaign(id) {
    Swal.fire({ title: 'تأكيد الحذف', icon: 'warning', showCancelButton: true, confirmButtonColor: '#991b1b', background: '#111827', color: '#e5e7eb'}).then((result) => {
        if (result.isConfirmed) {
            fetch('api/manage_campaign.php', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({ action: 'delete', campaign_id: id }) })
            .then(res => res.json())
            .then(data => { if(data.success) loadCampaigns('Send Retarget IG'); });
        }
    });
}

function editCampaign(id) {
    // جلب الحملة سواء كان المعرف id أو campaign_id
    const campaign = allCampaigns.find(c => Number(c.id) === Number(id) || Number(c.campaign_id) === Number(id));
    if (!campaign) return;
    
    editingCampaignId = id;
    document.getElementById('campaign-name').value = campaign.name;
    selectedAccounts = JSON.parse(campaign.token || '[]');
    renderSelectedAccounts();
    
    // 💡 التعديل هنا: نحدد مودال التعديل فقط عشان مانأثرش على مودال النتائج!
    // (استبدل 'campaignModal' بالـ id الحقيقي لمودال إضافة/تعديل الحملة عندك)
    const editModal = document.getElementById('campaignModal'); 
    if (editModal) {
        editModal.querySelector('.modal-title').innerHTML = '<i class="fas fa-edit"></i> تعديل الحملة';
        editModal.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ التعديلات';
        editModal.style.display = 'block'; // فتح المودال بالـ ID مباشرة بدلاً من دالة openModal العشوائية
    }
    
    // لو عندك Select inputs محتاج تعمل لها Prefill كملها هنا...
}

function downloadReport(id) {
    window.location.href = "api/save_data.php?camp_id=" + id + "&tool=Extract-Messages-IG-send"; // 👈 يجب إضافتها في ملف save_data.php
}

// ================= Contacts Modal & Table =================
let contactsState = { campaign_id: null, page: 1, per_page: 25, q: '' };

function sendMessage(campaignId) {
    console.log(campaignId);
    contactsState.campaign_id = campaignId;
    contactsState.page = 1;
    contactsState.q = '';
    
    const searchInput = document.getElementById('contactsSearch');
    if (searchInput) searchInput.value = '';
    
    const perPageSelect = document.getElementById('contactsPerPage');
    if (perPageSelect) perPageSelect.value = '25';
    
    const modal = document.getElementById('contactsModal');
    if (modal) {
        modal.style.display = 'block';
    } else {
        console.error("عنصر المودال contactsModal غير موجود في الصفحة!");
    }
    
    loadContacts();
}

function closeContactsModal() { document.getElementById('contactsModal').style.display = 'none'; }

let searchTimeout; // متغير لحفظ التايمر
function debouncedSearchContacts() { 
    clearTimeout(searchTimeout); // بنلغي أي طلب قديم لو العميل لسه بيكتب
    searchTimeout = setTimeout(() => { 
        contactsState.q = document.getElementById('contactsSearch').value.trim(); 
        reloadContactsPage(1); 
    }, 300); 
}
function reloadContactsPage(page) { contactsState.page = page; contactsState.per_page = parseInt(document.getElementById('contactsPerPage').value, 10) || 25; loadContacts(); }

function loadContacts() {
    const params = new URLSearchParams({ 
        table: 'ig_retarget', // 👈 هنا نحدد الجدول الجديد
        campaign_id: contactsState.campaign_id, 
        page: contactsState.page, 
        per_page: contactsState.per_page, 
        q: contactsState.q
    });
    console.log(contactsState.campaign_id);
    // 👈 نستدعي الملف المشترك
    fetch('api/ig_basic_info.php?' + params.toString()) 
    .then(r=>r.json())
    .then(j=>{
        if (!j.success) throw new Error(j.message || 'failed');
        renderContactsTable(j.data || []);
        
        console.log(j);
        // تحديث معلومات الصفحة
        document.getElementById('contactsInfo').textContent = `الصفحة ${contactsState.page} من ${j.total_pages} — إجمالي ${j.total || 0} معرف`;
        
        // رسم أزرار الصفحات (سأعطيك دالتها في الأسفل لتكتمل عندك)
        contactsState.total_pages = j.total_pages; 
        renderContactsPagination();
    }).catch(e=>{
        document.getElementById('contactsTableBody').innerHTML = `<tr><td colspan="2" style="text-align:center; color:#ef4444;">خطأ: ${e.message}</td></tr>`;
    });
}


function renderContactsTable(data) {
    const tb = document.getElementById('contactsTableBody');
    if (!data.length) { 
        tb.innerHTML = '<tr><td colspan="2" style="text-align:center; padding:20px;">لا توجد بيانات</td></tr>'; 
        return; 
    }
    
    tb.innerHTML = data.map(row => {
        let statusBadge = row.status === 'True' ? '<span class="status-badge" style="background:#10b981;">ناجح</span>' : 
                         (row.status === 'False' ? '<span class="status-badge" style="background:#ef4444;">فاشل</span>' : 
                         '<span class="status-badge" style="background:#f59e0b;">قيد الانتظار</span>');
        return `
        <tr>
            <td style="padding:12px; text-align:right; border-bottom:1px solid var(--border-color);">
                <span style="direction:ltr; display:inline-block; font-family:monospace;">${escapeHtml(row.ig_user_id || row.identifier || '')}</span>
            </td>
            <td style="padding:12px; text-align:center; border-bottom:1px solid var(--border-color);">
                ${statusBadge}
            </td>
        </tr>`;
    }).join('');
}

// دالة أزرار التصفح
function renderContactsPagination() {
    const el = document.getElementById('contactsPagination');
    const total = contactsState.total_pages;
    const cur = contactsState.page;
    if (!total || total <= 1) { el.innerHTML = ''; return; }
    let html = '';
    const btn = (p, t, dis=false) => `<button class="action-btn-small" style="background: linear-gradient(135deg, #3b82f6, #2563eb); ${dis?'opacity:.5; cursor:not-allowed;':''}" ${dis?'disabled':''} onclick="reloadContactsPage(${p})">${t}</button>`;
    
    html += btn(1, 'الأولى', cur===1);
    html += btn(Math.max(1, cur-1), 'السابق', cur===1);
    
    const start = Math.max(1, cur-2);
    const end = Math.min(total, cur+2);
    for (let p=start; p<=end; p++) {
        html += `<button class="action-btn-small" style="${p===cur?'background: linear-gradient(135deg, #10b981, #059669);':'background: linear-gradient(135deg, #6b7280, #4b5563);'}" onclick="reloadContactsPage(${p})">${p}</button>`;
    }
    
    html += btn(Math.min(total, cur+1), 'التالي', cur===total);
    html += btn(total, 'الأخيرة', cur===total);
    el.innerHTML = html;
}

function escapeHtml(s){ return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'}[c])); }
</script>

<?php include 'includes/footer.php'; ?>