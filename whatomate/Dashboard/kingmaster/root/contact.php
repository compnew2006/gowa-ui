<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>اتصل بنا | Kingmaster</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@400;500;600;700&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
  <link rel="stylesheet" href="css/styles.css">
  <link rel="stylesheet" href="css/rtl-ltr.css">
  <style>
    :root { color-scheme: dark; }
    body { font-family: 'Cairo', system-ui, -apple-system, Segoe UI, Roboto, "Helvetica Neue", Arial; }
    .bg-gradient { position: fixed; inset: 0; z-index: -1; }
    .container { max-width: 1000px; margin: 0 auto; padding: 2rem 1rem; }
    .header-bar { display:flex; align-items:center; justify-content:space-between; gap:1rem; margin-bottom:1rem; }
    .header-title { font-size:1.8rem; font-weight:700; color: var(--text-light); margin:0; }
    .back-link { display:inline-flex; align-items:center; gap:.5rem; padding:.5rem .75rem; border:1px solid var(--glass-border); border-radius:10px; color:#e5e7eb; text-decoration:none; background:var(--glass-bg); }
    .back-link:hover { background: rgba(255,255,255,.06); }
    .grid { display:grid; grid-template-columns: 1.2fr .8fr; gap: 1rem; }
    .card { background: var(--glass-bg); border:1px solid var(--glass-border); border-radius: 16px; backdrop-filter: blur(14px); box-shadow: 0 8px 28px rgba(0,0,0,.35); }
    .card-body { padding: 1.25rem; }
    label { color:#e5e7eb; margin-bottom:.35rem; display:block; }
    input, textarea { width:100%; padding:.75rem; border:1px solid var(--glass-border); border-radius:10px; background: rgba(17,24,39,0.85); color:#fff; }
    textarea { min-height: 140px; resize: vertical; }
    .btn { padding:.7rem 1.1rem; border:none; border-radius:10px; background: linear-gradient(135deg,var(--primary),var(--primary-light)); color:#fff; font-weight:700; cursor:pointer; }
    .list { list-style:none; padding:0; margin:0; }
    .list li { display:flex; gap:.6rem; align-items:center; color:#d1d5db; margin:.5rem 0; }
    @media (max-width: 768px){ .grid{ grid-template-columns: 1fr; } }
  </style>
</head>
<body class="rtl ar">
  <div class="bg-gradient"></div>
  <div class="container">
    <div class="header-bar">
      <h1 class="header-title"><i class="fas fa-envelope-open-text"></i> اتصل بنا</h1>
      <a class="back-link" href="landing"><i class="fas fa-arrow-right"></i><span>رجوع</span></a>
    </div>

    <div class="grid">
      <div class="card"><div class="card-body">
        <form id="contactForm">
          <div class="form-group">
            <label>الاسم</label>
            <input type="text" id="name" required />
          </div>
          <div class="form-group">
            <label>البريد الإلكتروني</label>
            <input type="email" id="email" required />
          </div>
          <div class="form-group">
            <label>الموضوع</label>
            <input type="text" id="subject" required />
          </div>
          <div class="form-group">
            <label>الرسالة</label>
            <textarea id="message" required></textarea>
          </div>
          <button class="btn" type="submit"><i class="fas fa-paper-plane"></i> إرسال</button>
        </form>
      </div></div>

      <div class="card"><div class="card-body">
        <h2 style="color:#fff; margin-top:0; font-size:1.2rem"><i class="fas fa-headset"></i> معلومات التواصل</h2>
        <ul class="list">
          <li><i class="fas fa-envelope"></i> البريد: info@kingmaster.org</li>
          <li><i class="fas fa-phone"></i> الهاتف: +201033685371</li>
          <li><i class="fas fa-phone"></i> الهاتف: +201025385693</li>

          <li><i class="fas fa-location-dot"></i> العنوان: الاسكندريه - العجمي الهانوفيل </li>
        </ul>
        <p style="color:#9ca3af; font-size:.9rem; margin-top:.75rem">بجوار نقطه فوزي معاذ سيتي بلاس</p>
      </div></div>
    </div>
  </div>

  <script src="https://cdn.jsdelivr.net/npm/sweetalert2@11"></script>
  <script>
    if (!window.Swal) { window.Swal = { fire: (o)=>alert((o.title||'')+'\n'+(o.text||'')) }; }
    document.getElementById('contactForm').addEventListener('submit', function(e){
      e.preventDefault();
      const name = document.getElementById('name').value.trim();
      const email = document.getElementById('email').value.trim();
      const subject = document.getElementById('subject').value.trim();
      const message = document.getElementById('message').value.trim();
      if(!name || !email || !subject || !message){ Swal.fire({icon:'warning', title:'تنبيه', text:'يرجى ملء جميع الحقول'}); return; }
      Swal.fire({icon:'success', title:'تم الإرسال', text:'تم إرسال رسالتك بنجاح (نموذج تجريبي).'});
      this.reset();
    });
  </script>
</body>
</html>
