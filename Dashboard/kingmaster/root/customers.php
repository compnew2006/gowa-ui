 
<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة جهات الاتصال | Kingmaster";
$page_css = ['/css/rightnavbar.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>
<main class="main-content">
  <!-- Header Section -->
  <div class="content-card" style="margin-bottom: 1.5rem;">
    <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
      <div>
        <h2 style="margin: 0 0 0.5rem 0;"><i class="fa-solid fa-address-book" style="color: #63E6BE;"></i> إدارة جهات الاتصال</h2>
        <p style="margin: 0; color: var(--text-gray); font-size: 0.9rem;">قم بإضافة وإدارة جهات الاتصال الخاصة بك</p>
      </div>
      <button class="btn-add-contact" onclick="openAddModal()">
        <i class="fas fa-plus"></i>
        إضافة جهات اتصال
      </button>
    </div>
  </div>

  <!-- Contacts Grid -->
  <div id="contacts-grid" class="contacts-grid">
    <!-- Contacts will be loaded here dynamically -->
    <div class="empty-state">
      <i class="fas fa-address-book" style="font-size: 4rem; color: var(--text-gray); opacity: 0.3;"></i>
      <p style="color: var(--text-gray); margin-top: 1rem;">
        لا توجد جهات اتصال حتى الآن
      </p>
      <button onclick="openAddModal()" class="btn-add-contact" style="margin-top: 1rem;">
        <i class="fas fa-plus"></i>
        إضافة أول جهة اتصال
      </button>
    </div>
  </div>
</main>

<!-- Add Contact Modal -->
<div class="modal" id="add-modal">
  <div class="modal-content" style="max-width: 700px;">
    <div class="modal-header">
      <h3><i class="fas fa-user-plus"></i> إضافة جهات اتصال جديدة</h3>
      <button class="close-btn" onclick="closeAddModal()">
        <i class="fas fa-times"></i>
      </button>
    </div>
    <div class="modal-body">
      <!-- Contact Name -->
      <div class="form-group">
        <label><i class="fas fa-tag" style="color: #FFD43B;"></i> اسم جهة الاتصال</label>
        <input type="text" id="contact-name" class="form-input" placeholder="مثال: عملاء فيسبوك - يناير 2024">
      </div>

      <!-- Platform Select -->
      <div class="form-group">
        <label><i class="fas fa-globe" style=" color: #667eea;"></i> المنصة</label>
        <select id="platform-select" class="form-input">
          <option value="">اختر المنصة</option>
          <option value="facebook">فيسبوك</option>
          <option value="whatsapp">واتساب</option>
          <option value="instagram">انستجرام</option>
          <option value="telegram">تليجرام</option>
          <option value="tiktok">تيك توك</option>
          <option value="email">بريد إلكتروني</option>
        </select>
      </div>

      <!-- Import Type -->
      <div class="form-group">
        <label><i class="fas fa-upload" style="color: #10b981;"></i> طريقة الإضافة</label>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 0.5rem;">
          <label class="radio-option" style="margin: 0;">
            <input type="radio" name="import-type" value="csv" checked onchange="toggleImportMethod()">
            <span>عبر ملف CSV</span>
          </label>
          <label class="radio-option" style="margin: 0;">
            <input type="radio" name="import-type" value="text" onchange="toggleImportMethod()">
            <span>عبر النصوص</span>
          </label>
        </div>
      </div>

      <!-- CSV Upload Section -->
      <div id="csv-section" class="import-section">
        <div class="form-group">
          <label><i class="fas fa-file-csv" style="color: #10b981;"></i> رفع ملف CSV</label>
          <div class="upload-area" onclick="document.getElementById('csv-input').click()">
            <i class="fas fa-cloud-upload-alt" style="font-size: 2rem; color: var(--primary-color); margin-bottom: 0.5rem;"></i>
            <p style="color: var(--text-light); font-weight: 600; margin-bottom: 0.3rem;">اضغط لاختيار ملف CSV</p>
            <p style="color: var(--text-gray); font-size: 0.85rem;">سيتم أخذ العمودين الأولين فقط (المعرف/الرقم، الاسم)</p>
            <input type="file" id="csv-input" accept=".csv" style="display: none;" onchange="handleCSVUpload(event)">
          </div>
          <div id="csv-preview" style="display: none; margin-top: 1rem;">
            <p style="color: var(--text-light); font-weight: 600; margin-bottom: 0.5rem;">
              <i class="fas fa-check-circle" style="color: #10b981;"></i> 
              تم تحميل الملف: <span id="csv-filename"></span>
            </p>
            <p style="color: var(--text-gray); font-size: 0.9rem;">
              عدد السجلات: <strong id="csv-count">0</strong>
            </p>
          </div>
        </div>
      </div>

      <!-- Text Input Section -->
      <div id="text-section" class="import-section" style="display: none;">
        <div class="form-group">
          <label><i class="fas fa-align-left" style="color: #667eea;"></i> أدخل البيانات (كل رقم/معرف في سطر)</label>
          <textarea id="text-input" class="form-textarea" rows="10" placeholder="01012345678&#10;01098765432&#10;example@email.com&#10;..."></textarea>
          <p style="color: var(--text-gray); font-size: 0.85rem; margin-top: 0.5rem;">
            <i class="fas fa-info-circle"></i> أقصى عدد: <strong>20,000 سطر</strong>
          </p>
        </div>
      </div>
    </div>
    <div class="modal-footer">
      <button class="btn-cancel" onclick="closeAddModal()">إلغاء</button>
      <button class="btn-save" onclick="saveContact()">
        <i class="fas fa-save"></i>
        حفظ
      </button>
    </div>
  </div>
</div>

<!-- Edit Contact Modal -->
<div class="modal" id="edit-modal">
  <div class="modal-content" style="max-width: 800px;">
    <div class="modal-header">
      <h3><i class="fas fa-edit"></i> تعديل جهة الاتصال</h3>
      <button class="close-btn" onclick="closeEditModal()">
        <i class="fas fa-times"></i>
      </button>
    </div>
    <div class="modal-body">
      <input type="hidden" id="edit-contact-id">
      
      <!-- Contact Name -->
      <div class="form-group">
        <label><i class="fas fa-tag" style="color: #FFD43B;"></i> اسم جهة الاتصال</label>
        <input type="text" id="edit-contact-name" class="form-input">
      </div>

      <!-- Platform Select -->
      <div class="form-group">
        <label><i class="fas fa-globe" style=" color: #667eea;"></i> المنصة</label>
        <select id="edit-platform-select" class="form-input">
          <option value="facebook">فيسبوك</option>
          <option value="whatsapp">واتساب</option>
          <option value="instagram">انستجرام</option>
          <option value="telegram">تليجرام</option>
          <option value="tiktok">تيك توك</option>
          <option value="email">بريد إلكتروني</option>
        </select>
      </div>

      <!-- Manage Contacts Section -->
      <div class="form-group" style="border-top: 1px solid var(--border-color); padding-top: 1.5rem; margin-top: 1.5rem;">
        <label><i class="fas fa-users-cog" style="color: #667eea;"></i> إدارة جهات الاتصال</label>
        
        <!-- Add More Contacts -->
        <div style="margin-bottom: 1rem;">
          <button class="btn-manage" onclick="showAddMoreSection()">
            <i class="fas fa-plus-circle"></i>
            إضافة جهات اتصال إضافية
          </button>
        </div>

        <!-- Add More Section -->
        <div id="add-more-section" style="display: none; margin-bottom: 1rem;">
          <div style="display: flex; gap: 1rem; margin-bottom: 1rem;">
            <label class="radio-option" style="flex: 1;">
              <input type="radio" name="add-more-type" value="csv" checked onchange="toggleAddMoreMethod()">
              <span>عبر CSV</span>
            </label>
            <label class="radio-option" style="flex: 1;">
              <input type="radio" name="add-more-type" value="text" onchange="toggleAddMoreMethod()">
              <span>عبر النص</span>
            </label>
          </div>

          <div id="add-more-csv" class="add-more-method">
            <div class="upload-area" style="padding: 1rem;" onclick="document.getElementById('add-more-csv-input').click()">
              <i class="fas fa-file-csv" style="font-size: 1.5rem; color: var(--primary-color);"></i>
              <p style="margin: 0.5rem 0 0 0; font-size: 0.9rem;">اضغط لاختيار ملف CSV</p>
              <input type="file" id="add-more-csv-input" accept=".csv" style="display: none;" onchange="handleAddMoreCSV(event)">
            </div>
          </div>

          <div id="add-more-text" class="add-more-method" style="display: none;">
            <textarea id="add-more-text-input" class="form-textarea" rows="5" placeholder="01012345678&#10;01098765432&#10;..."></textarea>
          </div>

          <button class="btn-manage" onclick="addMoreContacts()" style="margin-top: 0.5rem; width: 100%;">
            <i class="fas fa-check"></i>
            إضافة الآن
          </button>
        </div>

        <!-- Search and Remove -->
        <div>
          <button class="btn-manage btn-danger" onclick="showRemoveSection()">
            <i class="fas fa-search-minus"></i>
            البحث وحذف جهة اتصال
          </button>
        </div>

        <!-- Remove Section -->
        <div id="remove-section" style="display: none; margin-top: 1rem;">
          <div style="display: flex; gap: 0.5rem;">
            <input type="text" id="search-contact-input" class="form-input" placeholder="ابحث عن رقم أو معرف..." style="flex: 1;">
            <button class="btn-manage" onclick="searchAndRemove()">
              <i class="fas fa-search"></i>
              بحث وحذف
            </button>
          </div>
          <p style="color: var(--text-gray); font-size: 0.85rem; margin-top: 0.5rem;">
            <i class="fas fa-info-circle"></i> سيتم حذف جميع التطابقات من القائمة
          </p>
        </div>

        <!-- Statistics -->
        <div style="margin-top: 1rem; padding: 1rem; background: rgba(102, 126, 234, 0.05); border-radius: 8px;">
          <p style="margin: 0; color: var(--text-gray); font-size: 0.9rem;">
            <i class="fas fa-info-circle"></i> إجمالي جهات الاتصال: <strong id="edit-total-count">0</strong>
          </p>
        </div>
      </div>
    </div>
    <div class="modal-footer">
      <button class="btn-cancel" onclick="closeEditModal()">إلغاء</button>
      <button class="btn-save" onclick="updateContact()">
        <i class="fas fa-save"></i>
        حفظ التعديلات
      </button>
    </div>
  </div>
</div>


<script>
let contacts = [];
let csvData = [];
let addMoreCsvData = [];
let currentEditingContact = null;

// Load contacts from database
function loadContacts() {
  fetch('api/contacts_api.php?action=get_all')
    .then(response => response.json())
    .then(data => {
      if (data.success) {
        contacts = data.contacts;
        renderContacts();
      } else {
        console.error('Error loading contacts:', data.message);
        Swal.fire({
          icon: 'error',
          title: 'خطأ',
          text: data.message
        });
      }
    })
    .catch(error => {
      console.error('Error:', error);
    });
}

// Get platform icon and color
function getPlatformInfo(platform) {
  const platforms = {
    facebook: { icon: 'fa-brands fa-facebook', class: 'platform-facebook', name: 'فيسبوك' },
    whatsapp: { icon: 'fa-brands fa-whatsapp', class: 'platform-whatsapp', name: 'واتساب' },
    instagram: { icon: 'fa-brands fa-instagram', class: 'platform-instagram', name: 'انستجرام' },
    telegram: { icon: 'fa-brands fa-telegram', class: 'platform-telegram', name: 'تليجرام' },
    tiktok: { icon: 'fa-brands fa-tiktok', class: 'platform-tiktok', name: 'تيك توك' },
    email: { icon: 'fas fa-envelope', class: 'platform-email', name: 'بريد إلكتروني' }
  };
  return platforms[platform] || platforms.facebook;
}

// Render contacts
function renderContacts() {
  const grid = document.getElementById('contacts-grid');
  
  if (contacts.length === 0) {
    grid.innerHTML = `
      <div class="empty-state">
        <i class="fas fa-address-book" style="font-size: 4rem; color: var(--text-gray); opacity: 0.3;"></i>
        <p style="color: var(--text-gray); margin-top: 1rem;">لا توجد جهات اتصال حتى الآن</p>
        <button onclick="openAddModal()" class="btn-add-contact" style="margin-top: 1rem;">
          <i class="fas fa-plus"></i>
          إضافة أول جهة اتصال
        </button>
      </div>
    `;
    return;
  }

  grid.innerHTML = contacts.map(contact => {
    const platform = getPlatformInfo(contact.platform);
    const sent_count = contact.sent_count || 0;
    const progress = contact.count > 0 ? Math.round((sent_count / contact.count) * 100) : 0;
    const status = contact.sending_status || 'idle';
    
    let statusText = 'لم يتم الإرسال';
    let statusColor = 'var(--text-gray)';
    
    if(status === 'sending') {
      statusText = 'جاري الإرسال...';
      statusColor = '#3b82f6';
    } else if(status === 'paused') {
      statusText = 'متوقف';
      statusColor = '#f59e0b';
    } else if(status === 'completed') {
      statusText = 'مكتمل';
      statusColor = '#10b981';
    }
    
    return `
      <div class="contact-card">
        <div class="contact-platform ${platform.class}">
          <i class="${platform.icon}"></i>
          <span>${platform.name}</span>
        </div>
        <div class="contact-name" title="${contact.name}">${contact.name}</div>
        <div class="contact-info">
          <div class="info-item">
            <div class="info-label">العدد</div>
            <div class="info-value">${contact.count.toLocaleString()}</div>
          </div>
          <div class="info-item">
            <div class="info-label">النوع</div>
            <div class="info-value" style="font-size: 0.9rem;">${contact.type === 'csv' ? 'CSV' : 'نص'}</div>
          </div>
        </div>
        
        ${sent_count > 0 ? `
        <div style="margin: 1rem 0;">
          <div style="display: flex; justify-content: space-between; margin-bottom: 0.3rem;">
            <span style="font-size: 0.85rem; color: var(--text-gray);">التقدم</span>
            <span style="font-size: 0.85rem; font-weight: 600; color: ${statusColor};">${sent_count}/${contact.count} (${progress}%)</span>
          </div>
          <div style="width: 100%; height: 6px; background: rgba(102, 126, 234, 0.1); border-radius: 10px; overflow: hidden;">
            <div style="width: ${progress}%; height: 100%; background: linear-gradient(90deg, var(--primary-color), var(--secondary-color)); transition: width 0.3s ease;"></div>
          </div>
          <div style="text-align: center; margin-top: 0.3rem;">
            <span style="font-size: 0.8rem; color: ${statusColor};">${statusText}</span>
          </div>
        </div>
        ` : ''}
        
        <div class="contact-actions">
          <button class="btn-edit" onclick="editContact('${contact.id}')">
            <i class="fas fa-edit"></i>
            تعديل
          </button>
          <button class="btn-delete" onclick="deleteContact('${contact.id}')">
            <i class="fas fa-trash"></i>
            حذف
          </button>
        </div>
      </div>
    `;
  }).join('');
}

// Open Add Modal
function openAddModal() {
  document.getElementById('add-modal').classList.add('active');
  document.getElementById('contact-name').value = '';
  document.getElementById('platform-select').value = '';
  document.getElementById('csv-input').value = '';
  document.getElementById('text-input').value = '';
  document.getElementById('csv-preview').style.display = 'none';
  document.querySelector('input[name="import-type"][value="csv"]').checked = true;
  toggleImportMethod();
  csvData = [];
}

// Close Add Modal
function closeAddModal() {
  document.getElementById('add-modal').classList.remove('active');
}

// Toggle Import Method
function toggleImportMethod() {
  const importType = document.querySelector('input[name="import-type"]:checked').value;
  const csvSection = document.getElementById('csv-section');
  const textSection = document.getElementById('text-section');
  
  if (importType === 'csv') {
    csvSection.style.display = 'block';
    textSection.style.display = 'none';
  } else {
    csvSection.style.display = 'none';
    textSection.style.display = 'block';
  }
}

// Handle CSV Upload
function handleCSVUpload(event) {
  const file = event.target.files[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = function(e) {
    const text = e.target.result;
    const lines = text.split('\n').filter(line => line.trim());
    
    csvData = lines.map(line => {
      const cols = line.split(',');
      return {
        identifier: cols[0]?.trim() || '',
        name: cols[1]?.trim() || ''
      };
    }).filter(item => item.identifier);

    document.getElementById('csv-filename').textContent = file.name;
    document.getElementById('csv-count').textContent = csvData.length.toLocaleString();
    document.getElementById('csv-preview').style.display = 'block';
  };
  reader.readAsText(file);
}

// Save Contact
function saveContact() {
  const name = document.getElementById('contact-name').value.trim();
  const platform = document.getElementById('platform-select').value;
  const importType = document.querySelector('input[name="import-type"]:checked').value;

  // Validate contact name
  if (!name || name === '') {
    Swal.fire({
      icon: 'error',
      title: 'خطأ في الاسم',
      text: 'الرجاء إدخال اسم جهة الاتصال',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    return;
  }

  // Validate platform selection
  if (!platform || platform === '') {
    Swal.fire({
      icon: 'error',
      title: 'خطأ في المنصة',
      text: 'الرجاء اختيار المنصة',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    return;
  }

  let count = 0;
  let data = [];

  if (importType === 'csv') {
    if (csvData.length === 0) {
      Swal.fire({
        icon: 'error',
        title: 'خطأ في الملف',
        text: 'الرجاء رفع ملف CSV',
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea'
      });
      return;
    }
    count = csvData.length;
    data = csvData;
  } else {
    const text = document.getElementById('text-input').value.trim();
    if (!text || text === '') {
      Swal.fire({
        icon: 'error',
        title: 'خطأ في البيانات',
        text: 'الرجاء إدخال البيانات',
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea'
      });
      return;
    }

    const lines = text.split('\n').filter(line => line.trim());
    
    if (lines.length === 0) {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: 'الرجاء إدخال بيانات صحيحة',
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea'
      });
      return;
    }
    
    if (lines.length > 20000) {
      Swal.fire({
        icon: 'error',
        title: 'تجاوز الحد المسموح',
        text: `لا يمكن إضافة أكثر من 20,000 سطر. لديك ${lines.length.toLocaleString()} سطر`,
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea'
      });
      return;
    }

    count = lines.length;
    data = lines.map(line => ({ identifier: line.trim() }));
  }

  // Send to database
  const formData = new FormData();
  formData.append('action', 'add');
  formData.append('name', name);
  formData.append('platform', platform);
  formData.append('type', importType);
  formData.append('count', count);
  formData.append('data', JSON.stringify(data));

  fetch('api/contacts_api.php', {
    method: 'POST',
    body: formData
  })
  .then(response => response.json())
  .then(result => {
    if (result.success) {
      closeAddModal();
      loadContacts(); // Reload from database
      
      Swal.fire({
        icon: 'success',
        title: 'تم الحفظ!',
        text: `تم إضافة ${count.toLocaleString()} جهة اتصال بنجاح`,
        timer: 2000,
        showConfirmButton: false,
        timerProgressBar: true
      });
    } else {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: result.message
      });
    }
  })
  .catch(error => {
    console.error('Error:', error);
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ أثناء الحفظ'
    });
  });
}

