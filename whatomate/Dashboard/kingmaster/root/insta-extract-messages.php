<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];

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

$page_title = "استخراج رسائل إنستجرام | Kingmaster";
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
            <i class="fas fa-envelope-open-text fa-spin" style="--fa-animation-duration: 3s; color: #667eea;"></i>
            استخراج المستخدمين من الرسايل (DMs)
        </div>
        <button class="create-campaign-btn" onclick="openModal()">
            <i class="fas fa-plus-circle"></i> إنشاء حملة جديدة
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
        
        <button class="clear-filters-btn" onclick="clearFilters()">
            <i class="fas fa-times"></i> إعادة تعيين
        </button>
    </div>
    
    <div class="campaigns-grid" id="campaigns-grid">
        <div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);">
            <i class="fas fa-spinner fa-spin fa-2x"></i><br><br>جاري تحميل الحملات...
        </div>
    </div>
</div>

<div id="campaignModal" class="modal">
    <div class="modal-content" style="max-width: 500px;">
        <div class="modal-header">
            <div class="modal-title"><i class="fas fa-rocket"></i> إنشاء حملة استخراج رسائل</div>
            <span class="close-modal" onclick="closeModal()">&times;</span>
        </div>
        
        <form id="campaignForm">
            <div class="form-group">
                <label class="form-label"><i class="fas fa-tag"></i> اسم الحملة</label>
                <input type="text" class="form-input" id="campaign-name" placeholder="أدخل اسم الحملة" required>
            </div>
            
            <div class="form-group">
                <label class="form-label"><i class="fas fa-users"></i> الحساب المراد الاستخراج منه</label>
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
            
            <button type="submit" class="submit-btn" style="margin-top: 20px;">
                <i class="fas fa-save"></i> حفظ الحملة
            </button>
        </form>
    </div>
</div>

<div id="contactsModal" class="modal">
  <div class="modal-content" style="max-width: 900px; background: #1e293b; border-radius: 16px; border: 1px solid #334155; padding: 24px; box-shadow: 0 10px 25px rgba(0,0,0,0.5);">
    <div class="modal-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; border-bottom: 1px solid #334155; padding-bottom: 15px;">
      <div class="modal-title" style="font-size: 1.25rem; font-weight: bold; color: #f8fafc;">
        <i class="fas fa-address-book" style="color: #3b82f6; margin-left: 8px;"></i> المستخدمين المستخرجين
      </div>
      <span class="close-modal" onclick="closeContactsModal()" style="cursor: pointer; font-size: 1.5rem; color: #94a3b8;">&times;</span>
    </div>

    <div class="filters-section" style="display: flex; gap: 15px; margin-bottom: 20px; align-items: center; border:none; padding:0; background:transparent;">
      <div style="flex: 1; position: relative;">
         <i class="fas fa-search" style="position: absolute; right: 15px; top: 50%; transform: translateY(-50%); color: #94a3b8;"></i>
         <input type="text" id="contactsSearch" class="form-input" placeholder="بحث بالمعرف أو الاسم..." oninput="debouncedSearchContacts()" style="width: 100%; padding: 10px 40px 10px 15px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff; outline: none;">
      </div>
      <select id="contactsPerPage" class="form-select" onchange="reloadContactsPage(1)" style="width: 130px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; color: #fff; padding: 10px; cursor: pointer; outline: none;">
        <option value="25">25 نتيجة</option>
        <option value="50">50 نتيجة</option>
        <option value="100">100 نتيجة</option>
      </select>
      <button class="action-btn-small btn-save" onclick="openAddContactsModal()" style="white-space:nowrap; padding: 10px 15px; border-radius: 8px; background: linear-gradient(135deg, #10b981, #059669);">
        <i class="fas fa-user-plus"></i> حفظ في قائمة
      </button>
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
        <tbody id="contactsTableBody">
          <tr><td colspan="4" style="padding:30px; text-align:center; color:#94a3b8;">جاري التحميل...</td></tr>
        </tbody>
      </table>
    </div>

    <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 20px;">
      <div id="contactsInfo" style="color: #94a3b8; font-size: 0.95rem; font-weight: 500;">—</div>
      <div id="contactsPagination" style="display: flex; gap: 6px; flex-wrap: wrap;"></div>
    </div>
  </div>
