<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "إدارة الكوبونات | Kingmaster";
include 'includes/admin_head.php';
include 'includes/admin_navbar_top.php';
include 'includes/admin_navbar_actions.php';
include 'includes/admin_navbar_extra_actions.php';
include 'includes/admin_sidebar_right.php';
include 'includes/admin_sidebar_left.php';
?>

<style>
    .products-container {
        padding: 30px;
        max-width: 1400px;
        margin: 120px auto 0 auto;
    }

    .products-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 30px;
        flex-wrap: wrap;
        gap: 20px;
    }

    .products-header h1 {
        font-size: 32px;
        color: var(--text-primary);
        display: flex;
        align-items: center;
        gap: 15px;
    }

    .products-header h1 i {
        color: var(--primary-color);
        font-size: 36px;
    }

    .add-product-btn {
        padding: 12px 25px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        text-decoration: none;
        border-radius: 10px;
        display: flex;
        align-items: center;
        gap: 10px;
        font-weight: 600;
        transition: all 0.3s ease;
        border: none;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
    }

    .add-product-btn:hover {
        transform: translateY(-2px);
        box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
    }

    /* شريط البحث والفلاتر */
    .filters-section {
        background: var(--card-bg);
        padding: 25px;
        border-radius: 15px;
        margin-bottom: 30px;
        box-shadow: var(--shadow);
    }

    .search-bar {
        display: flex;
        gap: 15px;
        margin-bottom: 20px;
        flex-wrap: wrap;
    }

    .search-input-wrapper {
        flex: 1;
        min-width: 250px;
        position: relative;
    }

    .search-input-wrapper i {
        position: absolute;
        right: 15px;
        top: 50%;
        transform: translateY(-50%);
        color: var(--text-secondary);
        font-size: 18px;
    }

    .search-input-wrapper input {
        width: 100%;
        padding: 14px 45px 14px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        font-weight: 500;
        background: #1e293b;
        color: #f1f5f9;
        transition: all 0.3s ease;
    }

    .search-input-wrapper input:focus {
        outline: none;
        border-color: var(--primary-color);
    }

    .search-btn {
        padding: 14px 30px;
        background: var(--primary-color);
        color: white;
        border: none;
        border-radius: 10px;
        cursor: pointer;
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        font-weight: 700;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: all 0.3s ease;
    }

    .search-btn:hover {
        background: var(--primary-hover);
        transform: translateY(-2px);
    }

    .filters-row {
        display: flex;
        gap: 15px;
        flex-wrap: wrap;
    }

    .filter-group {
        flex: 1;
        min-width: 200px;
    }

    .filter-group label {
        display: block;
        margin-bottom: 10px;
        font-weight: 700;
        color: var(--text-primary);
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
    }

    .filter-group select,
    .filter-group input {
        width: 100%;
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        font-weight: 500;
        background: #1e293b;
        color: #f1f5f9;
        transition: all 0.3s ease;
    }

    .filter-group select option {
        background: #1e293b;
        color: #f1f5f9;
        font-family: 'Cairo', sans-serif;
        font-weight: 500;
        padding: 10px;
    }

    .filter-group select:focus,
    .filter-group input:focus {
        outline: none;
        border-color: var(--primary-color);
    }

    /* شبكة المنتجات */
    .products-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 25px;
        margin-bottom: 30px;
    }

    .product-card {
        background: var(--card-bg);
        border-radius: 15px;
        overflow: hidden;
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
        transition: all 0.3s ease;
        position: relative;
        border: 1px solid var(--border-color);
    }

    .product-card:hover {
        transform: translateY(-8px);
        box-shadow: 0 15px 35px rgba(0,0,0,0.2);
        border-color: var(--primary-color);
    }

    .product-image {
        width: 100%;
        height: 220px;
        object-fit: cover;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 80px;
        color: white;
        position: relative;
    }

    .product-image img {
        width: 100%;
        height: 100%;
        object-fit: cover;
    }

    .product-image i {
        position: absolute;
    }

    .product-badge {
        position: absolute;
        top: 15px;
        right: 15px;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 5px;
    }

    .badge-digital {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
    }

    .badge-physical {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        color: white;
    }

    .product-info {
        padding: 20px;
    }

    .product-name {
        font-size: 17px;
        font-weight: 700;
        color: var(--text-primary);
        margin-bottom: 10px;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
        line-height: 1.5;
        font-family: 'Cairo', sans-serif;
    }

    .product-description {
        font-size: 13px;
        color: var(--text-secondary);
        margin-bottom: 15px;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
        line-height: 1.7;
        font-family: 'Cairo', sans-serif;
        font-weight: 500;
    }

    .product-meta {
        display: flex;
        gap: 10px;
        margin-bottom: 15px;
        flex-wrap: wrap;
    }

    .meta-badge {
        padding: 6px 12px;
        border-radius: 15px;
        font-size: 12px;
        background: var(--bg-primary);
        color: var(--text-secondary);
        display: flex;
        align-items: center;
        gap: 6px;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
    }

    .product-footer {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        padding-top: 15px;
        border-top: 2px solid var(--border-color);
    }

    .product-price {
        font-size: 22px;
        font-weight: 800;
        color: var(--primary-color);
        display: flex;
        align-items: center;
        gap: 8px;
        font-family: 'Cairo', sans-serif;
    }

    .product-price i {
        font-size: 20px;
    }

    .product-actions {
        display: flex;
        gap: 8px;
    }

    .btn-action {
        padding: 8px 16px;
        border: none;
        border-radius: 8px;
        cursor: pointer;
        font-weight: 600;
        font-size: 13px;
        font-family: 'Cairo', sans-serif;
        display: flex;
        align-items: center;
        gap: 6px;
        transition: all 0.3s ease;
    }

    .btn-edit {
        background: rgba(34, 197, 94, 0.1);
        color: #16a34a;
    }

    .btn-edit:hover {
        background: #16a34a;
        color: white;
        transform: translateY(-2px);
    }

    .btn-delete {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
    }

    .btn-delete:hover {
        background: #dc2626;
        color: white;
        transform: translateY(-2px);
    }

    .stock-badge {
        position: absolute;
        top: 15px;
        left: 15px;
        padding: 6px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
        background: rgba(0,0,0,0.7);
        color: white;
    }

    .stock-low {
        background: rgba(255, 107, 107, 0.9) !important;
    }

    .stock-out {
        background: rgba(150, 150, 150, 0.9) !important;
    }

    /* Modal Styles */
    .modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        z-index: 9999;
        justify-content: center;
        align-items: center;
        backdrop-filter: blur(5px);
    }

    .modal.active {
        display: flex;
    }

    .modal-content {
        background: var(--card-bg);
        border-radius: 20px;
        padding: 30px;
        max-width: 600px;
        width: 90%;
        max-height: 90vh;
        overflow-y: auto;
        border: 2px solid var(--border-color);
        animation: modalSlideIn 0.3s ease;
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
    }

    .close-modal {
        background: none;
        border: none;
        font-size: 28px;
        color: var(--text-secondary);
        cursor: pointer;
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 50%;
        transition: all 0.3s ease;
    }

    .close-modal:hover {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-label {
        display: block;
        margin-bottom: 8px;
        font-weight: 600;
        color: var(--text-primary);
        font-family: 'Cairo', sans-serif;
        font-size: 15px;
    }

    .form-input,
    .form-textarea,
    .form-select {
        width: 100%;
        padding: 12px 15px;
        border: 2px solid var(--border-color);
        border-radius: 10px;
        background: #1e293b;
        color: #f1f5f9;
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .form-select option {
        background: #1e293b;
        color: #f1f5f9;
        font-family: 'Cairo', sans-serif;
        padding: 10px;
    }

    .form-input:focus,
    .form-textarea:focus,
    .form-select:focus {
        outline: none;
        border-color: #667eea;
    }

    .form-textarea {
        min-height: 100px;
        resize: vertical;
    }

    .form-checkbox-group {
        display: flex;
        gap: 20px;
        flex-wrap: wrap;
    }

    .checkbox-item {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .checkbox-item input[type="checkbox"] {
        width: 20px;
        height: 20px;
        cursor: pointer;
    }

    .checkbox-item label {
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        color: var(--text-primary);
    }

    .form-row {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 15px;
    }

    /* Tags Input Style */
    .tags-input-wrapper {
        border: 2px solid var(--border-color);
        border-radius: 10px;
        padding: 8px;
        background: #1e293b;
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        align-items: center;
        min-height: 50px;
    }

    .tags-container {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }

    .tag {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        padding: 6px 12px;
        border-radius: 20px;
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        font-family: 'Cairo', sans-serif;
        font-weight: 600;
        animation: tagSlideIn 0.3s ease;
    }

    @keyframes tagSlideIn {
        from {
            opacity: 0;
            transform: scale(0.8);
        }
        to {
            opacity: 1;
            transform: scale(1);
        }
    }

    .tag-remove {
        cursor: pointer;
        width: 18px;
        height: 18px;
        background: rgba(255, 255, 255, 0.3);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 12px;
        transition: all 0.3s ease;
    }

    .tag-remove:hover {
        background: rgba(255, 255, 255, 0.5);
        transform: scale(1.1);
    }

    .tags-input {
        flex: 1;
        min-width: 150px;
        border: none;
        outline: none;
        background: transparent;
        color: #f1f5f9;
        font-size: 15px;
        font-family: 'Cairo', sans-serif;
        padding: 5px;
    }

    .tags-input::placeholder {
        color: #94a3b8;
    }

    body.light-theme .tags-input-wrapper {
        background: #ffffff;
    }

    body.light-theme .tags-input {
        color: #2d3436;
    }

    body.light-theme .tags-input::placeholder {
        color: #636e72;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 25px;
        padding-top: 20px;
        border-top: 2px solid var(--border-color);
    }

    .btn-submit {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border: none;
        padding: 12px 30px;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .btn-submit:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 30px rgba(102, 126, 234, 0.4);
    }

    .btn-cancel {
        background: rgba(239, 68, 68, 0.1);
        color: #dc2626;
        border: none;
        padding: 12px 30px;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
    }

    .btn-cancel:hover {
        background: #dc2626;
        color: white;
    }

    /* رسالة فارغة */
    .empty-state {
        text-align: center;
        padding: 80px 20px;
    }

    .empty-state i {
        font-size: 120px;
        color: var(--text-secondary);
        margin-bottom: 20px;
        opacity: 0.3;
    }

    .empty-state h3 {
        font-size: 24px;
        color: var(--text-primary);
        margin-bottom: 10px;
    }

    .empty-state p {
        font-size: 16px;
        color: var(--text-secondary);
    }

    /* التحميل */
    .loading {
        text-align: center;
        padding: 60px 20px;
    }

    .spinner {
        border: 4px solid var(--border-color);
        border-top: 4px solid var(--primary-color);
        border-radius: 50%;
        width: 50px;
        height: 50px;
        animation: spin 1s linear infinite;
        margin: 0 auto 20px;
    }

    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }

    /* أيقونات ملونة ومتحركة */
    .products-header h1 i {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: bounce 2s ease-in-out infinite;
    }

    @keyframes bounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-10px); }
    }

    .add-product-btn i {
        animation: rotate-scale 2s ease-in-out infinite;
    }

    @keyframes rotate-scale {
        0%, 100% { transform: rotate(0deg) scale(1); }
        50% { transform: rotate(180deg) scale(1.2); }
    }

    .search-input-wrapper i {
        color: #3b82f6 !important;
        animation: pulse-search 2s ease-in-out infinite;
    }

    @keyframes pulse-search {
        0%, 100% { transform: scale(1); opacity: 1; }
        50% { transform: scale(1.2); opacity: 0.7; }
    }

    .search-btn i {
        animation: shake-search 1.5s ease-in-out infinite;
    }

    @keyframes shake-search {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-3px); }
        75% { transform: translateX(3px); }
    }

    .filter-group label i {
        animation: colorful-spin 3s linear infinite;
    }

    @keyframes colorful-spin {
        0% { color: #667eea; transform: rotate(0deg); }
        25% { color: #f093fb; transform: rotate(90deg); }
        50% { color: #fbbf24; transform: rotate(180deg); }
        75% { color: #10b981; transform: rotate(270deg); }
        100% { color: #667eea; transform: rotate(360deg); }
    }

    .product-badge i {
        animation: heartbeat 1.5s ease-in-out infinite;
    }

    @keyframes heartbeat {
        0%, 100% { transform: scale(1); }
        10%, 30% { transform: scale(1.1); }
        20%, 40% { transform: scale(0.9); }
    }

    .stock-badge i {
        animation: blink 2s ease-in-out infinite;
    }

    @keyframes blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.3; }
    }

    .meta-badge i {
        animation: wiggle 2s ease-in-out infinite;
    }

    .meta-badge:nth-child(1) i {
        color: #fbbf24;
    }

    .meta-badge:nth-child(2) i {
        color: #f43f5e;
        animation: fire-flicker 1s ease-in-out infinite;
    }

    .meta-badge:nth-child(3) i {
        color: #ec4899;
    }

    @keyframes wiggle {
        0%, 100% { transform: rotate(0deg); }
        25% { transform: rotate(-15deg); }
        75% { transform: rotate(15deg); }
    }

    @keyframes fire-flicker {
        0%, 100% { transform: scale(1); filter: brightness(1); }
        50% { transform: scale(1.2); filter: brightness(1.3); }
    }

    .product-price i {
        color: #fbbf24;
        animation: coin-flip 3s ease-in-out infinite;
    }

    @keyframes coin-flip {
        0%, 100% { transform: rotateY(0deg); }
        50% { transform: rotateY(180deg); }
    }

    .btn-edit i {
        color: #16a34a;
        animation: edit-bounce 2s ease-in-out infinite;
    }

    @keyframes edit-bounce {
        0%, 100% { transform: translateY(0) rotate(0deg); }
        50% { transform: translateY(-5px) rotate(10deg); }
    }

    .btn-delete i {
        color: #dc2626;
        animation: shake-delete 2s ease-in-out infinite;
    }

    @keyframes shake-delete {
        0%, 100% { transform: rotate(0deg); }
        10%, 30%, 50%, 70%, 90% { transform: rotate(-5deg); }
        20%, 40%, 60%, 80% { transform: rotate(5deg); }
    }

    .modal-title::before {
        content: '✨';
        margin-left: 10px;
        animation: sparkle 1.5s ease-in-out infinite;
    }

    @keyframes sparkle {
        0%, 100% { transform: scale(1) rotate(0deg); opacity: 1; }
        50% { transform: scale(1.3) rotate(180deg); opacity: 0.7; }
    }

    .close-modal {
        animation: rotate-close 3s linear infinite;
    }

    @keyframes rotate-close {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    .form-label i {
        color: #8b5cf6;
        animation: float-up-down 2s ease-in-out infinite;
    }

    @keyframes float-up-down {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-5px); }
    }

    .btn-submit i {
        animation: save-pulse 1.5s ease-in-out infinite;
    }

    @keyframes save-pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.2); }
    }

    .btn-cancel:hover {
        animation: cancel-shake 0.5s ease-in-out;
    }

    @keyframes cancel-shake {
        0%, 100% { transform: translateX(0); }
        25% { transform: translateX(-5px); }
        75% { transform: translateX(5px); }
    }

    .tag-remove {
        animation: rotate-x 2s linear infinite;
    }

    @keyframes rotate-x {
        0%, 100% { transform: rotate(0deg); }
        50% { transform: rotate(180deg); }
    }

    .empty-state i {
        background: linear-gradient(135deg, #667eea, #764ba2, #f093fb, #fbbf24);
        background-size: 400% 400%;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        animation: gradient-move 3s ease infinite, float 3s ease-in-out infinite;
    }

    @keyframes gradient-move {
        0% { background-position: 0% 50%; }
        50% { background-position: 100% 50%; }
        100% { background-position: 0% 50%; }
    }

    @keyframes float {
        0%, 100% { transform: translateY(0px); }
        50% { transform: translateY(-20px); }
    }

    .loading .spinner {
        border-top-color: #667eea;
        animation: spin 1s linear infinite, color-change 3s ease-in-out infinite;
    }

    @keyframes color-change {
        0%, 100% { border-top-color: #667eea; }
        25% { border-top-color: #f093fb; }
        50% { border-top-color: #fbbf24; }
        75% { border-top-color: #10b981; }
    }

    /* Light Theme Fixes */
    body.light-theme .filter-group select,
    body.light-theme .filter-group input,
    body.light-theme .search-input-wrapper input,
    body.light-theme .form-input,
    body.light-theme .form-textarea,
    body.light-theme .form-select {
        background: #ffffff !important;
        color: #2d3436 !important;
        border-color: rgba(0, 0, 0, 0.1);
    }

    body.light-theme .filter-group select option,
    body.light-theme .form-select option {
        background: #ffffff !important;
        color: #2d3436 !important;
    }

    body.light-theme .product-card,
    body.light-theme .filters-section,
    body.light-theme .modal-content {
        background: rgba(255, 255, 255, 0.95);
        border-color: rgba(0, 0, 0, 0.1);
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
    }

    body.light-theme .product-card:hover {
        box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
    }

    body.light-theme .meta-badge {
        background: #f5f6fa;
        color: #636e72;
    }

    body.light-theme .product-name,
    body.light-theme .filter-group label,
    body.light-theme .empty-state h3,
    body.light-theme .modal-title,
    body.light-theme .form-label {
        color: #2d3436;
    }

    body.light-theme .product-description,
    body.light-theme .empty-state p {
        color: #636e72;
    }

    /* Responsive */
    @media (max-width: 768px) {
        .products-container {
            padding: 20px 15px;
        }

        .products-header h1 {
            font-size: 26px;
        }

        .filters-row {
            flex-direction: column;
        }

        .products-grid {
            grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
            gap: 20px;
        }

        .form-row {
            grid-template-columns: 1fr;
        }
    }
</style>

<div class="products-container">
    <div class="products-header">
        <h1>
            <i class="fas fa-box-open"></i>
            إدارة المنتجات
        </h1>
        <button class="add-product-btn" onclick="openAddModal()">
            <i class="fas fa-plus"></i>
            إضافة منتج جديد
        </button>
    </div>

    <!-- قسم الفلاتر والبحث -->
    <div class="filters-section">
        <div class="search-bar">
            <div class="search-input-wrapper">
                <i class="fas fa-search"></i>
                <input type="text" id="searchInput" placeholder="ابحث عن المنتجات...">
            </div>
            <button class="search-btn" onclick="loadProducts()">
                <i class="fas fa-search"></i>
                بحث
            </button>
        </div>

        <div class="filters-row">
            <div class="filter-group">
                <label><i class="fas fa-layer-group" style="color: #8b5cf6;"></i> الفئة</label>
                <select id="categoryFilter" onchange="loadProducts()">
                    <option value="">جميع الفئات</option>
                    <option value="courses">دورات</option>
                    <option value="clothing">ملابس</option>
                    <option value="ebooks">كتب إلكترونية</option>
                    <option value="shoes">أحذية</option>
                    <option value="other">أخرى</option>
                </select>
            </div>

            <div class="filter-group">
                <label><i class="fas fa-cube" style="color: #10b981;"></i> النوع</label>
                <select id="typeFilter" onchange="loadProducts()">
                    <option value="">الكل</option>
                    <option value="0">منتجات حقيقية</option>
                    <option value="1">منتجات رقمية</option>
                </select>
            </div>

            <div class="filter-group">
                <label><i class="fas fa-signal" style="color: #f59e0b;"></i> الحالة</label>
                <select id="statusFilter" onchange="loadProducts()">
                    <option value="">جميع الحالات</option>
                    <option value="active">نشط</option>
                    <option value="inactive">غير نشط</option>
                    <option value="out_of_stock">نفذت الكمية</option>
                </select>
            </div>
        </div>
    </div>

    <!-- شبكة المنتجات -->
    <div id="productsGrid" class="products-grid">
        <div class="loading">
            <div class="spinner"></div>
            <p>جاري تحميل المنتجات...</p>
        </div>
    </div>
</div>

<!-- Add/Edit Product Modal -->
<div class="modal" id="productModal">
    <div class="modal-content">
        <div class="modal-header">
            <h2 class="modal-title" id="modalTitle">إضافة منتج جديد</h2>
            <button class="close-modal" onclick="closeModal()">&times;</button>
        </div>
        <form id="productForm" onsubmit="saveProduct(event)">
            <input type="hidden" id="productId" name="id">
            
            <div class="form-group">
                <label class="form-label">اسم المنتج *</label>
                <input type="text" class="form-input" id="productName" name="name" required>
            </div>

            <div class="form-group">
                <label class="form-label">الوصف</label>
                <textarea class="form-textarea" id="productDescription" name="description"></textarea>
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label class="form-label">السعر *</label>
                    <input type="number" step="0.01" class="form-input" id="productPrice" name="price" required>
                </div>
                <div class="form-group">
                    <label class="form-label">نسبة الخصم %</label>
                    <input type="number" step="0.01" class="form-input" id="productDiscount" name="discount_percentage" value="0">
                </div>
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label class="form-label">سعر العمولة</label>
                    <input type="number" step="0.01" class="form-input" id="productCommission" name="commission" value="0" placeholder="0.00">
                </div>
                <div class="form-group">
                    <label class="form-label">الفئة</label>
                    <select class="form-select" id="productCategory" name="category">
                        <option value="courses">دورات</option>
                        <option value="clothing">ملابس</option>
                        <option value="ebooks">كتب إلكترونية</option>
                        <option value="shoes">أحذية</option>
                        <option value="other">أخرى</option>
                    </select>
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">الكمية المتاحة</label>
                <input type="number" class="form-input" id="productStock" name="stock_quantity" value="0">
            </div>

            <div class="form-group">
                <label class="form-label">صورة المنتج</label>
                <input type="file" class="form-input" id="productImage" name="product_image" accept="image/*">
                <div id="imagePreview" style="margin-top: 10px; display: none;">
                    <img id="previewImg" src="" alt="Preview" style="max-width: 200px; border-radius: 10px; border: 2px solid var(--border-color);">
                </div>
                <input type="hidden" id="existingImage" name="existing_image">
            </div>

            <div class="form-group">
                <label class="form-label">الحالة</label>
                <select class="form-select" id="productStatus" name="status">
                    <option value="active">نشط</option>
                    <option value="inactive">غير نشط</option>
                    <option value="out_of_stock">نفذت الكمية</option>
                </select>
            </div>

            <div class="form-group">
                <label class="form-label">الألوان (اضغط Enter للإضافة)</label>
                <div class="tags-input-wrapper" id="colorsWrapper">
                    <div class="tags-container" id="colorsContainer"></div>
                    <input type="text" class="tags-input" id="colorInput" placeholder="أدخل اللون">
                </div>
                <input type="hidden" id="productColors" name="colors">
            </div>

            <div class="form-group">
                <label class="form-label">المقاسات (اضغط Enter للإضافة)</label>
                <div class="tags-input-wrapper" id="sizesWrapper">
                    <div class="tags-container" id="sizesContainer"></div>
                    <input type="text" class="tags-input" id="sizeInput" placeholder="أدخل المقاس">
                </div>
                <input type="hidden" id="productSizes" name="sizes">
            </div>

            <div class="form-group">
                <label class="form-label">الخصائص</label>
                <div class="form-checkbox-group">
                    <div class="checkbox-item">
                        <input type="checkbox" id="isDigital" name="is_digital">
                        <label for="isDigital">منتج رقمي</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="isNew" name="is_new">
                        <label for="isNew">جديد</label>
                    </div>
                    <div class="checkbox-item">
                        <input type="checkbox" id="isFeatured" name="is_featured">
                        <label for="isFeatured">مميز</label>
                    </div>
                </div>
            </div>

            <div class="modal-footer">
                <button type="button" class="btn-cancel" onclick="closeModal()">إلغاء</button>
                <button type="submit" class="btn-submit">
                    <i class="fas fa-save"></i> حفظ
                </button>
            </div>
        </form>
    </div>
</div>

<script>
let colors = [];
let sizes = [];

// تحميل المنتجات عند تحميل الصفحة
document.addEventListener('DOMContentLoaded', function() {
    loadProducts();
    initializeTags();
    initializeImagePreview();
});

// تحميل جميع المنتجات
function loadProducts() {
    const search = document.getElementById('searchInput').value;
    const category = document.getElementById('categoryFilter').value;
    const type = document.getElementById('typeFilter').value;
    const status = document.getElementById('statusFilter').value;

    fetch('api/get_products.php?admin=true')
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            let products = data.products;

            // فلترة
            if (search) {
                products = products.filter(p => 
                    p.name.toLowerCase().includes(search.toLowerCase()) ||
                    (p.description && p.description.toLowerCase().includes(search.toLowerCase()))
                );
            }
            if (category) {
                products = products.filter(p => p.category === category);
            }
            if (type !== '') {
                products = products.filter(p => p.is_digital == type);
            }
            if (status) {
                products = products.filter(p => p.status === status);
            }

            displayProducts(products);
        } else {
            console.error('Error loading products:', data.message);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        document.getElementById('productsGrid').innerHTML = `
            <div class="empty-state">
                <i class="fas fa-exclamation-triangle"></i>
                <h3>حدث خطأ</h3>
                <p>فشل تحميل المنتجات</p>
            </div>
        `;
    });
}

// عرض المنتجات
function displayProducts(products) {
    const grid = document.getElementById('productsGrid');
    
    if (!products || products.length === 0) {
        grid.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-box-open"></i>
                <h3>لا توجد منتجات</h3>
                <p>لم يتم العثور على أي منتجات</p>
            </div>
        `;
        return;
    }

    grid.innerHTML = products.map(product => `
        <div class="product-card">
            <div class="product-image">
                ${product.image_url ? `<img src=uploads/products/${product.image_url} alt="${product.name}">` : '<i class="fas fa-box"></i>'}
                <div class="product-badge ${product.is_digital == 1 ? 'badge-digital' : 'badge-physical'}">
                    <i class="fas ${product.is_digital == 1 ? 'fa-cloud' : 'fa-box'}"></i>
                    ${product.is_digital == 1 ? 'رقمي' : 'فيزيائي'}
                </div>
                ${product.stock_quantity <= 0 ? '<div class="stock-badge stock-out"><i class="fas fa-times"></i> نفذت الكمية</div>' :
                  product.stock_quantity < 10 ? '<div class="stock-badge stock-low"><i class="fas fa-exclamation"></i> كمية قليلة</div>' :
                  '<div class="stock-badge"><i class="fas fa-check"></i> متوفر</div>'}
            </div>
            <div class="product-info">
                <div class="product-name">${product.name}</div>
                <div class="product-description">${product.description || 'لا يوجد وصف'}</div>
                
                <div class="product-meta">
                    ${product.is_new == 1 ? '<span class="meta-badge"><i class="fas fa-star"></i> جديد</span>' : ''}
                    ${product.is_featured == 1 ? '<span class="meta-badge"><i class="fas fa-fire"></i> مميز</span>' : ''}
                    ${product.discount_percentage > 0 ? `<span class="meta-badge"><i class="fas fa-tag"></i> خصم ${product.discount_percentage}%</span>` : ''}
                </div>

                <div class="product-footer">
                    <div>
                        <div class="product-price">
                            <i class="fas fa-coins"></i>
                            ${product.final_price} ريال
                        </div>
                        ${product.discount_percentage > 0 ? `<small style="text-decoration: line-through; color: var(--text-secondary);">${product.price} ريال</small>` : ''}
                    </div>
                    <div class="product-actions">
                        <button class="btn-action btn-edit" onclick='editProduct(${JSON.stringify(product).replace(/'/g, "&apos;")})' title="تعديل">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn-action btn-delete" onclick="deleteProduct(${product.id})" title="حذف">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `).join('');
}

// فتح مودال الإضافة
function openAddModal() {
    document.getElementById('modalTitle').textContent = 'إضافة منتج جديد';
    document.getElementById('productForm').reset();
    document.getElementById('productId').value = '';
    document.getElementById('existingImage').value = '';
    document.getElementById('imagePreview').style.display = 'none';
    colors = [];
    sizes = [];
    renderTags();
    document.getElementById('productModal').classList.add('active');
}

// فتح مودال التعديل
function editProduct(product) {
    document.getElementById('modalTitle').textContent = 'تعديل المنتج';
    document.getElementById('productId').value = product.id;
    document.getElementById('productName').value = product.name;
    document.getElementById('productDescription').value = product.description || '';
    document.getElementById('productPrice').value = product.price;
    document.getElementById('productDiscount').value = product.discount_percentage || 0;
    document.getElementById('productCommission').value = product.commission || 0;
    document.getElementById('productStock').value = product.stock_quantity || 0;
    document.getElementById('productCategory').value = product.category || 'other';
    document.getElementById('productStatus').value = product.status || 'active';
    document.getElementById('isDigital').checked = product.is_digital == 1;
    document.getElementById('isNew').checked = product.is_new == 1;
    document.getElementById('isFeatured').checked = product.is_featured == 1;
    
    // تحميل الصورة
    document.getElementById('existingImage').value = product.image_url || '';
    if (product.image_url) {
        document.getElementById('previewImg').src = product.image_url;
        document.getElementById('imagePreview').style.display = 'block';
    } else {
        document.getElementById('imagePreview').style.display = 'none';
    }
    
    // تحميل الألوان والمقاسات
    colors = product.colors ? product.colors.split(',').filter(c => c.trim()) : [];
    sizes = product.sizes ? product.sizes.split(',').filter(s => s.trim()) : [];
    renderTags();
    
    document.getElementById('productModal').classList.add('active');
}

// إغلاق المودال
function closeModal() {
    document.getElementById('productModal').classList.remove('active');
}

// حفظ المنتج
function saveProduct(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    const productId = document.getElementById('productId').value;
    
    formData.set('is_digital', document.getElementById('isDigital').checked ? 1 : 0);
    formData.set('is_new', document.getElementById('isNew').checked ? 1 : 0);
    formData.set('is_featured', document.getElementById('isFeatured').checked ? 1 : 0);
    
    // إضافة الألوان والمقاسات
    console.log('Colors:', colors);
    console.log('Sizes:', sizes);
    formData.set('colors', colors.join(','));
    formData.set('sizes', sizes.join(','));
    console.log('Colors string:', formData.get('colors'));
    console.log('Sizes string:', formData.get('sizes'));
    
    const url = productId ? 'api/update_product.php' : 'api/add_product.php';
    
    fetch(url, {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            closeModal();
            loadProducts();
            alert('تم حفظ المنتج بنجاح!');
        } else {
            alert('حدث خطأ: ' + data.message);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        alert('حدث خطأ في الاتصال');
    });
}

// حذف منتج
function deleteProduct(productId) {
    if (!confirm('هل أنت متأكد من حذف هذا المنتج؟')) {
        return;
    }
    
    fetch('api/delete_product.php', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ id: productId })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            loadProducts();
            alert('تم حذف المنتج بنجاح!');
        } else {
            alert('حدث خطأ: ' + data.message);
        }
    })
    .catch(error => {
        console.error('Error:', error);
        alert('حدث خطأ في الاتصال');
    });
}