// Edit Contact
function editContact(id) {
  const contact = contacts.find(c => c.id == id); // Use == instead of === for type flexibility
  if (!contact) {
    console.error('Contact not found:', id);
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'جهة الاتصال غير موجودة'
    });
    return;
  }

  // Make sure data is an array
  if (typeof contact.data === 'string') {
    try {
      contact.data = JSON.parse(contact.data);
    } catch(e) {
      console.error('Error parsing contact data:', e);
      contact.data = [];
    }
  }
  if (!Array.isArray(contact.data)) {
    contact.data = [];
  }

  currentEditingContact = contact;
  document.getElementById('edit-contact-id').value = contact.id;
  document.getElementById('edit-contact-name').value = contact.name;
  document.getElementById('edit-platform-select').value = contact.platform;
  document.getElementById('edit-total-count').textContent = contact.count.toLocaleString();
  
  // Reset sections
  document.getElementById('add-more-section').style.display = 'none';
  document.getElementById('remove-section').style.display = 'none';
  document.getElementById('add-more-csv-input').value = '';
  document.getElementById('add-more-text-input').value = '';
  document.getElementById('search-contact-input').value = '';
  addMoreCsvData = [];
  
  document.getElementById('edit-modal').classList.add('active');
}

// Show/Hide Add More Section
function showAddMoreSection() {
  const section = document.getElementById('add-more-section');
  section.style.display = section.style.display === 'none' ? 'block' : 'none';
  document.getElementById('remove-section').style.display = 'none';
}

