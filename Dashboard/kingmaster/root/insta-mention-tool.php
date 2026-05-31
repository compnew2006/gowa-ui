


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



$page_title = "اداة المنشن | Kingmaster";
$page_css = ['https://kingmaster.info/css/f-w-i.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>


<div class="search-container">
  <div class="search-bar">
    <input id="q" type="text" class="search-input" placeholder="ابحث بالبروفايل" />
    <button id="searchBtn" class="search-btn"><i class="fas fa-search"></i> بحث</button>
  </div>
  <div class="toolbar">
    <div>
      <label class="select-all" for="selectAll" style="cursor:pointer;">
        <input type="checkbox" id="selectAll" />
        <i class="fas fa-check-double fa-beat" style="--fa-animation-duration: 2.2s;"></i>
        <span>تحديد الجميع</span>
      </label>
      <span class="sel-count-badge" id="selCount">—</span>
    </div>
    <div>
      <button id="exportBtn" class="export-btn" disabled><i class="fas fa-file-csv"></i> تصدير CSV</button>
    </div>
  </div>
  <div id="results" class="results-grid">
    <div class="muted" style="grid-column:1/-1; text-align:center; padding:24px;">اكتب كلمة بحث لعرض النتائج</div>
  </div>
</div>

<script>
let resultsData = [];
let selectedIds = new Set();

function renderResults(items){
  const grid = document.getElementById('results');
  if (!items.length){
    grid.innerHTML = '<div class="muted" style="grid-column:1/-1; text-align:center; padding:24px;">لا توجد نتائج</div>';
    return;
  }
  grid.innerHTML = items.map(row=>{
    const img = row.GroupImage || row.groupImage || '';
    const title = row.groupName || '';
    const desc = (row.groupDesc || '').toString();
    const cat = row.categoryName || '-';
    const country = row.country || '-';
    const link = row.groupLink || '#';
    const checked = selectedIds.has(String(row.id)) ? 'checked' : '';
    return `
      <div class="card">
        <img class="card-img" src="${img ? encodeURI(img) : 'https://via.placeholder.com/640x360?text=Group'}" alt="group" onerror="this.src='https://via.placeholder.com/640x360?text=Group'"/>
        <div class="card-body">
          <div class="card-title">${escapeHtml(title)}
            <label class="select-cell">
              <input type="checkbox" data-id="${row.id}" onchange="toggleSelect(this)" ${checked}/> تحديد
            </label>
          </div>
          <div class="card-desc">${escapeHtml(desc)}</div>
          <div class="badge-row">
            <span class="badge"><i class="fas fa-tag fa-beat" style="--fa-animation-duration: 2.5s;"></i> ${escapeHtml(cat)}</span>
            <span class="badge"><i class="fas fa-globe fa-spin" style="--fa-animation-duration: 6s;"></i> ${escapeHtml(country)}</span>
          </div>
          <div class="card-actions">
            <a class="join-btn" href="${encodeURI(link)}" target="_blank" rel="noopener"><i class="fas fa-user-plus fa-beat-fade" style="--fa-animation-duration: 2.2s;"></i> انضمام</a>
          </div>
        </div>
      </div>`;
  }).join('');
}

function updateSelUI(){
  const count = selectedIds.size;
  document.getElementById('selCount').textContent = count ? `المحدد: ${count}` : '—';
  document.getElementById('exportBtn').disabled = count===0;
  const allShownIds = resultsData.map(r=>String(r.id));
  const allSelected = allShownIds.length>0 && allShownIds.every(id=>selectedIds.has(id));
  document.getElementById('selectAll').checked = allSelected;
}

function toggleSelect(chk){
  const id = String(chk.getAttribute('data-id'));
  if (!id) return;
  if (chk.checked) selectedIds.add(id); else selectedIds.delete(id);
  updateSelUI();
}

function toggleSelectAll(){
  const all = document.getElementById('selectAll').checked;
  resultsData.forEach(r=>{
    const id = String(r.id);
    if (all) selectedIds.add(id); else selectedIds.delete(id);
  });
  renderResults(resultsData);
  updateSelUI();
}

async function search(){
  const q = document.getElementById('q').value.trim();
  const grid = document.getElementById('results');
  grid.innerHTML = '<div class="muted" style="grid-column:1/-1; text-align:center; padding:24px;">جاري البحث...</div>';
  try {
    const res = await fetch('api/search_groups.php?q=' + encodeURIComponent(q), { credentials: 'same-origin' });
    const j = await res.json();
    if (!j.success) throw new Error(j.message||'تعذر إتمام البحث');
    resultsData = j.results || [];
    // حافظ على التحديد السابق إن كان ضمن النتائج
    renderResults(resultsData);
    updateSelUI();
  } catch(e){
    grid.innerHTML = `<div class="muted" style="grid-column:1/-1; text-align:center; padding:24px; color:#ef4444;">خطأ: ${e.message}</div>`;
  }
}

async function exportCSV(){
  const ids = Array.from(selectedIds);
  if (!ids.length) return;
  try {
    const res = await fetch('api/export_groups_csv.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids })
    });
    if (!res.ok) throw new Error('تعذر إنشاء الملف');
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'groups_export_' + new Date().toISOString().slice(0,19).replace(/[:T]/g,'-') + '.csv';
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  } catch(e) {
    alert('خطأ في التصدير: ' + e.message);
  }
}

document.getElementById('searchBtn').addEventListener('click', search);

document.getElementById('q').addEventListener('keydown', (e)=>{ if (e.key==='Enter') { e.preventDefault(); search(); } });

document.getElementById('selectAll').addEventListener('change', toggleSelectAll);

document.getElementById('exportBtn').addEventListener('click', exportCSV);

// Helpers used elsewhere in project
function escapeHtml(s){
  return String(s||'')
    .replace(/&/g,'&amp;')
    .replace(/</g,'&lt;')
    .replace(/>/g,'&gt;')
    .replace(/"/g,'&quot;')
    .replace(/'/g,'&#039;');
}
</script>


<?php include 'includes/footer.php'; ?>
