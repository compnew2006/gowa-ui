<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];

$page_title = "استخراج البيانات | Kingmaster";
$page_css = ['/css/toppages.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';

$countries = [
    ['code' => 'all', 'name_ar' => 'جميع الدول', 'name_en' => 'All Countries', 'flag' => '🌍'],
    ['code' => 'EG', 'name_ar' => 'مصر', 'name_en' => 'Egypt', 'flag' => '🇪🇬'],
    ['code' => 'SA', 'name_ar' => 'السعودية', 'name_en' => 'Saudi Arabia', 'flag' => '🇸🇦'],
    ['code' => 'AE', 'name_ar' => 'الإمارات', 'name_en' => 'UAE', 'flag' => '🇦🇪'],
    ['code' => 'KW', 'name_ar' => 'الكويت', 'name_en' => 'Kuwait', 'flag' => '🇰🇼'],
    ['code' => 'QA', 'name_ar' => 'قطر', 'name_en' => 'Qatar', 'flag' => '🇶🇦'],
    ['code' => 'BH', 'name_ar' => 'البحرين', 'name_en' => 'Bahrain', 'flag' => '🇧🇭'],
    ['code' => 'OM', 'name_ar' => 'عمان', 'name_en' => 'Oman', 'flag' => '🇴🇲'],
    ['code' => 'JO', 'name_ar' => 'الأردن', 'name_en' => 'Jordan', 'flag' => '🇯🇴'],
    ['code' => 'LB', 'name_ar' => 'لبنان', 'name_en' => 'Lebanon', 'flag' => '🇱🇧'],
    ['code' => 'SY', 'name_ar' => 'سوريا', 'name_en' => 'Syria', 'flag' => '🇸🇾'],
    ['code' => 'IQ', 'name_ar' => 'العراق', 'name_en' => 'Iraq', 'flag' => '🇮🇶'],
    ['code' => 'PS', 'name_ar' => 'فلسطين', 'name_en' => 'Palestine', 'flag' => '🇵🇸'],
    ['code' => 'YE', 'name_ar' => 'اليمن', 'name_en' => 'Yemen', 'flag' => '🇾🇪'],
    ['code' => 'LY', 'name_ar' => 'ليبيا', 'name_en' => 'Libya', 'flag' => '🇱🇾'],
    ['code' => 'TN', 'name_ar' => 'تونس', 'name_en' => 'Tunisia', 'flag' => '🇹🇳'],
    ['code' => 'DZ', 'name_ar' => 'الجزائر', 'name_en' => 'Algeria', 'flag' => '🇩🇿'],
    ['code' => 'MA', 'name_ar' => 'المغرب', 'name_en' => 'Morocco', 'flag' => '🇲🇦'],
    ['code' => 'SD', 'name_ar' => 'السودان', 'name_en' => 'Sudan', 'flag' => '🇸🇩'],
    ['code' => 'US', 'name_ar' => 'الولايات المتحدة', 'name_en' => 'United States', 'flag' => '🇺🇸'],
    ['code' => 'GB', 'name_ar' => 'بريطانيا', 'name_en' => 'United Kingdom', 'flag' => '🇬🇧'],
    ['code' => 'FR', 'name_ar' => 'فرنسا', 'name_en' => 'France', 'flag' => '🇫🇷'],
    ['code' => 'DE', 'name_ar' => 'ألمانيا', 'name_en' => 'Germany', 'flag' => '🇩🇪'],
    ['code' => 'IT', 'name_ar' => 'إيطاليا', 'name_en' => 'Italy', 'flag' => '🇮🇹'],
    ['code' => 'ES', 'name_ar' => 'إسبانيا', 'name_en' => 'Spain', 'flag' => '🇪🇸'],
    ['code' => 'TR', 'name_ar' => 'تركيا', 'name_en' => 'Turkey', 'flag' => '🇹🇷'],
];
?>

<div class="data-container">
  <div class="data-header">
    <div class="data-title"><i class="fas fa-database"></i>استخراج البيانات</div>
    <button class="extract-btn" onclick="openMethodModal()"><i class="fas fa-download"></i>استخراج البيانات</button>
  </div>

  <div class="points-bar">
    <div>
      <div class="label">نقاطك الحالية</div>
      <div class="hint">سيتم الخصم عند ظهور النتائج (بعد تأكيدك)</div>
    </div>
    <div class="value" id="userPoints">--</div>
  </div>

  <div id="campaignsGrid" class="campaigns-grid"></div>
</div>

<div id="methodModal" class="modal">
  <div class="modal-content">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-tasks"></i>اختر طريقة الاستخراج</div>
      <span class="close-modal" onclick="closeModal('methodModal')">&times;</span>
    </div>
    <div class="method-selection">
      <div class="method-btn" onclick="selectMethod('uids')"><i class="fas fa-id-card"></i><span>عبر المعرفات</span></div>
      <div class="method-btn" onclick="selectMethod('keywords')"><i class="fas fa-keyboard"></i><span>عبر الكلمات</span></div>
    </div>
  </div>
</div>

<div id="keywordsModal" class="modal">
  <div class="modal-content">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-keyboard"></i>استخراج عبر الكلمات المفتاحية</div>
      <span class="close-modal" onclick="closeModal('keywordsModal')">&times;</span>
    </div>

    <form onsubmit="event.preventDefault(); submitCampaign('keywords');">
      <div class="form-group">
        <label class="form-label">اسم الحملة</label>
        <input type="text" class="form-input" id="campaignNameKeywords" placeholder="أدخل اسم الحملة" required>
      </div>

      <div class="form-group">
        <label class="form-label">الدولة</label>
        <select class="form-select" id="countryKeywords">
          <?php foreach($countries as $country): ?>
            <option value="<?php echo $country['code']; ?>"><?php echo $country['flag']; ?> <?php echo $country['name_ar']; ?></option>
          <?php endforeach; ?>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">العدد المطلوب استخراجه</label>
        <input type="number" class="form-input" id="limitKeywords" value="1000" min="1" max="50000" required>
        <small style="display:block;margin-top:5px;color:var(--text-secondary);font-size:12px;font-family:'Cairo',sans-serif;">
          سيتم إيقاف البحث فور الوصول لهذا العدد لضمان سرعة الاستخراج
        </small>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في العمل</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'workKeywords')">
        <div class="tags-container" id="workKeywordsTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في التعليم</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'educationKeywords')">
        <div class="tags-container" id="educationKeywordsTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في الموقع</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'locationKeywords')">
        <div class="tags-container" id="locationKeywordsTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في المدينة</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'cityKeywords')">
        <div class="tags-container" id="cityKeywordsTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">الحالة الاجتماعية</label>
        <select class="form-select" id="maritalStatusKeywords">
          <option value="">الكل</option>
          <option value="single">أعزب</option>
          <option value="engaged">مخطوب</option>
          <option value="married">متزوج</option>
        </select>
      </div>

      <button type="submit" class="submit-btn"><i class="fas fa-search"></i>بدء البحث</button>
    </form>
  </div>