// Show/Hide Remove Section
function showRemoveSection() {
  const section = document.getElementById('remove-section');
  section.style.display = section.style.display === 'none' ? 'block' : 'none';
  document.getElementById('add-more-section').style.display = 'none';
}

// Toggle Add More Method
function toggleAddMoreMethod() {
  const type = document.querySelector('input[name="add-more-type"]:checked').value;
  const csvSection = document.getElementById('add-more-csv');
  const textSection = document.getElementById('add-more-text');
  
  if (type === 'csv') {
    csvSection.style.display = 'block';
    textSection.style.display = 'none';
  } else {
    csvSection.style.display = 'none';
    textSection.style.display = 'block';
  }
}

// Handle Add More CSV
function handleAddMoreCSV(event) {
  const file = event.target.files[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = function(e) {
    const text = e.target.result;
    const lines = text.split('\n').filter(line => line.trim());
    
    addMoreCsvData = lines.map(line => {
      const cols = line.split(',');
      return {
        identifier: cols[0]?.trim() || '',
        name: cols[1]?.trim() || ''
      };
    }).filter(item => item.identifier);

    Swal.fire({
      icon: 'success',
      title: 'تم التحميل!',
      text: `تم تحميل ${addMoreCsvData.length} سجل`,
      timer: 1500,
      showConfirmButton: false
    });
  };
  reader.readAsText(file);
}

// Add More Contacts
function addMoreContacts() {
  if (!currentEditingContact) return;

  const type = document.querySelector('input[name="add-more-type"]:checked').value;
  let newData = [];

  if (type === 'csv') {
    if (addMoreCsvData.length === 0) {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: 'الرجاء رفع ملف CSV أولاً',
        confirmButtonColor: '#667eea'
      });
      return;
    }
    newData = addMoreCsvData;
  } else {
    const text = document.getElementById('add-more-text-input').value.trim();
    if (!text) {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: 'الرجاء إدخال البيانات',
        confirmButtonColor: '#667eea'
      });
      return;
    }
    const lines = text.split('\n').filter(line => line.trim());
    newData = lines.map(line => ({ identifier: line.trim() }));
  }

  // Add new data to existing contact data
  currentEditingContact.data = [...currentEditingContact.data, ...newData];
  currentEditingContact.count = currentEditingContact.data.length;

  // Update display
  document.getElementById('edit-total-count').textContent = currentEditingContact.count.toLocaleString();
  
  // Clear inputs
  document.getElementById('add-more-csv-input').value = '';
  document.getElementById('add-more-text-input').value = '';
  document.getElementById('add-more-section').style.display = 'none';
  addMoreCsvData = [];

  Swal.fire({
    icon: 'success',
    title: 'تمت الإضافة!',
    text: `تمت إضافة ${newData.length} جهة اتصال`,
    timer: 1500,
    showConfirmButton: false
  });
}

