<?php
session_start();

// إذا كان المستخدم مسجل دخول بالفعل، إعادة توجيهه
if (isset($_SESSION['user_id'])) {
    header('Location: index.php');
    exit;
}
?>
<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kingmaster - منصة التسويق الذكية</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@300;400;600;700;800;900&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="assets/css/landing.css">
</head>
<body>
    <!-- Navbar -->
    <nav class="navbar" id="navbar">
        <div class="nav-container">
            <div class="logo">
                <i class="fas fa-rocket"></i>
                Kingmaster
            </div>
            <ul class="nav-links">
                <li><a href="#home"><i class="fas fa-home"></i> الرئيسية</a></li>
                <li><a href="#features"><i class="fas fa-star"></i> المميزات</a></li>
                <li><a href="#pricing"><i class="fas fa-tags"></i> الأسعار</a></li>
                <li><a href="#faq"><i class="fas fa-question-circle"></i> الأسئلة</a></li>
            </ul>
            <div class="nav-actions">
                <div class="icon-btn" id="theme-toggle" onclick="toggleTheme()" title="تغيير الثيم">
                    <i class="fas fa-moon" id="themeIcon"></i>
                </div>
                <div class="language-dropdown">
                    <div class="icon-btn" onclick="toggleLanguageMenu()" title="تغيير اللغة">
                        <i class="fas fa-globe"></i>
                    </div>
                    <div class="language-menu" id="language-menu">
                        <div class="lang-option" onclick="changeLanguage('ar')">
                            <span style="font-size: 24px;">🇪🇬</span>
                            <span>مصر</span>
                        </div>
                        <div class="lang-option" onclick="changeLanguage('en')">
                            <span style="font-size: 24px;">🇺🇸</span>
                            <span>أمريكا</span>
                        </div>
                        <div class="lang-option" onclick="changeLanguage('fr')">
                            <span style="font-size: 24px;">🇫🇷</span>
                            <span>فرنسا</span>
                        </div>
                    </div>
                </div>
                <div class="auth-btns">
                    <a href="login.php" class="btn-login">
                        <i class="fas fa-sign-in-alt"></i>
                        تسجيل دخول
                    </a>
                    <a href="register.php" class="btn-register">
                        <i class="fas fa-user-plus"></i>
                        حساب جديد
                    </a>
                </div>
            </div>
        </div>
    </nav>

    <!-- Hero Section -->
    <section class="hero" id="home">
        <div class="stars-container" id="stars-container"></div>
        <div class="hero-content">
            <div class="hero-badge">
                <i class="fas fa-crown"></i>
                <span>المنصة رقم 1 في العالم العربي</span>
            </div>
            <h1>أقوى أدوات التسويق الرقمي</h1>
            <p>اكتشف مجموعة شاملة من الأدوات المتطورة لإدارة حملاتك التسويقية<br>وزيادة أرباحك بطريقة احترافية وآمنة</p>
            <div class="hero-buttons">
                <a href="#pricing" class="btn-primary">
                    <i class="fas fa-rocket"></i>
                    ابدأ مجاناً الآن
                </a>
                <a href="#video" class="btn-secondary">
                    <i class="fas fa-play-circle"></i>
                    شاهد العرض التوضيحي
                </a>
            </div>
            
            <!-- Stats -->
            <div class="hero-stats">
                <div class="hero-stat">
                    <div class="stat-icon">
                        <i class="fas fa-users"></i>
                    </div>
                    <div class="stat-number" data-target="50000" data-suffix="K+">0</div>
                    <div class="stat-label">مستخدم نشط</div>
                </div>
                <div class="hero-stat">
                    <div class="stat-icon">
                        <i class="fas fa-rocket"></i>
                    </div>
                    <div class="stat-number" data-target="1000000" data-suffix="M+">0</div>
                    <div class="stat-label">حملة نجحت</div>
                </div>
                <div class="hero-stat">
                    <div class="stat-icon">
                        <i class="fas fa-check-circle"></i>
                    </div>
                    <div class="stat-number" data-target="99.9" data-suffix="%">0</div>
                    <div class="stat-label">وقت التشغيل</div>
                </div>
            </div>
        </div>
    </section>

    <!-- Video Section -->
    <section class="video-section" id="video">
        <div class="video-container">
            <h2 class="section-title">شاهد كيف تعمل منصة Kingmaster</h2>
            <p class="video-subtitle">تعرف على المنصة ومميزاتها في أقل من 3 دقائق</p>
            
            <div class="video-wrapper">
                <div class="video-placeholder" id="video-placeholder">
                    <div class="play-button" onclick="loadVideo()">
                        <i class="fas fa-play"></i>
                    </div>
                    <div class="video-info">
                        <i class="fas fa-video"></i>
                        <span>اضغط لمشاهدة الفيديو</span>
                    </div>
                </div>
                <iframe 
                    id="youtube-video" 
                    width="100%" 
                    height="100%" 
                    src="" 
                    frameborder="0" 
                    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" 
                    allowfullscreen
                    style="display: none;"
                ></iframe>
            </div>
            
            <div class="video-features">
                <div class="video-feature">
                    <i class="fas fa-clock"></i>
                    <span>3 دقائق</span>
                </div>
                <div class="video-feature">
                    <i class="fas fa-language"></i>
                    <span>باللغة العربية</span>
                </div>
                <div class="video-feature">
                    <i class="fas fa-hd-video"></i>
                    <span>جودة عالية HD</span>
                </div>
            </div>
        </div>
    </section>

    <!-- Features Section -->
    <section class="features" id="features">
        <h2 class="section-title">لماذا تختار Kingmaster؟</h2>
        <div class="features-grid">
            <div class="feature-card">
                <div class="feature-icon"><i class="fas fa-users"></i></div>
                <h3>إدارة متعددة الحسابات</h3>
                <p>إدارة جميع حساباتك على المنصات المختلفة من مكان واحد</p>
            </div>
            <div class="feature-card">
                <div class="feature-icon"><i class="fas fa-paper-plane"></i></div>
                <h3>رسائل مجمعة</h3>
                <p>أرسل آلاف الرسائل في نفس الوقت لعملائك</p>
            </div>
            <div class="feature-card">
                <div class="feature-icon"><i class="fas fa-chart-line"></i></div>
                <h3>تقارير تفصيلية</h3>
                <p>احصل على تقارير دقيقة عن أداء حملاتك</p>
            </div>
            <div class="feature-card">
                <div class="feature-icon"><i class="fas fa-shield-alt"></i></div>
                <h3>أمان عالي</h3>
                <p>نظام حماية متطور لضمان سرية بياناتك</p>
            </div>
            <div class="feature-card">
                <div class="feature-icon"><i class="fas fa-robot"></i></div>
                <h3>أتمتة ذكية</h3>
                <p>أتمتة المهام التسويقية بذكاء اصطناعي</p>
            </div>
            <div class="feature-card">
                <div class="feature-icon"><i class="fas fa-headset"></i></div>
                <h3>دعم فني 24/7</h3>
                <p>فريق دعم متاح على مدار الساعة</p>
            </div>
        </div>
    </section>

    <!-- Pricing Section -->
    <section class="pricing" id="pricing">
        <h2 class="section-title">اختر الباقة المناسبة لك</h2>
        <div class="pricing-grid" id="pricingGrid">
            <div style="text-align: center; grid-column: 1 / -1; padding: 60px;">
                <i class="fas fa-spinner fa-spin" style="font-size: 48px; color: var(--primary);"></i>
                <p style="margin-top: 20px; font-size: 18px;">جاري تحميل الباقات...</p>
            </div>
        </div>
    </section>

    <!-- Screenshots Section -->
    <section class="screenshots" id="screenshots">
        <h2 class="section-title">لقطات من المنصة</h2>
        <div class="screenshots-grid">
            <div class="screenshot-item">
                <div class="screenshot-placeholder" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); height: 300px; display: flex; align-items: center; justify-content: center; color: white; font-size: 24px; font-weight: 800;">
                    <i class="fas fa-desktop" style="font-size: 64px;"></i>
                </div>
                <div class="screenshot-overlay">
                    <h3>لوحة التحكم</h3>
                    <p>واجهة سهلة وبسيطة</p>
                </div>
            </div>
            <div class="screenshot-item">
                <div class="screenshot-placeholder" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); height: 300px; display: flex; align-items: center; justify-content: center; color: white; font-size: 24px; font-weight: 800;">
                    <i class="fas fa-users" style="font-size: 64px;"></i>
                </div>
                <div class="screenshot-overlay">
                    <h3>إدارة الحسابات</h3>
                    <p>تحكم كامل بحساباتك</p>
                </div>
            </div>
            <div class="screenshot-item">
                <div class="screenshot-placeholder" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); height: 300px; display: flex; align-items: center; justify-content: center; color: white; font-size: 24px; font-weight: 800;">
                    <i class="fas fa-paper-plane" style="font-size: 64px;"></i>
                </div>
                <div class="screenshot-overlay">
                    <h3>إرسال الرسائل</h3>
                    <p>رسائل جماعية بكل سهولة</p>
                </div>
            </div>
            <div class="screenshot-item">
                <div class="screenshot-placeholder" style="background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); height: 300px; display: flex; align-items: center; justify-content: center; color: white; font-size: 24px; font-weight: 800;">
                    <i class="fas fa-chart-line" style="font-size: 64px;"></i>
                </div>
                <div class="screenshot-overlay">
                    <h3>التقارير</h3>
                    <p>إحصائيات تفصيلية</p>
                </div>
            </div>
        </div>
    </section>

    <!-- Testimonials Section -->
    <section class="testimonials" id="testimonials">
        <h2 class="section-title">ماذا يقول عملاؤنا</h2>
        <div class="testimonials-grid">
            <div class="testimonial-card">
                <div class="testimonial-avatar">
                    <i class="fas fa-user-circle"></i>
                </div>
                <div class="testimonial-content">
                    <p>"منصة رائعة ساعدتني في إدارة حساباتي بكل سهولة. النتائج مذهلة!"</p>
                    <div class="testimonial-rating">
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                    </div>
                </div>
                <div class="testimonial-author">
                    <h4>أحمد محمد</h4>
                    <p>صاحب متجر إلكتروني</p>
                </div>
            </div>

            <div class="testimonial-card">
                <div class="testimonial-avatar">
                    <i class="fas fa-user-circle"></i>
                </div>
                <div class="testimonial-content">
                    <p>"أفضل استثمار قمت به لتطوير أعمالي. الدعم الفني ممتاز!"</p>
                    <div class="testimonial-rating">
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                    </div>
                </div>
                <div class="testimonial-author">
                    <h4>فاطمة علي</h4>
                    <p>مسوقة رقمية</p>
                </div>
            </div>

            <div class="testimonial-card">
                <div class="testimonial-avatar">
                    <i class="fas fa-user-circle"></i>
                </div>
                <div class="testimonial-content">
                    <p>"وفرت علي الكثير من الوقت والجهد. أنصح الجميع باستخدامها!"</p>
                    <div class="testimonial-rating">
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                        <i class="fas fa-star"></i>
                    </div>
                </div>
                <div class="testimonial-author">
                    <h4>خالد السعيد</h4>
                    <p>مدير تسويق</p>
                </div>
            </div>
        </div>
    </section>

    <!-- FAQ Section -->
    <section class="faq" id="faq">
        <h2 class="section-title">الأسئلة الشائعة</h2>
        <div class="faq-container">
            <div class="faq-item">
                <div class="faq-question">
                    <div class="faq-icon">💡</div>
                    <h3>ما هي منصة كينج ماستر؟</h3>
                    <i class="fas fa-chevron-down faq-toggle"></i>
                </div>
                <div class="faq-answer">
                    <p>منصة تسويقية ذكية بتساعدك تدير التواصل مع عملائك آليًا وتزود مبيعاتك بسهولة و كمان بتساعدك علي الانتشار.</p>
                </div>
            </div>

            <div class="faq-item">
                <div class="faq-question">
                    <div class="faq-icon">🧠</div>
                    <h3>هل كينج ماستر آمنة في الاستخدام؟</h3>
                    <i class="fas fa-chevron-down faq-toggle"></i>
                </div>
                <div class="faq-answer">
                    <p>نعم، المنصة مبنية على أعلى معايير الأمان، وبيتم تشفير البيانات بشكل كامل لضمان خصوصية العملاء والمستخدمين.</p>
                </div>
            </div>

            <div class="faq-item">
                <div class="faq-question">
                    <div class="faq-icon">🎯</div>
                    <h3>كينج ماستر بتخدم مين؟</h3>
                    <i class="fas fa-chevron-down faq-toggle"></i>
                </div>
                <div class="faq-answer">
                    <p>بتخدم أصحاب المشاريع، المسوقين، والمتاجر اللي عايزين نظام تسويق آلي يجمع داتا العملاء ويرجعهم يشترو تاني.</p>
                </div>
            </div>

            <div class="faq-item">
                <div class="faq-question">
                    <div class="faq-icon">⚙️</div>
                    <h3>إيه المميزات اللي تخليني أختار كينج ماستر؟</h3>
                    <i class="fas fa-chevron-down faq-toggle"></i>
                </div>
                <div class="faq-answer">
                    <p>إرسال جماعي – بوت ذكي – تقارير وتحليلات – دعم عربي كامل – شغال 24/7.</p>
                </div>
            </div>

            <div class="faq-item">
                <div class="faq-question">
                    <div class="faq-icon">🚀</div>
                    <h3>إزاي أبدأ أستخدم المنصة؟</h3>
                    <i class="fas fa-chevron-down faq-toggle"></i>
                </div>
                <div class="faq-answer">
                    <p>سجل حسابك، اختار الباقة المناسبة، وابدأ حملتك فورًا بدون أي تعقيد.</p>
                </div>
            </div>

            <div class="faq-item">
                <div class="faq-question">
                    <div class="faq-icon">💰</div>
                    <h3>هل في خطط وأسعار مختلفة؟</h3>
                    <i class="fas fa-chevron-down faq-toggle"></i>
                </div>
                <div class="faq-answer">
                    <p>أيوه، في 4 خطط مرنة بتناسب كل الأحجام — من التجربة المجانية لحد الباقة الاحترافية.</p>
                </div>
            </div>

            <div class="faq-item">
                <div class="faq-question">
                    <div class="faq-icon">📞</div>
                    <h3>هل في دعم فني لو واجهت مشكلة؟</h3>
                    <i class="fas fa-chevron-down faq-toggle"></i>
                </div>
                <div class="faq-answer">
                    <p>طبعًا، تقدر تتواصل مع فريق الدعم في أي وقت من خلال واتساب أو من داخل المنصة.</p>
                </div>
            </div>
        </div>
    </section>

    <!-- Footer -->
    <footer class="footer" id="contact">
        <div class="footer-content">
            <div class="footer-section">
                <h3><i class="fas fa-rocket"></i> عن Kingmaster</h3>
                <p>منصة تسويقية متكاملة توفر لك جميع الأدوات للنجاح في التسويق الرقمي</p>
                <div class="footer-social">
                    <a href="#" class="social-icon"><i class="fab fa-facebook"></i></a>
                    <a href="#" class="social-icon"><i class="fab fa-instagram"></i></a>
                    <a href="#" class="social-icon"><i class="fab fa-linkedin"></i></a>
                </div>
            </div>
            
            <div class="footer-section">
                <h3>الشركة</h3>
                <a href="about">من نحن</a>
                <a href="#careers">الوظائف</a>
                <a href="contact">اتصل بنا</a>
                <a href="#blog">المدونة</a>
            </div>
            
            <div class="footer-section">
                <h3>الدعم</h3>
                <a href="help_center">مركز المساعدة</a>
                <a href="#faq">الأسئلة الشائعة</a>
                <a href="contact">اتصل بنا</a>
                <a href="#status">حالة الخدمة</a>
            </div>
            
            <div class="footer-section">
                <h3>قانوني</h3>
                <a href="privacy">سياسة الخصوصية</a>
                <a href="terms_privacy">الشروط والأحكام</a>
                <a href="cookies">سياسة ملفات تعريف الارتباط</a>
                <a href="security">الأمان</a>
                 <a href="refund">سياسة أسترجاع الاموال</a>
            </div>
        </div>
        <div class="footer-bottom">
            <p>&copy; 2024 Kingmaster. جميع الحقوق محفوظة.</p>
        </div>
    </footer>

    <script src="assets/js/landing.js"></script>
</body>
</html>