</div>

<div id="uidModal" class="modal">
  <div class="modal-content">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-id-card"></i>استخراج عبر المعرفات</div>
      <span class="close-modal" onclick="closeModal('uidModal')">&times;</span>
    </div>

    <form onsubmit="event.preventDefault(); submitCampaign('uids');">
      <div class="form-group">
        <label class="form-label">اسم الحملة</label>
        <input type="text" class="form-input" id="campaignName" placeholder="أدخل اسم الحملة" required>
      </div>

      <div class="form-group">
        <label class="form-label">الدولة</label>
        <select class="form-select" id="country">
          <?php foreach($countries as $country): ?>
            <option value="<?php echo $country['code']; ?>"><?php echo $country['flag']; ?> <?php echo $country['name_ar']; ?></option>
          <?php endforeach; ?>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">العدد المطلوب استخراجه</label>
        <input type="number" class="form-input" id="limitUids" value="1000" min="1" max="50000" required>
        <small style="display:block;margin-top:5px;color:var(--text-secondary);font-size:12px;font-family:'Cairo',sans-serif;">
          سيتم إيقاف البحث فور الوصول لهذا العدد لضمان سرعة الاستخراج
        </small>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في العمل</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'work')">
        <div class="tags-container" id="workTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في التعليم</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'education')">
        <div class="tags-container" id="educationTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في الموقع</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'location')">
        <div class="tags-container" id="locationTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">البحث في المدينة</label>
        <input type="text" class="form-input" placeholder="أدخل كلمة واضغط Enter" onkeypress="addTag(event,'city')">
        <div class="tags-container" id="cityTags"></div>
      </div>

      <div class="form-group">
        <label class="form-label">الحالة الاجتماعية</label>
        <select class="form-select" id="maritalStatus">
          <option value="">الكل</option>
          <option value="single">أعزب</option>
          <option value="engaged">مخطوب</option>
          <option value="married">متزوج</option>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">نوع المعرف</label>
        <select class="form-select" id="uidType">
          <option value="fb_id">FB ID (UID)</option>
          <option value="id">ID (Database Row)</option>
        </select>
        <small style="display:block;margin-top:5px;color:var(--text-secondary);font-size:12px;font-family:'Cairo',sans-serif;">
          اختر هل ستبحث بمعرف فيسبوك (fb_id) أو رقم السجل داخل قاعدة البيانات (id)
        </small>
      </div>

      <div class="form-group">
        <label class="form-label">المعرفات</label>
        <textarea class="form-input" id="uidsInput" rows="6" placeholder="كل معرف في سطر&#10;مثال:&#10;100012345678901&#10;100098765432109"></textarea>
        <small style="display:block;margin-top:5px;color:var(--text-secondary);font-size:12px;font-family:'Cairo',sans-serif;">
          يمكنك إدخال عدة معرفات، واحد في كل سطر
        </small>
      </div>

      <button type="submit" class="submit-btn"><i class="fas fa-search"></i>بدء البحث</button>
    </form>
  </div>