// Search and Remove Contact
function searchAndRemove() {
  if (!currentEditingContact) return;

  const searchTerm = document.getElementById('search-contact-input').value.trim();
  
  if (!searchTerm) {
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'الرجاء إدخال كلمة البحث',
      confirmButtonColor: '#667eea'
    });
    return;
  }

  const beforeCount = currentEditingContact.data.length;
  
  // Remove matching contacts
  currentEditingContact.data = currentEditingContact.data.filter(item => {
    return !item.identifier.includes(searchTerm) && 
           (!item.name || !item.name.includes(searchTerm));
  });

  const afterCount = currentEditingContact.data.length;
  const removedCount = beforeCount - afterCount;

  if (removedCount === 0) {
    Swal.fire({
      icon: 'info',
      title: 'لا توجد تطابقات',
      text: 'لم يتم العثور على أي تطابقات للحذف',
      confirmButtonColor: '#667eea'
    });
    return;
  }

  currentEditingContact.count = afterCount;
  document.getElementById('edit-total-count').textContent = afterCount.toLocaleString();
  document.getElementById('search-contact-input').value = '';

  Swal.fire({
    icon: 'success',
    title: 'تم الحذف!',
    text: `تم حذف ${removedCount} جهة اتصال`,
    timer: 1500,
    showConfirmButton: false
  });
}