</div>

<div id="addContactsModal" class="modal">
  <div class="modal-content" style="max-width: 400px; background: #1e293b; border: 1px solid #334155; border-radius: 12px;">
    <div class="modal-header" style="border-bottom: 1px solid #334155;">
      <div class="modal-title" style="color: #fff;"><i class="fas fa-list"></i> حفظ في جهات الاتصال</div>
      <span class="close-modal" onclick="closeAddContactsModal()" style="color: #94a3b8; cursor: pointer;">&times;</span>
    </div>
    <div class="form-group" style="margin-top: 15px;">
      <label class="form-label" style="color: #cbd5e1;"><i class="fas fa-tag"></i> اسم القائمة الجديدة</label>
      <input type="text" id="contactsListName" class="form-input" style="background: #0f172a; border: 1px solid #334155; color: #fff;" placeholder="مثال: عملاء رسائل إنستجرام" />
    </div>
    <div class="hint" id="addContactsCountHint" style="color: #10b981; margin-bottom: 15px;">—</div>
    <button class="submit-btn" style="width: 100%; background: #3b82f6;" onclick="submitAddContactsList()"><i class="fas fa-save"></i> حفظ القائمة</button>
  </div>
</div>

<script>
let selectedAccounts = [];
let allAccounts = [];
let allCampaigns = [];
let editingCampaignId = null;

// المتغير الخاص بالأداة (مهم جداً للباك إند)
const TOOL_NAME = 'Extract DMs IG';

document.addEventListener('DOMContentLoaded', function() {
    loadAccounts();
    loadCampaigns(TOOL_NAME);
});

// إغلاق المودالز
window.onclick = function(event) {
    if (event.target == document.getElementById('campaignModal')) closeModal();
    if (event.target == document.getElementById('contactsModal')) closeContactsModal();
    if (event.target == document.getElementById('addContactsModal')) closeAddContactsModal();
}

function openModal() {
    document.getElementById('campaignModal').style.display = 'block';
    try { document.body.style.overflow = 'hidden'; } catch(e){}
}

function closeModal() {
    document.getElementById('campaignModal').style.display = 'none';
    try { document.body.style.overflow = ''; } catch(e){}
    document.getElementById('campaignForm').reset();
    selectedAccounts = [];
    document.getElementById('selected-accounts').innerHTML = '';
    editingCampaignId = null;
    document.querySelector('.modal-title').innerHTML = '<i class="fas fa-rocket"></i> إنشاء حملة استخراج رسائل';
    document.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ الحملة';
}

function loadAccounts() {
    fetch('api/get_accounts_ig.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            allAccounts = data.accounts;
            const select = document.getElementById('account-select');
            data.accounts.forEach(account => {
                const option = document.createElement('option');
                option.value = account.account_uid;
                option.textContent = account.name;
                option.dataset.name = account.name;
                select.appendChild(option);
            });
        }
    });
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
    
    // يفضل في الاستخراج تحديد حساب واحد فقط للحملة الواحدة
    if (selectedAccounts.length >= 1) {
        Swal.fire({ icon: 'info', title: 'تنبيه', text: 'يمكنك اختيار حساب واحد فقط في كل حملة استخراج', background: '#111827', color: '#e5e7eb' });
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

// ================= Form Submit =================
document.getElementById('campaignForm').addEventListener('submit', function(e) {
    e.preventDefault();
    if (selectedAccounts.length === 0) {
        Swal.fire({icon: 'warning', title: 'تنبيه', text: 'يرجى إضافة حساب واحد على الأقل', background: '#111827', color: '#e5e7eb'});
        return;
    }

    // لا يوجد Contact أو Content أو Interval
    const data = {
        name: document.getElementById('campaign-name').value,
        accounts: selectedAccounts,
        tools: TOOL_NAME,
        paltform: "Instagram"
    };

    if (editingCampaignId) {
        data.action = 'update';
        data.campaign_id = editingCampaignId;
        
        Swal.fire({ title: 'جاري الحفظ...', allowOutsideClick: false, didOpen: () => Swal.showLoading() });
        fetch('api/manage_campaign.php', {
            method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)
        }).then(res => res.json()).then(resData => {
            if (resData.success) {
                Swal.fire({icon: 'success', title: 'تم!', text: resData.message, timer: 2000, showConfirmButton: false, background: '#111827', color: '#e5e7eb'});
                closeModal();
                loadCampaigns(TOOL_NAME);
            } else {
                Swal.fire({icon: 'error', title: 'خطأ', text: resData.message, background: '#111827', color: '#e5e7eb'});
            }
        });
    } else {
        Swal.fire({ title: 'جاري الحفظ...', allowOutsideClick: false, didOpen: () => Swal.showLoading() });
        fetch('api/create_campaign.php', {
            method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data)
        }).then(res => res.json()).then(resData => {
            if (resData.success) {
                Swal.fire({icon: 'success', title: 'تم!', text: resData.message, timer: 2000, showConfirmButton: false, background: '#111827', color: '#e5e7eb'});
                closeModal();
                loadCampaigns(TOOL_NAME);
            } else {
                Swal.fire({icon: 'error', title: 'خطأ', text: resData.message, background: '#111827', color: '#e5e7eb'});
            }
        });
    }
});

