// Toggle Theme (Light/Dark)
function toggleTheme() {
    const body = document.body;
    body.classList.toggle('light-theme');
    
    // حفظ الثيم في localStorage
    const theme = body.classList.contains('light-theme') ? 'light' : 'dark';
    localStorage.setItem('theme', theme);
    
    // تأثير بصري عند التبديل
    const themeBtn = document.getElementById('theme-toggle');
    themeBtn.style.transform = 'rotate(360deg) scale(1.2)';
    setTimeout(() => {
        themeBtn.style.transform = '';
    }, 500);
}

// Toggle Language Menu
function toggleLanguageMenu() {
    const menu = document.getElementById('language-menu');
    menu.classList.toggle('active');
}

// تبديل اللغة
function changeLanguage(lang) {
    console.log('اللغة المختارة:', lang);
    
    // هنا يمكن إضافة كود لتحديث لغة الصفحة فعلياً
    // مثل: تغيير محتوى النصوص أو إعادة تحميل الصفحة بلغة مختلفة
    
    // حفظ اللغة في localStorage
    localStorage.setItem('language', lang);
    
    // إغلاق القائمة
    toggleLanguageMenu();
    
    // إشعار للمستخدم
    showNotification(`تم تغيير اللغة إلى ${lang}`);
}

// إظهار إشعار مؤقت
function showNotification(message) {
    const isMobile = window.innerWidth <= 480;

    // إنشاء عنصر الإشعار
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: ${isMobile ? '80px' : '100px'};
        ${isMobile ? 'left: 12px; right: 12px;' : 'right: 20px; max-width: 420px;'}
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
        padding: 15px 20px;
        border-radius: 12px;
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
        z-index: 9999;
        font-weight: 600;
        text-align: center;
        animation: slideInRight 0.5s ease;
    `;
    notification.textContent = message;

    document.body.appendChild(notification);

    // إزالة الإشعار بعد 3 ثوانٍ
    setTimeout(() => {
        notification.style.animation = 'slideOutRight 0.5s ease';
        setTimeout(() => notification.remove(), 500);
    }, 3000);
}

// Mobile Navbar (Hamburger)
function initMobileNav() {
    const navbar = document.getElementById('navbar');
    const toggle = document.getElementById('navToggle');
    const overlay = document.getElementById('navOverlay');

    if (!navbar || !toggle || !overlay) return;

    const icon = toggle.querySelector('i');

    const closeNav = () => {
        navbar.classList.remove('nav-open');
        overlay.classList.remove('active');
        toggle.setAttribute('aria-expanded', 'false');
        if (icon) icon.className = 'fas fa-bars';
    };

    const openNav = () => {
        navbar.classList.add('nav-open');
        overlay.classList.add('active');
        toggle.setAttribute('aria-expanded', 'true');
        if (icon) icon.className = 'fas fa-times';
    };

    toggle.addEventListener('click', (e) => {
        e.stopPropagation();
        if (navbar.classList.contains('nav-open')) closeNav();
        else openNav();
    });

    overlay.addEventListener('click', closeNav);

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') closeNav();
    });

    navbar.querySelectorAll('.nav-links a').forEach((a) => {
        a.addEventListener('click', closeNav);
    });

    window.addEventListener('resize', () => {
        if (window.innerWidth > 768) closeNav();
    });
}

// Navbar scroll effect
window.addEventListener('scroll', () => {
    const navbar = document.querySelector('.navbar');
    if (window.scrollY > 50) {
        navbar.classList.add('scrolled');
    } else {
        navbar.classList.remove('scrolled');
    }
});

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

// إغلاق قائمة اللغة عند النقر خارجها
document.addEventListener('click', (e) => {
    const languageDropdown = document.querySelector('.language-dropdown');
    const languageMenu = document.getElementById('language-menu');
    
    if (!languageDropdown.contains(e.target)) {
        languageMenu.classList.remove('active');
    }
});

// تحميل الثيم المحفوظ عند فتح الصفحة
window.addEventListener('DOMContentLoaded', () => {
    const savedTheme = localStorage.getItem('theme');
    const savedLanguage = localStorage.getItem('language');
    
    if (savedTheme === 'light') {
        document.body.classList.add('light-theme');
    }
    
    if (savedLanguage) {
        console.log('اللغة المحفوظة:', savedLanguage);
    }
});

// Intersection Observer للتأثيرات عند ظهور العناصر
const observerOptions = {
    threshold: 0.2,
    rootMargin: '0px 0px -100px 0px'
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.style.opacity = '1';
            entry.target.style.transform = 'translateY(0)';
        }
    });
}, observerOptions);

// تطبيق المراقبة على البطاقات
document.addEventListener('DOMContentLoaded', () => {
    const cards = document.querySelectorAll('.feature-card, .pricing-card, .testimonial-card, .screenshot-item');
    cards.forEach(card => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(30px)';
        card.style.transition = 'all 0.6s ease';
        observer.observe(card);
    });
});

// تحميل الباقات من API
async function loadPricingPackages() {
    try {
        const response = await fetch('get_packages.php');
        const data = await response.json();
        
        if (data.success && data.data && data.data.length > 0) {
            displayPackages(data.data);
        } else {
            // عرض رسالة في حالة عدم وجود باقات
            const pricingGrid = document.getElementById('pricingGrid');
            if (pricingGrid) {
                pricingGrid.innerHTML = `
                    <div style="text-align: center; grid-column: 1 / -1; padding: 60px;">
                        <i class="fas fa-info-circle" style="font-size: 48px; color: var(--warning);"></i>
                        <p style="margin-top: 20px; font-size: 18px;">${data.message || 'لا توجد باقات متاحة حالياً'}</p>
                    </div>
                `;
            }
        }
    } catch (error) {
        console.error('خطأ في الاتصال بالـ API:', error);
        const pricingGrid = document.getElementById('pricingGrid');
        if (pricingGrid) {
            pricingGrid.innerHTML = `
                <div style="text-align: center; grid-column: 1 / -1; padding: 60px;">
                    <i class="fas fa-exclamation-triangle" style="font-size: 48px; color: var(--warning);"></i>
                    <p style="margin-top: 20px; font-size: 18px;">حدث خطأ في تحميل الباقات. يرجى المحاولة لاحقاً.</p>
                </div>
            `;
        }
    }
}

// عرض الباقات في الصفحة
function displayPackages(packages) {
    const pricingGrid = document.getElementById('pricingGrid');
    
    if (!pricingGrid) return;
    
    pricingGrid.innerHTML = '';
    
    packages.forEach((pkg, index) => {
        // الحصول على المميزات باللغة العربية
        const features = pkg.features?.ar || [];
        const platforms = pkg.supported_platforms || [];
        
        const platformIcons = {
            'facebook': { icon: 'fa-facebook', class: 'platform-facebook' },
            'whatsapp': { icon: 'fa-whatsapp', class: 'platform-whatsapp' },
            'telegram': { icon: 'fa-telegram', class: 'platform-telegram' },
            'instagram': { icon: 'fa-instagram', class: 'platform-instagram' },
            'email': { icon: 'fa-envelope', class: 'platform-email' }
        };
        
        const platformBadges = platforms.map(platform => {
            const platformData = platformIcons[platform] || { icon: 'fa-globe', class: 'platform-business' };
            const platformName = platform.charAt(0).toUpperCase() + platform.slice(1);
            
            return `<span class="platform-badge ${platformData.class}">
                <i class="fab ${platformData.icon}"></i> ${platformName}
            </span>`;
        }).join('');
        
        const price = pkg.monthly_price || 0;
        const originalPrice = pkg.has_discount ? (price + (pkg.monthly_discount || 0)) : 0;
        const discountPercent = pkg.monthly_discount_percentage || 0;
        const hasDiscount = pkg.has_discount && discountPercent > 0;
        
        const popularBadge = pkg.is_popular ? 
            '<span class="popular-badge"><i class="fas fa-fire"></i> الأكثر طلباً</span>' : '';
        
        const card = `
            <div class="pricing-card">
                <div class="pricing-header">
                    ${popularBadge}
                    <div class="pricing-name">${pkg.name_ar || pkg.name}</div>
                    <div class="pricing-description">${pkg.description_ar || pkg.description || ''}</div>
                </div>
                
                <div class="pricing-body">
                    <div class="pricing-price">
                        ${hasDiscount ? `<span class="discount-badge"><i class="fas fa-tag"></i> خصم ${discountPercent.toFixed(0)}%</span>` : ''}
                        <div class="price-amount">
                            <span class="price-currency">$</span>${price.toFixed(2)}
                        </div>
                        ${hasDiscount ? `<div class="price-original">$${originalPrice.toFixed(2)}</div>` : ''}
                    </div>
                    
                    <div class="pricing-stats">
                        <div class="pricing-stat">
                            <div class="stat-icon accounts"><i class="fas fa-users"></i></div>
                            <div class="stat-value">${pkg.accounts || 1}</div>
                            <div class="stat-label">حسابات</div>
                        </div>
                        <div class="pricing-stat">
                            <div class="stat-icon messages"><i class="fas fa-paper-plane"></i></div>
                            <div class="stat-value">${pkg.messages || 0}</div>
                            <div class="stat-label">رسائل</div>
                        </div>
                        <div class="pricing-stat">
                            <div class="stat-icon points"><i class="fas fa-coins"></i></div>
                            <div class="stat-value">${pkg.points || 0}</div>
                            <div class="stat-label">نقاط</div>
                        </div>
                    </div>
                    
                    <div class="pricing-platforms">
                        ${platformBadges}
                    </div>
                    
                    <div class="pricing-features">
                        ${features.map(feature => `
                            <div class="pricing-feature">
                                <i class="fas fa-check-circle"></i>
                                <span>${feature}</span>
                            </div>
                        `).join('')}
                    </div>
                    
                    <button class="pricing-button" onclick="orderPackage(${pkg.id})">
                        <i class="fas fa-shopping-cart"></i> اطلب الآن
                    </button>
                </div>
            </div>
        `;
        
        pricingGrid.innerHTML += card;
    });
    
    // إعادة تطبيق Observer على البطاقات الجديدة
    const newCards = pricingGrid.querySelectorAll('.pricing-card');
    newCards.forEach(card => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(30px)';
        card.style.transition = 'all 0.6s ease';
        observer.observe(card);
    });
}

// دالة الطلب
function orderPackage(packageId) {
    // التحقق من تسجيل الدخول
    // يمكن إضافة كود للتحقق من الجلسة هنا
    
    // إعادة التوجيه لصفحة الطلب
    window.location.href = `order.php?package_id=${packageId}`;
}

// إنشاء النجوم في الخلفية
function createStars() {
    const starsContainer = document.getElementById('stars-container');
    if (!starsContainer) return;

    const numberOfStars = window.innerWidth < 768 ? 60 : 150;
    
    for (let i = 0; i < numberOfStars; i++) {
        const star = document.createElement('div');
        star.className = 'star';
        
        // حجم عشوائي للنجمة
        const size = Math.random() * 3 + 1;
        star.style.width = size + 'px';
        star.style.height = size + 'px';
        
        // موقع عشوائي
        star.style.left = Math.random() * 100 + '%';
        star.style.top = Math.random() * 100 + '%';
        
        // مدة الأنيميشن العشوائية
        const duration = Math.random() * 3 + 2;
        star.style.animationDuration = duration + 's';
        
        // تأخير عشوائي
        const delay = Math.random() * 3;
        star.style.animationDelay = delay + 's';
        
        starsContainer.appendChild(star);
    }
}

// عداد متحرك للأرقام
function animateCounter(element, target, suffix, duration = 2000) {
    const start = 0;
    const increment = target / (duration / 16);
    let current = start;
    
    const timer = setInterval(() => {
        current += increment;
        if (current >= target) {
            current = target;
            clearInterval(timer);
        }
        
        // تنسيق الرقم
        let displayValue;
        if (suffix === 'K+') {
            displayValue = Math.floor(current / 1000);
        } else if (suffix === 'M+') {
            displayValue = (current / 1000000).toFixed(1);
        } else if (suffix === '%') {
            displayValue = current.toFixed(1);
        } else {
            displayValue = Math.floor(current);
        }
        
        element.textContent = displayValue + suffix;
    }, 16);
}

// بدء العداد عند ظهور العنصر
function startCounters() {
    const statNumbers = document.querySelectorAll('.stat-number[data-target]');
    
    const observerOptions = {
        threshold: 0.5,
        rootMargin: '0px'
    };
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting && !entry.target.classList.contains('counted')) {
                entry.target.classList.add('counted');
                const target = parseFloat(entry.target.getAttribute('data-target'));
                const suffix = entry.target.getAttribute('data-suffix') || '';
                animateCounter(entry.target, target, suffix);
            }
        });
    }, observerOptions);
    
    statNumbers.forEach(stat => observer.observe(stat));
}

// YouTube Video Loader
function loadVideo() {
    const placeholder = document.getElementById('video-placeholder');
    const iframe = document.getElementById('youtube-video');
    
    // ضع رابط فيديو YouTube هنا (غير VIDEO_ID برقم الفيديو)
    const videoId = 'LW1lbgcUQHE'; // مثال - غير هذا برقم الفيديو الحقيقي
    const videoUrl = `https://www.youtube.com/embed/${videoId}?autoplay=1&rel=0`;
    
    // إخفاء الـ placeholder وإظهار الفيديو
    placeholder.style.display = 'none';
    iframe.src = videoUrl;
    iframe.style.display = 'block';
}

// FAQ Accordion
function initFAQ() {
    const faqItems = document.querySelectorAll('.faq-item');
    
    faqItems.forEach(item => {
        const question = item.querySelector('.faq-question');
        
        question.addEventListener('click', () => {
            // إغلاق جميع العناصر الأخرى
            const isActive = item.classList.contains('active');
            
            faqItems.forEach(otherItem => {
                otherItem.classList.remove('active');
            });
            
            // فتح العنصر الحالي إذا لم يكن مفتوحًا
            if (!isActive) {
                item.classList.add('active');
            }
        });
    });
    
    // فتح أول سؤال بشكل افتراضي
    if (faqItems.length > 0) {
        faqItems[0].classList.add('active');
    }
}

// تحميل الباقات عند تحميل الصفحة
document.addEventListener('DOMContentLoaded', () => {
    createStars();
    startCounters();
    loadPricingPackages();
    initFAQ();
    initMobileNav();
});

// إضافة CSS للأنيميشنات
const style = document.createElement('style');
style.textContent = `
    @keyframes slideInRight {
        from {
            transform: translateX(400px);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOutRight {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(400px);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);