</div>

<div id="resultsModal" class="modal">
  <div class="modal-content" style="max-width:1200px;">
    <div class="modal-header">
      <div class="modal-title"><i class="fas fa-table"></i>نتائج الحملة</div>
      <span class="close-modal" onclick="closeModal('resultsModal')">&times;</span>
    </div>

    <div style="display:flex;gap:10px;flex-wrap:wrap;margin-bottom:12px;">
      <input id="resultsSearch" class="form-input" placeholder="بحث داخل النتائج..." style="flex:1;min-width:240px;">
      <button class="action-btn btn-download" onclick="downloadCurrentResults()">
        <i class="fas fa-download"></i> تنزيل CSV
      </button>
    </div>

    <div style="overflow:auto;border:1px solid var(--border-color);border-radius:12px;">
      <table style="width:100%;border-collapse:collapse;font-family:'Cairo',sans-serif;">
        <thead id="resultsThead"></thead>
        <tbody id="resultsTbody"></tbody>
      </table>
    </div>

    <div style="display:flex;justify-content:space-between;align-items:center;margin-top:12px;">
      <div id="resultsCount" style="color:var(--text-secondary)"></div>
      <div style="display:flex;gap:8px;">
        <button class="action-btn btn-pause" onclick="prevPage()">السابق</button>
        <button class="action-btn btn-pause" onclick="nextPage()">التالي</button>
      </div>
    </div>
  </div>
</div>

<script>
let campaigns = [];
let tags = {
  work: [], education: [], location: [], city: [],
  workKeywords: [], educationKeywords: [], locationKeywords: [], cityKeywords: []
};

let currentCampaignId = null;
let currentPage = 1;
let pageSize = 5000;
let currentFiltered = [];

const tableHeaders = ['id','fb_id','name','mobile_phone','gender','birthday','location','relationship','email','work','education'];

/* =========================
   Points UI
========================= */
let userPointsValue = null;

async function refreshPoints() {
  try {
    const res = await fetch('/api/points.php', { method:'GET', credentials:'same-origin' });
    const json = await res.json().catch(()=> ({}));

    if (json && json.success) {
      userPointsValue = Number(json.points || 0);
      document.getElementById('userPoints').innerText = userPointsValue.toLocaleString();
    } else {
      document.getElementById('userPoints').innerText = '--';
      console.log('POINTS ERROR:', json);
    }
  } catch(e) {
    document.getElementById('userPoints').innerText = '--';
    console.log('POINTS EXCEPTION:', e);
  }
}

/* =========================
   Modals
========================= */
function openMethodModal() {
  const m = document.getElementById('methodModal');
  if (m) m.style.display = 'block';
}
function closeModal(modalId) {
  const m = document.getElementById(modalId);
  if (m) m.style.display = 'none';
}
function selectMethod(method) {
  closeModal('methodModal');
  if (method === 'uids') document.getElementById('uidModal').style.display = 'block';
  if (method === 'keywords') document.getElementById('keywordsModal').style.display = 'block';
}

