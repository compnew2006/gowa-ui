<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}

require_once 'includes/functions.php';
require_once 'config/database.php';

$user_id = $_SESSION['user_id'];
$package_id = isset($_GET['package_id']) ? intval($_GET['package_id']) : 0;

if ($package_id <= 0) {
    header('Location: packages.php');
    exit;
}

// جلب معلومات الباقة
$conn = getDB();
$stmt = $conn->prepare("SELECT * FROM packages WHERE id = :id");
$stmt->execute([':id' => $package_id]);
$package = $stmt->fetch(PDO::FETCH_ASSOC);

if (!$package) {
    header('Location: packages.php');
    exit;
}

// جلب معلومات المستخدم
$stmt = $conn->prepare("SELECT * FROM users WHERE user_id = :id");
$stmt->execute([':id' => $user_id]);
$user = $stmt->fetch(PDO::FETCH_ASSOC);

if (!$user) {
    header('Location: landing.php');
    exit;
}

// جلب رصيد المحفظة
$stmt = $conn->prepare("SELECT balance FROM users_wallet WHERE user_id = :user_id");
$stmt->execute([':user_id' => $user_id]);
$wallet = $stmt->fetch(PDO::FETCH_ASSOC);
 
// جلب المندوبين
$stmt = $conn->query("SELECT id, name, phone FROM representative");
$representatives = $stmt->fetchAll(PDO::FETCH_ASSOC);

$page_title = "إتمام الشراء | Kingmaster";
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>

