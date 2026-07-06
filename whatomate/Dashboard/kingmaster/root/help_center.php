<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>مركز المساعدة | Kingmaster</title>
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
    .container { max-width: 1100px; margin: 0 auto; padding: 2rem 1rem; }
    .header-bar { display:flex; align-items:center; justify-content:space-between; gap:1rem; margin-bottom:1rem; }
    .header-title { font-size:1.8rem; font-weight:700; color: var(--text-light); margin:0; }
    .back-link { display:inline-flex; align-items:center; gap:.5rem; padding:.5rem .75rem; border:1px solid var(--glass-border); border-radius:10px; color:#e5e7eb; text-decoration:none; background:var(--glass-bg); }
    .back-link:hover { background: rgba(255,255,255,.06); }
    .grid { display:grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1rem; }
    .card { background: var(--glass-bg); border:1px solid var(--glass-border); border-radius: 16px; backdrop-filter: blur(14px); box-shadow: 0 8px 28px rgba(0,0,0,.35); }
    .card-body { padding: 1.25rem; }
    h2 { font-size:1.25rem; color:#fff; margin:0 0 .75rem; display:flex; align-items:center; gap:.5rem; }
    p, li, a { color:#d1d5db; line-height:1.9; }
    a { color:#93c5fd; text-decoration: underline; }
    a:hover { color:#bfdbfe; }
  </style>
</head>
<body class="rtl ar">
  <div class="bg-gradient"></div>
  <div class="container">
    <div class="header-bar">
      <h1 class="header-title"><i class="fas fa-circle-question"></i> مركز المساعدة</h1>
      <a class="back-link" href="landing"><i class="fas fa-arrow-right"></i><span>رجوع</span></a>
    </div>

    <div class="grid">
      <div class="card"><div class="card-body">
        <h2><i class="fas fa-video"></i> فيديوهات الشرح</h2>
        <p>راجع الشروحات لتتعرف على الأدوات والخصائص.</p>
        <p><a href="https://www.youtube.com/@kingmaster6297" target="_blank" rel="noopener">فتح قائمة الشروحات</a></p>
      </div></div>

      <div class="card"><div class="card-body">
        <h2><i class="fas fa-book"></i> أسئلة شائعة</h2>
        <ul>
          <li>كيف أفعّل الترخيص؟ — من لوحة الحساب، اختر “إدخال الترخيص”.</li>
          <li>لا تعمل أداة ما؟ — تحقق من نظام التشغيل والاعتمادات أولًا.</li>
          <li>مشاكل تسجيل الدخول — جرّب استرجاع كلمة المرور أو تواصل مع الدعم.</li>
        </ul>
      </div></div>

      <div class="card"><div class="card-body">
        <h2><i class="fas fa-headset"></i> الدعم الفني</h2>
        <p>للمساعدة التقنية، استخدم صفحة <a href="contact">اتصل بنا</a> وحدّد نوع المشكلة بالتفصيل.</p>
      </div></div>
    </div>
  </div>
</body>
</html>