/* =========================
   Tags
========================= */
function addTag(event, type) {
  if (event.key === 'Enter') {
    event.preventDefault();
    const input = event.target;
    const value = (input.value || '').trim();
    if (value && !tags[type].includes(value)) {
      tags[type].push(value);
      renderTags(type);
      input.value = '';
    }
  }
}
function renderTags(type) {
  const container = document.getElementById(type + 'Tags');
  if (!container) return;
  container.innerHTML = tags[type].map((tag, index) => `
    <div class="tag">
      <span>${escapeHtml(tag)}</span>
      <i class="fas fa-times tag-remove" onclick="removeTag('${type}', ${index})"></i>
    </div>
  `).join('');
}
function removeTag(type, index) {
  tags[type].splice(index, 1);
  renderTags(type);
}

/* =========================
   Helpers
========================= */
function escapeHtml(str) {
  return String(str)
    .replaceAll('&','&amp;')
    .replaceAll('<','&lt;')
    .replaceAll('>','&gt;')
    .replaceAll('"','&quot;')
    .replaceAll("'","&#039;");
}
function toastLoading(title='جاري...') {
  if (window.Swal) Swal.fire({ title, allowOutsideClick:false, didOpen:()=>Swal.showLoading() });
  else console.log(title);
}
function toastError(msg) {
  if (window.Swal) Swal.fire({ icon:'error', title:'خطأ', text: msg });
  else alert(msg);
}
function toastSuccess(msg) {
  if (window.Swal) Swal.fire({ icon:'success', title:'تم!', text: msg, confirmButtonText:'حسناً' });
  else alert(msg);
}

/* =========================
   API
========================= */
function buildPayload(type='uids') {
  if (type === 'uids') {
    const uidsText = document.getElementById('uidsInput').value || '';
    const uids = uidsText.split('\n').map(x=>x.trim()).filter(Boolean);
    const country = document.getElementById('country')?.value || 'all';
    const uidType = document.getElementById('uidType')?.value || 'fb_id';
    
    // سحب العدد المطلوب
    const limit = parseInt(document.getElementById('limitUids').value) || 1000;

    return {
      type: 'uids',
      country,
      uidType,
      work: tags.work,
      education: tags.education,
      location: tags.location,
      city: tags.city,
      maritalStatus: document.getElementById('maritalStatus').value || '',
      uids,
      limit: limit
    };
  }

  const country = document.getElementById('countryKeywords')?.value || 'all';
  // سحب العدد المطلوب
  const limit = parseInt(document.getElementById('limitKeywords').value) || 1000;

  return {
    type: 'keywords',
    country,
    work: tags.workKeywords,
    education: tags.educationKeywords,
    location: tags.locationKeywords,
    city: tags.cityKeywords,
    maritalStatus: document.getElementById('maritalStatusKeywords').value || '',
    limit: limit
  };
}

async function callSearchAPI(payload, action = 'count') {
  payload.action = action;
  const res = await fetch('/api/data_fb_search.php', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    credentials: 'same-origin',
    body: JSON.stringify(payload)
  });
  const json = await res.json().catch(()=> ({}));
  if (!res.ok) throw new Error(json.message || ('HTTP ' + res.status));
  if (!json.success) throw new Error(json.message || 'فشلت العملية');
  return json;
}