<style>
    .checkout-container {
        padding: 30px;
        max-width: 1200px;
        margin: 120px auto 0 auto;
    }

    .checkout-header {
        text-align: center;
        margin-bottom: 40px;
    }

    .checkout-title {
        font-size: 32px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 10px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 12px;
    }

    .checkout-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: iconFloat 3s ease-in-out infinite, iconRotate 4s ease-in-out infinite;
    }

    @keyframes iconFloat {
        0%, 100% { transform: translateY(0px) rotate(0deg); }
        50% { transform: translateY(-10px) rotate(5deg); }
    }

    .checkout-subtitle {
        color: var(--text-secondary);
        font-size: 16px;
        font-family: 'Cairo', sans-serif;
    }

    .checkout-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 30px;
        margin-bottom: 40px;
    }

    .package-summary {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 30px;
        border: 1px solid var(--border-color);
        animation: slideInRight 0.5s ease;
    }

    @keyframes slideInRight {
        from {
            opacity: 0;
            transform: translateX(30px);
        }
        to {
            opacity: 1;
            transform: translateX(0);
        }
    }

    .summary-title {
        font-size: 20px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 20px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .summary-title i {
        color: #667eea;
        animation: iconPulse 2s ease-in-out infinite;
    }

    .summary-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 15px 0;
        border-bottom: 1px solid var(--border-color);
        font-family: 'Cairo', sans-serif;
    }

    .summary-item:last-child {
        border-bottom: none;
    }

    .summary-label {
        color: var(--text-secondary);
        font-size: 14px;
    }

    .summary-value {
        color: var(--text-primary);
        font-size: 16px;
        font-weight: 700;
    }

    .total-price {
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1));
        padding: 20px;
        border-radius: 10px;
        margin-top: 20px;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .total-label {
        font-size: 18px;
        font-weight: 700;
        color: var(--text-primary);
    }

    .total-amount {
        font-size: 32px;
        font-weight: 800;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .payment-methods {
        background: var(--card-bg);
        border-radius: 15px;
        padding: 30px;
        border: 1px solid var(--border-color);
        animation: slideInLeft 0.5s ease;
    }

    @keyframes slideInLeft {
        from {
            opacity: 0;
            transform: translateX(-30px);
        }
        to {
            opacity: 1;
            transform: translateX(0);
        }
    }

    .payment-title {
        font-size: 20px;
        font-weight: 800;
        color: var(--text-primary);
        margin-bottom: 25px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .payment-title i {
        color: #667eea;
        animation: iconPulse 2s ease-in-out infinite;
    }

    .payment-option {
        background: var(--bg-secondary);
        border: 2px solid var(--border-color);
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 15px;
        cursor: pointer;
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
    }

    .payment-option::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        width: 4px;
        height: 100%;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        transform: scaleY(0);
        transition: transform 0.3s ease;
    }

    .payment-option:hover {
        border-color: #667eea;
        transform: translateX(-5px);
        box-shadow: 0 5px 20px rgba(102, 126, 234, 0.3);
    }

    .payment-option:hover::before {
        transform: scaleY(1);
    }

    .payment-option.selected {
        border-color: #667eea;
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1));
    }

    .payment-option.selected::before {
        transform: scaleY(1);
    }

    .payment-header {
        display: flex;
        align-items: center;
        gap: 15px;
        margin-bottom: 10px;
    }

    .payment-icon {
        width: 50px;
        height: 50px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 24px;
        transition: all 0.3s ease;
    }

    .payment-icon i {
        animation: paymentIconFloat 3s ease-in-out infinite;
    }

    @keyframes paymentIconFloat {
        0%, 100% {
            transform: translateY(0px) rotate(0deg);
        }
        25% {
            transform: translateY(-3px) rotate(-5deg);
        }
        75% {
            transform: translateY(-3px) rotate(5deg);
        }
    }

    .payment-option:hover .payment-icon {
        transform: scale(1.1) rotate(5deg);
    }

    .payment-option:hover .payment-icon i {
        animation: paymentIconSpin 0.8s ease;
    }

    @keyframes paymentIconSpin {
        0% { transform: rotate(0deg) scale(1); }
        50% { transform: rotate(180deg) scale(1.2); }
        100% { transform: rotate(360deg) scale(1); }
    }

    .payment-option.selected .payment-icon {
        animation: iconPulse 1s ease infinite;
    }

    @keyframes iconPulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }

    .icon-balance {
        background: linear-gradient(135deg, #667eea, #764ba2);
        color: white;
    }

    .icon-online {
        background: linear-gradient(135deg, #00b894, #00d2d3);
        color: white;
    }

    .icon-delivery {
        background: linear-gradient(135deg, #f093fb, #f5576c);
        color: white;
    }

    .payment-info {
        flex: 1;
    }

    .payment-name {
        font-size: 16px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 5px;
        font-family: 'Cairo', sans-serif;
    }

    .payment-desc {
        font-size: 13px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
    }

    .payment-check {
        width: 24px;
        height: 24px;
        border: 2px solid var(--border-color);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.3s ease;
    }

    .payment-option.selected .payment-check {
        border-color: #667eea;
        background: #667eea;
    }

    .payment-check i {
        color: white;
        font-size: 12px;
        opacity: 0;
        transition: opacity 0.3s ease;
    }

    .payment-option.selected .payment-check i {
        opacity: 1;
        animation: checkBounce 0.5s ease;
    }

    @keyframes checkBounce {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.3); }
    }

    .balance-info {
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1));
        padding: 15px;
        border-radius: 10px;
        margin-top: 10px;
        display: none;
        animation: fadeIn 0.3s ease;
    }

    .payment-option.selected .balance-info {
        display: block;
    }

    @keyframes fadeIn {
        from { opacity: 0; transform: translateY(-10px); }
        to { opacity: 1; transform: translateY(0); }
    }

    .coupon-section {
        margin-top: 15px;
        padding-top: 15px;
        border-top: 1px solid var(--border-color);
    }

    .coupon-label {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 10px;
        color: var(--text-primary);
        font-weight: 600;
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
    }

    .coupon-label i {
        color: #f39c12;
        animation: couponShine 2s ease-in-out infinite;
    }

    @keyframes couponShine {
        0%, 100% { transform: scale(1) rotate(0deg); }
        50% { transform: scale(1.2) rotate(10deg); }
    }

    .coupon-input-group {
        display: flex;
        gap: 10px;
    }

    .coupon-input {
        flex: 1;
        padding: 10px 15px;
        border: 2px solid var(--border-color);
        border-radius: 8px;
        background: var(--bg-primary);
        color: var(--text-primary);
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .coupon-input:focus {
        outline: none;
        border-color: #667eea;
        box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.15);
    }

    .coupon-input::placeholder {
        color: var(--text-secondary);
        opacity: 0.6;
    }

    .apply-coupon-btn {
        padding: 10px 20px;
        background: linear-gradient(135deg, #f39c12, #e67e22);
        color: white;
        border: none;
        border-radius: 8px;
        font-size: 14px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        white-space: nowrap;
    }

    .apply-coupon-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(243, 156, 18, 0.4);
        background: linear-gradient(135deg, #e67e22, #f39c12);
    }

    .apply-coupon-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
        transform: none;
    }

    .coupon-result {
        margin-top: 10px;
        padding: 10px;
        border-radius: 8px;
        font-size: 13px;
        font-family: 'Cairo', sans-serif;
        display: none;
        animation: fadeIn 0.3s ease;
    }

    .coupon-result.success {
        background: rgba(0, 184, 148, 0.1);
        color: #00b894;
        border: 1px solid rgba(0, 184, 148, 0.3);
        display: block;
    }

    .coupon-result.error {
        background: rgba(255, 107, 107, 0.1);
        color: #ff6b6b;
        border: 1px solid rgba(255, 107, 107, 0.3);
        display: block;
    }

    .coupon-result i {
        margin-left: 5px;
    }

    body.light-theme .coupon-input {
        background: #ffffff;
        color: #2d3436;
    }

    /* Duration Section */
    .duration-section {
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.05), rgba(118, 75, 162, 0.05));
        border: 1px solid var(--border-color);
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 20px;
    }

    .duration-title {
        font-size: 18px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 15px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .duration-title i {
        color: #667eea;
        animation: iconPulse 2s ease-in-out infinite;
    }

    .duration-options {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
        gap: 10px;
    }

    .duration-option {
        padding: 12px 8px;
        background: var(--bg-secondary);
        border: 2px solid var(--border-color);
        border-radius: 10px;
        cursor: pointer;
        transition: all 0.3s ease;
        text-align: center;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
        font-size: 14px;
        position: relative;
        overflow: hidden;
    }

    .duration-option::before {
        content: '';
        position: absolute;
        top: 0;
        left: -100%;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(102, 126, 234, 0.3), transparent);
        transition: left 0.5s ease;
    }

    .duration-option:hover::before {
        left: 100%;
    }

    .duration-option:hover {
        border-color: #667eea;
        transform: translateY(-3px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.3);
    }

    .duration-option.selected {
        border-color: #667eea;
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.15), rgba(118, 75, 162, 0.15));
        color: #667eea;
        transform: translateY(-3px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.3);
    }

    .duration-period {
        display: block;
        font-size: 16px;
        font-weight: 800;
        margin-bottom: 3px;
    }

    .duration-discount {
        font-size: 11px;
        color: #00b894;
        font-weight: 700;
        display: none;
    }

    .duration-option.selected .duration-discount,
    .duration-option[data-discount]:hover .duration-discount {
        display: block;
        animation: discountPulse 1s ease infinite;
    }

    @keyframes discountPulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.1); }
    }

    .price-breakdown {
        background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1));
        padding: 15px;
        border-radius: 10px;
        margin-top: 15px;
        border: 1px solid rgba(102, 126, 234, 0.2);
    }

    .price-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
    }

    .price-row:last-child {
        margin-bottom: 0;
        font-size: 16px;
        font-weight: 700;
        border-top: 1px solid var(--border-color);
        padding-top: 10px;
        margin-top: 10px;
    }

    .price-label {
        color: var(--text-secondary);
    }

    .price-value {
        color: var(--text-primary);
        font-weight: 600;
    }

    .total-value {
        color: #667eea;
        font-weight: 800;
        font-size: 18px;
    }

    .balance-row {
        display: flex;
        justify-content: space-between;
        margin-bottom: 8px;
        font-family: 'Cairo', sans-serif;
        font-size: 14px;
    }

    .balance-row:last-child {
        margin-bottom: 0;
    }

    .balance-label {
        color: var(--text-secondary);
    }

    .balance-value {
        font-weight: 700;
        color: var(--text-primary);
        animation: numberGlow 2s ease-in-out infinite;
    }

    @keyframes numberGlow {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.8; transform: scale(1.02); }
    }

    .balance-value.insufficient {
        color: #ff6b6b;
        animation: numberShake 0.5s ease infinite;
    }

    @keyframes numberShake {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-2px); }
        75% { transform: translateX(2px); }
    }

    .balance-value.sufficient {
        color: #00b894;
        animation: numberPulse 1.5s ease-in-out infinite;
    }

    @keyframes numberPulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.05); }
    }

    .confirm-btn {
        width: 100%;
        padding: 18px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        border-radius: 12px;
        font-size: 18px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.4s cubic-bezier(0.68, -0.55, 0.265, 1.55);
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        margin-top: 30px;
        position: relative;
        overflow: hidden;
    }

    .confirm-btn::before {
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

    .confirm-btn:hover::before {
        width: 400px;
        height: 400px;
    }

    .confirm-btn:hover {
        transform: translateY(-3px) scale(1.02);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.5);
    }

    .confirm-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
        transform: none;
    }

    .confirm-btn i {
        animation: iconShake 2s ease-in-out infinite;
    }

    @keyframes iconShake {
        0%, 100% { transform: rotate(0deg); }
        25% { transform: rotate(-10deg); }
        75% { transform: rotate(10deg); }
    }

    .confirm-btn:hover i {
        animation: iconRotate 0.6s ease;
    }

    @keyframes iconRotate {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    /* Light Theme */
    body.light-theme .package-summary,
    body.light-theme .payment-methods {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .payment-option {
        background: #f8f9fa;
    }

    /* Modal Styles */
    .modal {
        display: none;
        position: fixed;
        z-index: 1000;
        left: 0;
        top: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        backdrop-filter: blur(5px);
    }

    .modal-content {
        background: var(--card-bg);
        margin: 5% auto;
        padding: 30px;
        border-radius: 20px;
        width: 90%;
        max-width: 600px;
        max-height: 80vh;
        overflow-y: auto;
        animation: modalSlideIn 0.3s ease;
        border: 1px solid var(--border-color);
    }

    @keyframes modalSlideIn {
        from {
            opacity: 0;
            transform: translateY(-50px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 25px;
        padding-bottom: 15px;
        border-bottom: 2px solid var(--border-color);
    }

    .modal-title {
        font-size: 24px;
        font-weight: 800;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .modal-title i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        font-size: 28px;
        animation: modalIconRotate 3s ease-in-out infinite;
    }

    @keyframes modalIconRotate {
        0%, 100% { transform: rotate(0deg) scale(1); }
        50% { transform: rotate(10deg) scale(1.1); }
    }

    .close-modal {
        font-size: 28px;
        font-weight: bold;
        color: var(--text-secondary);
        cursor: pointer;
        transition: all 0.3s ease;
        width: 35px;
        height: 35px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 50%;
    }

    .close-modal:hover {
        color: var(--text-primary);
        background: var(--bg-secondary);
        transform: rotate(90deg);
    }

    .representatives-list {
        display: grid;
        gap: 15px;
    }

    .representative-card {
        background: var(--bg-secondary);
        border: 2px solid var(--border-color);
        border-radius: 12px;
        padding: 20px;
        transition: all 0.3s ease;
        animation: cardSlideUp 0.4s ease;
    }

    .representative-card:hover {
        border-color: #667eea;
        transform: translateY(-3px);
        box-shadow: 0 5px 20px rgba(102, 126, 234, 0.3);
    }

    .rep-header {
        display: flex;
        align-items: center;
        gap: 15px;
        margin-bottom: 15px;
    }

    .rep-icon {
        width: 60px;
        height: 60px;
        border-radius: 50%;
        background: linear-gradient(135deg, #f093fb, #f5576c);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 28px;
        color: white;
        animation: iconPulse 2s ease infinite;
    }

    .rep-info {
        flex: 1;
    }

    .rep-name {
        font-size: 18px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 5px;
        font-family: 'Cairo', sans-serif;
    }

    .rep-phone {
        font-size: 14px;
        color: var(--text-secondary);
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 5px;
    }

    .rep-phone i {
        color: #667eea;
        animation: phoneRing 2s ease-in-out infinite;
    }

    @keyframes phoneRing {
        0%, 100% { transform: rotate(0deg); }
        10%, 30% { transform: rotate(-15deg); }
        20%, 40% { transform: rotate(15deg); }
    }

    .whatsapp-btn {
        width: 100%;
        padding: 12px 20px;
        background: linear-gradient(135deg, #25d366, #128c7e);
        color: white;
        border: none;
        border-radius: 10px;
        font-size: 15px;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.3s ease;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
    }

    .whatsapp-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(37, 211, 102, 0.4);
        background: linear-gradient(135deg, #128c7e, #25d366);
    }

    .whatsapp-btn i {
        font-size: 18px;
        animation: iconShake 2s ease-in-out infinite;
    }

    body.light-theme .modal-content {
        background: rgba(255, 255, 255, 0.98);
    }

    body.light-theme .representative-card {
        background: #f8f9fa;
    }

    /* Responsive */
    @media (max-width: 768px) {
        .checkout-container {
            padding: 20px;
            margin-top: 100px;
        }

        .checkout-grid {
            grid-template-columns: 1fr;
        }

        .checkout-title {
            font-size: 24px;
        }

        .modal-content {
            width: 95%;
            padding: 20px;
        }
    }
</style>

<div class="checkout-container">
    <div class="checkout-header">
        <h1 class="checkout-title">
            <i class="fas fa-shopping-bag"></i>
            إتمام عملية الشراء
        </h1>
        <p class="checkout-subtitle">اختر طريقة الدفع المناسبة لك</p>
    </div>

    <div class="checkout-grid">
        <div class="payment-methods">
            <h2 class="payment-title">
                <i class="fas fa-credit-card"></i>
                طرق الدفع
            </h2>

            <div class="payment-option" onclick="selectPayment('balance')" id="payment-balance">
                <div class="payment-header">
                    <div class="payment-icon icon-balance">
                        <i class="fas fa-wallet"></i>
                    </div>
                    <div class="payment-info">
                        <div class="payment-name">الدفع من الرصيد</div>
                        <div class="payment-desc">استخدم رصيدك المتاح في المحفظة</div>
                    </div>
                    <div class="payment-check">
                        <i class="fas fa-check"></i>
                    </div>
                </div>
                <div class="balance-info">
                    <div class="balance-row">
                        <span class="balance-label">رصيدك الحالي:</span>
                        <span class="balance-value"><?php echo number_format($wallet['balance'] ?? 0, 2); ?> USD</span>
                    </div>
                    <div class="balance-row">
                        <span class="balance-label">المطلوب:</span>
                        <span class="balance-value" id="required-amount"><?php echo number_format($package['price'], 2); ?> USD</span>
                    </div>
                    <div class="balance-row">
                        <span class="balance-label">الرصيد المتبقي:</span>
                        <span class="balance-value <?php echo (($wallet['balance'] ?? 0) >= $package['price']) ? 'sufficient' : 'insufficient'; ?>" id="remaining-amount">
                            <?php echo number_format(($wallet['balance'] ?? 0) - $package['price'], 2); ?> USD
                        </span>
                    </div>
                    
                    <div class="coupon-section">
                        <div class="coupon-label">
                            <i class="fas fa-tag"></i>
                            هل لديك كوبون خصم؟
                        </div>
                        <div class="coupon-input-group">
                            <input type="text" class="coupon-input" id="couponCode" placeholder="أدخل رمز الكوبون">
                            <button class="apply-coupon-btn" onclick="applyCoupon()">
                                <i class="fas fa-check"></i> تطبيق
                            </button>
                        </div>
                        <div class="coupon-result" id="couponResult"></div>
                    </div>
                </div>
            </div>

            <div class="payment-option" onclick="selectPayment('online')" id="payment-online">
                <div class="payment-header">
                    <div class="payment-icon icon-online">
                        <i class="fas fa-credit-card"></i>
                    </div>
                    <div class="payment-info">
                        <div class="payment-name">الدفع الإلكتروني</div>
                        <div class="payment-desc">الدفع بواسطة البطاقة الائتمانية أو PayPal</div>
                    </div>
                    <div class="payment-check">
                        <i class="fas fa-check"></i>
                    </div>
                </div>
            </div>

            <div class="payment-option" onclick="selectPayment('representative')" id="payment-representative">
                <div class="payment-header">
                    <div class="payment-icon icon-delivery">
                        <i class="fas fa-user-tie"></i>
                    </div>
                    <div class="payment-info">
                        <div class="payment-name">الدفع من خلال المندوب</div>
                        <div class="payment-desc">تواصل مع المندوب عبر واتساب لإتمام عملية الشراء</div>
                    </div>
                    <div class="payment-check">
                        <i class="fas fa-check"></i>
                    </div>
                </div>
            </div>
        </div>

        <div class="package-summary">
            <h2 class="summary-title">
                <i class="fas fa-receipt"></i>
                ملخص الطلب
            </h2>

            <div class="duration-section">
                <div class="duration-title">
                    <i class="fas fa-calendar-alt"></i>
                    مدة الاشتراك
                </div>
                <div class="duration-options">
                    <div class="duration-option selected" data-months="1" onclick="selectDuration(1)">
                        <span class="duration-period">شهر</span>
                        <span class="duration-discount"></span>
                    </div>
                    <div class="duration-option" data-months="3" data-discount="5" onclick="selectDuration(3)">
                        <span class="duration-period">3 شهور</span>
                        <span class="duration-discount">وفر 5%</span>
                    </div>
                    <div class="duration-option" data-months="6" data-discount="10" onclick="selectDuration(6)">
                        <span class="duration-period">6 شهور</span>
                        <span class="duration-discount">وفر 10%</span>
                    </div>
                    <div class="duration-option" data-months="12" data-discount="20" onclick="selectDuration(12)">
                        <span class="duration-period">سنة</span>
                        <span class="duration-discount">وفر 20%</span>
                    </div>
                </div>
                <div class="price-breakdown" id="priceBreakdown">
                    <div class="price-row">
                        <span class="price-label">السعر الشهري:</span>
                        <span class="price-value" id="monthlyPrice"><?php echo number_format($package['price'], 2); ?> USD</span>
                    </div>
                    <div class="price-row">
                        <span class="price-label">المدة:</span>
                        <span class="price-value" id="selectedDuration">شهر واحد</span>
                    </div>
                    <div class="price-row" id="discountRow" style="display: none;">
                        <span class="price-label">خصم المدة:</span>
                        <span class="price-value" id="durationDiscount" style="color: #00b894;">0%</span>
                    </div>
                    <div class="price-row">
                        <span class="price-label">المجموع:</span>
                        <span class="price-value total-value" id="totalPrice"><?php echo number_format($package['price'], 2); ?> USD</span>
                    </div>
                </div>
            </div>

            <div class="summary-item">
                <span class="summary-label">اسم الباقة</span>
                <span class="summary-value"><?php echo htmlspecialchars($package['name']); ?></span>
            </div>

            <div class="summary-item">
                <span class="summary-label">عدد الحسابات</span>
                <span class="summary-value"><?php echo $package['accounts_count']; ?> حساب</span>
            </div>

            <div class="summary-item">
                <span class="summary-label">عدد الرسائل</span>
                <span class="summary-value"><?php echo number_format($package['messages_count']); ?> رسالة</span>
            </div>

            <div class="summary-item">
                <span class="summary-label">النقاط</span>
                <span class="summary-value"><?php echo number_format($package['points']); ?> نقطة</span>
            </div>

            <div class="total-price">
                <span class="total-label">المجموع النهائي:</span>
                <span class="total-amount" id="finalTotalAmount"><?php echo number_format($package['price'], 2); ?> USD</span>
            </div>

            <button class="confirm-btn" onclick="confirmPurchase()" id="confirmBtn" disabled>
                <i class="fas fa-check-circle"></i>
                تأكيد الشراء
            </button>
        </div>
    </div>
</div>

<!-- Representatives Modal -->
<div id="representativesModal" class="modal">
    <div class="modal-content">
        <div class="modal-header">
            <div class="modal-title">
                <i class="fas fa-users"></i>
                اختر المندوب
            </div>
            <span class="close-modal" onclick="closeRepresentativesModal()">&times;</span>
        </div>
        <div class="representatives-list">
            <?php foreach($representatives as $rep): ?>
                <div class="representative-card">
                    <div class="rep-header">
                        <div class="rep-icon">
                            <i class="fas fa-user-tie"></i>
                        </div>
                        <div class="rep-info">
                            <div class="rep-name"><?php echo htmlspecialchars($rep['name']); ?></div>
                            <div class="rep-phone">
                                <i class="fas fa-phone"></i>
                                <?php echo htmlspecialchars($rep['phone']); ?>
                            </div>
                        </div>
                    </div>
                    <button class="whatsapp-btn" onclick="contactRepresentative('<?php echo htmlspecialchars($rep['phone']); ?>', '<?php echo htmlspecialchars($rep['name']); ?>')">
                        <i class="fab fa-whatsapp"></i>
                        تواصل عبر واتساب
                    </button>
                </div>
            <?php endforeach; ?>
        </div>
    </div>
</div>

<script>
    let selectedPayment = null;
    const packagePrice = <?php echo $package['price']; ?>;
    const userBalance = <?php echo floatval($wallet['balance'] ?? 0); ?>;
    
    console.log('Package Price:', packagePrice);
    console.log('User Balance:', userBalance);

    function selectPayment(type) {
        // إزالة التحديد من جميع الخيارات
        document.querySelectorAll('.payment-option').forEach(option => {
            option.classList.remove('selected');
        });

        // تحديد الخيار المختار
        document.getElementById('payment-' + type).classList.add('selected');
        selectedPayment = type;

        // إذا اختار الدفع من خلال المندوب، فتح المودال
        if (type === 'representative') {
            document.getElementById('representativesModal').style.display = 'block';
            document.getElementById('confirmBtn').disabled = true;
            return;
        }

        // تفعيل زر التأكيد
        const confirmBtn = document.getElementById('confirmBtn');
        
        // التحقق من كفاية الرصيد في حالة الدفع من الرصيد
        if (type === 'balance' && userBalance < packagePrice) {
            confirmBtn.disabled = true;
            Swal.fire({
                icon: 'warning',
                title: 'رصيد غير كافٍ',
                text: 'رصيدك الحالي غير كافٍ لإتمام عملية الشراء. يرجى اختيار طريقة دفع أخرى أو شحن المحفظة.',
                confirmButtonColor: '#667eea',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
        } else {
            confirmBtn.disabled = false;
        }
    }

    function closeRepresentativesModal() {
        document.getElementById('representativesModal').style.display = 'none';
    }

    function contactRepresentative(phone, repName) {
        const packageName = '<?php echo htmlspecialchars($package['name']); ?>';
        const packagePrice = '<?php echo number_format($package['price'], 2); ?>';
        
        // تنسيق رقم الهاتف للواتساب (إزالة أي رموز غير رقمية)
        const cleanPhone = phone.replace(/[^0-9]/g, '');
        
        // رسالة واتساب
        const message = `السلام عليكم ${repName}\n\nأريد شراء باقة:\n📦 اسم الباقة: ${packageName}\n💰 السعر: ${packagePrice} USD\n\nيرجى التواصل معي لإتمام عملية الشراء.`;
        
        // فتح واتساب
        const whatsappUrl = `https://wa.me/${cleanPhone}?text=${encodeURIComponent(message)}`;
        window.open(whatsappUrl, '_blank');
        
        // إغلاق المودال
        closeRepresentativesModal();
        
        // عرض رسالة نجاح
        Swal.fire({
            icon: 'success',
            title: 'تم فتح واتساب',
            text: 'يرجى إكمال عملية الشراء مع المندوب عبر واتساب',
            confirmButtonColor: '#667eea',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });
    }

    // إغلاق المودال عند النقر خارجه
    window.onclick = function(event) {
        const modal = document.getElementById('representativesModal');
        if (event.target == modal) {
            closeRepresentativesModal();
        }
    }

    let appliedDiscount = 0;
    let appliedCoupon = null;
    let originalPrice = packagePrice;
    let selectedMonths = 1;
    let durationDiscount = 0;
    const monthlyPrice = packagePrice;

    function applyCoupon() {
        const couponCode = document.getElementById('couponCode').value.trim();
        const resultDiv = document.getElementById('couponResult');
        const applyBtn = document.querySelector('.apply-coupon-btn');

        if (!couponCode) {
            resultDiv.className = 'coupon-result error';
            resultDiv.innerHTML = '<i class="fas fa-exclamation-circle"></i> يرجى إدخال رمز الكوبون';
            return;
        }

        // تعطيل زر التطبيق أثناء التحقق
        applyBtn.disabled = true;
        applyBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> جاري التحقق...';

        // إرسال طلب للتحقق من الكوبون
        fetch('api/validate_coupon.php', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                coupon_code: couponCode,
                package_id: <?php echo $package_id; ?>,
                package_price: originalPrice
            })
        })
        .then(response => response.json())
        .then(data => {
            applyBtn.disabled = false;
            applyBtn.innerHTML = '<i class="fas fa-check"></i> تطبيق';

            if (data.success) {
                appliedDiscount = data.discount;
                appliedCoupon = couponCode; // حفظ رمز الكوبون
                const newPrice = originalPrice - appliedDiscount;

                // تحديث السعر
                document.getElementById('required-amount').textContent = newPrice.toFixed(2) + ' USD';
                document.querySelector('.total-amount').textContent = newPrice.toFixed(2) + ' USD';

                // تحديث الرصيد المتبقي
                const remaining = userBalance - newPrice;
                const remainingSpan = document.getElementById('remaining-amount');
                remainingSpan.textContent = remaining.toFixed(2) + ' USD';
                remainingSpan.className = 'balance-value ' + (remaining >= 0 ? 'sufficient' : 'insufficient');

                // عرض رسالة نجاح
                resultDiv.className = 'coupon-result success';
                resultDiv.innerHTML = `<i class="fas fa-check-circle"></i> تم تطبيق الكوبون! خصم ${data.discount.toFixed(2)} USD`;

                // تعطيل الحقل والزر
                document.getElementById('couponCode').disabled = true;
                applyBtn.disabled = true;
                applyBtn.innerHTML = '<i class="fas fa-check"></i> تم التطبيق';

                // إعادة التحقق من كفاية الرصيد
                if (selectedPayment === 'balance') {
                    const confirmBtn = document.getElementById('confirmBtn');
                    if (userBalance >= newPrice) {
                        confirmBtn.disabled = false;
                    } else {
                        confirmBtn.disabled = true;
                    }
                }
            } else {
                resultDiv.className = 'coupon-result error';
                resultDiv.innerHTML = `<i class="fas fa-times-circle"></i> ${data.message || 'كوبون غير صالح'}`;
            }
        })
        .catch(error => {
            applyBtn.disabled = false;
            applyBtn.innerHTML = '<i class="fas fa-check"></i> تطبيق';
            resultDiv.className = 'coupon-result error';
            resultDiv.innerHTML = '<i class="fas fa-exclamation-circle"></i> حدث خطأ في التحقق من الكوبون';
            console.error('Error:', error);
        });
    }

    function selectDuration(months) {
        // إزالة التحديد من جميع الخيارات
        document.querySelectorAll('.duration-option').forEach(option => {
            option.classList.remove('selected');
        });

        // تحديد الخيار المختار
        const selectedOption = document.querySelector(`[data-months="${months}"]`);
        selectedOption.classList.add('selected');

        selectedMonths = months;

        // حساب خصم المدة
        const discountPercentage = parseFloat(selectedOption.getAttribute('data-discount') || 0);
        durationDiscount = discountPercentage;

        // حساب السعر الإجمالي للمدة
        let totalBeforeDiscount = monthlyPrice * months;
        let durationDiscountAmount = (totalBeforeDiscount * discountPercentage) / 100;
        let totalAfterDurationDiscount = totalBeforeDiscount - durationDiscountAmount;

        // تحديث originalPrice ليشمل المدة
        originalPrice = totalAfterDurationDiscount;

        // تحديث العرض
        updatePriceDisplay();
    }

    function updatePriceDisplay() {
        // تحديث وصف المدة
        const durationText = {
            1: 'شهر واحد',
            3: '3 شهور',
            6: '6 شهور',
            12: 'سنة كاملة'
        };
        document.getElementById('selectedDuration').textContent = durationText[selectedMonths];

        // عرض/إخفاء خصم المدة
        const discountRow = document.getElementById('discountRow');
        if (durationDiscount > 0) {
            discountRow.style.display = 'flex';
            document.getElementById('durationDiscount').textContent = durationDiscount + '%';
        } else {
            discountRow.style.display = 'none';
        }

        // حساب السعر النهائي مع خصم الكوبون
        const finalPrice = originalPrice - appliedDiscount;

        // تحديث الأسعار
        document.getElementById('totalPrice').textContent = finalPrice.toFixed(2) + ' USD';
        document.getElementById('finalTotalAmount').textContent = finalPrice.toFixed(2) + ' USD';

        // تحديث السعر المطلوب في معلومات الرصيد
        const requiredAmountElem = document.getElementById('required-amount');
        if (requiredAmountElem) {
            requiredAmountElem.textContent = finalPrice.toFixed(2) + ' USD';

            // تحديث الرصيد المتبقي
            const remaining = userBalance - finalPrice;
            const remainingSpan = document.getElementById('remaining-amount');
            if (remainingSpan) {
                remainingSpan.textContent = remaining.toFixed(2) + ' USD';
                remainingSpan.className = 'balance-value ' + (remaining >= 0 ? 'sufficient' : 'insufficient');
            }
        }

        // إعادة التحقق من كفاية الرصيد
        if (selectedPayment === 'balance') {
            const confirmBtn = document.getElementById('confirmBtn');
            if (userBalance >= finalPrice) {
                confirmBtn.disabled = false;
            } else {
                confirmBtn.disabled = true;
            }
        }
    }

    function confirmPurchase() {
        if (!selectedPayment) {
            Swal.fire({
                icon: 'warning',
                title: 'تنبيه',
                text: 'يرجى اختيار طريقة الدفع',
                confirmButtonColor: '#667eea',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
            return;
        }

        // حساب السعر النهائي
        const finalPrice = originalPrice - appliedDiscount;
        const hasDiscount = appliedDiscount > 0;

        let priceHTML = '';
        if (hasDiscount) {
            priceHTML = `
                <div style="margin-bottom: 8px;">
                    <strong>السعر الأصلي:</strong> 
                    <span style="text-decoration: line-through; color: #95a5a6;">${originalPrice.toFixed(2)} USD</span>
                </div>
                <div style="margin-bottom: 8px;">
                    <strong>الخصم:</strong> 
                    <span style="color: #00b894; font-weight: bold;">-${appliedDiscount.toFixed(2)} USD</span>
                </div>
                <div style="margin-bottom: 8px; font-size: 18px;">
                    <strong>السعر النهائي:</strong> 
                    <span style="color: #667eea; font-weight: bold;">${finalPrice.toFixed(2)} USD</span>
                </div>
            `;
        } else {
            priceHTML = `
                <div style="margin-bottom: 8px;">
                    <strong>السعر:</strong> ${finalPrice.toFixed(2)} USD
                </div>
            `;
        }

        Swal.fire({
            title: 'تأكيد عملية الشراء',
            html: `
                <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                    <p style="font-size: 16px; color: #636e72; margin-bottom: 15px;">
                        هل أنت متأكد من إتمام عملية الشراء؟
                    </p>
                    <div style="background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1)); padding: 15px; border-radius: 10px;">
                        <div style="margin-bottom: 8px;">
                            <strong>الباقة:</strong> <?php echo htmlspecialchars($package['name']); ?>
                        </div>
                        ${priceHTML}
                        <div>
                            <strong>طريقة الدفع:</strong> <span id="paymentMethodName"></span>
                        </div>
                    </div>
                </div>
            `,
            icon: 'question',
            showCancelButton: true,
            confirmButtonText: '<i class="fas fa-check"></i> تأكيد',
            cancelButtonText: '<i class="fas fa-times"></i> إلغاء',
            confirmButtonColor: '#667eea',
            cancelButtonColor: '#95a5a6',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb',
            didOpen: () => {
                const paymentNames = {
                    'balance': 'الدفع من الرصيد',
                    'online': 'الدفع الإلكتروني',
                    'representative': 'الدفع من خلال المندوب'
                };
                document.getElementById('paymentMethodName').textContent = paymentNames[selectedPayment];
            }
        }).then((result) => {
            if (result.isConfirmed) {
                processPurchase();
            }
        });
    }

    function processPurchase() {
        // سؤال عن رمز الإحالة
        Swal.fire({
            title: '🎁 رمز الإحالة',
            html: `
                <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                    <p style="font-size: 16px; color: #636e72; margin-bottom: 20px;">
                        هل لديك رمز إحالة؟
                    </p>
                </div>
            `,
            icon: 'question',
            showDenyButton: true,
            showCancelButton: true,
            confirmButtonText: '<i class="fas fa-check"></i> نعم، لدي رمز',
            denyButtonText: '<i class="fas fa-times"></i> لا، ليس لدي',
            cancelButtonText: 'إلغاء',
            confirmButtonColor: '#667eea',
            denyButtonColor: '#95a5a6',
            cancelButtonColor: '#e74c3c',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        }).then((result) => {
            if (result.isConfirmed) {
                // لديه رمز إحالة
                askForReferralCode();
            } else if (result.isDenied) {
                // ليس لديه رمز
                completePurchase(null);
            }
            // إذا ضغط إلغاء، لا شيء يحدث
        });
    }

    function askForReferralCode() {
        Swal.fire({
            title: 'أدخل رمز الإحالة',
            html: `
                <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                    <p style="font-size: 14px; color: #636e72; margin-bottom: 15px;">
                        مثال: USR9A866D735F9738E2
                    </p>
                    <input type="text" id="referralCodeInput" class="swal2-input" placeholder="أدخل رمز الإحالة" style="font-family: 'Cairo', sans-serif;">
                </div>
            `,
            icon: 'info',
            showCancelButton: true,
            confirmButtonText: '<i class="fas fa-search"></i> تحقق',
            cancelButtonText: 'إلغاء',
            confirmButtonColor: '#667eea',
            cancelButtonColor: '#95a5a6',
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb',
            preConfirm: () => {
                const code = document.getElementById('referralCodeInput').value;
                if (!code) {
                    Swal.showValidationMessage('يرجى إدخال رمز الإحالة');
                }
                return code;
            }
        }).then((result) => {
            if (result.isConfirmed) {
                verifyReferralCode(result.value);
            } else {
                // إذا ألغى، اسأل مرة أخرى
                processPurchase();
            }
        });
    }

    function verifyReferralCode(code) {
        // عرض رسالة تحميل
        Swal.fire({
            title: 'جاري التحقق...',
            html: '<i class="fas fa-spinner fa-spin" style="font-size: 48px; color: #667eea;"></i>',
            showConfirmButton: false,
            allowOutsideClick: false,
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });

        // التحقق من رمز الإحالة
        fetch('api/verify_referral.php', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ referral_code: code })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                // رمز صحيح، اسأل للتأكيد
                Swal.fire({
                    title: 'تأكيد رمز الإحالة',
                    html: `
                        <div style="text-align: center; font-family: 'Cairo', sans-serif;">
                            <p style="font-size: 16px; color: #636e72; margin-bottom: 15px;">
                                هل تريد التسجيل عبر:
                            </p>
                            <div style="background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1)); padding: 15px; border-radius: 10px;">
                                <h3 style="color: #667eea; margin-bottom: 5px;">${data.first_name} ${data.last_name}</h3>
                                <p style="font-size: 14px; color: #95a5a6;">رمز الإحالة: ${code}</p>
                            </div>
                        </div>
                    `,
                    icon: 'question',
                    showDenyButton: true,
                    showCancelButton: false,
                    confirmButtonText: '<i class="fas fa-check"></i> موافق',
                    denyButtonText: '<i class="fas fa-times"></i> غير موافق',
                    confirmButtonColor: '#667eea',
                    denyButtonColor: '#e74c3c',
                    background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                    color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
                }).then((result) => {
                    if (result.isConfirmed) {
                        // موافق
                        completePurchase(code);
                    } else {
                        // غير موافق، ارجع للسؤال الأول
                        processPurchase();
                    }
                });
            } else {
                // رمز غير صحيح
                Swal.fire({
                    icon: 'error',
                    title: 'رمز غير صحيح',
                    text: data.message || 'رمز الإحالة غير صحيح',
                    confirmButtonColor: '#667eea',
                    background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                    color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
                }).then(() => {
                    askForReferralCode();
                });
            }
        })
        .catch(error => {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: 'حدث خطأ في التحقق من الرمز',
                confirmButtonColor: '#667eea',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
        });
    }

    function completePurchase(referralCode) {
        // عرض رسالة التحميل
        Swal.fire({
            title: 'جاري معالجة الطلب...',
            html: '<i class="fas fa-spinner fa-spin" style="font-size: 48px; color: #667eea;"></i>',
            showConfirmButton: false,
            allowOutsideClick: false,
            background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
            color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
        });

        // حساب السعر النهائي
        const finalPrice = originalPrice - appliedDiscount;
        
        // إعداد البيانات
        const requestData = {
            package_id: <?php echo $package_id; ?>,
            payment_method: selectedPayment,
            duration_months: selectedMonths,
            final_amount: finalPrice,
            coupon_code: appliedCoupon || null,
            referral_code: referralCode || null
        };
        
        console.log('Sending purchase request:', requestData);

        // إرسال الطلب للخادم
        fetch('api/process_package_purchase.php', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                // إغلاق رسالة التحميل
                Swal.close();
                
                // إظهار الاحتفال
                showCelebration();
                
                // عرض رسالة النجاح بعد 3 ثواني
                setTimeout(() => {
                    Swal.fire({
                        icon: 'success',
                        title: '🎉 مبروك!',
                        html: `
                            <div style="font-family: 'Cairo', sans-serif;">
                                <p style="font-size: 18px; margin-bottom: 10px;">تم شراء الباقة بنجاح!</p>
                                <p style="font-size: 14px; color: #95a5a6;">سيتم توجيهك للصفحة الرئيسية...</p>
                            </div>
                        `,
                        showConfirmButton: false,
                        timer: 2000,
                        background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                        color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
                    }).then(() => {
                        window.location.href = 'index.php';
                    });
                }, 3000);
            } else {
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ',
                    text: data.message || 'حدث خطأ أثناء معالجة الطلب',
                    confirmButtonColor: '#667eea',
                    background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                    color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
                });
            }
        })
        .catch(error => {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: 'حدث خطأ في الاتصال بالخادم',
                confirmButtonColor: '#667eea',
                background: document.body.classList.contains('light-theme') ? '#ffffff' : '#1a1d29',
                color: document.body.classList.contains('light-theme') ? '#2d3436' : '#e4e6eb'
            });
        });
    }

    // دالة إظهار الاحتفال
    function showCelebration() {
        // إنشاء canvas للكونفيتي
        const canvas = document.createElement('canvas');
        canvas.style.position = 'fixed';
        canvas.style.top = '0';
        canvas.style.left = '0';
        canvas.style.width = '100%';
        canvas.style.height = '100%';
        canvas.style.pointerEvents = 'none';
        canvas.style.zIndex = '9999';
        document.body.appendChild(canvas);

        const ctx = canvas.getContext('2d');
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;

        // مصفوفة الكونفيتي
        const confetti = [];
        const colors = ['#667eea', '#764ba2', '#f093fb', '#4facfe', '#00f2fe', '#43e97b', '#fa709a', '#fee140'];
        
        // إنشاء الكونفيتي
        for (let i = 0; i < 150; i++) {
            confetti.push({
                x: Math.random() * canvas.width,
                y: Math.random() * canvas.height - canvas.height,
                r: Math.random() * 6 + 4,
                d: Math.random() * 150 + 10,
                color: colors[Math.floor(Math.random() * colors.length)],
                tilt: Math.floor(Math.random() * 10) - 10,
                tiltAngleIncremental: Math.random() * 0.07 + 0.05,
                tiltAngle: 0
            });
        }

        // رسم الكونفيتي
        function draw() {
            ctx.clearRect(0, 0, canvas.width, canvas.height);

            confetti.forEach((c, index) => {
                ctx.beginPath();
                ctx.lineWidth = c.r / 2;
                ctx.strokeStyle = c.color;
                ctx.moveTo(c.x + c.tilt + c.r / 4, c.y);
                ctx.lineTo(c.x + c.tilt, c.y + c.tilt + c.r / 4);
                ctx.stroke();

                // تحديث الموقع
                c.tiltAngle += c.tiltAngleIncremental;
                c.y += (Math.cos(c.d) + 3 + c.r / 2) / 2;
                c.tilt = Math.sin(c.tiltAngle - index / 3) * 15;

                // إذا خرج من الشاشة، إعادة تعيينه
                if (c.y > canvas.height) {
                    confetti.splice(index, 1);
                }
            });

            if (confetti.length > 0) {
                requestAnimationFrame(draw);
            } else {
                // إزالة canvas بعد الانتهاء
                document.body.removeChild(canvas);
            }
        }

        draw();

        // إضافة أصوات الاحتفال (إموجي متحرك)
        const emojiContainer = document.createElement('div');
        emojiContainer.style.position = 'fixed';
        emojiContainer.style.top = '50%';
        emojiContainer.style.left = '50%';
        emojiContainer.style.transform = 'translate(-50%, -50%)';
        emojiContainer.style.fontSize = '100px';
        emojiContainer.style.zIndex = '10000';
        emojiContainer.style.animation = 'celebrationBounce 1s ease-in-out';
        emojiContainer.innerHTML = '🎉🎊🎆';
        document.body.appendChild(emojiContainer);

        // إضافة animation CSS
        if (!document.getElementById('celebrationStyle')) {
            const style = document.createElement('style');
            style.id = 'celebrationStyle';
            style.innerHTML = `
                @keyframes celebrationBounce {
                    0%, 100% { transform: translate(-50%, -50%) scale(0); opacity: 0; }
                    50% { transform: translate(-50%, -50%) scale(1.2); opacity: 1; }
                }
            `;
            document.head.appendChild(style);
        }

        // إزالة الإموجي بعد ثانية
        setTimeout(() => {
            if (document.body.contains(emojiContainer)) {
                document.body.removeChild(emojiContainer);
            }
        }, 1000);
    }
</script>

<?php include 'includes/footer.php'; ?>
