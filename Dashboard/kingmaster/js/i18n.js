/**
 * King Master Dashboard - Internationalization (i18n)
 * نظام الترجمة متعدد اللغات
 */

// قاموس الترجمات
const translations = {
    ar: {
        maneg_account: "إدارة الحسابات",
        // صفحة التسجيل
        create_account: "إنشاء حساب جديد",
        create_account_subtitle: "انضم إلينا واستمتع بأفضل أدوات التسويق الرقمي",
        full_name: "الاسم الكامل",
        email: "البريد الإلكتروني",
        password: "كلمة المرور",
        phone: "رقم الهاتف",
        timezone: "المنطقة الزمنية",
        country: "الدولة",
        terms_agreement: "أوافق على",
        terms_conditions: "الشروط والأحكام",
        privacy_policy: "سياسة الخصوصية",
        create_account_btn: "إنشاء الحساب",
        already_have_account: "لديك حساب بالفعل؟",
        login: "تسجيل الدخول",
        
        // Placeholders
        enter_full_name: "أدخل اسمك الكامل",
        enter_email: "أدخل بريدك الإلكتروني",
        enter_password: "أدخل كلمة المرور",
        enter_phone: "أدخل رقم الهاتف",
        search_timezone: "ابحث عن المنطقة الزمنية...",
        auto_detect_country: "سيتم اكتشافها تلقائياً",
        select_timezone: "اختر المنطقة الزمنية",
        
        // اللغات
        arabic: "العربية",
        english: "الإنجليزية", 
        french: "الفرنسية",
        
        // رسائل التحقق
        fill_all_fields: "يرجى ملء جميع الحقول المطلوبة",
        agree_to_terms: "يجب الموافقة على الشروط والأحكام",
        invalid_email: "يرجى إدخال بريد إلكتروني صحيح",
        password_too_short: "يجب أن تكون كلمة المرور 6 أحرف على الأقل",
        account_created: "تم إنشاء الحساب بنجاح! مرحباً بك في King Master",
        creating_account: "جاري إنشاء الحساب...",
        
        // رسائل اكتشاف البلد
        detecting_country: "جاري اكتشاف البلد...",
        country_detected: "تم اكتشاف البلد بنجاح",
        country_not_detected: "لم يتم التعرف على البلد",
        invalid_phone_format: "تنسيق رقم الهاتف غير صحيح",
        
        // صفحة تسجيل الدخول
        login_title: "تسجيل الدخول",
        login_subtitle: "أدخل بياناتك للوصول إلى حسابك",
        remember_me: "تذكرني",
        forgot_password: "نسيت كلمة المرور؟",
        login_btn: "تسجيل الدخول",
        no_account: "ليس لديك حساب؟",
        create_account_link: "إنشاء حساب جديد",
        logging_in: "جاري تسجيل الدخول...",
        login_success: "تم تسجيل الدخول بنجاح! مرحباً بك",
        
        // صفحة نسيت كلمة المرور
        forgot_password_title: "نسيت كلمة المرور؟",
        forgot_password_subtitle: "أدخل بريدك الإلكتروني وسنرسل لك رابط إعادة تعيين كلمة المرور",
        reset_password_info: "ستصلك رسالة على بريدك الإلكتروني خلال دقائق قليلة تحتوي على رابط إعادة تعيين كلمة المرور. تحقق من صندوق البريد الوارد أو الرسائل غير المرغوب فيها.",
        send_reset_link: "إرسال رابط الاستعادة",
        back_to_login: "العودة لتسجيل الدخول",
        email_sent: "تم إرسال البريد الإلكتروني!",
        email_sent_description: "تحقق من صندوق الوارد الخاص بك واتبع التعليمات لإعادة تعيين كلمة المرور.",
        sending: "جاري الإرسال...",
        email_required: "البريد الإلكتروني مطلوب",
        check_email: "تحقق من بريدك الإلكتروني",
        
        // صفحة OTP
        otp_title: "تأكيد الهوية",
        otp_subtitle: "أدخل رمز التحقق المرسل إليك",
        verify_otp: "تحقق من الرمز",
        invalid_otp: "رمز التحقق غير صحيح، يرجى المحاولة مرة أخرى",
        otp_success: "تم التحقق بنجاح!",
        no_code_received: "لم تستلم الرمز؟",
        resend_code: "إعادة الإرسال",
        verifying: "جاري التحقق...",
        otp_resent: "تم إعادة إرسال الرمز بنجاح",
        
        // صفحة اللاندينغ
        brand_name: "King Master",
        nav_home: "الرئيسية",
        nav_features: "المميزات",
        nav_pricing: "الأسعار",
        nav_screenshots: "لقطات الشاشة",
        nav_reviews: "آراء العملاء",
        nav_login: "تسجيل الدخول",
        nav_register: "حساب جديد",
        
        // قسم البطل
        hero_title: "أقوى أدوات التسويق الرقمي",
        hero_subtitle: "اكتشف مجموعة شاملة من الأدوات المتطورة لإدارة حملاتك التسويقية وزيادة أرباحك بطريقة احترافية وآمنة",
        hero_start_now: "ابدأ مجاناً الآن",
        hero_watch_demo: "شاهد العرض التوضيحي",
        stat_users: "مستخدم نشط",
        stat_campaigns: "حملة نجحت",
        stat_uptime: "وقت التشغيل",
        
        // المميزات
        features_title: "مميزات لا تُقاوم",
        features_subtitle: "نوفر لك كل ما تحتاجه لإدارة حملاتك التسويقية بكفاءة عالية ونتائج مضمونة",
        feature_whatsapp_title: "أدوات واتساب المتطورة",
        feature_whatsapp_desc: "إرسال جماعي، استخراج أرقام، إدارة المجموعات، والمزيد من الأدوات الاحترافية",
        feature_facebook_title: "تسويق فيسبوك الذكي",
        feature_facebook_desc: "إدارة الصفحات، جدولة المنشورات، تحليل الأداء، واستهداف دقيق للجمهور",
        feature_instagram_title: "نمو انستغرام السريع",
        feature_instagram_desc: "زيادة المتابعين، تحليل الهاشتاغات، جدولة المحتوى، وتقارير مفصلة",
        feature_analytics_title: "تحليلات متقدمة",
        feature_analytics_desc: "تقارير شاملة، إحصائيات دقيقة، وتحليل عميق لأداء حملاتك التسويقية",
        feature_security_title: "أمان مطلق",
        feature_security_desc: "تشفير عالي المستوى، حماية البيانات، وضمان الخصوصية التامة لجميع معلوماتك",
        feature_support_title: "دعم 24/7",
        feature_support_desc: "فريق دعم متخصص متاح على مدار الساعة لمساعدتك في تحقيق أهدافك",
        
        // الأسعار
        pricing_title: "خطط مرنة تناسب احتياجاتك",
        pricing_subtitle: "اختر الخطة المناسبة لك واستمتع بأقوى أدوات التسويق الرقمي",
        plan_basic_badge: "أساسية",
        plan_basic_title: "الباقة الأساسية",
        plan_pro_badge: "احترافية",
        plan_pro_title: "الباقة الاحترافية",
        plan_enterprise_badge: "مؤسسية",
        plan_enterprise_title: "الباقة المؤسسية",
        plan_monthly: "/شهرياً",
        plan_yearly: "/سنوياً",
        plan_choose: "اختر هذه الخطة",
        
        // مميزات الباقات
        feature_whatsapp_basic: "أدوات واتساب الأساسية",
        feature_facebook_basic: "إدارة فيسبوك البسيطة",
        feature_support_email: "دعم عبر البريد الإلكتروني",
        feature_reports_basic: "تقارير أساسية",
        feature_users_5: "حتى 5 مستخدمين",
        feature_whatsapp_pro: "أدوات واتساب المتقدمة",
        feature_facebook_pro: "إدارة فيسبوك الكاملة",
        feature_instagram_full: "أدوات انستغرام كاملة",
        feature_analytics_advanced: "تحليلات متقدمة",
        feature_support_priority: "دعم ذو أولوية",
        feature_users_unlimited: "مستخدمين غير محدود",
        feature_all_tools: "جميع الأدوات والمميزات",
        feature_custom_integration: "تكامل مخصص",
        feature_dedicated_support: "دعم مخصص",
        feature_white_label: "علامة تجارية مخصصة",
        feature_api_access: "وصول كامل للـ API",
        feature_training: "تدريب مجاني",
        
        annual_discount: "وفر 20% عند الاشتراك السنوي",
        view_annual_pricing: "عرض الأسعار السنوية",
        view_monthly_pricing: "عرض الأسعار الشهرية",
        
        // الفيديو
        video_title: "شاهد كيف يعمل النظام",
        video_subtitle: "تعرف على جميع مميزات النظام من خلال هذا العرض التوضيحي المفصل",
        
        // لقطات الشاشة
        screenshots_title: "لقطات من داخل النظام",
        screenshots_subtitle: "اطلع على واجهة المستخدم البديهية والتصميم العصري لأدواتنا",
        screenshot_dashboard: "لوحة التحكم الرئيسية",
        screenshot_whatsapp: "أدوات واتساب",
        screenshot_analytics: "التحليلات والتقارير",
        screenshot_campaigns: "إدارة الحملات",
        screenshot_settings: "الإعدادات",
        screenshot_mobile: "التطبيق المحمول",
        
        // آراء العملاء
        reviews_title: "ماذا يقول عملاؤنا",
        reviews_subtitle: "آراء حقيقية من عملاء راضين عن خدماتنا وأدواتنا المتطورة",
        review_role_marketer: "مسوق رقمي",
        review_role_owner: "صاحبة متجر إلكتروني",
        review_role_entrepreneur: "رائد أعمال",
        review_1: "أدوات رائعة ساعدتني في زيادة مبيعاتي بنسبة 300% خلال شهرين فقط. الدعم الفني ممتاز والواجهة سهلة الاستخدام.",
        review_2: "منصة متكاملة وشاملة لجميع احتياجاتي التسويقية. أصبحت إدارة حملاتي أسهل بكثير والنتائج مذهلة!",
        review_3: "استثمار يستحق كل ريال. الأدوات قوية والتقارير مفصلة، ساعدني في فهم جمهوري بشكل أفضل وزيادة التفاعل.",
        
        // الفوتر
        footer_company: "الشركة",
        footer_about: "من نحن",
        footer_careers: "الوظائف",
        footer_press: "الصحافة",
        footer_blog: "المدونة",
        footer_products: "المنتجات",
        footer_whatsapp_tools: "أدوات واتساب",
        footer_facebook_tools: "أدوات فيسبوك",
        footer_instagram_tools: "أدوات انستغرام",
        footer_analytics: "التحليلات",
        footer_support: "الدعم",
        footer_help_center: "مركز المساعدة",
        footer_contact: "اتصل بنا",
        footer_documentation: "التوثيق",
        footer_api: "واجهة برمجة التطبيقات",
        footer_legal: "قانوني",
        footer_privacy: "سياسة الخصوصية",
        footer_terms: "الشروط والأحكام",
        footer_cookies: "سياسة ملفات تعريف الارتباط",
        footer_security: "الأمان",
        footer_copyright: "© 2025 King Master. جميع الحقوق محفوظة."
    },
    
    en: {
                maneg_account: "Maneage Accounts",

        // Registration Page
        create_account: "Create New Account",
        create_account_subtitle: "Join us and enjoy the best digital marketing tools",
        full_name: "Full Name",
        email: "Email Address",
        password: "Password",
        phone: "Phone Number",
        timezone: "Time Zone",
        country: "Country",
        terms_agreement: "I agree to the",
        terms_conditions: "Terms and Conditions",
        privacy_policy: "Privacy Policy",
        create_account_btn: "Create Account",
        already_have_account: "Already have an account?",
        login: "Log In",
        
        // Placeholders
        enter_full_name: "Enter your full name",
        enter_email: "Enter your email address",
        enter_password: "Enter your password",
        enter_phone: "Enter your phone number",
        search_timezone: "Search for timezone...",
        auto_detect_country: "Will be auto-detected",
        select_timezone: "Select timezone",
        
        // Languages
        arabic: "Arabic",
        english: "English",
        french: "French",
        
        // Validation Messages
        fill_all_fields: "Please fill in all required fields",
        agree_to_terms: "You must agree to the terms and conditions",
        invalid_email: "Please enter a valid email address",
        password_too_short: "Password must be at least 6 characters long",
        account_created: "Account created successfully! Welcome to King Master",
        creating_account: "Creating account...",
        
        // Country Detection Messages
        detecting_country: "Detecting country...",
        country_detected: "Country detected successfully",
        country_not_detected: "Could not detect country",
        invalid_phone_format: "Invalid phone number format",
        
        // Login Page
        login_title: "Sign In",
        login_subtitle: "Enter your credentials to access your account",
        remember_me: "Remember me",
        forgot_password: "Forgot password?",
        login_btn: "Sign In",
        no_account: "Don't have an account?",
        create_account_link: "Create new account",
        logging_in: "Signing in...",
        login_success: "Login successful! Welcome back",
        
        // Forgot Password Page
        forgot_password_title: "Forgot Password?",
        forgot_password_subtitle: "Enter your email and we'll send you a password reset link",
        reset_password_info: "You will receive an email within a few minutes containing a password reset link. Check your inbox or spam folder.",
        send_reset_link: "Send Reset Link",
        back_to_login: "Back to Login",
        email_sent: "Email Sent!",
        email_sent_description: "Check your inbox and follow the instructions to reset your password.",
        sending: "Sending...",
        email_required: "Email is required",
        check_email: "Check your email",
        
        // OTP Page
        otp_title: "Verify Identity",
        otp_subtitle: "Enter the verification code sent to you",
        verify_otp: "Verify Code",
        invalid_otp: "Invalid verification code, please try again",
        otp_success: "Verification successful!",
        no_code_received: "Didn't receive the code?",
        resend_code: "Resend Code",
        verifying: "Verifying...",
        otp_resent: "Code resent successfully",
        
        // Landing Page
        brand_name: "King Master",
        nav_home: "Home",
        nav_features: "Features",
        nav_pricing: "Pricing",
        nav_screenshots: "Screenshots",
        nav_reviews: "Reviews",
        nav_login: "Login",
        nav_register: "Sign Up",
        
        // Hero section
        hero_title: "The Most Powerful Digital Marketing Tools",
        hero_subtitle: "Discover a comprehensive suite of advanced tools to manage your marketing campaigns and increase your profits professionally and securely",
        hero_start_now: "Start Free Now",
        hero_watch_demo: "Watch Demo",
        stat_users: "Active Users",
        stat_campaigns: "Successful Campaigns",
        stat_uptime: "Uptime",
        
        // Features
        features_title: "Irresistible Features",
        features_subtitle: "We provide everything you need to manage your marketing campaigns with high efficiency and guaranteed results",
        feature_whatsapp_title: "Advanced WhatsApp Tools",
        feature_whatsapp_desc: "Bulk messaging, number extraction, group management, and more professional tools",
        feature_facebook_title: "Smart Facebook Marketing",
        feature_facebook_desc: "Page management, post scheduling, performance analysis, and precise audience targeting",
        feature_instagram_title: "Fast Instagram Growth",
        feature_instagram_desc: "Increase followers, hashtag analysis, content scheduling, and detailed reports",
        feature_analytics_title: "Advanced Analytics",
        feature_analytics_desc: "Comprehensive reports, accurate statistics, and deep analysis of your marketing campaign performance",
        feature_security_title: "Absolute Security",
        feature_security_desc: "High-level encryption, data protection, and complete privacy assurance for all your information",
        feature_support_title: "24/7 Support",
        feature_support_desc: "Specialized support team available around the clock to help you achieve your goals",
        
        // Pricing
        pricing_title: "Flexible Plans to Suit Your Needs",
        pricing_subtitle: "Choose the right plan for you and enjoy the most powerful digital marketing tools",
        plan_basic_badge: "Basic",
        plan_basic_title: "Basic Plan",
        plan_pro_badge: "Professional",
        plan_pro_title: "Professional Plan",
        plan_enterprise_badge: "Enterprise",
        plan_enterprise_title: "Enterprise Plan",
        plan_monthly: "/monthly",
        plan_yearly: "/yearly",
        plan_choose: "Choose This Plan",
        
        // Plan features
        feature_whatsapp_basic: "Basic WhatsApp Tools",
        feature_facebook_basic: "Simple Facebook Management",
        feature_support_email: "Email Support",
        feature_reports_basic: "Basic Reports",
        feature_users_5: "Up to 5 Users",
        feature_whatsapp_pro: "Advanced WhatsApp Tools",
        feature_facebook_pro: "Complete Facebook Management",
        feature_instagram_full: "Full Instagram Tools",
        feature_analytics_advanced: "Advanced Analytics",
        feature_support_priority: "Priority Support",
        feature_users_unlimited: "Unlimited Users",
        feature_all_tools: "All Tools and Features",
        feature_custom_integration: "Custom Integration",
        feature_dedicated_support: "Dedicated Support",
        feature_white_label: "White Label",
        feature_api_access: "Full API Access",
        feature_training: "Free Training",
        
        annual_discount: "Save 20% with annual subscription",
        view_annual_pricing: "View Annual Pricing",
        view_monthly_pricing: "View Monthly Pricing",
        
        // Video
        video_title: "See How It Works",
        video_subtitle: "Learn about all system features through this detailed demo",
        
        // Screenshots
        screenshots_title: "Inside the System",
        screenshots_subtitle: "Explore the intuitive user interface and modern design of our tools",
        screenshot_dashboard: "Main Dashboard",
        screenshot_whatsapp: "WhatsApp Tools",
        screenshot_analytics: "Analytics & Reports",
        screenshot_campaigns: "Campaign Management",
        screenshot_settings: "Settings",
        screenshot_mobile: "Mobile App",
        
        // Reviews
        reviews_title: "What Our Customers Say",
        reviews_subtitle: "Real testimonials from satisfied customers about our advanced services and tools",
        review_role_marketer: "Digital Marketer",
        review_role_owner: "E-commerce Store Owner",
        review_role_entrepreneur: "Entrepreneur",
        review_1: "Amazing tools helped me increase my sales by 300% in just two months. Excellent technical support and easy-to-use interface.",
        review_2: "A comprehensive and complete platform for all my marketing needs. Managing my campaigns became much easier and the results are amazing!",
        review_3: "An investment worth every penny. Powerful tools and detailed reports helped me understand my audience better and increase engagement.",
        
        // Footer
        footer_company: "Company",
        footer_about: "About Us",
        footer_careers: "Careers",
        footer_press: "Press",
        footer_blog: "Blog",
        footer_products: "Products",
        footer_whatsapp_tools: "WhatsApp Tools",
        footer_facebook_tools: "Facebook Tools",
        footer_instagram_tools: "Instagram Tools",
        footer_analytics: "Analytics",
        footer_support: "Support",
        footer_help_center: "Help Center",
        footer_contact: "Contact Us",
        footer_documentation: "Documentation",
        footer_api: "API",
        footer_legal: "Legal",
        footer_privacy: "Privacy Policy",
        footer_terms: "Terms & Conditions",
        footer_cookies: "Cookie Policy",
        footer_security: "Security",
        footer_copyright: "© 2025 King Master. All rights reserved."
    },
    
    fr: {
                        maneg_account: "Maneage Accoudnts",

        // Page d'inscription
        create_account: "Créer un nouveau compte",
        create_account_subtitle: "Rejoignez-nous et profitez des meilleurs outils de marketing numérique",
        full_name: "Nom complet",
        email: "Adresse e-mail",
        password: "Mot de passe",
        phone: "Numéro de téléphone",
        timezone: "Fuseau horaire",
        country: "Pays",
        terms_agreement: "J'accepte les",
        terms_conditions: "Termes et Conditions",
        privacy_policy: "Politique de confidentialité",
        create_account_btn: "Créer un compte",
        already_have_account: "Vous avez déjà un compte?",
        login: "Se connecter",
        
        // Placeholders
        enter_full_name: "Entrez votre nom complet",
        enter_email: "Entrez votre adresse e-mail",
        enter_password: "Entrez votre mot de passe",
        enter_phone: "Entrez votre numéro de téléphone",
        search_timezone: "Rechercher un fuseau horaire...",
        auto_detect_country: "Sera détecté automatiquement",
        select_timezone: "Sélectionner le fuseau horaire",
        
        // Langues
        arabic: "Arabe",
        english: "Anglais",
        french: "Français",
        
        // Messages de validation
        fill_all_fields: "Veuillez remplir tous les champs requis",
        agree_to_terms: "Vous devez accepter les termes et conditions",
        invalid_email: "Veuillez entrer une adresse e-mail valide",
        password_too_short: "Le mot de passe doit contenir au moins 6 caractères",
        account_created: "Compte créé avec succès! Bienvenue chez King Master",
        creating_account: "Création du compte...",
        
        // Messages de détection de pays
        detecting_country: "Détection du pays...",
        country_detected: "Pays détecté avec succès",
        country_not_detected: "Impossible de détecter le pays",
        invalid_phone_format: "Format de numéro de téléphone invalide",
        
        // Page de connexion
        login_title: "Se connecter",
        login_subtitle: "Entrez vos identifiants pour accéder à votre compte",
        remember_me: "Se souvenir de moi",
        forgot_password: "Mot de passe oublié?",
        login_btn: "Se connecter",
        no_account: "Vous n'avez pas de compte?",
        create_account_link: "Créer un nouveau compte",
        logging_in: "Connexion en cours...",
        login_success: "Connexion réussie! Bienvenue",
        
        // Page mot de passe oublié
        forgot_password_title: "Mot de passe oublié?",
        forgot_password_subtitle: "Entrez votre email et nous vous enverrons un lien de réinitialisation",
        reset_password_info: "Vous recevrez un email dans quelques minutes contenant un lien de réinitialisation du mot de passe. Vérifiez votre boîte de réception ou vos spams.",
        send_reset_link: "Envoyer le lien de réinitialisation",
        back_to_login: "Retour à la connexion",
        email_sent: "Email envoyé!",
        email_sent_description: "Vérifiez votre boîte de réception et suivez les instructions pour réinitialiser votre mot de passe.",
        sending: "Envoi en cours...",
        email_required: "L'email est requis",
        check_email: "Vérifiez votre email",
        
        // Page OTP
        otp_title: "Vérifier l'identité",
        otp_subtitle: "Entrez le code de vérification qui vous a été envoyé",
        verify_otp: "Vérifier le code",
        invalid_otp: "Code de vérification invalide, veuillez réessayer",
        otp_success: "Vérification réussie!",
        no_code_received: "Vous n'avez pas reçu le code?",
        resend_code: "Renvoyer le code",
        verifying: "Vérification en cours...",
        otp_resent: "Code renvoyé avec succès",
        
        // Page d'accueil
        brand_name: "King Master",
        nav_home: "Accueil",
        nav_features: "Fonctionnalités",
        nav_pricing: "Tarifs",
        nav_screenshots: "Captures d'écran",
        nav_reviews: "Avis clients",
        nav_login: "Connexion",
        nav_register: "S'inscrire",
        
        // Section héro
        hero_title: "Les Outils de Marketing Numérique les Plus Puissants",
        hero_subtitle: "Découvrez une suite complète d'outils avancés pour gérer vos campagnes marketing et augmenter vos profits de manière professionnelle et sécurisée",
        hero_start_now: "Commencer Gratuitement",
        hero_watch_demo: "Voir la Démo",
        stat_users: "Utilisateurs Actifs",
        stat_campaigns: "Campagnes Réussies",
        stat_uptime: "Temps de Fonctionnement",
        
        // Fonctionnalités
        features_title: "Fonctionnalités Irrésistibles",
        features_subtitle: "Nous fournissons tout ce dont vous avez besoin pour gérer vos campagnes marketing avec une haute efficacité et des résultats garantis",
        feature_whatsapp_title: "Outils WhatsApp Avancés",
        feature_whatsapp_desc: "Envoi en masse, extraction de numéros, gestion de groupes, et plus d'outils professionnels",
        feature_facebook_title: "Marketing Facebook Intelligent",
        feature_facebook_desc: "Gestion des pages, planification des publications, analyse des performances, et ciblage précis de l'audience",
        feature_instagram_title: "Croissance Instagram Rapide",
        feature_instagram_desc: "Augmentation des followers, analyse des hashtags, planification du contenu, et rapports détaillés",
        feature_analytics_title: "Analyses Avancées",
        feature_analytics_desc: "Rapports complets, statistiques précises, et analyse approfondie des performances de vos campagnes marketing",
        feature_security_title: "Sécurité Absolue",
        feature_security_desc: "Chiffrement de haut niveau, protection des données, et assurance de confidentialité complète pour toutes vos informations",
        feature_support_title: "Support 24/7",
        feature_support_desc: "Équipe de support spécialisée disponible 24h/24 pour vous aider à atteindre vos objectifs",
        
        // Tarifs
        pricing_title: "Plans Flexibles Adaptés à Vos Besoins",
        pricing_subtitle: "Choisissez le plan qui vous convient et profitez des outils de marketing numérique les plus puissants",
        plan_basic_badge: "Basique",
        plan_basic_title: "Plan Basique",
        plan_pro_badge: "Professionnel",
        plan_pro_title: "Plan Professionnel",
        plan_enterprise_badge: "Entreprise",
        plan_enterprise_title: "Plan Entreprise",
        plan_monthly: "/mensuel",
        plan_yearly: "/annuel",
        plan_choose: "Choisir ce Plan",
        
        // Fonctionnalités des plans
        feature_whatsapp_basic: "Outils WhatsApp de Base",
        feature_facebook_basic: "Gestion Facebook Simple",
        feature_support_email: "Support Email",
        feature_reports_basic: "Rapports de Base",
        feature_users_5: "Jusqu'à 5 Utilisateurs",
        feature_whatsapp_pro: "Outils WhatsApp Avancés",
        feature_facebook_pro: "Gestion Facebook Complète",
        feature_instagram_full: "Outils Instagram Complets",
        feature_analytics_advanced: "Analyses Avancées",
        feature_support_priority: "Support Prioritaire",
        feature_users_unlimited: "Utilisateurs Illimités",
        feature_all_tools: "Tous les Outils et Fonctionnalités",
        feature_custom_integration: "Intégration Personnalisée",
        feature_dedicated_support: "Support Dédié",
        feature_white_label: "Marque Blanche",
        feature_api_access: "Accès API Complet",
        feature_training: "Formation Gratuite",
        
        annual_discount: "Économisez 20% avec un abonnement annuel",
        view_annual_pricing: "Voir les Tarifs Annuels",
        view_monthly_pricing: "Voir les Tarifs Mensuels",
        
        // Vidéo
        video_title: "Voir Comment Ça Marche",
        video_subtitle: "Découvrez toutes les fonctionnalités du système à travers cette démo détaillée",
        
        // Captures d'écran
        screenshots_title: "À l'Intérieur du Système",
        screenshots_subtitle: "Explorez l'interface utilisateur intuitive et le design moderne de nos outils",
        screenshot_dashboard: "Tableau de Bord Principal",
        screenshot_whatsapp: "Outils WhatsApp",
        screenshot_analytics: "Analyses et Rapports",
        screenshot_campaigns: "Gestion des Campagnes",
        screenshot_settings: "Paramètres",
        screenshot_mobile: "Application Mobile",
        
        // Avis
        reviews_title: "Ce Que Disent Nos Clients",
        reviews_subtitle: "Témoignages réels de clients satisfaits de nos services et outils avancés",
        review_role_marketer: "Marketeur Numérique",
        review_role_owner: "Propriétaire de Boutique E-commerce",
        review_role_entrepreneur: "Entrepreneur",
        review_1: "Outils fantastiques qui m'ont aidé à augmenter mes ventes de 300% en seulement deux mois. Excellent support technique et interface facile à utiliser.",
        review_2: "Une plateforme complète et globale pour tous mes besoins marketing. La gestion de mes campagnes est devenue beaucoup plus facile et les résultats sont incroyables!",
        review_3: "Un investissement qui vaut chaque centime. Outils puissants et rapports détaillés m'ont aidé à mieux comprendre mon audience et augmenter l'engagement.",
        
        // Pied de page
        footer_company: "Entreprise",
        footer_about: "À Propos",
        footer_careers: "Carrières",
        footer_press: "Presse",
        footer_blog: "Blog",
        footer_products: "Produits",
        footer_whatsapp_tools: "Outils WhatsApp",
        footer_facebook_tools: "Outils Facebook",
        footer_instagram_tools: "Outils Instagram",
        footer_analytics: "Analyses",
        footer_support: "Support",
        footer_help_center: "Centre d'Aide",
        footer_contact: "Nous Contacter",
        footer_documentation: "Documentation",
        footer_api: "API",
        footer_legal: "Légal",
        footer_privacy: "Politique de Confidentialité",
        footer_terms: "Conditions Générales",
        footer_cookies: "Politique des Cookies",
        footer_security: "Sécurité",
        footer_copyright: "© 2025 King Master. Tous droits réservés."
    }
};