// Update Contact
function updateContact() {
  const id = document.getElementById('edit-contact-id').value;
  const name = document.getElementById('edit-contact-name').value.trim();
  const platform = document.getElementById('edit-platform-select').value;

  if (!name || name === '') {
    Swal.fire({
      icon: 'error',
      title: 'خطأ في الاسم',
      text: 'الرجاء إدخال اسم جهة الاتصال',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    return;
  }

  if (!platform || platform === '') {
    Swal.fire({
      icon: 'error',
      title: 'خطأ في المنصة',
      text: 'الرجاء اختيار المنصة',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    return;
  }

  // Update in database
  const formData = new FormData();
  formData.append('action', 'update');
  formData.append('id', id);
  formData.append('name', name);
  formData.append('platform', platform);
  formData.append('count', currentEditingContact.count);
  formData.append('data', JSON.stringify(currentEditingContact.data));

  fetch('api/contacts_api.php', {
    method: 'POST',
    body: formData
  })
  .then(response => response.json())
  .then(result => {
    if (result.success) {
      closeEditModal();
      loadContacts();
      
      Swal.fire({
        icon: 'success',
        title: 'تم التحديث!',
        text: 'تم تحديث جهة الاتصال بنجاح',
        timer: 2000,
        showConfirmButton: false
      });
    } else {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: result.message
      });
    }
  })
  .catch(error => {
    console.error('Error:', error);
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ أثناء التحديث'
    });
  });
}

