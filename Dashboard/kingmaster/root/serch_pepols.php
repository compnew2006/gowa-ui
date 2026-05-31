




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
                <label class="form-label">
                    <i class="fas fa-tag"></i> كلمه البحث 
                </label>
                <input type="text" class="form-input" id="keyword" placeholder="أدخل كلمه البحث" required>
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
            <th style="padding:10px; border-bottom:1px solid var(--border-color); text-align:right;">الاسم</th>
            <th style="padding:10px; border-bottom:1px solid var(--border-color); text-align:right;">المعرف</th>
            <th style="padding:10px; border-bottom:1px solid var(--border-color); text-align:right;">العدد</th>
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
    loadIntervals();
    loadCampaigns('sech_pp');
    loadContent();
});

function openModal() {
    document.getElementById('campaignModal').style.display = 'block';
}

function closeModal() {
    document.getElementById('campaignModal').style.display = 'none';
    document.getElementById('campaignForm').reset();
    selectedAccounts = [];
    document.getElementById('selected-accounts').innerHTML = '';
    editingCampaignId = null;
    
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
    fetch('api/get_accounts_fb.php')
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

function loadContent() {
    fetch('api/get_content.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const select = document.getElementById('content-select');
            data.content.forEach(content => {
                const option = document.createElement('option');
                option.value = content.id;
                option.textContent = content.name;
                select.appendChild(option);
            });
        }
    });
}

function loadIntervals() {
    fetch('api/get_intervals.php')
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            const select = document.getElementById('interval-select');
            const commentsSelect = document.getElementById('comments-interval-select');
            data.intervals.forEach(interval => {
                const option1 = document.createElement('option');
                option1.value = interval.id;
                option1.textContent = interval.settings_name;
                select.appendChild(option1);
                
                const option2 = document.createElement('option');
                option2.value = interval.id;
                option2.textContent = interval.settings_name;
                commentsSelect.appendChild(option2);
            });
        }
    });
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
        // Update existing campaign
        const data = {
            action: 'update',
            campaign_id: editingCampaignId,
            name: document.getElementById('campaign-name').value,
            accounts: selectedAccounts
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
                loadCampaigns('sech_pp');
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
        // Create new campaign
        const data = {
            name: document.getElementById('campaign-name').value,
            accounts: selectedAccounts,
            tools: "sech_pp",
            paltform: "facebook",
             keyword:  document.getElementById('keyword').value,
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
                loadCampaigns('sech_pp');
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
                    <span class="campaign-info-label"><i class="fas fa-check-double fa-beat" style="--fa-animation-duration: 2s; color: #10b981;"></i> العدد:</span>
                    <span class="campaign-info-value">${campaign.true_count}</span>
                </div>
                <div class="campaign-info-item">
                    <span class="campaign-info-label"><i class="fa-brands fa-whatsapp fa-beat" style="--fa-animation-duration: 2s; color: #1dc717ff;"></i> المنصة:</span>
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
                       
                         <button class="action-btn-full btn-send" onclick="sendMessage(${campaign.campaign_id})">
                            <i class="fa-solid fa-table-cells fa-flip" style="--fa-animation-duration: 3s;"></i> عرض في جدول
                        </button>

                    </div>
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
                    loadCampaigns('sech_pp');
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
    const campaign = allCampaigns.find(c => c.id === id);
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
}


function saveCampaign(id) {
   
     
     window.location.href = "api/save_data.php?camp_id=" + id+"&tool=Extract-serch-pg_b";
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
                    loadCampaigns('sech_pp');
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
  fetch('api/serchpag.php?' + params.toString())
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
    const checked = contactsState.selected.has(id) ? 'checked' : '';
    return `
      <tr>
        <td style="padding:8px; text-align:center;"><input type="checkbox" data-id="${id}" data-phone="${escapeHtml(row.page_id||'')}" data-name="${escapeHtml((row.name||row.followers_count||'').toString())}" onchange="toggleSelectContact(this)" ${checked}></td>
        <td style="padding:8px;">${escapeHtml(row.name || '')}</td>
        <td style="padding:8px; direction:ltr;">${escapeHtml(row.page_id || '')}</td>
        <td style="padding:8px;">${escapeHtml(row.followers_count || '')}</td>
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
            loadCampaigns('sech_pp');
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
});
</script>

<?php include 'includes/footer.php'; ?>