// دالة الحصول على النص المترجم
function t(key, lang = null) {
    const currentLang = lang || getCurrentLanguage();
    return translations[currentLang]?.[key] || translations['ar'][key] || key;
}

// دالة الحصول على اللغة الحالية
function getCurrentLanguage() {
    return document.documentElement.lang || 'ar';
}

// دالة تطبيق الترجمات على الصفحة
function applyTranslations(lang = 'ar') {
    // تحديث العناصر التي تحتوي على data-i18n
    document.querySelectorAll('[data-i18n]').forEach(element => {
        const key = element.getAttribute('data-i18n');
        element.textContent = t(key, lang);
    });
    
    // تحديث placeholders
    document.querySelectorAll('[data-i18n-placeholder]').forEach(element => {
        const key = element.getAttribute('data-i18n-placeholder');
        element.placeholder = t(key, lang);
    });
    
    // تحديث خصائص HTML أخرى
    document.querySelectorAll('[data-i18n-title]').forEach(element => {
        const key = element.getAttribute('data-i18n-title');
        element.title = t(key, lang);
    });
    
    // تحديث معلومات الوثيقة
    document.documentElement.lang = lang;
}

// دالة تغيير اللغة (بدون تغيير الاتجاه)
function changeLanguage(lang) {
    // تطبيق الترجمات فقط
    applyTranslations(lang);
    
    // تحديث نص اللغة الحالية
    const currentLangElement = document.getElementById('current-language');
    if (currentLangElement) {
        currentLangElement.textContent = t(lang, lang);
    }
    
    // إعادة تعبئة المناطق الزمنية بناءً على اللغة الجديدة
    if (typeof populateTimezoneSelectWithSearch === 'function') {
        populateTimezoneSelectWithSearch('timezone', {
            defaultSelected: 'UTC+03:00',
            autoDetect: true,
            language: lang
        });
    }
    
    // حفظ اللغة المختارة
    localStorage.setItem('preferred-language', lang);
    
    // إغلاق قائمة اللغات
    const dropdown = document.getElementById('language-dropdown');
    if (dropdown) {
        dropdown.style.display = 'none';
    }
    
    // إظهار رسالة نجاح
    if (typeof Swal !== 'undefined') {
        const langNames = {
            'ar': 'العربية',
            'en': 'English',
            'fr': 'Français'
        };
        
        Swal.fire({
            icon: 'success',
            title: t('language_changed', lang) || 'تم تغيير اللغة!',
            text: `${langNames[lang]}`,
            timer: 2000,
            showConfirmButton: false,
            position: 'top-end',
            toast: true
        });
    }
}

// دالة تهيئة اللغة عند تحميل الصفحة
function initializeLanguage() {
    // الحصول على اللغة المحفوظة أو الافتراضية
    const savedLang = localStorage.getItem('preferred-language') || 'ar';
    changeLanguage(savedLang);
    
    // تحديث حالة القائمة المنسدلة
    document.querySelectorAll('[data-lang]').forEach(el => {
        el.classList.toggle('active', el.dataset.lang === savedLang);
    });
}

// تصدير الدوال للاستخدام العام
if (typeof window !== 'undefined') {
    window.t = t;
    window.changeLanguage = changeLanguage;
    window.applyTranslations = applyTranslations;
    window.initializeLanguage = initializeLanguage;
}