// Close Edit Modal
function closeEditModal() {
  document.getElementById('edit-modal').classList.remove('active');
}

// Delete Contact
function deleteContact(id) {
  Swal.fire({
    title: 'هل أنت متأكد؟',
    text: 'سيتم حذف جهة الاتصال نهائياً',
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#ef4444',
    cancelButtonColor: '#667eea',
    confirmButtonText: 'نعم، احذف!',
    cancelButtonText: 'إلغاء'
  }).then((result) => {
    if (result.isConfirmed) {
      const formData = new FormData();
      formData.append('action', 'delete');
      formData.append('id', id);

      fetch('api/contacts_api.php', {
        method: 'POST',
        body: formData
      })
      .then(response => response.json())
      .then(data => {
        if (data.success) {
          loadContacts();
          
          Swal.fire({
            icon: 'success',
            title: 'تم الحذف!',
            text: 'تم حذف جهة الاتصال بنجاح',
            timer: 2000,
            showConfirmButton: false
          });
        } else {
          Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: data.message
          });
        }
      })
      .catch(error => {
        console.error('Error:', error);
        Swal.fire({
          icon: 'error',
          title: 'خطأ',
          text: 'حدث خطأ أثناء الحذف'
        });
      });
    }
  });
}

// Close modals on outside click
window.onclick = function(event) {
  const addModal = document.getElementById('add-modal');
  const editModal = document.getElementById('edit-modal');
  
  if (event.target === addModal) {
    closeAddModal();
  }
  if (event.target === editModal) {
    closeEditModal();
  }
}

// Initialize
document.addEventListener('DOMContentLoaded', function() {
  loadContacts();
});
</script>

<?php include 'includes/footer.php'; ?>
