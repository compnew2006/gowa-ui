/**
 * King Master Dashboard v2.0
 * الملف الرئيسي للوظائف التفاعلية
 */

(function installCsrfFetchGuard() {
    if (window.__kmCsrfFetchGuardInstalled || typeof window.fetch !== 'function') return;
    window.__kmCsrfFetchGuardInstalled = true;
    const nativeFetch = window.fetch.bind(window);

    window.fetch = function(input, init = {}) {
        const url = typeof input === 'string' ? input : (input && input.url) || '';
        const method = String(init.method || (input && input.method) || 'GET').toUpperCase();
        const isUnsafe = ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method);
        const isSameOrigin = !/^https?:\/\//i.test(url) || url.startsWith(window.location.origin);
        const token = window.KM_CSRF_TOKEN || document.querySelector('meta[name="csrf-token"]')?.content || '';

        if (isUnsafe && isSameOrigin && token) {
            init = { ...init };
            const headers = new Headers(init.headers || (input && input.headers) || {});
            headers.set('X-CSRF-Token', token);
            init.headers = headers;

            if (init.body instanceof FormData && !init.body.has('csrf_token')) {
                init.body.append('csrf_token', token);
            } else if (typeof init.body === 'string' && headers.get('Content-Type')?.includes('application/json')) {
                try {
                    const parsed = JSON.parse(init.body || '{}');
                    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed) && !parsed.csrf_token) {
                        parsed.csrf_token = token;
                        init.body = JSON.stringify(parsed);
                    }
                } catch (e) {}
            }
        }

        return nativeFetch(input, init);
    };
})();

// عناصر DOM
const htmlElement = document.documentElement;
const body = document.body;
const sidebar = document.getElementById('sidebar');
const sidebarToggle = document.getElementById('sidebar-toggle');
const mobileSidebarToggle = document.getElementById('mobile-sidebar-toggle');
const mainContent = document.querySelector('.main-content');
const themeToggle = document.getElementById('theme-toggle');
const languageDropdown = document.querySelector('.language-dropdown');
const userDropdown = document.querySelector('.user-dropdown-menu');

// اللغة الحالية
let currentLanguage = localStorage.getItem('language') || 'ar';

// نسخ الرسوم البيانية
let balanceChart, pointsUsageChart, toolsUsageChart;

/**
 * تبديل حالة الشريط الجانبي
 */
function toggleSidebar() {
    sidebar.classList.toggle('collapsed');
    
    // حفظ حالة الشريط الجانبي
    const isCollapsed = sidebar.classList.contains('collapsed');
    localStorage.setItem('sidebarCollapsed', isCollapsed);
}

/**
 * تبديل الشريط الجانبي على الأجهزة المحمولة
 */
function toggleMobileSidebar() {
    sidebar.classList.toggle('active');

    const isActive = sidebar.classList.contains('active');
    // تحريك الصفحة عند فتح القائمة على الموبايل
    document.body.classList.toggle('sidebar-open', isActive);
    
    // إضافة/إزالة طبقة التعتيم عند فتح/إغلاق القائمة
    const overlay = document.querySelector('.mobile-overlay');
    if (isActive) {
        if (!overlay) {
            const newOverlay = document.createElement('div');
            newOverlay.className = 'mobile-overlay';
            document.body.appendChild(newOverlay);
            
            // إغلاق القائمة عند النقر على الطبقة
            newOverlay.addEventListener('click', toggleMobileSidebar);
        }
    } else {
        if (overlay) {
            overlay.remove();
        }
    }
}

/**
 * تبديل وضع السمة (داكن/فاتح)
 */
function toggleTheme() {
    body.classList.toggle('light-theme');
    
    // تحديث أيقونة السمة
    const themeIcon = themeToggle.querySelector('i');
    if (body.classList.contains('light-theme')) {
        themeIcon.className = 'fas fa-moon';
        localStorage.setItem('theme', 'light');
    } else {
        themeIcon.className = 'fas fa-sun';
        localStorage.setItem('theme', 'dark');
    }
    
    // إعادة تهيئة الرسوم البيانية لتتناسب مع السمة الجديدة
    initCharts();
}

/**
 * التحقق من السمة المحفوظة
 */
function checkTheme() {
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'light') {
        body.classList.add('light-theme');
        const themeIcon = themeToggle.querySelector('i');
        themeIcon.className = 'fas fa-moon';
    }
}

/**
 * تهيئة القوائم المنسدلة
 */
