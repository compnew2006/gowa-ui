<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'];

$page_title = "الباقات المتاحة | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<style>
    .packages-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }

    .packages-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
    }

    .packages-title {
        font-size: 28px;
        font-weight: 800;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 12px;
        font-family: 'Cairo', sans-serif;
    }

    .packages-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: iconFloat 3s ease-in-out infinite;
    }

    @keyframes iconFloat {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-10px); }
    }

    .packages-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 25px;
        margin-top: 30px;
    }

    .package-card {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 25px;
        border: 1px solid var(--border-color);
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
        animation: cardSlideUp 0.5s ease;
        display: flex;
        flex-direction: column;
    }

    @keyframes cardSlideUp {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .package-card:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3);
    }

    .package-card.popular {
        border: 2px solid #667eea;
    }

    .package-card.popular::before {
        content: '⭐ الأكثر شعبية';
        position: absolute;
        top: 20px;
        right: -35px;
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
        padding: 8px 40px;
        transform: rotate(45deg);
        font-size: 12px;
        font-weight: 700;
        box-shadow: 0 3px 10px rgba(0,0,0,0.2);
    }

    .package-name {
        font-size: 24px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 15px;
        text-align: center;
        font-family: 'Cairo', sans-serif;
    }

    .package-description {
        text-align: center;
        color: var(--text-secondary);
        margin-bottom: 25px;
        font-size: 14px;
        line-height: 1.6;
        font-family: 'Cairo', sans-serif;
    }

    .package-features li span {
        color: var(--text-primary);
    }
        
        .package-price {
            text-align: center;
            margin-bottom: 30px;
        }
        
        .price-amount {
            font-size: 48px;
            font-weight: 800;
            color: #667eea;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 5px;
        }
        
        .price-currency {
            font-size: 20px;
            color: var(--text-secondary);
        }
        
        .original-price {
            text-decoration: line-through;
            color: var(--text-secondary);
            opacity: 0.7;
            font-size: 20px;
            margin-top: 5px;
        }
        
        .discount-badge {
            display: inline-block;
            background: #00b894;
            color: white;
            padding: 5px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 700;
            margin-top: 10px;
        }
        
        .package-features {
            list-style: none;
            margin-bottom: 30px;
        }
        
        .package-features li {
            padding: 12px 0;
            border-bottom: 1px solid var(--border-color);
            color: var(--text-primary);
            display: flex;
            align-items: center;
            gap: 12px;
            transition: all 0.3s ease;
        }

        .package-features li:hover {
            padding-right: 10px;
            background: linear-gradient(90deg, transparent, rgba(102, 126, 234, 0.05));
        }
        
        .package-features li:last-child {
            border-bottom: none;
        }
        
        .package-features li i {
            font-size: 18px;
            min-width: 24px;
            height: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: linear-gradient(135deg, #00b894, #00d2d3);
            color: white;
            border-radius: 50%;
            padding: 4px;
            animation: featureIconPulse 2s ease-in-out infinite;
            transition: all 0.3s ease;
        }

        @keyframes featureIconPulse {
            0%, 100% {
                transform: scale(1);
                box-shadow: 0 0 0 0 rgba(0, 184, 148, 0.4);
            }
            50% {
                transform: scale(1.1);
                box-shadow: 0 0 0 8px rgba(0, 184, 148, 0);
            }
        }

        .package-features li:hover i {
            animation: featureIconBounce 0.6s ease;
            background: linear-gradient(135deg, #667eea, #764ba2);
        }

        @keyframes featureIconBounce {
            0%, 100% {
                transform: scale(1) rotate(0deg);
            }
            25% {
                transform: scale(1.2) rotate(-10deg);
            }
            75% {
                transform: scale(1.2) rotate(10deg);
            }
        }

        .package-features li span {
            font-weight: 500;
            transition: all 0.3s ease;
        }

        .package-features li:hover span {
            font-weight: 600;
            color: #667eea;
        }
        
        .package-stats {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 15px;
            margin-bottom: 25px;
        }
        
        .stat-item {
            text-align: center;
            padding: 20px 15px;
            background: linear-gradient(135deg, rgba(102, 126, 234, 0.15) 0%, rgba(118, 75, 162, 0.15) 100%);
            border-radius: 12px;
            position: relative;
            overflow: hidden;
            transition: all 0.3s ease;
            border: 1px solid rgba(102, 126, 234, 0.2);
        }

        .stat-item::before {
            content: '';
            position: absolute;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background: linear-gradient(135deg, transparent, rgba(102, 126, 234, 0.1), transparent);
            transform: rotate(45deg);
            animation: shimmer 3s infinite;
        }

        @keyframes shimmer {
            0% { transform: translateX(-100%) translateY(-100%) rotate(45deg); }
            100% { transform: translateX(100%) translateY(100%) rotate(45deg); }
        }

        .stat-item:hover {
            transform: translateY(-3px);
            box-shadow: 0 5px 20px rgba(102, 126, 234, 0.3);
            background: linear-gradient(135deg, rgba(102, 126, 234, 0.25) 0%, rgba(118, 75, 162, 0.25) 100%);
        }
        
        .stat-value {
            font-size: 28px;
            font-weight: 800;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            display: block;
            position: relative;
            animation: statPulse 2s ease-in-out infinite;
        }

        @keyframes statPulse {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.05); }
        }
        
    .stat-label {
        font-size: 13px;
        color: var(--text-secondary);
        margin-top: 8px;
        font-weight: 600;
        position: relative;
    }

    .stat-icon {
        font-size: 32px;
        margin-bottom: 12px;
        display: block;
        position: relative;
        filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.2));
    }

    /* User Icon - Blue Gradient */
    .stat-item:nth-child(1) .stat-icon {
        background: linear-gradient(135deg, #667eea, #4facfe);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: userIconFloat 3s ease-in-out infinite;
    }

    @keyframes userIconFloat {
        0%, 100% {
            transform: translateY(0px) rotate(0deg);
        }
        25% {
            transform: translateY(-8px) rotate(-5deg);
        }
        75% {
            transform: translateY(-8px) rotate(5deg);
        }
    }

    .stat-item:nth-child(1):hover .stat-icon {
        animation: userIconSpin 0.8s ease;
    }

    @keyframes userIconSpin {
        0% { transform: rotate(0deg) scale(1); }
        50% { transform: rotate(180deg) scale(1.3); }
        100% { transform: rotate(360deg) scale(1); }
    }

    /* Envelope Icon - Purple Gradient */
    .stat-item:nth-child(2) .stat-icon {
        background: linear-gradient(135deg, #764ba2, #f093fb);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: envelopeShake 2.5s ease-in-out infinite;
    }

    @keyframes envelopeShake {
        0%, 100% {
            transform: rotate(0deg) translateX(0px);
        }
        10%, 30%, 50%, 70%, 90% {
            transform: rotate(-3deg) translateX(-2px);
        }
        20%, 40%, 60%, 80% {
            transform: rotate(3deg) translateX(2px);
        }
    }

    .stat-item:nth-child(2):hover .stat-icon {
        animation: envelopeOpen 0.6s ease;
    }

    @keyframes envelopeOpen {
        0%, 100% { transform: rotateX(0deg); }
        50% { transform: rotateX(180deg) scale(1.2); }
    }

    /* Coins Icon - Gold Gradient */
    .stat-item:nth-child(3) .stat-icon {
        background: linear-gradient(135deg, #f093fb, #f5576c, #ffd89b);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: coinsRotate 3s linear infinite;
    }

    @keyframes coinsRotate {
        0% {
            transform: rotateY(0deg);
        }
        50% {
            transform: rotateY(180deg) scale(1.1);
        }
        100% {
            transform: rotateY(360deg);
        }
    }

    .stat-item:nth-child(3):hover .stat-icon {
        animation: coinsBounce 0.6s ease infinite;
    }

    @keyframes coinsBounce {
        0%, 100% {
            transform: translateY(0) scale(1);
        }
        25% {
            transform: translateY(-10px) scale(1.15);
        }
        50% {
            transform: translateY(0) scale(1.1) rotateZ(10deg);
        }
        75% {
            transform: translateY(-5px) scale(1.15) rotateZ(-10deg);
        }
    }
        
        .platforms {
            display: flex;
            justify-content: center;
            gap: 10px;
            margin-bottom: 25px;
            flex-wrap: wrap;
        }
        
        .platform-icon {
            width: 45px;
            height: 45px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            background: var(--bg-secondary);
            font-size: 20px;
            transition: all 0.4s cubic-bezier(0.68, -0.55, 0.265, 1.55);
            cursor: pointer;
            position: relative;
            overflow: hidden;
            animation: platformFloat 3s ease-in-out infinite;
        }

        @keyframes platformFloat {
            0%, 100% { transform: translateY(0px); }
            50% { transform: translateY(-8px); }
        }

        .platform-icon::before {
            content: '';
            position: absolute;
            width: 100%;
            height: 100%;
            border-radius: 50%;
            background: inherit;
            opacity: 0;
            transition: all 0.3s ease;
        }
        
        .platform-icon:hover {
            transform: scale(1.15) rotate(5deg);
            box-shadow: 0 8px 20px rgba(0, 0, 0, 0.3);
        }

        .platform-icon:hover::before {
            opacity: 0.5;
            transform: scale(1.3);
        }

        .platform-icon i {
            position: relative;
            z-index: 1;
            animation: iconSpin 4s ease-in-out infinite;
        }

        @keyframes iconSpin {
            0%, 100% { transform: rotate(0deg); }
            25% { transform: rotate(-10deg); }
            75% { transform: rotate(10deg); }
        }

        .platform-icon:hover i {
            animation: iconRotate 0.6s ease;
        }

        @keyframes iconRotate {
            0% { transform: rotate(0deg); }
            50% { transform: rotate(180deg); }
            100% { transform: rotate(360deg); }
        }

        /* Platform-specific colors */
        .platform-icon:nth-child(1) {
            background: linear-gradient(135deg, #1877f2, #0c63d4);
            color: white;
        }

        .platform-icon:nth-child(2) {
            background: linear-gradient(135deg, #25d366, #128c7e);
            color: white;
        }

        .platform-icon:nth-child(3) {
            background: linear-gradient(135deg, #0088cc, #0077b5);
            color: white;
        }

        .platform-icon:nth-child(4) {
            background: linear-gradient(135deg, #e4405f, #d62976, #962fbf, #4f5bd5);
            color: white;
        }

        .platform-icon:nth-child(5) {
            background: linear-gradient(135deg, #ea4335, #c5221f);
            color: white;
        }

        .platform-icon:nth-child(6) {
            background: linear-gradient(135deg, #0077b5, #005582);
            color: white;
        }

        .platform-icon:hover {
            filter: brightness(1.2);
        }

    .package-features {
        flex-grow: 1;
    }

    .buy-btn {
        width: 100%;
        padding: 15px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 12px;
        font-size: 16px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.4s cubic-bezier(0.68, -0.55, 0.265, 1.55);
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        font-family: 'Cairo', sans-serif;
        margin-top: auto;
        position: relative;
        overflow: hidden;
    }

    .buy-btn::before {
        content: '';
        position: absolute;
        top: 50%;
        left: 50%;
        width: 0;
        height: 0;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.3);
        transform: translate(-50%, -50%);
        transition: width 0.6s, height 0.6s;
    }

    .buy-btn:hover::before {
        width: 300px;
        height: 300px;
    }

    .buy-btn i {
        animation: cartShake 2s ease-in-out infinite;
        transition: all 0.3s ease;
    }

    @keyframes cartShake {
        0%, 100% {
            transform: translateX(0) rotate(0deg);
        }
        25% {
            transform: translateX(-3px) rotate(-5deg);
        }
        75% {
            transform: translateX(3px) rotate(5deg);
        }
    }

    .buy-btn:hover i {
        animation: cartBounce 0.6s ease infinite;
    }

    @keyframes cartBounce {
        0%, 100% {
            transform: translateY(0) scale(1);
        }
        50% {
            transform: translateY(-5px) scale(1.2);
        }
    }

    .buy-btn:hover {
        transform: translateY(-3px) scale(1.02);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.5);
        background: linear-gradient(135deg, #764ba2 0%, #667eea 100%);
    }

    .buy-btn:active {
        transform: translateY(-1px) scale(0.98);
    }

    .loading {
        text-align: center;
        padding: 60px 20px;
        color: var(--text-primary);
        font-size: 18px;
        font-family: 'Cairo', sans-serif;
    }

    .loading i {
        font-size: 48px;
        margin-bottom: 20px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    .empty-state {
        grid-column: 1/-1;
        text-align: center;
        padding: 60px 20px;
        background: var(--card-bg);
        border-radius: 15px;
        border: 1px solid var(--border-color);
    }

    .empty-state i {
        font-size: 80px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        margin-bottom: 20px;
        opacity: 0.5;
    }

    .empty-state h3 {
        font-size: 24px;
        color: var(--text-primary);
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
    }

    .empty-state p {
        color: var(--text-secondary);
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
    }

    /* Light Theme */
    body.light-theme .package-card,
    body.light-theme .empty-state {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .package-name,
    body.light-theme .packages-title,
    body.light-theme .loading {
        color: #2d3436;
    }

    body.light-theme .package-description {
        color: #636e72;
    }

    body.light-theme .stat-item {
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.08) 0%, rgba(118, 75, 162, 0.08) 100%);
        border-color: rgba(102, 126, 234, 0.15);
    }

    body.light-theme .stat-item:hover {
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.15) 0%, rgba(118, 75, 162, 0.15) 100%);
    }

    body.light-theme .package-features li {
        border-bottom-color: #f0f0f0;
    }

    /* Responsive */
    @media (max-width: 768px) {
        .packages-container {
            padding: 20px;
            margin-top: 100px;
        }

        .packages-grid {
            grid-template-columns: 1fr;
        }

        .package-stats {
            grid-template-columns: 1fr;
        }
    }
</style>

<div class="packages-container">
    <div class="packages-header">
        <div class="packages-title">
            <i class="fas fa-box-open"></i>
            الباقات المتاحة
        </div>
    </div>

    <div id="packagesContainer" class="packages-grid">
        <div class="loading">
            <i class="fas fa-spinner"></i>
            <p>جاري تحميل الباقات...</p>
        </div>
    </div>
</div>

    <script>
        // تحميل الباقات عند فتح الصفحة
        document.addEventListener('DOMContentLoaded', loadPackages);
        
        // دالة تحميل الباقات
        async function loadPackages() {
            const container = document.getElementById('packagesContainer');
            
            try {
                const response = await fetch('api/get_packages.php');
                const data = await response.json();
                
                if (data.success && data.packages.length > 0) {
                    renderPackages(data.packages);
                } else {
                    showEmptyState();
                }
            } catch (error) {
                console.error('خطأ في تحميل الباقات:', error);
                showError('حدث خطأ في تحميل الباقات. يرجى المحاولة مرة أخرى.');
            }
        }
        
        // عرض الباقات
        function renderPackages(packages) {
            const container = document.getElementById('packagesContainer');
            container.innerHTML = '';
            
            packages.forEach(pkg => {
                const card = createPackageCard(pkg);
                container.appendChild(card);
            });
        }
        
        // إنشاء بطاقة باقة
        function createPackageCard(pkg) {
            const card = document.createElement('div');
            card.className = 'package-card' + (pkg.is_popular ? ' popular' : '');
            
            // تحويل المميزات من JSON إلى array
            let features = [];
            try {
                features = typeof pkg.features === 'string' ? JSON.parse(pkg.features) : pkg.features;
            } catch(e) {
                features = [];
            }
            
            // تحويل المنصات من JSON إلى array
            let platforms = [];
            try {
                platforms = typeof pkg.platforms === 'string' ? JSON.parse(pkg.platforms) : pkg.platforms;
            } catch(e) {
                platforms = [];
            }
            
            // بناء HTML الباقة
            card.innerHTML = `
                <h2 class="package-name">${escapeHtml(pkg.name)}</h2>
                <p class="package-description">${escapeHtml(pkg.description || '')}</p>
                
                <div class="package-price">
                    <div class="price-amount">
                        <span>${parseFloat(pkg.price).toFixed(2)}</span>
                        <span class="price-currency">${pkg.currency || 'USD'}</span>
                    </div>
                    ${pkg.has_discount && pkg.original_price ? `
                        <div class="original-price">${parseFloat(pkg.original_price).toFixed(2)} ${pkg.currency || 'USD'}</div>
                        <span class="discount-badge">
                            <i class="fas fa-tag"></i>
                            خصم ${Math.round(((pkg.original_price - pkg.price) / pkg.original_price) * 100)}%
                        </span>
                    ` : ''}
                </div>
                
                <div class="package-stats">
                    <div class="stat-item">
                        <i class="fas fa-user stat-icon"></i>
                        <span class="stat-value">${pkg.accounts_count || 0}</span>
                        <span class="stat-label">حساب</span>
                    </div>
                    <div class="stat-item">
                        <i class="fas fa-envelope stat-icon"></i>
                        <span class="stat-value">${pkg.messages_count || 0}</span>
                        <span class="stat-label">رسالة</span>
                    </div>
                    <div class="stat-item">
                        <i class="fas fa-coins stat-icon"></i>
                        <span class="stat-value">${pkg.points || 0}</span>
                        <span class="stat-label">نقطة</span>
                    </div>
                </div>
                
                ${platforms.length > 0 ? `
                    <div class="platforms">
                        ${platforms.map(platform => getPlatformIcon(platform)).join('')}
                    </div>
                ` : ''}
                
                ${features.length > 0 ? `
                    <ul class="package-features">
                        ${features.map(feature => `
                            <li>
                                <i class="fas fa-check-circle"></i>
                                <span>${escapeHtml(feature)}</span>
                            </li>
                        `).join('')}
                    </ul>
                ` : ''}
                
                <button class="buy-btn" onclick="buyPackage(${pkg.id}, '${escapeHtml(pkg.name)}')">
                    <i class="fas fa-shopping-cart"></i>
                    اشتري الآن
                </button>
            `;
            
            return card;
        }
        
        // الحصول على أيقونة المنصة
        function getPlatformIcon(platform) {
            const icons = {
                'facebook': 'fab fa-facebook',
                'whatsapp': 'fab fa-whatsapp',
                'telegram': 'fab fa-telegram',
                'instagram': 'fab fa-instagram',
                'email': 'fas fa-envelope',
                'business': 'fas fa-briefcase'
            };
            
            const icon = icons[platform] || 'fas fa-globe';
            const title = platform.charAt(0).toUpperCase() + platform.slice(1);
            
            return `<div class="platform-icon" title="${title}"><i class="${icon}"></i></div>`;
        }
        
        // دالة شراء الباقة
        function buyPackage(packageId, packageName) {
            Swal.fire({
                title: '🎉 شراء الباقة',
                html: `
                    <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                        <p style="font-size: 18px; color: #2d3436; margin-bottom: 20px;">
                            هل تريد شراء باقة <strong style="color: #667eea;">${packageName}</strong>؟
                        </p>
                        <div style="background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1)); padding: 15px; border-radius: 10px; margin-top: 15px;">
                            <i class="fas fa-info-circle" style="color: #667eea; margin-left: 5px;"></i>
                            <span style="color: #636e72;">سيتم توجيهك لصفحة الدفع لإتمام عملية الشراء</span>
                        </div>
                    </div>
                `,
                icon: 'question',
                showCancelButton: true,
                confirmButtonText: '<i class="fas fa-shopping-cart"></i> متابعة الشراء',
                cancelButtonText: '<i class="fas fa-times"></i> إلغاء',
                confirmButtonColor: '#667eea',
                cancelButtonColor: '#95a5a6',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb',
                customClass: {
                    popup: 'swal-rtl',
                    confirmButton: 'swal-confirm-btn',
                    cancelButton: 'swal-cancel-btn'
                },
                showClass: {
                    popup: 'animate__animated animate__zoomIn animate__faster'
                },
                hideClass: {
                    popup: 'animate__animated animate__zoomOut animate__faster'
                }
            }).then((result) => {
                if (result.isConfirmed) {
                    // عرض رسالة تحميل
                    Swal.fire({
                        title: 'جاري التحميل...',
                        html: '<i class="fas fa-spinner fa-spin" style="font-size: 48px; color: #667eea;"></i>',
                        showConfirmButton: false,
                        allowOutsideClick: false,
                        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb',
                        didOpen: () => {
                            // الانتظار قليلاً ثم التوجيه
                            setTimeout(() => {
                                window.location.href = `checkout.php?package_id=${packageId}`;
                            }, 800);
                        }
                    });
                }
            });
        }
        
        // عرض حالة فارغة
        function showEmptyState() {
            const container = document.getElementById('packagesContainer');
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-box-open"></i>
                    <h3>لا توجد باقات متاحة</h3>
                    <p>لم يتم العثور على أي باقات في الوقت الحالي</p>
                </div>
            `;
        }

        // عرض رسالة خطأ
        function showError(message) {
            const container = document.getElementById('packagesContainer');
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-exclamation-triangle"></i>
                    <h3>حدث خطأ</h3>
                    <p>${message}</p>
                </div>
            `;
        }
        
        // دالة لتأمين النصوص من XSS
        function escapeHtml(text) {
            const map = {
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                '"': '&quot;',
                "'": '&#039;'
            };
            return String(text).replace(/[&<>"']/g, m => map[m]);
        }
    </script>

<?php include 'includes/footer.php'; ?>