// ================= Load Campaigns =================
function loadCampaigns(tool) {
    fetch('api/get_campaigns.php', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ tool: tool })
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            allCampaigns = data.campaigns;
            applyFilters();
        } else {
            console.error('حدث خطأ:', data.message);
        }
    }).catch(err => console.error('خطأ في الاتصال:', err));
}

function applyFilters() {
    const statusFilter = document.getElementById('status-filter').value;
    const dateFilter = document.getElementById('date-filter').value;
    let filtered = allCampaigns;
    
    if (statusFilter !== 'all') filtered = filtered.filter(c => c.status === statusFilter);
    if (dateFilter) filtered = filtered.filter(c => {
        const campaignDate = new Date(c.created_at).toISOString().split('T')[0];
        return campaignDate === dateFilter;
    });
    
    renderCampaigns(filtered);
}

function clearFilters() {
    document.getElementById('status-filter').value = 'all';
    document.getElementById('date-filter').value = '';
    renderCampaigns(allCampaigns);
}

function renderCampaigns(campaigns) {
    const grid = document.getElementById('campaigns-grid');
    if (campaigns.length === 0) {
        grid.innerHTML = '<div style="grid-column: 1/-1; text-align: center; padding: 40px; color: var(--text-secondary);"><i class="fas fa-inbox fa-3x" style="margin-bottom: 20px;"></i><br>لا توجد حملات حتى الآن</div>';
        return;
    }
    
    grid.innerHTML = campaigns.map(campaign => `
        <div class="campaign-card">
            <div class="campaign-name">
                <i class="fas fa-bullhorn"></i> ${campaign.name}
            </div>
            
            <div class="campaign-info">
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fas fa-users fa-beat" style="--fa-animation-duration: 2s; color: #10b981;"></i> المستخرج:</span>
                    <span class="campaign-info-value">${campaign.true_count || 0}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fa-brands fa-instagram fa-beat" style="--fa-animation-duration: 2s; color: #1dc717ff;"></i> المنصة:</span>
                    <span class="campaign-info-value">${campaign.paltform}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label">${getStatusIcon(campaign.status)} الحالة:</span>
                    <span class="status-badge status-${campaign.status}">
                        ${getStatusText(campaign.status)}
                    </span>
                </div>
            </div>
            
            <button class="campaign-actions-btn" onclick="toggleAccordion(${campaign.id})">
                <i class="fas fa-cog"></i> الإجراءات
            </button>
            
            <div class="accordion-content" id="accordion-${campaign.id}">
                <div class="accordion-section">
                    <div class="accordion-section-title">
                        <i class="fas fa-sliders-h"></i> الإعدادات
                    </div>
                    <div class="action-btns" id="actions-${campaign.id}">
                        ${getActionButtons(campaign)}
                    </div>
                </div>
                
                 <div class="accordion-section">
                    <div class="accordion-section-title">
                        <i class="fas fa-tasks"></i> الإجراء
                    </div>
                    <div class="action-btns">
                         <button class="action-btn-full btn-send" onclick="viewResults('${campaign.campaign_id || campaign.id}')">
                            <i class="fa-solid fa-table-cells fa-flip" style="--fa-animation-duration: 3s;"></i> عرض النتائج
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `).join('');
}

