
<?php

session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];
$user = getUserByUserId($user_id);

$page_title = 'Visual Flow Builder | Kingmaster';
include 'includes/head.php';
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


include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
 
?>

<style>
  .fb-container { padding:30px; max-width:100%; margin:120px auto 0 auto; }
  .fb-header { display:flex; gap:12px; align-items:center; flex-wrap:wrap; }
  .form-select, .form-input, .form-textarea { width:100%; padding:12px 14px; border:2px solid var(--border-color); border-radius:10px; background:#1f2937; color:#e5e7eb; font-weight:700; }
  .form-textarea { min-height:88px; resize:vertical; }
.fb-grid { display:grid; grid-template-columns: 3fr 1fr; gap:16px; margin-top:16px; }
  .fb-card { background:var(--card-bg); border:2px solid var(--border-color); border-radius:14px; padding:14px; display:flex; flex-direction:column; gap:10px; }
  .fb-card h4 { margin:0; color:#fff; font-weight:900; display:flex; align-items:center; gap:8px; }
  .btn { border:0; padding:10px 12px; border-radius:10px; font-weight:800; cursor:pointer; color:#fff; }
  .btn-primary { background:linear-gradient(135deg,#667eea,#764ba2); }
  .btn-green { background:linear-gradient(135deg,#10b981,#059669); }
  .btn-red { background:linear-gradient(135deg,#ef4444,#dc2626); }
  .btn-gray { background:linear-gradient(135deg,#6b7280,#4b5563); }
  .thin-muted { color:var(--text-secondary); font-size:12px; }
  .list { background:var(--card-bg); border:2px solid var(--border-color); border-radius:12px; padding:12px; margin-top:16px; }
  .flow-list-item { display:flex; align-items:center; justify-content:space-between; border-bottom:1px solid rgba(255,255,255,.06); padding:10px 0; }
  .flow-list-item:last-child { border-bottom:0; }
  /* Board */
.board-wrap { position:relative; height:calc(100vh - 200px); min-height:760px; border:2px dashed rgba(255,255,255,.12); border-radius:12px; background: repeating-linear-gradient(45deg, rgba(255,255,255,.02) 0 10px, transparent 10px 20px); overflow:auto; }
#board-inner { position:relative; width:100%; height:100%; transform-origin:0 0; }
#fbEdges { position:absolute; inset:0; width:100%; height:100%; pointer-events:none; z-index:2; }
  #fbNodes { position:absolute; inset:0; z-index:3; }
  .fb-node { position:absolute; min-width:240px; max-width:420px; background:#111827; color:#e5e7eb; border:2px solid var(--border-color); border-radius:12px; box-shadow:0 12px 30px rgba(0,0,0,.35); }
  .fb-node .head { display:flex; align-items:center; justify-content:space-between; gap:8px; padding:10px; background:linear-gradient(135deg,#1f2937,#0f172a); border-bottom:1px solid rgba(255,255,255,.06); cursor:move; border-top-left-radius:10px; border-top-right-radius:10px; }
  .fb-node .title { font-weight:900; display:flex; align-items:center; gap:8px; }
  .fb-node .body { padding:10px; font-size:12px; color:#a1a1aa; }
.badge-type { display:inline-flex; align-items:center; gap:6px; padding:3px 8px; border-radius:999px; background:rgba(124,58,237,.12); color:#c4b5fd; border:1px solid rgba(124,58,237,.3); font-size:11px; }
  .match-pill { display:inline-flex; align-items:center; gap:6px; padding:4px 10px; border-radius:999px; background:linear-gradient(135deg, rgba(14,165,233,.2), rgba(99,102,241,.2)); color:#e0f2fe; border:1px solid rgba(59,130,246,.35); font-size:11px; box-shadow: 0 1px 8px rgba(59,130,246,.18) inset; }
  .match-pill.equal { background:linear-gradient(135deg, rgba(16,185,129,.22), rgba(99,102,241,.18)); color:#d1fae5; border-color: rgba(16,185,129,.35); }
  .handle { width:12px; height:12px; border-radius:50%; background:linear-gradient(135deg,#667eea,#7c3aed); border:2px solid #1f2937; box-shadow:0 0 0 2px rgba(102,126,234,.35); cursor:crosshair; }
.handles { position:absolute; right:-8px; top:50%; transform:translateY(-50%); display:flex; flex-direction:column; gap:6px; }
  .handles-left { right:auto; left:-8px; }
  .edit-btn { background:transparent; border:0; color:#9ca3af; cursor:pointer; }
  .edit-btn:hover { color:#fff; }
  .connecting .handle { animation: pulse 1s ease-in-out infinite; }
  @keyframes pulse { 0%,100%{ box-shadow:0 0 0 2px rgba(102,126,234,.35);} 50%{ box-shadow:0 0 0 6px rgba(102,126,234,.15);} }
  /* Palette & sessions */
  .palette { display:grid; gap:8px; }
  .palette-item { display:flex; align-items:center; gap:8px; padding:10px; border:2px dashed rgba(255,255,255,.15); border-radius:12px; background:rgba(255,255,255,.03); cursor:grab; }
  .palette-item:hover { background:rgba(255,255,255,.06); }
  .sessions { display:grid; gap:8px; max-height:220px; overflow:auto; border:2px solid var(--border-color); border-radius:12px; padding:8px; }
  .session-card { display:flex; align-items:center; justify-content:space-between; padding:8px 10px; border-radius:10px; background:rgba(255,255,255,.04); border:2px solid transparent; transition:.15s; }
  .session-card:hover { background:rgba(255,255,255,.06); }
  .session-card.selected { background:rgba(16,185,129,.12); border-color:rgba(16,185,129,.35); box-shadow:0 2px 10px rgba(16,185,129,.2) inset; }
  .session-card .sel-badge { display:inline-flex; align-items:center; gap:6px; padding:3px 8px; border-radius:999px; background:rgba(16,185,129,.2); color:#bbf7d0; border:1px solid rgba(16,185,129,.35); font-size:12px; }
  .session-card button { padding:6px 10px; border:0; border-radius:8px; background:linear-gradient(135deg,#3b82f6,#2563eb); color:#fff; font-weight:800; cursor:pointer; }
  /* Modal */
  .modal { display:none; position:fixed; z-index:10000; inset:0; background:rgba(0,0,0,.75); backdrop-filter: blur(5px); }
  .modal .content { background: var(--card-bg); border:2px solid var(--border-color); border-radius:16px; width:90%; max-width:720px; margin:5% auto; padding:16px; display:flex; flex-direction:column; max-height:80vh; overflow:hidden; }
  .modal .modal-body { flex:1 1 auto; overflow:auto; margin-top:8px; }
  .modal .modal-footer { display:flex; justify-content:flex-end; gap:8px; padding-top:10px; margin-top:10px; border-top:1px solid rgba(255,255,255,.08); }
  .form-select, .form-input, .form-textarea { width:100%; padding:12px 14px; border:2px solid var(--border-color); border-radius:10px; background:#1f2937; color:#e5e7eb; font-weight:700; }
  .form-textarea { min-height:100px; resize:vertical; }
.chips { display:flex; flex-wrap:wrap; gap:6px; margin-top:6px; }
  .chip { display:inline-flex; align-items:center; gap:6px; padding:6px 12px; border-radius:999px; background:linear-gradient(135deg, rgba(124,58,237,.22), rgba(99,102,241,.22)); border:1px solid rgba(124,58,237,.35); color:#e9d5ff; font-size:12px; box-shadow: 0 1px 8px rgba(99,102,241,.18) inset; }
  .chip.more { background:linear-gradient(135deg, rgba(59,130,246,.2), rgba(99,102,241,.18)); border-color:rgba(59,130,246,.35); color:#bfdbfe; }
  .chip .btn-x { width:16px; height:16px; display:inline-flex; align-items:center; justify-content:center; border-radius:999px; background:linear-gradient(135deg, #ef4444, #dc2626); color:#fff; font-size:10px; cursor:pointer; box-shadow: 0 1px 6px rgba(239,68,68,.35); }
  .chip .btn-x:hover { filter: brightness(1.08); }
.segmented { display:inline-flex; background: rgba(17,24,39,0.6); border:2px solid rgba(99,102,241,.25); border-radius:999px; overflow:hidden; box-shadow: 0 6px 18px rgba(0,0,0,.25) inset, 0 2px 10px rgba(99,102,241,.15); backdrop-filter: saturate(140%) blur(2px); }
  .segmented button { appearance:none; background:transparent; color:#cbd5e1; border:0; padding:8px 16px; cursor:pointer; font-weight:900; display:inline-flex; align-items:center; gap:6px; transition:.15s; }
  .segmented button i { opacity:.9; }
  .segmented button:hover { background: rgba(99,102,241,.10); color:#e5e7eb; }
  .segmented button.selected { background: linear-gradient(135deg, #22d3ee, #6366f1); color:#fff; box-shadow: 0 0 0 2px rgba(99,102,241,.25) inset, 0 4px 14px rgba(99,102,241,.35); }
  /* Image node */
  .img-thumb { max-width:100%; max-height:240px; border-radius:10px; border:1px solid rgba(255,255,255,.08); display:block; }
  .img-uploader { display:flex; flex-direction:column; gap:8px; align-items:flex-start; }
  .img-uploader .btn { padding:8px 10px; }
</style>

<div class="fb-container">
  <div class="fb-header" style="justify-content:flex-end;">
    <div style="display:flex; gap:8px; align-items:center;">
      <button class="btn btn-primary" onclick="saveFlow()"><i class="fas fa-save fa-beat"></i> حفظ</button>
      <button class="btn btn-gray" onclick="resetBuilder()"><i class="fas fa-rotate-left"></i> إعادة تعيين</button>
    </div>
  </div>

  <div class="fb-grid">
    <div class="fb-card">
      <div style="display:flex; justify-content:flex-end; gap:6px; margin-bottom:6px; align-items:center;">
        <button class="btn btn-gray" style="padding:4px 8px; font-size:11px;" onclick="setBoardZoom(boardZoom - 0.1)"><i class="fas fa-search-minus"></i></button>
        <span class="thin-muted" id="boardZoomLabel" style="min-width:42px; text-align:center;">100%</span>
        <button class="btn btn-gray" style="padding:4px 8px; font-size:11px;" onclick="setBoardZoom(boardZoom + 0.1)"><i class="fas fa-search-plus"></i></button>
      </div>
      <div class="board-wrap" ondragover="event.preventDefault()" ondrop="onBoardDrop(event)">
        <div id="board-inner">
          <svg id="fbEdges"></svg>
          <div id="fbNodes"></div>
        </div>
      </div>
    </div>
    <div class="fb-card">
      <h4><i class="fas fa-sitemap fa-bounce"></i> Workflow Builder</h4>
      <div class="thin-muted">جلساتك</div>
      <div id="sessionsList" class="sessions">—</div>
      <div class="thin-muted" style="margin-top:10px;">لوحة الأدوات</div>
      <div class="palette">
        <div class="palette-item" draggable="true" ondragstart="onPaletteDrag(event,'keywords')"><i class="fas fa-key fa-beat"></i> كلمات مفتاحية</div>
        <div class="palette-item" draggable="true" ondragstart="onPaletteDrag(event,'text')"><i class="fas fa-align-left"></i> رسالة نصية</div>
        <div class="palette-item" draggable="true" ondragstart="onPaletteDrag(event,'image')"><i class="fas fa-image"></i> صورة</div>
        <div class="palette-item" draggable="true" ondragstart="onPaletteDrag(event,'poll')"><i class="fas fa-poll-h"></i> استطلاع رأي</div>
        <div class="palette-item" draggable="true" ondragstart="onPaletteDrag(event,'location')"><i class="fas fa-map-marker-alt"></i> موقع</div>
        <div class="palette-item" draggable="true" ondragstart="onPaletteDrag(event,'file')"><i class="fas fa-file-pdf"></i> ملف PDF</div>
        <div class="palette-item" draggable="true" ondragstart="onPaletteDrag(event,'list')"><i class="fas fa-list"></i> قائمة</div>
      </div>

      <!-- كارت إدخال عنوان / موقع داخل لوحة الأدوات -->
     

      <div class="thin-muted" style="margin-top:10px;">التدفقات</div>
      <div id="flowsList" class="list">—</div>
    </div>
  </div>
</div>

<!-- Node Editor Modal -->
<div id="nodeModal" class="modal">
  <div class="content">
    <div style="display:flex; align-items:center; justify-content:space-between;">
      <h3 style="margin:0; color:#fff; font-weight:900;">تحرير العقدة</h3>
      <button class="edit-btn" onclick="closeNodeEditor()"><i class="fas fa-times"></i></button>
    </div>
    <div class="modal-body">
      <div class="thin-muted" style="margin-bottom:8px;">اضبط بيانات العقدة — الكلمات المفتاحية أو النص أو الاستطلاع حسب النوع.</div>
      <div class="row" style="gap:16px; align-items:end;">
        <div>
          <label class="thin-muted">عنوان العقدة</label>
          <input id="ne-title" class="form-input" />
        </div>
        <div id="ne-match-wrap">
          <label class="thin-muted">نوع المطابقة</label>
          <div class="segmented" id="neMatchSeg">
            <button type="button" data-v="contains" class="selected"><i class="fas fa-asterisk"></i> تحتوي</button>
            <button type="button" data-v="equals"><i class="fas fa-equals"></i> تساوي</button>
          </div>
          <input type="hidden" id="ne-match" value="contains" />
        </div>
      </div>
      <div id="ne-keywords-wrap">
        <label class="thin-muted">الكلمات المفتاحية</label>
        <div id="ne-keywords-chips" class="chips"></div>
        <input id="ne-keyword-input" class="form-input" placeholder="اكتب ثم اضغط Enter أو فاصلة" />
      </div>
      <div id="ne-reply-wrap">
        <label class="thin-muted">نص الرسالة</label>
        <textarea id="ne-reply" class="form-textarea" placeholder="اكتب نص الرسالة..."></textarea>
        <div id="ne-join-wrap" style="margin-top:10px;">
          <label class="thin-muted">تشغيل بعد</label>
          <div class="segmented" id="neJoinSeg">
            <button type="button" data-j="any" class="selected"><i class="fas fa-share"></i> أي مدخل</button>
            <button type="button" data-j="all"><i class="fas fa-layer-group"></i> كل المداخل</button>
          </div>
          <input type="hidden" id="ne-join" value="any" />
        </div>
      </div>
      <div id="ne-poll-wrap" style="display:none; margin-top:16px;">
        <label class="thin-muted">سؤال الاستطلاع</label>
        <textarea id="ne-poll-question" class="form-textarea" placeholder="اكتب نص السؤال..."></textarea>
        <label class="thin-muted" style="margin-top:8px; display:block;">الاختيارات</label>
        <div id="ne-poll-options"></div>
      </div>
      <div id="ne-location-wrap" style="display:none; margin-top:16px;">
        <label class="thin-muted">إحداثيات الموقع</label>
        <div class="row" style="display:flex; gap:8px; margin-bottom:8px;">
          <input id="ne-lat" class="form-input" placeholder="lat" />
          <input id="ne-lng" class="form-input" placeholder="lng" />
        </div>
        <label class="thin-muted">عنوان الموقع</label>
        <input id="ne-loc-title" class="form-input" placeholder="title" />
        <label class="thin-muted" style="margin-top:8px; display:block;">الوصف / العنوان التفصيلي</label>
        <textarea id="ne-address" class="form-textarea" placeholder="address"></textarea>
      </div>
      <div id="ne-list-wrap" style="display:none; margin-top:16px;">
        <label class="thin-muted">زر القائمة</label>
        <input id="ne-list-button" class="form-input" placeholder="نص الزر (مثال: افتح القائمة)" />
        <label class="thin-muted" style="margin-top:8px; display:block;">الأقسام والعناصر</label>
        <div id="ne-list-sections"></div>
        <button type="button" class="btn btn-green" style="margin-top:8px;" onclick="addListSection()"><i class="fas fa-plus"></i> إضافة قسم</button>
      </div>
    </div>
    <div class="modal-footer">
      <button class="btn btn-primary" onclick="saveNodeEditor()"><i class="fas fa-save"></i> حفظ</button>
    </div>
  </div>
</div>
<script src="https://cdn.jsdelivr.net/npm/sweetalert2@11"></script>
<script>
let flowState = { id:null, account_uid:'', name:'', nodes: [] };
let connectingFrom = null;
let connectCursor = { x: null, y: null };
let dragging = { id:null, dx:0, dy:0 };
let boardZoom = 1;
let editingNodeId = null;
let neKeywords = [];
let nePollOptions = [];
let neListConfig = null;
let allAccountsCache = [];
let selectedAccountUid = '';

function loadAccounts(){
  fetch('api/get_accounts_wa.php', { credentials:'same-origin' })
    .then(r=>r.json()).then(j=>{
      const sel = document.getElementById('accountSelect');
      if (sel) sel.innerHTML = '<option value="">— اختر الحساب —</option>';
      if(j && j.success && Array.isArray(j.accounts)){
        allAccountsCache = j.accounts;
        if (sel) {
          j.accounts.forEach(a=>{
            const o = document.createElement('option'); o.value = a.account_uid; o.textContent = a.name; sel.appendChild(o);
          });
        }
        renderSessions();
      }
    });
}
function loadFlows(){
  fetch('api/flows.php?action=list', { credentials:'same-origin' })
    .then(r=>r.json()).then(j=>{
      const box = document.getElementById('flowsList');
      if (!j.success || !Array.isArray(j.flows) || !j.flows.length){ box.innerHTML='لا يوجد تدفقات بعد'; return; }
      box.innerHTML = j.flows.map(f=>`
        <div class="flow-list-item">
          <div>
            <div style="font-weight:800; color:#fff;">${escapeHtml(f.flow_name)}</div>
            <div class="thin-muted">${escapeHtml(f.account_uid||'')}</div>
          </div>
          <div style="display:flex; gap:8px;">
            <button class="btn btn-primary" onclick='editFlow(${f.id})'><i class="fas fa-pen"></i> تعديل</button>
            <button class="btn btn-red" onclick='deleteFlow(${f.id})'><i class="fas fa-trash"></i> حذف</button>
          </div>
        </div>`).join('');
    });
}

function addNode(){
  const board = document.querySelector('.board-wrap');
  const bx = board.getBoundingClientRect();
  const centerX = (bx.width / boardZoom) / 2;
  const centerY = (bx.height / boardZoom) / 2;
  addNodeFromType('keywords', Math.round(centerX - 110), Math.round(centerY - 60));
}
function addNodeFromType(type, x, y){
  const id = 'n' + (Date.now()) + '_' + Math.floor(Math.random()*999);
  const node = {
    id,
    x,
    y,
    type,
    title: (
      type==='text'     ? 'رسالة نصية'   :
      type==='image'    ? 'صورة'         :
      type==='poll'     ? 'استطلاع رأي' :
      type==='location' ? 'موقع'         :
      type==='file'     ? 'ملف PDF'      :
      type==='list'     ? 'قائمة'        :
                          'كلمات مفتاحية'
    ),
    transitions: []
  };
  if (type==='keywords') node.trigger  = { keywords: [], match:'contains' };
  if (type==='text')     node.reply    = { text:'' };
  if (type==='image')    node.media    = { url:'' };
  if (type==='poll')     node.poll     = { question:'', options:[] };
  if (type==='location') node.location = { lat:'', lng:'', title:'', address:'' };
  if (type==='file')     node.file     = { url:'', name:'' };
  if (type==='list')     node.list_config = { buttonText:'افتح القائمه', sections: [] };
  flowState.nodes.push(node);
  renderBoard();
}

// إنشاء عقدة موقع في منتصف البورد من كارت العنوان في الشريط الجانبي
function createLocationNodeFromCard(){
  const board = document.querySelector('.board-wrap');
  if (!board) return;
  const bx = board.getBoundingClientRect();
  const centerX = Math.round((bx.width / boardZoom) / 2 - 110);
  const centerY = Math.round((bx.height / boardZoom) / 2 - 60);

  // إنشاء عقدة موقع جديدة
  addNodeFromType('location', centerX, centerY);
  const node = flowState.nodes[flowState.nodes.length - 1];
  if (!node || node.type !== 'location') return;

  // قراءة القيم من الكارت
  const lat    = document.getElementById('addr-lat')?.value || '';
  const lng    = document.getElementById('addr-lng')?.value || '';
  const title  = document.getElementById('addr-title')?.value || '';
  const addr   = document.getElementById('addr-address')?.value || '';

  node.location = {
    lat: lat.trim(),
    lng: lng.trim(),
    title: title.trim(),
    address: addr.trim()
  };

  renderBoard();
}
function onPaletteDrag(e, type){ e.dataTransfer.setData('type', type); }
function removeNode(id){
  flowState.nodes = flowState.nodes.filter(n=>n.id!==id);
  flowState.nodes.forEach(n=>{ n.transitions = (n.transitions||[]).filter(t=>t.to!==id); });
  renderBoard();
}
function startConnect(ev, id){
  if (ev && ev.stopPropagation) ev.stopPropagation();
  connectingFrom = id; document.getElementById('fbNodes').classList.add('connecting');
  document.addEventListener('mousemove', onConnectMouseMove);
  document.addEventListener('mouseup', onConnectMouseUp);
  const br = document.querySelector('.board-wrap').getBoundingClientRect();
  const baseX = ev ? ev.clientX : (br.left + 1);
  const baseY = ev ? ev.clientY : (br.top + 1);
  connectCursor.x = (baseX - br.left) / boardZoom;
  connectCursor.y = (baseY - br.top) / boardZoom;
  drawEdges();
}
function onConnectMouseMove(e){
  if (!connectingFrom) return;
  const br = document.querySelector('.board-wrap').getBoundingClientRect();
  connectCursor.x = (e.clientX - br.left) / boardZoom;
  connectCursor.y = (e.clientY - br.top) / boardZoom;
  drawEdges();
}
function onConnectMouseUp(){ /* keep connection mode until node click */ }
function endConnect(targetId){
  if (!connectingFrom || connectingFrom===targetId) return cancelConnect();
  const src = flowState.nodes.find(n=>n.id===connectingFrom); if(!src) return cancelConnect();
  src.transitions = src.transitions||[];
  src.transitions.push({ keywords:[], match:'contains', to: targetId });
  cancelConnect();
  renderBoard();
}
function cancelConnect(){ connectingFrom=null; document.getElementById('fbNodes').classList.remove('connecting'); document.removeEventListener('mousemove', onConnectMouseMove); document.removeEventListener('mouseup', onConnectMouseUp); connectCursor={x:null,y:null}; drawEdges(); }

function onBoardDrop(e){
  const type = e.dataTransfer.getData('type'); if (!type) return;
  const br = e.currentTarget.getBoundingClientRect();
  const logicalX = (e.clientX - br.left) / boardZoom;
  const logicalY = (e.clientY - br.top) / boardZoom;
  const x = Math.round(logicalX - 110);
  const y = Math.round(logicalY - 40);
  addNodeFromType(type, Math.max(0,x), Math.max(0,y));
}
function onDragStart(e, id){
  const n = flowState.nodes.find(n=>n.id===id); if(!n) return;
  dragging.id = id;
  const el = document.getElementById('node-'+id);
  const rect = el.getBoundingClientRect();
  dragging.dx = e.clientX - rect.left; dragging.dy = e.clientY - rect.top;
  document.addEventListener('mousemove', onDragMove);
  document.addEventListener('mouseup', onDragEnd);
  e.preventDefault();
}
function onDragMove(e){
  if(!dragging.id) return;
  const board = document.querySelector('.board-wrap'); const br = board.getBoundingClientRect();
  let x = ((e.clientX - dragging.dx) - br.left) / boardZoom;
  let y = ((e.clientY - dragging.dy) - br.top) / boardZoom;
  const maxX = (br.width / boardZoom) - 220;
  const maxY = (br.height / boardZoom) - 40;
  x = Math.max(0, Math.min(maxX, x));
  y = Math.max(0, Math.min(maxY, y));
  const n = flowState.nodes.find(nn=>nn.id===dragging.id); if(!n) return;
  n.x = Math.round(x); n.y = Math.round(y);
  const el = document.getElementById('node-'+n.id); if(el){ el.style.left = n.x+'px'; el.style.top = n.y+'px'; }
  drawEdges();
}
function onDragEnd(){ dragging.id=null; document.removeEventListener('mousemove', onDragMove); document.removeEventListener('mouseup', onDragEnd); }

// Image upload handlers
function onNodeImagePick(id){ const inp = document.getElementById('file-'+id); if (inp) inp.click(); }
function handleNodeFileChange(id, input){
  try {
    const file = input && input.files && input.files[0]; if(!file) return;
    const n = flowState.nodes.find(nn=>nn.id===id); if(!n) return;
    n._uploading = true; renderBoard();
    const fd = new FormData(); fd.append('file', file);
    fetch('api/upload_bot_image.php', { method:'POST', body: fd })
      .then(r=>r.json())
      .then(j=>{ if(!j.success) throw new Error(j.message||'فشل الرفع'); n.media = { url: j.url, name: j.name }; delete n._uploading; renderBoard(); })
      .catch(e=>{
        Swal.fire({
          icon: 'error',
          title: 'خطأ في الرفع',
          text: e.message || 'حدث خطأ أثناء رفع الصورة'
        });
        delete n._uploading; renderBoard();
      });
  } catch(err){
    Swal.fire({
      icon: 'error',
      title: 'تعذر رفع الصورة',
      text: 'حدث خطأ غير متوقع أثناء رفع الصورة.'
    });
  }
}

// File (PDF) upload handlers
function onNodeFilePick(id){ const inp = document.getElementById('pdf-'+id); if (inp) inp.click(); }
function handleNodePdfChange(id, input){
  try {
    const file = input && input.files && input.files[0]; if(!file) return;
    const n = flowState.nodes.find(nn=>nn.id===id); if(!n) return;
    n._uploading = true; renderBoard();
    const fd = new FormData(); fd.append('file', file);
    fetch('api/upload_bot_pdf.php', { method:'POST', body: fd })
      .then(r=>r.json())
      .then(j=>{ if(!j.success) throw new Error(j.message||'فشل رفع الملف'); n.file = { url: j.url, name: j.name }; delete n._uploading; renderBoard(); })
      .catch(e=>{
        Swal.fire({
          icon: 'error',
          title: 'خطأ في رفع الملف',
          text: e.message || 'حدث خطأ أثناء رفع ملف PDF'
        });
        delete n._uploading; renderBoard();
      });
  } catch(err){
    Swal.fire({
      icon: 'error',
      title: 'تعذر رفع الملف',
      text: 'حدث خطأ غير متوقع أثناء رفع الملف.'
    });
  }
}

function openNodeEditor(id){
  editingNodeId = id; const n = flowState.nodes.find(n=>n.id===id); if(!n) return;
  const titleEl = document.getElementById('ne-title'); if (titleEl) titleEl.value = n.title||'';
  const kwWrap = document.getElementById('ne-keywords-wrap');
  const rpWrap = document.getElementById('ne-reply-wrap');
  const pollWrap = document.getElementById('ne-poll-wrap');
  const pollQuestionEl = document.getElementById('ne-poll-question');
  const locWrap = document.getElementById('ne-location-wrap');
  const latEl = document.getElementById('ne-lat');
  const lngEl = document.getElementById('ne-lng');
  const locTitleEl = document.getElementById('ne-loc-title');
  const addrEl = document.getElementById('ne-address');
  const listWrap = document.getElementById('ne-list-wrap');
  const listButtonEl = document.getElementById('ne-list-button');
  const matchWrap = document.getElementById('ne-match-wrap');
  const matchHid = document.getElementById('ne-match');
  const matchSeg = document.getElementById('neMatchSeg');
  const joinWrap = document.getElementById('ne-join-wrap');
  const joinHid = document.getElementById('ne-join');
  const joinSeg = document.getElementById('neJoinSeg');
  if (matchSeg) matchSeg.querySelectorAll('button').forEach(b=>b.classList.remove('selected'));
  if (joinSeg) joinSeg.querySelectorAll('button').forEach(b=>b.classList.remove('selected'));
  if (kwWrap) kwWrap.style.display='none';
  if (rpWrap) rpWrap.style.display='none';
  if (pollWrap) pollWrap.style.display='none';
  if (locWrap) locWrap.style.display='none';
  if (listWrap) listWrap.style.display='none';
  if (matchWrap) matchWrap.style.display='none';
  if (joinWrap) joinWrap.style.display='none';
  neListConfig = null;
  if (n.type==='keywords') {
    if (kwWrap) kwWrap.style.display=''; if (matchWrap) matchWrap.style.display='';
    neKeywords = Array.isArray(n.trigger?.keywords)? [...n.trigger.keywords] : [];
    if (matchHid) matchHid.value = n.trigger?.match||'contains';
    if (matchSeg && matchHid) { const btn = matchSeg.querySelector(`button[data-v=\"${matchHid.value}\"]`); if (btn) btn.classList.add('selected'); }
  } else if (n.type==='text') {
    if (rpWrap) rpWrap.style.display=''; if (joinWrap) joinWrap.style.display='';
    const rep = document.getElementById('ne-reply'); if (rep) rep.value = n.reply?.text||'';
    if (joinHid) joinHid.value = n.join||'any';
    if (joinSeg && joinHid) { const b = joinSeg.querySelector(`button[data-j=\"${joinHid.value}\"]`); if (b) b.classList.add('selected'); }
  } else if (n.type==='poll') {
    if (pollWrap) pollWrap.style.display='';
    nePollOptions = Array.isArray(n.poll?.options) ? [...n.poll.options] : [];
    if (pollQuestionEl) pollQuestionEl.value = n.poll?.question || '';
    renderNePollOptions();
  } else if (n.type==='location') {
    if (locWrap) locWrap.style.display='';
    if (latEl) latEl.value = n.location?.lat || '';
    if (lngEl) lngEl.value = n.location?.lng || '';
    if (locTitleEl) locTitleEl.value = n.location?.title || '';
    if (addrEl) addrEl.value = n.location?.address || '';
  } else if (n.type==='list') {
    if (listWrap) listWrap.style.display='';
    neListConfig = n.list_config && typeof n.list_config === 'object'
      ? JSON.parse(JSON.stringify(n.list_config))
      : { buttonText:'افتح القائمه', sections: [] };
    if (!Array.isArray(neListConfig.sections)) neListConfig.sections = [];
    if (listButtonEl) listButtonEl.value = neListConfig.buttonText || 'افتح القائمه';
    if (listButtonEl) {
      listButtonEl.oninput = (e)=>{
        if (!neListConfig) neListConfig = { buttonText:'', sections: [] };
        neListConfig.buttonText = e.target.value;
      };
    }
    renderNeListSections();
  }
  renderNeChips();
  const modal = document.getElementById('nodeModal'); if (modal) modal.style.display='block';
}
function renderNeChips(){
  const box = document.getElementById('ne-keywords-chips'); if (!box) return;
  box.innerHTML = (neKeywords||[]).map((k,i)=>`<span class=\"chip\"><i class=\"fas fa-tag\"></i> ${escapeHtml(k)} <span class=\"btn-x\" onclick=\"neKeywords.splice(${i},1);renderNeChips();\">×</span></span>`).join('');
}
function renderNePollOptions(){
  const box = document.getElementById('ne-poll-options'); if (!box) return;
  if (!Array.isArray(nePollOptions) || !nePollOptions.length) nePollOptions = [''];
  box.innerHTML = nePollOptions.map((opt,i)=>`<div style=\"display:flex; gap:8px; margin-bottom:6px;\"><input class=\"form-input\" style=\"flex:1;\" value=\"${escapeHtml(opt)}\" oninput=\"nePollOptions[${i}] = this.value\" placeholder=\"نص الاختيار\" /><button type=\"button\" class=\"btn btn-red\" onclick=\"removePollOption(${i})\"><i class=\"fas fa-trash\"></i></button></div>`).join('') + `<button type=\"button\" class=\"btn btn-green\" onclick=\"addPollOption()\"><i class=\"fas fa-plus\"></i> إضافة اختيار</button>`;
}
function addPollOption(){ nePollOptions.push(''); renderNePollOptions(); }
function removePollOption(i){ nePollOptions.splice(i,1); renderNePollOptions(); }

function ensureListConfig(){
  if (!neListConfig) neListConfig = { buttonText:'', sections: [] };
  if (!Array.isArray(neListConfig.sections)) neListConfig.sections = [];
}
function renderNeListSections(){
  const box = document.getElementById('ne-list-sections'); if (!box) return;
  ensureListConfig();
  box.innerHTML = '';
  neListConfig.sections.forEach((sec, si)=>{
    const secDiv = document.createElement('div');
    secDiv.style.border = '1px solid rgba(255,255,255,.1)';
    secDiv.style.borderRadius = '10px';
    secDiv.style.padding = '8px';
    secDiv.style.marginBottom = '8px';
    secDiv.style.display = 'flex';
    secDiv.style.flexDirection = 'column';
    secDiv.style.gap = '6px';

    const headerRow = document.createElement('div');
    headerRow.style.display = 'flex';
    headerRow.style.gap = '8px';
    headerRow.style.alignItems = 'center';

    const titleInput = document.createElement('input');
    titleInput.className = 'form-input';
    titleInput.placeholder = 'عنوان القسم';
    titleInput.value = sec && sec.title ? sec.title : '';
    titleInput.style.flex = '1';
    titleInput.addEventListener('input', (e)=>{
      ensureListConfig();
      if (!neListConfig.sections[si]) neListConfig.sections[si] = { title:'', rows:[] };
      neListConfig.sections[si].title = e.target.value;
    });

    const delSecBtn = document.createElement('button');
    delSecBtn.type = 'button';
    delSecBtn.className = 'btn btn-red';
    delSecBtn.innerHTML = '<i class="fas fa-trash"></i>';
    delSecBtn.onclick = ()=>{ removeListSection(si); };

    headerRow.appendChild(titleInput);
    headerRow.appendChild(delSecBtn);
    secDiv.appendChild(headerRow);

    const rowsWrap = document.createElement('div');
    rowsWrap.style.display = 'flex';
    rowsWrap.style.flexDirection = 'column';
    rowsWrap.style.gap = '6px';

    const rows = Array.isArray(sec.rows) ? sec.rows : [];
    rows.forEach((row, ri)=>{
      const rowBox = document.createElement('div');
      rowBox.style.border = '1px solid rgba(255,255,255,.08)';
      rowBox.style.borderRadius = '8px';
      rowBox.style.padding = '6px';
      rowBox.style.display = 'flex';
      rowBox.style.flexDirection = 'column';
      rowBox.style.gap = '4px';

      const topRow = document.createElement('div');
      topRow.style.display = 'flex';
      topRow.style.gap = '6px';
      topRow.style.alignItems = 'center';

      const rowIdInput = document.createElement('input');
      rowIdInput.className = 'form-input';
      rowIdInput.placeholder = 'rowId';
      rowIdInput.value = row && row.rowId ? row.rowId : '';
      rowIdInput.style.flex = '1';
      rowIdInput.addEventListener('input', (e)=>{
        ensureListConfig();
        const s = neListConfig.sections[si];
        if (!s.rows[ri]) s.rows[ri] = { rowId:'', title:'', description:'' };
        s.rows[ri].rowId = e.target.value;
      });

      const delRowBtn = document.createElement('button');
      delRowBtn.type = 'button';
      delRowBtn.className = 'btn btn-red';
      delRowBtn.innerHTML = '<i class="fas fa-trash"></i>';
      delRowBtn.onclick = ()=>{ removeListRow(si, ri); };

      topRow.appendChild(rowIdInput);
      topRow.appendChild(delRowBtn);
      rowBox.appendChild(topRow);

      const titleRowInput = document.createElement('input');
      titleRowInput.className = 'form-input';
      titleRowInput.placeholder = 'العنوان';
      titleRowInput.value = row && row.title ? row.title : '';
      titleRowInput.addEventListener('input', (e)=>{
        ensureListConfig();
        const s = neListConfig.sections[si];
        if (!s.rows[ri]) s.rows[ri] = { rowId:'', title:'', description:'' };
        s.rows[ri].title = e.target.value;
      });
      rowBox.appendChild(titleRowInput);

      const descInput = document.createElement('textarea');
      descInput.className = 'form-textarea';
      descInput.placeholder = 'الوصف';
      descInput.value = row && row.description ? row.description : '';
      descInput.addEventListener('input', (e)=>{
        ensureListConfig();
        const s = neListConfig.sections[si];
        if (!s.rows[ri]) s.rows[ri] = { rowId:'', title:'', description:'' };
        s.rows[ri].description = e.target.value;
      });
      rowBox.appendChild(descInput);

      rowsWrap.appendChild(rowBox);
    });

    const addRowBtn = document.createElement('button');
    addRowBtn.type = 'button';
    addRowBtn.className = 'btn btn-green';
    addRowBtn.innerHTML = '<i class="fas fa-plus"></i> إضافة عنصر';
    addRowBtn.onclick = ()=>{ addListRow(si); };

    rowsWrap.appendChild(addRowBtn);
    secDiv.appendChild(rowsWrap);

    box.appendChild(secDiv);
  });
}
function addListSection(){
  ensureListConfig();
  neListConfig.sections.push({ title:'قسم جديد', rows: [] });
  renderNeListSections();
}
function removeListSection(i){
  ensureListConfig();
  neListConfig.sections.splice(i,1);
  renderNeListSections();
}
function addListRow(sectionIndex){
  ensureListConfig();
  const sec = neListConfig.sections[sectionIndex];
  if (!sec.rows) sec.rows = [];
  sec.rows.push({ rowId:'', title:'', description:'' });
  renderNeListSections();
}
function removeListRow(sectionIndex, rowIndex){
  ensureListConfig();
  const sec = neListConfig.sections[sectionIndex];
  if (!sec || !Array.isArray(sec.rows)) return;
  sec.rows.splice(rowIndex,1);
  renderNeListSections();
}

function closeNodeEditor(){ const m=document.getElementById('nodeModal'); if(m) m.style.display='none'; editingNodeId=null; neKeywords=[]; nePollOptions=[]; neListConfig=null; }
function saveNodeEditor(){
  const n = flowState.nodes.find(n=>n.id===editingNodeId); if(!n) return;
  n.title = document.getElementById('ne-title').value.trim()||n.title;
  if (n.type==='keywords') {
    n.trigger = { match: document.getElementById('ne-match').value || 'contains', keywords: (neKeywords||[]).map(s=>String(s).trim()).filter(Boolean) };
  } else if (n.type==='text') {
    n.reply = { text: document.getElementById('ne-reply').value||'' };
    n.join = document.getElementById('ne-join')?.value || 'any';
  } else if (n.type==='poll') {
    const qEl = document.getElementById('ne-poll-question');
    const question = qEl ? qEl.value||'' : '';
    n.poll = { question, options: (nePollOptions||[]).map(s=>String(s).trim()).filter(Boolean) };
  } else if (n.type==='location') {
    const lat = document.getElementById('ne-lat')?.value || '';
    const lng = document.getElementById('ne-lng')?.value || '';
    const locTitle = document.getElementById('ne-loc-title')?.value || '';
    const address = document.getElementById('ne-address')?.value || '';
    n.location = {
      lat: lat.trim(),
      lng: lng.trim(),
      title: locTitle.trim(),
      address: address.trim()
    };
  } else if (n.type==='list') {
    const btnEl = document.getElementById('ne-list-button');
    const btnText = btnEl ? btnEl.value || '' : (neListConfig && neListConfig.buttonText) || '';
    const cfg = { buttonText: (btnText || 'افتح القائمه').trim(), sections: [] };
    if (neListConfig && Array.isArray(neListConfig.sections)) {
      cfg.sections = neListConfig.sections.map(sec=>{
        const title = (sec && sec.title) ? String(sec.title).trim() : '';
        const rows = Array.isArray(sec.rows) ? sec.rows.map(row=>({
          rowId: row && row.rowId ? String(row.rowId).trim() : '',
          title: row && row.title ? String(row.title).trim() : '',
          description: row && row.description ? String(row.description).trim() : ''
        })).filter(r=>r.rowId || r.title || r.description) : [];
        return { title, rows };
      }).filter(sec=>sec.title || (sec.rows && sec.rows.length));
    }
    n.list_config = cfg;
  }
  closeNodeEditor();
  renderBoard();
}
// segmented toggle for match
(function(){
  const seg = document.getElementById('neMatchSeg'); const hid = document.getElementById('ne-match');
  if (!seg || !hid) return;
  seg.addEventListener('click', (e)=>{ const b = e.target.closest('button[data-v]'); if (!b) return; seg.querySelectorAll('button').forEach(x=>x.classList.remove('selected')); b.classList.add('selected'); hid.value = b.getAttribute('data-v'); });
})();
// segmented toggle for join (text nodes)
(function(){
  const seg = document.getElementById('neJoinSeg'); const hid = document.getElementById('ne-join');
  if (!seg || !hid) return;
  seg.addEventListener('click', (e)=>{ const b = e.target.closest('button[data-j]'); if (!b) return; seg.querySelectorAll('button').forEach(x=>x.classList.remove('selected')); b.classList.add('selected'); hid.value = b.getAttribute('data-j'); });
})();
// keywords input add
(function(){
  const input = document.getElementById('ne-keyword-input'); if (!input) return;
  input.addEventListener('keydown', (e)=>{
    if (e.key==='Enter' || e.key===',') { e.preventDefault(); const v = input.value.trim(); if (v) { (v.split(',')).forEach(x=>{ const s = x.trim(); if(s) neKeywords.push(s); }); input.value=''; renderNeChips(); } }
  });
})();

function centerBoard(){ drawEdges(); }

function setBoardZoom(z){
  const min = 0.5;
  const max = 2.0;
  boardZoom = Math.max(min, Math.min(max, z));
  const inner = document.getElementById('board-inner');
  if (inner) inner.style.transform = 'scale(' + boardZoom + ')';
  const lbl = document.getElementById('boardZoomLabel');
  if (lbl) lbl.textContent = Math.round(boardZoom * 100) + '%';
  drawEdges();
}

function renderBoard(){
  const wrap = document.getElementById('fbNodes');
  if (!wrap) return;

  wrap.innerHTML = flowState.nodes.map(n=>{
    const title = escapeHtml(n.title || 'عقدة');

    let typeBadge = '';
    if (n.type === 'text') {
      typeBadge = `<span class="badge-type"><i class="fas fa-align-left"></i> رسالة${n.join === 'all' ? ' · كل المداخل' : ''}</span>`;
    } else if (n.type === 'image') {
      typeBadge = '<span class="badge-type"><i class="fas fa-image"></i> صورة</span>';
    } else if (n.type === 'poll') {
      typeBadge = '<span class="badge-type"><i class="fas fa-poll-h"></i> استطلاع</span>';
    } else if (n.type === 'location') {
      typeBadge = '<span class="badge-type"><i class="fas fa-map-marker-alt"></i> موقع</span>';
    } else if (n.type === 'file') {
      typeBadge = '<span class="badge-type"><i class="fas fa-file-pdf"></i> ملف PDF</span>';
    } else if (n.type === 'list') {
      typeBadge = '<span class="badge-type"><i class="fas fa-list"></i> قائمة</span>';
    } else {
      typeBadge = '<span class="badge-type"><i class="fas fa-key"></i> كلمات</span>';
    }

    let body = '';
    if (n.type === 'text') {
      const txt = (n.reply && n.reply.text) ? n.reply.text : '';
      body = `<div style="white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">${escapeHtml(txt.slice(0,60)) || '—'}</div>`;
    } else if (n.type === 'keywords') {
      const kws = (n.trigger && Array.isArray(n.trigger.keywords)) ? n.trigger.keywords : [];
      const shown = kws.slice(0,3);
      const rest = Math.max(0, kws.length - shown.length);
      const chips = shown.map(k=>`<span class="chip"><i class="fas fa-tag"></i> ${escapeHtml(k)}</span>`).join('');
      const more = rest>0 ? `<span class="chip more">+${rest}</span>` : '';
      const match = n.trigger && n.trigger.match === 'equals' ? 'equals' : 'contains';
      const mp = match === 'equals'
        ? '<span class="match-pill equal"><i class="fas fa-equals"></i> تساوي</span>'
        : '<span class="match-pill"><i class="fas fa-asterisk"></i> تحتوي</span>';
      body = `<div style="display:flex; flex-direction:column; gap:6px;">${mp}<div class="chips">${chips}${more}</div></div>`;
    } else if (n.type === 'image') {
      const uploading = !!n._uploading;
      const has = !!(n.media && n.media.url);
      const img = has ? `<img class="img-thumb" src="${escapeHtml(n.media.url)}" alt="">` : '';
      const spin = uploading ? '<div class="thin-muted"><i class="fas fa-spinner fa-spin"></i> جاري الرفع...</div>' : '';
      const btnDis = uploading ? 'disabled style="opacity:.7"' : '';
      body = `<div class="img-uploader">${img}<button class="btn btn-primary" onclick="onNodeImagePick('${n.id}')" ${btnDis}>${has ? 'تغيير الصورة' : 'رفع صورة'}</button>${spin}<input id="file-${n.id}" type="file" accept="image/*" style="display:none" onchange="handleNodeFileChange('${n.id}', this)" /></div>`;
    } else if (n.type === 'poll') {
      const q = escapeHtml((n.poll && n.poll.question) || 'استطلاع رأي');
      const opts = (n.poll && Array.isArray(n.poll.options)) ? n.poll.options : [];
      const shown = opts.slice(0,3);
      const rest = Math.max(0, opts.length - shown.length);
      const chips = shown.map(o=>`<span class="chip"><i class="fas fa-dot-circle"></i> ${escapeHtml(o)}</span>`).join('');
      const more = rest>0 ? `<span class="chip more">+${rest}</span>` : '';
      const chipsHtml = chips + more || '<span class="thin-muted">لا يوجد اختيارات</span>';
      body = `<div style="display:flex; flex-direction:column; gap:6px;"><div style="white-space:nowrap; overflow:hidden; text-overflow:ellipsis; font-weight:700; color:#e5e7eb;">${q}</div><div class="chips">${chipsHtml}</div></div>`;
    } else if (n.type === 'location') {
      const lt  = escapeHtml(n.location && n.location.title  ? n.location.title  : 'موقع');
      const lat = escapeHtml(n.location && n.location.lat    ? n.location.lat    : '');
      const lng = escapeHtml(n.location && n.location.lng    ? n.location.lng    : '');
      const addr = escapeHtml(n.location && n.location.address ? n.location.address.slice(0,60) : '');
      const coords = (lat || lng) ? `${lat}, ${lng}` : 'لم يتم إدخال الإحداثيات';
      body = `<div style="display:flex; flex-direction:column; gap:4px;"><div style="font-weight:700; color:#e5e7eb; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">${lt}</div><div class="thin-muted">${coords}</div><div style="white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">${addr}</div></div>`;
    } else if (n.type === 'file') {
      const has = !!(n.file && n.file.url);
      const name = escapeHtml(n.file && n.file.name ? n.file.name : 'ملف PDF');
      const uploading = !!n._uploading;
      const spin = uploading ? '<div class="thin-muted"><i class="fas fa-spinner fa-spin"></i> جاري الرفع...</div>' : '';
      const btnDis = uploading ? 'disabled style="opacity:.7"' : '';
      const preview = has
        ? `<div style="display:flex; flex-direction:column; gap:4px;"><div style="font-weight:700; color:#e5e7eb; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;"><i class="fas fa-file-pdf"></i> ${name}</div><a href="${escapeHtml(n.file.url)}" target="_blank" class="thin-muted">فتح المعاينة</a></div>`
        : '<span class="thin-muted">لم يتم رفع ملف بعد</span>';
      body = `<div class="img-uploader">${preview}<button class="btn btn-primary" onclick="onNodeFilePick('${n.id}')" ${btnDis}>${has ? 'تغيير الملف' : 'رفع ملف PDF'}</button>${spin}<input id="pdf-${n.id}" type="file" accept="application/pdf" style="display:none" onchange="handleNodePdfChange('${n.id}', this)" /></div>`;
    } else if (n.type === 'list') {
      const cfg = n.list_config || {};
      const btn = escapeHtml(cfg.buttonText || 'افتح القائمه');
      const sections = Array.isArray(cfg.sections) ? cfg.sections : [];
      let rowsCount = 0;
      sections.forEach(sec=>{
        if (sec && Array.isArray(sec.rows)) rowsCount += sec.rows.length;
      });
      const secCount = sections.length;
      body = `<div style="display:flex; flex-direction:column; gap:4px;">
        <div style="font-weight:700; color:#e5e7eb; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">زر: ${btn}</div>
        <div class="thin-muted">أقسام: ${secCount} · عناصر: ${rowsCount}</div>
      </div>`;
    } else {
      body = '<span class="thin-muted">لا توجد بيانات</span>';
    }

    return `<div id="node-${n.id}" class="fb-node" style="left:${n.x||20}px; top:${n.y||20}px;" ondblclick="openNodeEditor('${n.id}')">
      <div class="head" onmousedown="onDragStart(event,'${n.id}')">
        <div class="title"><i class="fas fa-node fa-bounce"></i> ${title}</div>
        <div style="display:flex; align-items:center; gap:6px;">${typeBadge}
          <button class="edit-btn" title="اتصال" onclick="startConnect('${n.id}')"><i class="fas fa-link"></i></button>
          <button class="edit-btn" title="تحرير" onclick="openNodeEditor('${n.id}')"><i class="fas fa-pen"></i></button>
          <button class="edit-btn" title="حذف" onclick="removeNode('${n.id}')"><i class="fas fa-trash"></i></button>
        </div>
      </div>
      <div class="body">${body}</div>
      <div class="handles handles-left"><div class="handle handle-in" title="نقطة دخول"></div></div>
      <div class="handles"><div class="handle" onmousedown="startConnect(event,'${n.id}')" title="ابدأ اتصال"></div></div>
    </div>`;
  }).join('');

  // click target to end connect
  wrap.querySelectorAll('.fb-node').forEach(el=>{
    el.addEventListener('click', (e)=>{
      if (!connectingFrom) return; const id = el.id.replace('node-',''); endConnect(id);
      e.stopPropagation();
    });
  });

  drawEdges();
}
 
function renderSessions(){
  const box = document.getElementById('sessionsList'); if (!box) return;
  if (!Array.isArray(allAccountsCache) || !allAccountsCache.length){ box.innerHTML='—'; return; }
  box.innerHTML = allAccountsCache.map(a=>{
    const sel = String(a.account_uid)===String(selectedAccountUid);
    const badge = sel ? `<span class=\"sel-badge\"><i class=\"fas fa-check\"></i> محددة</span>` : '';
    const btn = sel ? `<button disabled style=\"opacity:.6; cursor:not-allowed;\">محددة</button>` : `<button onclick=\"selectAccount('${a.account_uid}')\">اختيار</button>`;
    return `<div class=\"session-card ${sel?'selected':''}\"><div style=\"display:flex; align-items:center; gap:8px;\"><i class=\"fab fa-whatsapp\" style=\"color:#25D366\"></i> ${escapeHtml(a.name||a.account_uid)} ${badge}</div>${btn}</div>`;
  }).join('');
}
function selectAccount(uid){
  selectedAccountUid = uid;
  flowState.account_uid = uid;
  const sel = document.getElementById('accountSelect'); if (sel) sel.value = uid;
  renderSessions();
}

function getAnchorXY(id, side){
  const el = document.getElementById('node-'+id); const board = document.querySelector('.board-wrap'); if (!el||!board) return {x:0,y:0};
  const r = el.getBoundingClientRect(); const br = board.getBoundingClientRect();
  const xScreen = side==='left' ? (r.left - br.left) : (r.right - br.left);
  const yScreen = (r.top - br.top) + 28; // near header line
  const x = xScreen / boardZoom;
  const y = yScreen / boardZoom;
  return { x: Math.round(x), y: Math.round(y) };
}
function drawEdges(){
  const svg = document.getElementById('fbEdges');
  const board = document.querySelector('.board-wrap');
  const br = board.getBoundingClientRect();
  const vw = Math.round(br.width / boardZoom);
  const vh = Math.round(br.height / boardZoom);
  svg.setAttribute('viewBox', `0 0 ${vw} ${vh}`);
  svg.innerHTML = '';
  const paths = [];
  flowState.nodes.forEach(src=>{
    (src.transitions||[]).forEach(t=>{
      if (!t.to) return; const tgt = flowState.nodes.find(n=>n.id===t.to); if(!tgt) return;
      const s = getAnchorXY(src.id, 'right'); const txy = getAnchorXY(tgt.id, 'left');
      const dx = Math.abs(txy.x - s.x) * 0.5; const c1x = s.x + dx, c1y = s.y; const c2x = txy.x - dx, c2y = txy.y;
paths.push(`<path d=\"M ${s.x} ${s.y} C ${c1x} ${c1y}, ${c2x} ${c2y}, ${txy.x} ${txy.y}\" stroke=\"#7c3aed\" stroke-width=\"3.2\" fill=\"none\" opacity=\"0.95\" stroke-linecap=\"round\" />`);
    });
  });
  // Preview line while connecting
  if (connectingFrom && connectCursor.x!==null) {
    const s = getAnchorXY(connectingFrom, 'right');
    const dx = Math.abs(connectCursor.x - s.x) * 0.5; const c1x = s.x + dx, c1y = s.y; const c2x = connectCursor.x - dx, c2y = connectCursor.y;
    paths.push(`<path d=\"M ${s.x} ${s.y} C ${c1x} ${c1y}, ${c2x} ${c2y}, ${connectCursor.x} ${connectCursor.y}\" stroke=\"#60a5fa\" stroke-dasharray=\"8 6\" stroke-width=\"3\" fill=\"none\" stroke-linecap=\"round\" />`);
  }
  svg.innerHTML += paths.join('');
}

function resetBuilder(){ flowState = { id:null, account_uid:'', name:'', nodes: [] }; const accEl=document.getElementById('accountSelect'); if(accEl) accEl.value=''; const fn=document.getElementById('flowName'); if(fn) fn.value=''; renderBoard(); }

function saveFlow(){
  const accEl = document.getElementById('accountSelect');
  const acc = accEl ? accEl.value : (flowState.account_uid || selectedAccountUid || '');
  const nameEl = document.getElementById('flowName');
  const name = nameEl ? (nameEl.value||'').trim() : (flowState.name||'');
  const payload = { action:'save', id: flowState.id, account_uid: acc, flow_name: name||'Workflow', config: flowState };
  fetch('api/flows.php', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(payload) })
    .then(r=>r.json()).then(j=>{
      if(!j.success) throw new Error(j.message||'تعذر الحفظ');
      flowState.id = j.id;
      Swal.fire({
        icon: 'success',
        title: 'تم الحفظ بنجاح',
        timer: 1500,
        showConfirmButton: false
      });
      loadFlows();
    }).catch(e=>{
      Swal.fire({
        icon: 'error',
        title: 'خطأ في الحفظ',
        text: e.message || 'تعذر حفظ التدفق.'
      });
    });
}

function editFlow(id){
  fetch('api/flows.php?action=get&id='+encodeURIComponent(id))
    .then(r=>r.json()).then(j=>{
      if(!j.success) throw new Error(j.message||'');
      const accEl = document.getElementById('accountSelect'); if (accEl) accEl.value = j.flow.account_uid || '';
      const nameEl = document.getElementById('flowName'); if (nameEl) nameEl.value = j.flow.flow_name || '';
      try { flowState = j.flow.config && typeof j.flow.config==='string' ? JSON.parse(j.flow.config) : (j.flow.config || {}); } catch(e){ flowState = j.flow.config || {}; }
      if(!flowState || !flowState.nodes) flowState = { id: j.flow.id, account_uid: j.flow.account_uid, name: j.flow.flow_name, nodes: [] };
      flowState.id = j.flow.id;
      renderBoard();
    }).catch(e=>{
      Swal.fire({
        icon: 'error',
        title: 'خطأ في تحميل التدفق',
        text: e.message || 'تعذر تحميل بيانات التدفق.'
      });
    });
}
function deleteFlow(id){
  if(!confirm('تأكيد الحذف؟')) return;
  fetch('api/flows.php', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ action:'delete', id }) })
    .then(r=>r.json()).then(j=>{
      if(!j.success) throw new Error(j.message||'');
      resetBuilder();
      loadFlows();
      Swal.fire({
        icon: 'success',
        title: 'تم حذف التدفق',
        timer: 1200,
        showConfirmButton: false
      });
    })
    .catch(e=>{
      Swal.fire({
        icon: 'error',
        title: 'خطأ في الحذف',
        text: e.message || 'تعذر حذف التدفق.'
      });
    });
}

function escapeHtml(s){ return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#039;'); }

document.addEventListener('DOMContentLoaded', ()=>{
  loadAccounts();
  loadFlows();
  renderBoard();
  setBoardZoom(1);
  const board = document.querySelector('.board-wrap');
  if (board) {
    board.addEventListener('wheel', (e)=>{
      if (e.ctrlKey) {
        e.preventDefault();
        const delta = e.deltaY > 0 ? -0.1 : 0.1;
        setBoardZoom(boardZoom + delta);
      }
    }, { passive:false });
  }
});
</script>

<?php include 'includes/footer.php'; ?>