function initDropdowns() {
    // قائمة اللغة
    const languageBtn = document.querySelector('.language-btn');
    if (languageBtn) {
        languageBtn.addEventListener('click', function(e) {
            e.stopPropagation();
            languageDropdown.classList.toggle('active');
            
            // إخفاء القائمة المنسدلة الأخرى
            if (userDropdown) {
                userDropdown.classList.remove('active');
            }
        });
    }
    
    // قائمة المستخدم
    const userDropdownBtn = document.querySelector('.user-dropdown-btn');
    if (userDropdownBtn) {
        userDropdownBtn.addEventListener('click', function(e) {
            e.stopPropagation();
            userDropdown.classList.toggle('active');
            
            // إخفاء القائمة المنسدلة الأخرى
            if (languageDropdown) {
                languageDropdown.classList.remove('active');
            }
        });
    }
    
    // قائمة المسؤول في Navbar
    const navbarDropdowns = document.querySelectorAll('.top-navbar .dropdown');
    navbarDropdowns.forEach(dropdown => {
        const toggle = dropdown.querySelector('.dropdown-toggle');
        if (toggle) {
            toggle.addEventListener('click', function(e) {
                e.preventDefault();
                e.stopPropagation();
                
                // إغلاق القوائم الأخرى
                navbarDropdowns.forEach(other => {
                    if (other !== dropdown) {
                        other.classList.remove('active');
                    }
                });
                
                // تبديل القائمة الحالية
                dropdown.classList.toggle('active');
            });
        }
    });
    
    // إغلاق القوائم المنسدلة عند النقر خارجها
    document.addEventListener('click', function() {
        if (languageDropdown) {
            languageDropdown.classList.remove('active');
        }
        if (userDropdown) {
            userDropdown.classList.remove('active');
        }
        navbarDropdowns.forEach(dropdown => {
            dropdown.classList.remove('active');
        });
    });
}

/**
 * تغيير اللغة (بدون تغيير الاتجاه)
 * @param {string} lang - رمز اللغة (ar, en, fr)
 */
function changeLanguage(lang, isAutoInit = false) {
    currentLanguage = lang;
    localStorage.setItem('language', lang);
    
    // تحديث اللغة فقط بدون تغيير الاتجاه
    htmlElement.setAttribute('lang', lang);
    
    // تحديث عرض اللغة الحالية
    const currentLanguageText = document.getElementById("current-language");
    if (currentLanguageText) {
        currentLanguageText.textContent = translations[lang].current_language_display;
    }

    // تحديث جميع العناصر التي تحتوي على سمة data-i18n
    const elements = document.querySelectorAll('[data-i18n]');
    elements.forEach(element => {
        const key = element.getAttribute('data-i18n');
        if (translations[lang][key]) {
            element.textContent = translations[lang][key];
        }
    });
    
    // تحديث العناصر التي تحتوي على سمة data-i18n-placeholder
    const placeholders = document.querySelectorAll('[data-i18n-placeholder]');
    placeholders.forEach(element => {
        const key = element.getAttribute('data-i18n-placeholder');
        if (translations[lang][key]) {
            element.setAttribute("placeholder", translations[lang][key]);
        }
    });
    
    // إعادة تهيئة الرسوم البيانية باللغة الجديدة
    initCharts();
    
    // إغلاق قائمة اللغات
    if (languageDropdown) {
        languageDropdown.classList.remove('active');
    }
    
    // إظهار رسالة وإعادة تحميل الصفحة فقط إذا لم يكن تهيئة تلقائية
    if (!isAutoInit) {
        if (typeof Swal !== 'undefined') {
            const langNames = {
                'ar': 'العربية',
                'en': 'English',
                'fr': 'Français'
            };
            
            Swal.fire({
                icon: 'success',
                title: 'تم تغيير اللغة!',
                text: langNames[lang],
                timer: 1000,
                showConfirmButton: false,
                position: 'center',
                toast: false,
                didClose: () => {
                    // إعادة تحميل الصفحة
                    window.location.reload();
                }
            });
        } else {
            // إذا لم يكن SweetAlert متاح، إعادة التحميل مباشرة
            window.location.reload();
        }
    }
}

/**
 * تبديل قائمة اللغات
 */
function toggleLanguageDropdown() {
    const dropdown = document.getElementById('language-dropdown');
    if (dropdown) {
        dropdown.classList.toggle('active');
    }
}

/**
 * تطبيق تنسيقات الاتجاه
 * @param {string} direction - الاتجاه (rtl, ltr)
 */