// =============== Tags System ===============
function initializeTags() {
    const colorInput = document.getElementById('colorInput');
    const sizeInput = document.getElementById('sizeInput');
    
    colorInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            addTag('color', this.value.trim());
            this.value = '';
        }
    });
    
    sizeInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            addTag('size', this.value.trim());
            this.value = '';
        }
    });
}

function addTag(type, value) {
    if (!value) return;
    
    if (type === 'color') {
        if (!colors.includes(value)) {
            colors.push(value);
            renderTags();
        }
    } else if (type === 'size') {
        if (!sizes.includes(value)) {
            sizes.push(value);
            renderTags();
        }
    }
}

function removeTag(type, index) {
    if (type === 'color') {
        colors.splice(index, 1);
    } else if (type === 'size') {
        sizes.splice(index, 1);
    }
    renderTags();
}

function renderTags() {
    // عرض الألوان
    const colorsContainer = document.getElementById('colorsContainer');
    colorsContainer.innerHTML = colors.map((color, index) => `
        <div class="tag">
            <span>${color}</span>
            <span class="tag-remove" onclick="removeTag('color', ${index})">&times;</span>
        </div>
    `).join('');
    
    // عرض المقاسات
    const sizesContainer = document.getElementById('sizesContainer');
    sizesContainer.innerHTML = sizes.map((size, index) => `
        <div class="tag">
            <span>${size}</span>
            <span class="tag-remove" onclick="removeTag('size', ${index})">&times;</span>
        </div>
    `).join('');
    
    // تحديث الحقول المخفية
    document.getElementById('productColors').value = colors.join(',');
    document.getElementById('productSizes').value = sizes.join(',');
}

// =============== Image Preview ===============
function initializeImagePreview() {
    const imageInput = document.getElementById('productImage');
    imageInput.addEventListener('change', function(e) {
        const file = e.target.files[0];
        if (file) {
            const reader = new FileReader();
            reader.onload = function(e) {
                document.getElementById('previewImg').src = e.target.result;
                document.getElementById('imagePreview').style.display = 'block';
            };
            reader.readAsDataURL(file);
        }
    });
}
</script>

<?php include 'includes/admin_footer.php'; ?>