/* =========================
   دالة موحدة لمعالجة الاستخراج والخصم بشكل آمن
========================= */
async function processCampaign(campaignName, country, type, payload, modalId) {
  // 1. طلب العد فقط
  toastLoading('جاري حساب عدد النتائج المتاحة...');
  const countResult = await callSearchAPI(payload, 'count');
  const toCharge = Number(countResult.count || 0);

  if (window.Swal) Swal.close();

  if (toCharge > 0) {
    // 2. سؤال المستخدم
    const r = await Swal.fire({
      icon: 'warning',
      title: 'تأكيد الخصم',
      html: `تم العثور على <b>${toCharge.toLocaleString()}</b> نتيجة متاحة وفقاً للعدد المطلوب.<br>سيتم خصم <b>${toCharge.toLocaleString()}</b> نقطة لاستخراجهم.`,
      showCancelButton: true,
      confirmButtonText: 'موافق واخصم',
      cancelButtonText: 'إلغاء'
    });
    
    if (!r.isConfirmed) return;

    // 3. طلب الاستخراج والخصم من السيرفر
    toastLoading('جاري الاستخراج وخصم النقاط...');
    const extractResult = await callSearchAPI(payload, 'extract');

    await refreshPoints();
    if (window.Swal) Swal.close();

    // 4. إضافة البيانات للجدول
    campaigns.push({
      id: Date.now(),
      name: campaignName,
      count: toCharge,
      status: 'finished',
      country,
      type: type,
      filters: payload,
      data: extractResult.data || []
    });

  } else {
    toastSuccess('لا توجد نتائج — لن يتم خصم نقاط');
    campaigns.push({
      id: Date.now(),
      name: campaignName,
      count: 0,
      status: 'finished',
      country,
      type: type,
      filters: payload,
      data: []
    });
  }

  renderCampaigns();
  closeModal(modalId);

  // تصفير الحقول
  if (type === 'uids') {
    document.getElementById('campaignName').value = '';
    document.getElementById('uidsInput').value = '';
    tags.work=[]; tags.education=[]; tags.location=[]; tags.city=[];
    renderTags('work'); renderTags('education'); renderTags('location'); renderTags('city');
  } else {
    document.getElementById('campaignNameKeywords').value = '';
    tags.workKeywords=[]; tags.educationKeywords=[]; tags.locationKeywords=[]; tags.cityKeywords=[];
    renderTags('workKeywords'); renderTags('educationKeywords'); renderTags('locationKeywords'); renderTags('cityKeywords');
  }

  if (toCharge > 0) {
    toastSuccess(`تم استخراج ${toCharge.toLocaleString()} نتيجة بنجاح ✅`);
  }
}

/* =========================
   Submit المحدثة
========================= */
async function submitCampaign(type='uids') {
  try {
    let campaignName = '';
    let country = '';

    if (type === 'uids') {
      campaignName = (document.getElementById('campaignName').value || '').trim();
      country = document.getElementById('country').value || '';
      if (!campaignName) return toastError('يرجى إدخال اسم الحملة');

      const payload = buildPayload('uids');
      if (!payload.uids.length) return toastError('يرجى إدخال معرف واحد على الأقل');

      await processCampaign(campaignName, country, type, payload, 'uidModal');

    } else {
      campaignName = (document.getElementById('campaignNameKeywords').value || '').trim();
      country = document.getElementById('countryKeywords').value || '';
      if (!campaignName) return toastError('يرجى إدخال اسم الحملة');

      const payload = buildPayload('keywords');
      await processCampaign(campaignName, country, type, payload, 'keywordsModal');
    }

  } catch (e) {
    if (window.Swal) Swal.close();
    toastError(e.message || 'حدث خطأ غير متوقع');
  }
}

/* =========================
   Campaigns UI
========================= */
function getStatusLabel(status) {
  const labels = { stopped:'إيقاف', running:'جاري', paused:'إيقاف مؤقت', finished:'انتهاء' };
  return labels[status] || status;
}
function renderCampaigns() {
  const grid = document.getElementById('campaignsGrid');
  if (!grid) return;

  if (!campaigns.length) {
    grid.innerHTML = `
      <div style="grid-column: 1/-1; text-align:center; padding:60px 20px;">
        <i class="fas fa-folder-open" style="font-size:80px; color:#667eea; margin-bottom:20px; opacity:.5;"></i>
        <h3 style="color:var(--text-primary); font-family:'Cairo',sans-serif;">لا توجد حملات</h3>
        <p style="color:var(--text-secondary); font-family:'Cairo',sans-serif;">ابدأ بإنشاء حملة استخراج جديدة</p>
      </div>
    `;
    return;
  }

  grid.innerHTML = campaigns.map(c => `
    <div class="campaign-card">
      <div class="campaign-header">
        <div class="campaign-name">${escapeHtml(c.name)}</div>
        <span class="campaign-status status-${c.status}">${getStatusLabel(c.status)}</span>
      </div>
      <div class="campaign-count">${Number(c.count || 0).toLocaleString()}</div>
      <div class="campaign-actions">
        <button class="action-btn btn-delete" onclick="deleteCampaign(${c.id})"><i class="fas fa-trash"></i> حذف</button>
        <button class="action-btn btn-pause" onclick="toggleStatus(${c.id}, 'paused')"><i class="fas fa-pause"></i> إيقاف مؤقت</button>
        <button class="action-btn btn-stop" onclick="toggleStatus(${c.id}, 'stopped')"><i class="fas fa-stop"></i> إيقاف</button>
        <button class="action-btn btn-download" onclick="openResults(${c.id})"><i class="fas fa-table"></i> عرض</button>
      </div>
    </div>
  `).join('');
}
function deleteCampaign(id) {
  if (window.Swal) {
    Swal.fire({
      title:'تأكيد الحذف',
      text:'هل أنت متأكد من حذف هذه الحملة؟',
      icon:'warning',
      showCancelButton:true,
      confirmButtonText:'نعم، احذف',
      cancelButtonText:'إلغاء',
      confirmButtonColor:'#ef4444'
    }).then((r)=>{
      if (!r.isConfirmed) return;
      campaigns = campaigns.filter(x=>x.id!==id);
      renderCampaigns();
    });
  } else {
    if (confirm('حذف الحملة؟')) {
      campaigns = campaigns.filter(x=>x.id!==id);
      renderCampaigns();
    }
  }
}
function toggleStatus(id, newStatus) {
  const c = campaigns.find(x=>x.id===id);
  if (!c) return;
  c.status = newStatus;
  renderCampaigns();
}

