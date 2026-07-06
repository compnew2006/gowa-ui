<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kingmaster - منصة التسويق الذكية</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@300;400;600;700;800;900&display=swap" rel="stylesheet">
    
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        :root {
            --primary: #667eea;
            --secondary: #764ba2;
            --accent: #f093fb;
            --dark: #0f172a;
            --light: #f1f5f9;
            --success: #10b981;
            --warning: #fbbf24;
        }

        body {
            font-family: 'Cairo', sans-serif;
            background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
            color: #f1f5f9;
            overflow-x: hidden;
            position: relative;
            transition: all 0.3s ease;
        }

        body.light-theme {
            background: linear-gradient(135deg, #f1f5f9 0%, #e2e8f0 100%);
            color: #0f172a;
        }

        /* خلفية متحركة */
        body::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: 
                radial-gradient(circle at 20% 30%, rgba(102, 126, 234, 0.15) 0%, transparent 40%),
                radial-gradient(circle at 80% 70%, rgba(118, 75, 162, 0.15) 0%, transparent 40%),
                radial-gradient(circle at 50% 50%, rgba(240, 147, 251, 0.1) 0%, transparent 50%);
            animation: gradientMove 15s ease infinite;
            z-index: -1;
        }

        @keyframes gradientMove {
            0%, 100% {
                transform: translate(0, 0) scale(1);
            }
            33% {
                transform: translate(50px, -50px) scale(1.1);
            }
            66% {
                transform: translate(-50px, 50px) scale(0.9);
            }
        }

        /* Navbar */
        .navbar {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            background: rgba(15, 23, 42, 0.95);
            backdrop-filter: blur(20px);
            padding: 15px 0;
            z-index: 1000;
            border-bottom: 2px solid rgba(102, 126, 234, 0.2);
            box-shadow: 0 5px 30px rgba(0, 0, 0, 0.3);
            transition: all 0.3s ease;
        }

        body.light-theme .navbar {
            background: rgba(255, 255, 255, 0.95);
            border-bottom: 2px solid rgba(102, 126, 234, 0.3);
            box-shadow: 0 5px 30px rgba(0, 0, 0, 0.1);
        }

        .navbar.scrolled {
            padding: 10px 0;
            box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
        }

        .nav-container {
            max-width: 1600px;
            margin: 0 auto;
            padding: 0 40px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 40px;
        }

        .logo {
            font-size: 28px;
            font-weight: 900;
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .logo i {
            font-size: 32px;
            animation: rotate 3s linear infinite;
        }

        .nav-links {
            display: flex;
            gap: 30px;
            list-style: none;
            align-items: center;
        }

        .nav-links li {
            position: relative;
        }

        .nav-links a {
            color: #f1f5f9;
            text-decoration: none;
            font-weight: 600;
            font-size: 15px;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 8px 12px;
            border-radius: 10px;
        }

        body.light-theme .nav-links a {
            color: #0f172a;
        }

        .nav-links a i {
            font-size: 16px;
            transition: all 0.3s ease;
        }

        .nav-links a:hover {
            background: rgba(102, 126, 234, 0.1);
            transform: translateY(-2px);
        }

        .nav-links a:hover i {
            transform: scale(1.2);
        }

        /* أيقونات ملونة */
        .nav-links a[href="#home"] i { color: #667eea; animation: bounce 2s infinite; }
        .nav-links a[href="#features"] i { color: #10b981; animation: pulse-grow 2s infinite; }
        .nav-links a[href="#pricing"] i { color: #fbbf24; animation: swing 2s infinite; }
        .nav-links a[href="#screenshots"] i { color: #f093fb; animation: rotate 3s linear infinite; }
        .nav-links a[href="#testimonials"] i { color: #3b82f6; animation: pulse 2s infinite; }

        .nav-actions {
            display: flex;
            align-items: center;
            gap: 15px;
        }

        .icon-btn {
            width: 40px;
            height: 40px;
            border-radius: 50%;
            background: rgba(102, 126, 234, 0.1);
            border: 2px solid rgba(102, 126, 234, 0.3);
            color: #667eea;
            display: flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            transition: all 0.3s ease;
            font-size: 18px;
        }

        .icon-btn:hover {
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            color: white;
            transform: rotate(360deg) scale(1.1);
            box-shadow: 0 5px 20px rgba(102, 126, 234, 0.4);
        }

        .icon-btn i {
            animation: pulse 2s infinite;
        }

        .auth-btns {
            display: flex;
            gap: 10px;
        }

        .btn-login {
            background: rgba(102, 126, 234, 0.1);
            color: #667eea;
            padding: 10px 25px;
            border-radius: 25px;
            text-decoration: none;
            font-weight: 700;
            font-size: 14px;
            border: 2px solid rgba(102, 126, 234, 0.3);
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        body.light-theme .btn-login {
            color: #667eea;
            border-color: rgba(102, 126, 234, 0.4);
        }

        .btn-login:hover {
            background: rgba(102, 126, 234, 0.2);
            transform: translateY(-2px);
        }

        .btn-register {
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            color: white;
            padding: 10px 25px;
            border-radius: 25px;
            text-decoration: none;
            font-weight: 700;
            font-size: 14px;
            transition: all 0.3s ease;
            box-shadow: 0 5px 20px rgba(102, 126, 234, 0.4);
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .btn-register:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 30px rgba(102, 126, 234, 0.6);
        }

        .cta-btn {
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            color: white;
            padding: 12px 30px;
            border-radius: 50px;
            text-decoration: none;
            font-weight: 700;
            transition: all 0.3s ease;
            box-shadow: 0 5px 20px rgba(102, 126, 234, 0.4);
        }

        .cta-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 30px rgba(102, 126, 234, 0.6);
        }

        /* Hero Section */
        .hero {
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            text-align: center;
            padding: 120px 40px 80px;
            position: relative;
            overflow: hidden;
        }

        .hero::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: 
                radial-gradient(circle at 20% 50%, rgba(102, 126, 234, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 80% 80%, rgba(118, 75, 162, 0.1) 0%, transparent 50%);
            z-index: 0;
        }

        .hero-content {
            max-width: 900px;
            position: relative;
            z-index: 1;
        }

        .hero h1 {
            font-size: 64px;
            font-weight: 900;
            margin-bottom: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            animation: fadeInUp 1s ease;
        }

        .hero p {
            font-size: 24px;
            color: #cbd5e1;
            margin-bottom: 40px;
            animation: fadeInUp 1s ease 0.2s backwards;
        }

        .hero-buttons {
            display: flex;
            gap: 20px;
            justify-content: center;
            animation: fadeInUp 1s ease 0.4s backwards;
        }

        .btn-primary {
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            color: white;
            padding: 18px 40px;
            border-radius: 50px;
            text-decoration: none;
            font-weight: 700;
            font-size: 18px;
            transition: all 0.3s ease;
            box-shadow: 0 8px 25px rgba(102, 126, 234, 0.4);
            display: inline-flex;
            align-items: center;
            gap: 10px;
        }

        .btn-primary:hover {
            transform: translateY(-3px);
            box-shadow: 0 12px 35px rgba(102, 126, 234, 0.6);
        }

        .btn-secondary {
            background: rgba(102, 126, 234, 0.1);
            color: white;
            padding: 18px 40px;
            border-radius: 50px;
            text-decoration: none;
            font-weight: 700;
            font-size: 18px;
            border: 2px solid rgba(102, 126, 234, 0.3);
            transition: all 0.3s ease;
            display: inline-flex;
            align-items: center;
            gap: 10px;
        }

        .btn-secondary:hover {
            background: rgba(102, 126, 234, 0.2);
            border-color: var(--primary);
        }

        /* Features Section */
        .features {
            padding: 100px 40px;
            background: rgba(30, 41, 59, 0.5);
        }

        .section-title {
            text-align: center;
            font-size: 48px;
            font-weight: 900;
            margin-bottom: 60px;
            background: linear-gradient(135deg, #667eea, #764ba2);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .features-grid {
            max-width: 1400px;
            margin: 0 auto;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 40px;
        }

        .feature-card {
            background: rgba(30, 41, 59, 0.8);
            padding: 40px;
            border-radius: 24px;
            border: 2px solid rgba(102, 126, 234, 0.2);
            transition: all 0.3s ease;
            text-align: center;
        }

        .feature-card:hover {
            transform: translateY(-10px);
            border-color: var(--primary);
            box-shadow: 0 20px 50px rgba(102, 126, 234, 0.3);
        }

        .feature-icon {
            font-size: 64px;
            margin-bottom: 20px;
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .feature-card h3 {
            font-size: 24px;
            margin-bottom: 15px;
            font-weight: 800;
        }

        .feature-card p {
            color: #cbd5e1;
            line-height: 1.8;
        }

        /* Pricing Section */
        .pricing {
            padding: 100px 40px;
        }

        .pricing-grid {
            max-width: 1400px;
            margin: 0 auto;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
            gap: 40px;
        }

        .pricing-card {
            background: rgba(30, 41, 59, 0.8);
            border-radius: 24px;
            padding: 0;
            border: 2px solid rgba(102, 126, 234, 0.2);
            transition: all 0.4s ease;
            display: flex;
            flex-direction: column;
            position: relative;
            overflow: hidden;
        }

        .pricing-card:hover {
            transform: translateY(-10px);
            box-shadow: 0 25px 60px rgba(102, 126, 234, 0.4);
            border-color: var(--primary);
        }

        .pricing-header {
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            padding: 30px;
            text-align: center;
            position: relative;
        }

        .popular-badge {
            position: absolute;
            top: 15px;
            right: 15px;
            background: rgba(255, 255, 255, 0.25);
            backdrop-filter: blur(10px);
            color: white;
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 700;
            border: 1px solid rgba(255, 255, 255, 0.3);
            animation: pulse 2s infinite;
        }

        .popular-badge i {
            animation: rotate 3s linear infinite;
        }

        .pricing-name {
            font-size: 28px;
            font-weight: 900;
            color: white;
            margin-bottom: 10px;
        }

        .pricing-description {
            color: rgba(255, 255, 255, 0.9);
            font-size: 14px;
        }

        .pricing-body {
            padding: 40px 30px;
            flex: 1;
            display: flex;
            flex-direction: column;
        }

        .pricing-price {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 30px;
            border-bottom: 2px dashed rgba(102, 126, 234, 0.2);
        }

        .price-amount {
            font-size: 48px;
            font-weight: 900;
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .price-currency {
            font-size: 20px;
        }

        .discount-badge {
            display: inline-block;
            background: linear-gradient(135deg, #ef4444, #dc2626);
            color: white;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 13px;
            font-weight: 700;
            margin-top: 10px;
        }

        .price-original {
            font-size: 20px;
            color: #64748b;
            text-decoration: line-through;
            margin-top: 5px;
        }

        .pricing-features {
            flex: 1;
            margin-bottom: 30px;
        }

        .pricing-feature {
            display: flex;
            align-items: flex-start;
            gap: 12px;
            margin-bottom: 15px;
            padding: 10px;
            border-radius: 10px;
            transition: all 0.3s ease;
        }

        .pricing-feature:hover {
            background: rgba(102, 126, 234, 0.05);
        }

        .pricing-feature i {
            color: var(--success);
            font-size: 18px;
            margin-top: 3px;
            animation: pulse-grow 2s infinite;
        }

        .pricing-stats {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 15px;
            margin-bottom: 30px;
        }

        .pricing-stat {
            text-align: center;
            padding: 15px;
            background: linear-gradient(135deg, rgba(102, 126, 234, 0.08), rgba(118, 75, 162, 0.08));
            border-radius: 12px;
            border: 1px solid rgba(102, 126, 234, 0.15);
        }

        .stat-icon {
            font-size: 24px;
            margin-bottom: 8px;
        }

        .stat-icon.accounts { color: var(--primary); animation: bounce 2s infinite; }
        .stat-icon.messages { color: var(--success); animation: swing 2s infinite; }
        .stat-icon.points { color: var(--warning); animation: pulse-grow 2s infinite; }

        .stat-value {
            font-size: 20px;
            font-weight: 900;
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .stat-label {
            font-size: 11px;
            color: #64748b;
            font-weight: 600;
            text-transform: uppercase;
        }

        .pricing-platforms {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 1px solid rgba(102, 126, 234, 0.2);
        }

        .platform-badge {
            padding: 8px 14px;
            border-radius: 25px;
            font-size: 12px;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .platform-facebook { background: rgba(59, 89, 152, 0.1); color: #3b5998; }
        .platform-whatsapp { background: rgba(37, 211, 102, 0.1); color: #25d366; }
        .platform-telegram { background: rgba(0, 136, 204, 0.1); color: #0088cc; }
        .platform-instagram { background: rgba(193, 53, 132, 0.1); color: #c13584; }
        .platform-email { background: rgba(234, 67, 53, 0.1); color: #ea4335; }
        .platform-business { background: rgba(102, 126, 234, 0.1); color: #667eea; }

        .pricing-button {
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            color: white;
            padding: 16px;
            border: none;
            border-radius: 12px;
            font-weight: 700;
            font-size: 16px;
            cursor: pointer;
            transition: all 0.3s ease;
            margin-top: auto;
        }

        .pricing-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 30px rgba(102, 126, 234, 0.5);
        }

        /* Footer */
        .footer {
            background: rgba(15, 23, 42, 0.8);
            padding: 60px 40px 30px;
            border-top: 1px solid rgba(102, 126, 234, 0.2);
        }

        .footer-content {
            max-width: 1400px;
            margin: 0 auto;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 40px;
            margin-bottom: 40px;
        }

        .footer-section h3 {
            font-size: 20px;
            margin-bottom: 20px;
            font-weight: 800;
        }

        .footer-section p, .footer-section a {
            color: #cbd5e1;
            text-decoration: none;
            display: block;
            margin-bottom: 10px;
            transition: all 0.3s ease;
        }

        .footer-section a:hover {
            color: var(--primary);
            transform: translateX(-5px);
        }

        .footer-bottom {
            text-align: center;
            padding-top: 30px;
            border-top: 1px solid rgba(102, 126, 234, 0.2);
            color: #64748b;
        }

        /* Animations */
        @keyframes fadeInUp {
            from {
                opacity: 0;
                transform: translateY(30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        @keyframes rotate {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
        }

        @keyframes bounce {
            0%, 100% { transform: translateY(0); }
            50% { transform: translateY(-8px); }
        }

        @keyframes swing {
            0%, 100% { transform: rotate(0deg); }
            25% { transform: rotate(-10deg); }
            75% { transform: rotate(10deg); }
        }

        @keyframes pulse {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.05); }
        }

        @keyframes pulse-grow {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.15); }
        }

        /* Responsive */
        @media (max-width: 768px) {
            .hero h1 {
                font-size: 40px;
            }

            .hero p {
                font-size: 18px;
            }

            .nav-links {
                display: none;
            }

            .pricing-grid,
            .features-grid {
                grid-template-columns: 1fr;
            }

            .hero-buttons {
                flex-direction: column;
            }
        }
    </style>
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
                <li><a href="#screenshots"><i class="fas fa-images"></i> لقطات الشاشة</a></li>
                <li><a href="#testimonials"><i class="fas fa-comments"></i> آراء العملاء</a></li>
            </ul>
            <div class="nav-actions">
                <div class="icon-btn" onclick="toggleTheme()" title="تغيير الثيم">
                    <i class="fas fa-moon" id="themeIcon"></i>
                </div>
                <div class="icon-btn" onclick="toggleLanguage()" title="تغيير اللغة">
                    <i class="fas fa-language"></i>
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
        <div class="hero-content">
            <h1>منصة التسويق الذكية</h1>
            <p>حلول تسويقية متكاملة عبر جميع المنصات الاجتماعية</p>
            <div class="hero-buttons">
                <a href="#pricing" class="btn-primary">
                    <i class="fas fa-rocket"></i>
                    ابدأ الآن
                </a>
                <a href="#features" class="btn-secondary">
                    <i class="fas fa-info-circle"></i>
                    اعرف المزيد
                </a>
            </div>
        </div>
    </section>

    <!-- Features Section -->
    <section class="features" id="features">
        <h2 class="section-title">لماذا تختار Kingmaster؟</h2>
        <div class="features-grid">
            <div class="feature-card">
                <div class="feature-icon">
                    <i class="fas fa-users"></i>
                </div>
                <h3>إدارة متعددة الحسابات</h3>
                <p>إدارة جميع حساباتك على المنصات المختلفة من مكان واحد بكل سهولة وأمان</p>
            </div>

            <div class="feature-card">
                <div class="feature-icon">
                    <i class="fas fa-paper-plane"></i>
                </div>
                <h3>رسائل مجمعة</h3>
                <p>أرسل آلاف الرسائل في نفس الوقت لعملائك عبر جميع المنصات</p>
            </div>

            <div class="feature-card">
                <div class="feature-icon">
                    <i class="fas fa-chart-line"></i>
                </div>
                <h3>تقارير تفصيلية</h3>
                <p>احصل على تقارير دقيقة عن أداء حملاتك التسويقية</p>
            </div>

            <div class="feature-card">
                <div class="feature-icon">
                    <i class="fas fa-shield-alt"></i>
                </div>
                <h3>أمان عالي</h3>
                <p>نظام حماية متطور لضمان سرية بياناتك وحساباتك</p>
            </div>

            <div class="feature-card">
                <div class="feature-icon">
                    <i class="fas fa-robot"></i>
                </div>
                <h3>أتمتة ذكية</h3>
                <p>أتمتة المهام التسويقية بذكاء اصطناعي متطور</p>
            </div>

            <div class="feature-card">
                <div class="feature-icon">
                    <i class="fas fa-headset"></i>
                </div>
                <h3>دعم فني 24/7</h3>
                <p>فريق دعم متاح على مدار الساعة لمساعدتك</p>
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

    <!-- Footer -->
    <footer class="footer" id="contact">
        <div class="footer-content">
            <div class="footer-section">
                <h3>عن Kingmaster</h3>
                <p>منصة تسويقية متكاملة توفر لك جميع الأدوات التي تحتاجها للنجاح في التسويق الرقمي</p>
            </div>

            <div class="footer-section">
                <h3>روابط سريعة</h3>
                <a href="#home">الرئيسية</a>
                <a href="#features">المميزات</a>
                <a href="#pricing">الأسعار</a>
                <a href="login.php">تسجيل الدخول</a>
            </div>

            <div class="footer-section">
                <h3>تواصل معنا</h3>
                <p><i class="fas fa-envelope"></i> info@kingmaster.com</p>
                <p><i class="fas fa-phone"></i> +966 50 123 4567</p>
                <p><i class="fas fa-map-marker-alt"></i> السعودية، الرياض</p>
            </div>

            <div class="footer-section">
                <h3>تابعنا</h3>
                <a href="#"><i class="fab fa-facebook"></i> فيسبوك</a>
                <a href="#"><i class="fab fa-twitter"></i> تويتر</a>
                <a href="#"><i class="fab fa-instagram"></i> إنستجرام</a>
                <a href="#"><i class="fab fa-linkedin"></i> لينكدإن</a>
            </div>
        </div>

        <div class="footer-bottom">
            <p>&copy; 2024 Kingmaster. جميع الحقوق محفوظة.</p>
        </div>
    </footer>

    <script>
        // Load packages from API
        document.addEventListener('DOMContentLoaded', function() {
            loadPricing();
        });

        function loadPricing() {
            fetch('api/get_packages.php')
                .then(response => response.json())
                .then(data => {
                    if (data.success && data.packages.length > 0) {
                        displayPricing(data.packages);
                    } else {
                        document.getElementById('pricingGrid').innerHTML = `
                            <div style="text-align: center; grid-column: 1 / -1; padding: 60px;">
                                <i class="fas fa-box-open" style="font-size: 80px; color: var(--primary); opacity: 0.5;"></i>
                                <p style="margin-top: 20px; font-size: 18px;">لا توجد باقات متاحة حالياً</p>
                            </div>
                        `;
                    }
                })
                .catch(error => {
                    console.error('Error:', error);
                });
        }

        function displayPricing(packages) {
            const grid = document.getElementById('pricingGrid');
            
            const currencySymbols = {
                'EGP': 'جنيه',
                'USD': '$',
                'SAR': 'ريال',
                'AED': 'درهم',
                'KWD': 'د.ك',
                'QAR': 'ر.ق',
                'EUR': '€',
                'GBP': '£'
            };

            const platformIcons = {
                'facebook': '<i class="fab fa-facebook"></i>',
                'whatsapp': '<i class="fab fa-whatsapp"></i>',
                'telegram': '<i class="fab fa-telegram"></i>',
                'instagram': '<i class="fab fa-instagram"></i>',
                'email': '<i class="fas fa-envelope"></i>',
                'business': '<i class="fas fa-briefcase"></i>'
            };

            const platformNames = {
                'facebook': 'فيسبوك',
                'whatsapp': 'واتساب',
                'telegram': 'تليجرام',
                'instagram': 'إنستجرام',
                'email': 'بريد',
                'business': 'أعمال'
            };

            grid.innerHTML = packages.map(pkg => {
                const features = JSON.parse(pkg.features || '[]');
                const platforms = JSON.parse(pkg.platforms || '[]');
                const currency = currencySymbols[pkg.currency] || pkg.currency || 'جنيه';
                const discountPercent = pkg.has_discount && pkg.original_price > 0 
                    ? Math.round(((pkg.original_price - pkg.price) / pkg.original_price) * 100) 
                    : 0;

                return `
                    <div class="pricing-card">
                        <div class="pricing-header">
                            ${pkg.is_popular ? '<div class="popular-badge"><i class="fas fa-star"></i> الأكثر مبيعاً</div>' : ''}
                            <div class="pricing-name">${pkg.name}</div>
                            ${pkg.description ? `<div class="pricing-description">${pkg.description}</div>` : ''}
                        </div>
                        
                        <div class="pricing-body">
                            <div class="pricing-price">
                                ${pkg.has_discount && discountPercent > 0 ? `<div class="discount-badge">خصم ${discountPercent}%</div>` : ''}
                                <div class="price-amount">${pkg.price} <span class="price-currency">${currency}</span></div>
                                ${pkg.has_discount && pkg.original_price ? `<div class="price-original">${pkg.original_price} ${currency}</div>` : ''}
                            </div>

                            <div class="pricing-stats">
                                <div class="pricing-stat">
                                    <div class="stat-icon accounts"><i class="fas fa-users"></i></div>
                                    <div class="stat-value">${pkg.accounts_count}</div>
                                    <div class="stat-label">حساب</div>
                                </div>
                                <div class="pricing-stat">
                                    <div class="stat-icon messages"><i class="fas fa-envelope"></i></div>
                                    <div class="stat-value">${pkg.messages_count}</div>
                                    <div class="stat-label">رسالة</div>
                                </div>
                                <div class="pricing-stat">
                                    <div class="stat-icon points"><i class="fas fa-star"></i></div>
                                    <div class="stat-value">${pkg.points}</div>
                                    <div class="stat-label">نقطة</div>
                                </div>
                            </div>

                            <div class="pricing-features">
                                ${features.map(feature => `
                                    <div class="pricing-feature">
                                        <i class="fas fa-check-circle"></i>
                                        <span>${feature}</span>
                                    </div>
                                `).join('')}
                            </div>

                            <div class="pricing-platforms">
                                ${platforms.map(platform => `
                                    <span class="platform-badge platform-${platform}">
                                        ${platformIcons[platform] || ''} ${platformNames[platform] || platform}
                                    </span>
                                `).join('')}
                            </div>

                            <button class="pricing-button" onclick="orderPackage(${pkg.id})">
                                <i class="fas fa-shopping-cart"></i> اطلب الآن
                            </button>
                        </div>
                    </div>
                `;
            }).join('');
        }

        function orderPackage(packageId) {
            // Redirect to order page or show order modal
            window.location.href = `order.php?package=${packageId}`;
        }

        // Smooth scroll for navigation links
        document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', function (e) {
                e.preventDefault();
                const target = document.querySelector(this.getAttribute('href'));
                if (target) {
                    target.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
            });
        });
    </script>
</body>
</html>