function applyDirectionStyles(direction) {
    const isRTL = direction === 'rtl';
    
    // تحديث اتجاه الرسوم البيانية
    updateChartsDirection(isRTL);
}

/**
 * تحديث اتجاه الرسوم البيانية
 * @param {boolean} isRTL - هل الاتجاه من اليمين إلى اليسار
 */
function updateChartsDirection(isRTL) {
    if (balanceChart) {
        balanceChart.updateOptions({
            chart: {
                dir: isRTL ? 'rtl' : 'ltr'
            },
            yaxis: {
                reversed: isRTL
            }
        });
    }
    
    if (pointsUsageChart) {
        pointsUsageChart.updateOptions({
            chart: {
                dir: isRTL ? 'rtl' : 'ltr'
            },
            xaxis: {
                position: isRTL ? 'top' : 'bottom'
            },
            yaxis: {
                reversed: isRTL
            }
        });
    }
    
    if (toolsUsageChart) {
        toolsUsageChart.updateOptions({
            chart: {
                dir: isRTL ? 'rtl' : 'ltr'
            }
        });
    }
}

/**
 * تطبيق اللغة المحفوظة
 */
function applySavedLanguage() {
    const savedLanguage = localStorage.getItem('language') || 'ar';
    changeLanguage(savedLanguage, true);
    
    // تحديث العنصر النشط في قائمة اللغات
    const languageLinks = document.querySelectorAll('.language-dropdown a');
    languageLinks.forEach(link => {
        link.classList.remove('active');
        if (link.getAttribute('data-lang') === savedLanguage) {
            link.classList.add('active');
        }
    });
}

/**
 * تهيئة الرسوم البيانية
 */