/* =========================
   Results Modal
========================= */
function openResults(campaignId) {
  currentCampaignId = campaignId;
  currentPage = 1;

  const c = campaigns.find(x=>x.id===campaignId);
  if (!c) return;

  document.getElementById('resultsThead').innerHTML = `
    <tr>
      ${tableHeaders.map(h=>`<th style="text-align:left;padding:10px;border-bottom:1px solid var(--border-color);position:sticky;top:0;background:var(--card-bg);">${h}</th>`).join('')}
    </tr>
  `;

  currentFiltered = Array.isArray(c.data) ? c.data : [];
  const search = document.getElementById('resultsSearch');
  search.value = '';
  search.oninput = () => applySearch();

  renderResultsTable();
  document.getElementById('resultsModal').style.display = 'block';
}
function applySearch() {
  const c = campaigns.find(x=>x.id===currentCampaignId);
  if (!c) return;

  const q = (document.getElementById('resultsSearch').value || '').trim().toLowerCase();
  const rows = Array.isArray(c.data) ? c.data : [];

  if (!q) currentFiltered = rows;
  else currentFiltered = rows.filter(r => tableHeaders.some(h => String(r[h] ?? '').toLowerCase().includes(q)));

  currentPage = 1;
  renderResultsTable();
}
function renderResultsTable() {
  const tbody = document.getElementById('resultsTbody');
  const total = currentFiltered.length;

  const start = (currentPage - 1) * pageSize;
  const end = Math.min(start + pageSize, total);
  const slice = currentFiltered.slice(start, end);

  tbody.innerHTML = slice.map(r => `
    <tr>
      ${tableHeaders.map(h=>`<td style="padding:10px;border-bottom:1px solid var(--border-color);white-space:nowrap;">${escapeHtml(r[h] ?? '')}</td>`).join('')}
    </tr>
  `).join('');

  document.getElementById('resultsCount').innerText =
    total ? `عرض ${start+1}-${end} من ${total} نتيجة (صفحة ${currentPage})` : `لا توجد نتائج`;
}
function nextPage() {
  const totalPages = Math.ceil(currentFiltered.length / pageSize) || 1;
  if (currentPage < totalPages) { currentPage++; renderResultsTable(); }
}
function prevPage() {
  if (currentPage > 1) { currentPage--; renderResultsTable(); }
}

/* =========================
   Download CSV
========================= */
async function downloadCurrentResults() {
  if (!currentCampaignId) return;
  const c = campaigns.find(x=>x.id===currentCampaignId);
  if (!c) return;

  const rows = currentFiltered;
  if (!rows.length) return toastError('لا توجد بيانات لتنزيلها');

  const BOM = '\uFEFF';
  const csvLines = [
    tableHeaders.join(','),
    ...rows.map(r => tableHeaders.map(h => `"${String(r[h] ?? '').replaceAll('"','""')}"`).join(','))
  ];
  const csv = BOM + csvLines.join('\n');

  const blob = new Blob([csv], {type:'text/csv;charset=utf-8;'});
  const url = URL.createObjectURL(blob);

  const a = document.createElement('a');
  a.href = url;
  a.download = (c.name || 'data') + '_filtered.csv';
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);

  toastSuccess('تم تحميل الملف ✅');
}

/* close on outside click */
window.onclick = function(event) {
  if (event.target && event.target.classList && event.target.classList.contains('modal')) {
    event.target.style.display = 'none';
  }
};

/* init */
renderCampaigns();
refreshPoints();
</script>

<?php include 'includes/footer.php'; ?>