function getStatusText(status) {
    const statusMap = { 'pending': 'قيد الانتظار', 'running': 'قيد التشغيل', 'paused': 'متوقف مؤقتاً', 'stopped': 'متوقف', 'finished': 'منتهي' };
    return statusMap[status] || status;
}

function getStatusIcon(status) {
    const iconMap = {
        'pending': '<i class="fas fa-hourglass-half fa-spin" style="color: #f59e0b;"></i>',
        'running': '<i class="fas fa-spinner fa-spin" style="color: #3b82f6;"></i>',
        'paused': '<i class="fas fa-pause-circle fa-beat" style="color: #f59e0b;"></i>',
        'stopped': '<i class="fas fa-stop-circle fa-fade" style="color: #ef4444;"></i>',
        'finished': '<i class="fas fa-check-circle fa-bounce" style="color: #10b981;"></i>'
    };
    return iconMap[status] || '<i class="fas fa-info-circle"></i>';
}

function getActionButtons(campaign) {
    const status = campaign.status;
    let buttons = '';
    
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
        <button class="action-btn-small btn-save" onclick="downloadReport('${campaign.campaign_id || campaign.id}')">
            <i class="fas fa-download fa-beat" style="--fa-animation-duration: 2s;"></i> تحميل Excel
        </button>
    `;
    
    buttons += `
        <button class="action-btn-full btn-delete" style="margin-top: 10px;" onclick="deleteCampaign(${campaign.id})">
            <i class="fas fa-trash fa-shake" style="--fa-animation-duration: 3s;"></i> حذف الحملة
        </button>
    `;
    return buttons;
}

function toggleAccordion(id) {
    document.getElementById(`accordion-${id}`).classList.toggle('active');
}

function changeStatus(id, newStatus) {
    const statusText = { 'pending': 'تشغيل', 'paused': 'إيقاف مؤقت', 'stopped': 'إيقاف' }[newStatus] || newStatus;
    Swal.fire({
        title: 'تأكيد ' + statusText, text: 'هل أنت متأكد؟', icon: 'warning', showCancelButton: true,
        confirmButtonText: 'نعم، ' + statusText, cancelButtonText: 'إلغاء', background: '#111827', color: '#e5e7eb',
        confirmButtonColor: newStatus === 'stopped' ? '#ef4444' : '#f59e0b'
    }).then((result) => {
        if (result.isConfirmed) {
            Swal.fire({ title: 'جاري المعالجة...', allowOutsideClick: false, didOpen: () => Swal.showLoading() });
            fetch('api/manage_campaign.php', {
                method: 'POST', headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({ action: 'change_status', campaign_id: id, status: newStatus })
            }).then(res => res.json()).then(data => {
                if (data.success) {
                    Swal.fire({ icon: 'success', title: 'تم!', text: data.message, timer: 2000, showConfirmButton: false, background: '#111827', color: '#e5e7eb' });
                    loadCampaigns(TOOL_NAME);
                } else {
                    Swal.fire({ icon: 'error', title: 'خطأ', text: data.message, background: '#111827', color: '#e5e7eb', confirmButtonColor: '#667eea' });
                }
            });
        }
    });
}

function editCampaign(id) {
    const campaign = allCampaigns.find(c => Number(c.id) === Number(id) || Number(c.campaign_id) === Number(id));
    if (!campaign) return;
    
    editingCampaignId = id;
    document.getElementById('campaign-name').value = campaign.name;
    selectedAccounts = JSON.parse(campaign.token || '[]');
    renderSelectedAccounts();
    
    const editModal = document.getElementById('campaignModal');
    if(editModal){
        editModal.querySelector('.modal-title').innerHTML = '<i class="fas fa-edit"></i> تعديل الحملة';
        editModal.querySelector('.submit-btn').innerHTML = '<i class="fas fa-save"></i> حفظ التعديلات';
        editModal.style.display = 'block';
    }
}

function downloadReport(id) {
    window.location.href = "api/save_data.php?camp_id=" + id + "&tool=" + encodeURIComponent(TOOL_NAME);
}

// ===== Contacts Table (modal) =====
let contactsState = { campaign_id: null, page: 1, per_page: 25, q: '', data: [], total: 0, total_pages: 0, selected: new Set(), selectedData: {} };
let contactsSearchTimer = null;

function viewResults(campaignId) {
  contactsState.campaign_id = campaignId;
  contactsState.page = 1;
  contactsState.q = '';
  contactsState.selected = new Set();
  contactsState.selectedData = {};
  
  const searchInp = document.getElementById('contactsSearch');
  if(searchInp) searchInp.value = '';
  
  document.getElementById('contactsPerPage').value = '25';
  document.getElementById('contactsModal').style.display = 'block';
  loadContacts();
}

function closeContactsModal() { document.getElementById('contactsModal').style.display = 'none'; }

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
    table: 'ig_dms', // 👈 اسم الجدول الجديد في الداتا بيز
    campaign_id: contactsState.campaign_id,
    page: contactsState.page,
    per_page: contactsState.per_page,
    q: contactsState.q
  });
  
  fetch('api/ig_basic_info.php?' + params.toString())
    .then(r=>r.json())
    .then(j=>{
      if (!j.success) throw new Error(j.message || 'failed');
      contactsState.data = j.data || [];
      contactsState.total = j.total || 0;
      contactsState.total_pages = j.total_pages || 0;
      renderContactsTable();
      renderContactsPagination();
      document.getElementById('contactsInfo').textContent = `الصفحة ${contactsState.page} من ${contactsState.total_pages} — إجمالي ${contactsState.total} مستخدم`;
    })
    .catch(e=>{
      const tb = document.getElementById('contactsTableBody');
      tb.innerHTML = `<tr><td colspan="4" style="padding:20px; text-align:center; color:#ef4444;">خطأ: ${e.message}</td></tr>`;
    });
}

function renderContactsTable() {
  const tb = document.getElementById('contactsTableBody');
  if (!contactsState.data.length) {
    tb.innerHTML = '<tr><td colspan="4" style="padding:40px; text-align:center; color:#94a3b8;"><i class="fas fa-box-open fa-3x" style="opacity:0.5; margin-bottom:10px;"></i><br>لا توجد بيانات</td></tr>';
    return;
  }
  
  tb.innerHTML = contactsState.data.map((row, idx) => {
    // بناءً على هيكلة الداتا المتوقعة: ig_user_id هو الأساسي
    const id = row.ig_user_id || row.identifier || `temp-${idx}`;
    const username = row.username || '---';
    const fullName = row.full_name || row.name || '---';
    const checked = contactsState.selected.has(id) ? 'checked' : '';
    
    return `
      <tr style="border-bottom: 1px solid #1e293b; transition: all 0.2s ease;" onmouseover="this.style.background='#1e293b'" onmouseout="this.style.background='transparent'">
        <td style="padding:12px; text-align:center;">
            <input type="checkbox" data-id="${id}" data-igid="${escapeHtml(row.ig_user_id || '')}" data-username="${escapeHtml(username)}" data-fullname="${escapeHtml(fullName)}" onchange="toggleSelectContact(this)" ${checked}>
        </td>
        <td style="padding:12px; text-align:right;">${escapeHtml(username)}</td>
        <td style="padding:12px; text-align:right;">${escapeHtml(fullName)}</td>
        <td style="padding:12px; text-align:center;">
            <span style="direction:ltr; display:inline-block; font-family:monospace; color:#cbd5e1;">${escapeHtml(id)}</span>
        </td>
      </tr>`;
  }).join('');
  
  const selAll = document.getElementById('contactsSelectAll');
  if (selAll) selAll.checked = false;
}

function renderContactsPagination() {
  const el = document.getElementById('contactsPagination');
  const total = contactsState.total_pages;
  const cur = contactsState.page;
  if (!total || total <= 1) { el.innerHTML = ''; return; }
  
  let html = '';
  const btn = (p, text, isActive = false, isDisabled = false) => {
      const bg = isActive ? 'background: #3b82f6; color: #fff;' : 'background: #1e293b; color: #cbd5e1;';
      const border = isActive ? 'border: 1px solid #3b82f6;' : 'border: 1px solid #334155;';
      const opacity = isDisabled ? 'opacity: 0.5; cursor: not-allowed;' : 'cursor: pointer;';
      const hover = (!isDisabled && !isActive) ? `onmouseover="this.style.background='#334155'" onmouseout="this.style.background='#1e293b'"` : '';
      return `<button style="padding: 6px 14px; border-radius: 8px; font-weight: bold; outline: none; ${bg} ${border} ${opacity} transition: 0.2s;" ${isDisabled ? 'disabled' : ''} onclick="reloadContactsPage(${p})" ${hover}>${text}</button>`;
  };
  
  html += btn(1, '<i class="fas fa-angle-double-right"></i>', false, cur === 1);
  html += btn(Math.max(1, cur - 1), '<i class="fas fa-angle-right"></i>', false, cur === 1);
  
  const start = Math.max(1, cur - 2);
  const end = Math.min(total, cur + 2);
  for (let p = start; p <= end; p++) html += btn(p, p, p === cur);
  
  html += btn(Math.min(total, cur + 1), '<i class="fas fa-angle-left"></i>', false, cur === total);
  html += btn(total, '<i class="fas fa-angle-double-left"></i>', false, cur === total);
  
  el.innerHTML = html;
}

function toggleSelectAllContacts(chk) {
  const inputs = document.querySelectorAll('#contactsTableBody input[type=checkbox]');
  inputs.forEach(i=>{
    i.checked = chk.checked;
    const id = i.getAttribute('data-id');
    const igid = i.getAttribute('data-igid');
    const username = i.getAttribute('data-username');
    const fullname = i.getAttribute('data-fullname');
    if (!id) return;
    if (chk.checked) {
      contactsState.selected.add(id);
      if (igid) contactsState.selectedData[id] = { identifier: igid, name: fullname || username };
    } else {
      contactsState.selected.delete(id);
      delete contactsState.selectedData[id];
    }
  });
}

function toggleSelectContact(input) {
  const id = input.getAttribute('data-id');
  const igid = input.getAttribute('data-igid');
  const username = input.getAttribute('data-username');
  const fullname = input.getAttribute('data-fullname');
  if (!id) return;
  if (input.checked) {
    contactsState.selected.add(id);
    if (igid) contactsState.selectedData[id] = { identifier: igid, name: fullname || username };
  } else {
    contactsState.selected.delete(id);
    delete contactsState.selectedData[id];
  }
}

function openAddContactsModal(){
  const count = contactsState.selected.size;
  if (count === 0) {
    Swal.fire({ icon:'warning', title:'تنبيه', text:'يرجى تحديد مستخدم واحد على الأقل', background:'#111827', color:'#e5e7eb' });
    return;
  }
  const defName = 'Users From DMs ' + new Date().toISOString().split('T')[0];
  document.getElementById('contactsListName').value = defName;
  document.getElementById('addContactsCountHint').textContent = `سيتم إضافة ${count} مستخدم للقائمة`;
  document.getElementById('addContactsModal').style.display = 'block';
}

function closeAddContactsModal(){ document.getElementById('addContactsModal').style.display = 'none'; }

async function submitAddContactsList(){
  const name = document.getElementById('contactsListName').value.trim();
  const count = contactsState.selected.size;
  if (!name) {
    Swal.fire({ icon:'warning', title:'تنبيه', text:'يرجى إدخال اسم القائمة', background:'#111827', color:'#e5e7eb' });
    return;
  }
  const dataArr = Object.values(contactsState.selectedData);
  try {
    const res = await fetch('api/contacts_add.php', {
      method:'POST', headers:{'Content-Type':'application/json'}, credentials: 'same-origin',
      body: JSON.stringify({ name, platform:'instagram', type:'csv', count, data: dataArr })
    });
    const j = await res.json();
    if (!j.success) throw new Error(j.message||'failed');
    
    Swal.fire({ icon:'success', title:'تم!', text:'تم إضافة جهات الاتصال للقائمة بنجاح', timer:2000, showConfirmButton:false, background:'#111827', color:'#e5e7eb' });
    closeAddContactsModal();
    
    const selAll = document.getElementById('contactsSelectAll');
    if (selAll) {
        selAll.checked = false;
        toggleSelectAllContacts(selAll);
    }
  } catch(e) {
    Swal.fire({ icon:'error', title:'خطأ', text:e.message, background:'#111827', color:'#e5e7eb' });
  }
}

function escapeHtml(s){ return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'}[c])); }
</script>

<?php include 'includes/footer.php'; ?>