function initCharts() {
    // تدمير الرسوم البيانية الموجودة إذا كانت موجودة
    if (balanceChart) { try{ balanceChart.destroy(); }catch(_){} balanceChart = null; }
    if (pointsUsageChart) { try{ pointsUsageChart.destroy(); }catch(_){} pointsUsageChart = null; }
    if (toolsUsageChart) { try{ toolsUsageChart.destroy(); }catch(_){} toolsUsageChart = null; }

    // مراجع الحاويات (قد لا تكون موجودة في بعض الصفحات)
    const elBalance = document.getElementById('balance-chart');
    const elPoints = document.getElementById('points-usage-chart');
    const elTools = document.getElementById('tools-usage-chart');

    // إذا لم توجد أي حاوية، لا شيء لنرسمه
    if (!elBalance && !elPoints && !elTools) return;

    // رسم بياني للرصيد
    const balanceChartOptions = {
        series: [{
            name: translations[currentLanguage].your_balance || 'رصيدك',
            data: [350, 420, 380, 500, 450, 550, 500]
        }],
        chart: {
            height: 150,
            type: 'area',
            toolbar: {
                show: false
            },
            sparkline: {
                enabled: true
            },
            fontFamily: currentLanguage === 'ar' ? 'Cairo, sans-serif' : 'Roboto, sans-serif',
            dir: currentLanguage === 'ar' ? 'rtl' : 'ltr'
        },
        colors: ['#ffffff'],
        dataLabels: {
            enabled: false
        },
        stroke: {
            curve: 'smooth',
            width: 2
        },
        fill: {
            type: 'gradient',
            gradient: {
                shadeIntensity: 1,
                opacityFrom: 0.7,
                opacityTo: 0.3,
                stops: [0, 90, 100]
            }
        },
        xaxis: {
            categories: [
                translations[currentLanguage].sunday || 'الأحد', 
                translations[currentLanguage].monday || 'الإثنين', 
                translations[currentLanguage].tuesday || 'الثلاثاء', 
                translations[currentLanguage].wednesday || 'الأربعاء', 
                translations[currentLanguage].thursday || 'الخميس', 
                translations[currentLanguage].friday || 'الجمعة', 
                translations[currentLanguage].saturday || 'السبت'
            ],
            labels: {
                show: false
            },
            axisBorder: {
                show: false
            },
            axisTicks: {
                show: false
            }
        },
        yaxis: {
            labels: {
                show: false
            },
            reversed: currentLanguage === 'ar'
        },
        tooltip: {
            x: {
                show: false
            }
        },
        grid: {
            show: false
        }
    };
    
    if (elBalance){
        balanceChart = new ApexCharts(elBalance, balanceChartOptions);
        balanceChart.render();
    }
    
    // رسم بياني لاستخدام النقاط
    const pointsUsageChartOptions = {
        series: [{
            name: translations[currentLanguage].points_usage || 'استخدام النقاط',
            data: [100, 55, 57, 56, 61, 58, 63, 60, 66]
        }],
        chart: {
            type: 'bar',
            height: 250,
            toolbar: {
                show: false
            },
            fontFamily: currentLanguage === 'ar' ? 'Cairo, sans-serif' : 'Roboto, sans-serif',
            dir: currentLanguage === 'ar' ? 'rtl' : 'ltr'
        },
        plotOptions: {
            bar: {
                horizontal: false,
                columnWidth: '55%',
                borderRadius: 5
            },
        },
        colors: ['#6c5ce7'],
        dataLabels: {
            enabled: false
        },
        stroke: {
            show: true,
            width: 2,
            colors: ['transparent']
        },
        xaxis: {
            categories: [
                translations[currentLanguage].january || 'يناير', 
                translations[currentLanguage].february || 'فبراير', 
                translations[currentLanguage].march || 'مارس', 
                translations[currentLanguage].april || 'أبريل', 
                translations[currentLanguage].may || 'مايو', 
                translations[currentLanguage].june || 'يونيو', 
                translations[currentLanguage].july || 'يوليو', 
                translations[currentLanguage].august || 'أغسطس', 
                translations[currentLanguage].september || 'سبتمبر'
            ],
            position: currentLanguage === 'ar' ? 'top' : 'bottom',
            labels: {
                style: {
                    colors: body.classList.contains('light-theme') ? '#2d3436' : '#f5f6fa',
                    fontFamily: currentLanguage === 'ar' ? 'Cairo, sans-serif' : 'Roboto, sans-serif'
                }
            }
        },
        yaxis: {
            title: {
                text: translations[currentLanguage].points || 'النقاط',
                style: {
                    color: body.classList.contains('light-theme') ? '#2d3436' : '#f5f6fa',
                    fontFamily: currentLanguage === 'ar' ? 'Cairo, sans-serif' : 'Roboto, sans-serif'
                }
            },
            reversed: currentLanguage === 'ar',
            labels: {
                style: {
                    colors: body.classList.contains('light-theme') ? '#2d3436' : '#f5f6fa',
                    fontFamily: currentLanguage === 'ar' ? 'Cairo, sans-serif' : 'Roboto, sans-serif'
                }
            }
        },
        fill: {
            opacity: 1
        },
        tooltip: {
            y: {
                formatter: function (val) {
                    return val + " " + (translations[currentLanguage].points || 'نقطة')
                }
            }
        },
        grid: {
            borderColor: body.classList.contains('light-theme') ? '#e2e8f0' : '#1e2a45',
            strokeDashArray: 5
        }
    };
    
    if (elPoints){
        pointsUsageChart = new ApexCharts(elPoints, pointsUsageChartOptions);
        pointsUsageChart.render();
    }
    
    // رسم بياني لاستخدام الأدوات
    const toolsUsageChartOptions = {
        series: [44, 55, 13, 43],
        chart: {
            type: 'donut',
            height: 250,
            fontFamily: currentLanguage === 'ar' ? 'Cairo, sans-serif' : 'Roboto, sans-serif',
            dir: currentLanguage === 'ar' ? 'rtl' : 'ltr'
        },
        labels: [
            translations[currentLanguage].whatsapp_tools || 'واتساب', 
            translations[currentLanguage].facebook_tools || 'فيسبوك', 
            translations[currentLanguage].instagram_tools || 'انستغرام', 
            translations[currentLanguage].telegram_tools || 'تليجرام'
        ],
        colors: ['#25D366', '#1877F2', '#E4405F', '#0088cc'],
        plotOptions: {
            pie: {
                donut: {
                    size: '65%'
                }
            }
        },
        dataLabels: {
            enabled: false
        },
        legend: {
            position: 'bottom',
            horizontalAlign: 'center',
            fontFamily: currentLanguage === 'ar' ? 'Cairo, sans-serif' : 'Roboto, sans-serif',
            labels: {
                colors: body.classList.contains('light-theme') ? '#2d3436' : '#f5f6fa'
            }
        },
        responsive: [{
            breakpoint: 480,
            options: {
                chart: {
                    width: 200
                },
                legend: {
                    position: 'bottom'
                }
            }
        }]
    };
    
    if (elTools){
        toolsUsageChart = new ApexCharts(elTools, toolsUsageChartOptions);
        toolsUsageChart.render();
    }
}

/**
 * إضافة تأثيرات حركية للعناصر
 */
function addAnimations() {
    const welcomeCard = document.querySelector('.welcome-card');
    const balanceCard = document.querySelector('.balance-card');
    const statsCards = document.querySelectorAll('.stats-card');
    const dashboardCards = document.querySelectorAll('.dashboard-card');
    const quickActionCards = document.querySelectorAll('.quick-action-card');
    
    if (welcomeCard) {
        welcomeCard.classList.add('fade-in');
    }
    
    if (balanceCard) {
        balanceCard.classList.add('slide-in-right');
    }
    
    statsCards.forEach((card, index) => {
        card.classList.add('slide-in-up');
        card.style.animationDelay = `${index * 0.1}s`;
    });
    
    dashboardCards.forEach((card, index) => {
        card.classList.add('fade-in');
        card.style.animationDelay = `${0.3 + index * 0.1}s`;
    });
    
    quickActionCards.forEach((card, index) => {
        card.classList.add('slide-in-up');
        card.style.animationDelay = `${0.5 + index * 0.1}s`;
    });
}

/**
 * تهيئة الصفحة عند تحميلها
 */
document.addEventListener('DOMContentLoaded', function() {
    // التحقق من حالة الشريط الجانبي المحفوظة
    const sidebarCollapsed = localStorage.getItem('sidebarCollapsed') === 'true';
    if (sidebarCollapsed) {
        sidebar.classList.add('collapsed');
    }
    
    // تهيئة الوظائف
    checkTheme();
    applySavedLanguage();
    initDropdowns();
    initCharts();
    addAnimations();
    initToolCardCollapse();
    
    // إضافة مستمعي الأحداث
    if (sidebarToggle) {
        sidebarToggle.addEventListener('click', toggleSidebar);
    }
    
    if (mobileSidebarToggle) {
        mobileSidebarToggle.addEventListener('click', toggleMobileSidebar);
    }
    
    if (themeToggle) {
        themeToggle.addEventListener('click', toggleTheme);
    }
    
    // مستمعي أحداث تغيير اللغة
    const languageItems = document.querySelectorAll('.language-item');
    languageItems.forEach(item => {
        item.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            
            // الحصول على اللؾة من onclick attribute
            const onclickAttr = this.getAttribute('onclick');
            if (onclickAttr) {
                const langMatch = onclickAttr.match(/changeLanguage\('(\w+)'\)/);
                if (langMatch && langMatch[1]) {
                    const lang = langMatch[1];
                    changeLanguage(lang);
                }
            }
        });
    });
    
    // إغلاق الشريط الجانبي عند النقر خارجه على الأجهزة المحمولة
    document.addEventListener('click', function(e) {
        if (window.innerWidth < 992) {
            if (!sidebar.contains(e.target) && !mobileSidebarToggle.contains(e.target) && sidebar.classList.contains('active')) {
                toggleMobileSidebar();
            }
        }
    });
    
    // التعامل مع تغيير حجم النافذة
    window.addEventListener('resize', function() {
        if (window.innerWidth < 992) {
            sidebar.classList.remove('collapsed');
        }
    });
});


/**
 * تهيئة وظيفة طي/فتح بطاقات الأدوات
 */
function initToolCardCollapse() {
    document.querySelectorAll(".toggle-card-body").forEach(button => {
        button.addEventListener("click", function(e) {
            e.preventDefault();
            e.stopPropagation();
            const targetId = this.dataset.target;
            const targetElement = document.querySelector(targetId);
            const container = this.closest('.tools-page-content') || document;

            if (!targetElement) return;

            const isOpening = targetElement.classList.contains('collapse');

            // Close all bodies in container and reset icons
            container.querySelectorAll('.card-body').forEach(el => {
                if (el !== targetElement){ el.classList.add('collapse'); el.classList.remove('show'); }
            });
            container.querySelectorAll('.toggle-card-body i').forEach(ic => {
                ic.classList.remove('fa-chevron-up');
                ic.classList.add('fa-chevron-down');
            });

            // Toggle target
            if (isOpening){
                targetElement.classList.remove('collapse');
                targetElement.classList.add('show');
                const ic = this.querySelector('i'); if (ic){ ic.classList.remove('fa-chevron-down'); ic.classList.add('fa-chevron-up'); }
            } else {
                targetElement.classList.add('collapse');
                targetElement.classList.remove('show');
                const ic = this.querySelector('i'); if (ic){ ic.classList.remove('fa-chevron-up'); ic.classList.add('fa-chevron-down'); }
            }
        });
    });
}

