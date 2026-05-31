 

<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    header('Location: landing.php');
    exit;
}
require_once 'includes/functions.php';
$user_id = $_SESSION['user_id'] ; // مثال

$page_title = "المنتجات | Kingmaster";
$page_css = ['/css/product.css'];
include 'includes/head.php';
include 'includes/navbar_top.php';
include 'includes/navbar_actions.php';
include 'includes/navbar_extra_actions.php';
include 'includes/sidebar_right.php';
include 'includes/sidebar_left.php';
?>


<div class="products-container">
    <div class="products-header">
        <h1>
            <i class="fas fa-shopping-bag"></i>
            المنتجات
        </h1>
        <a href="orders.php" class="my-orders-btn">
            <i class="fas fa-shopping-cart"></i>
            طلباتي
        </a>
    </div>

    <!-- قسم الفلاتر والبحث -->
    <div class="filters-section products-css">
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
            <div class="filter-group products-group">
                <label><i class="fas fa-layer-group"></i> الفئة</label>
                <select id="categoryFilter">
                    <option value="">جميع الفئات</option>
                    <option value="electronics">إلكترونيات</option>
                    <option value="fashion">أزياء</option>
                    <option value="courses">دورات</option>
                    <option value="books">كتب</option>
                </select>
            </div>

            <div class="filter-group products-group">
                <label><i class="fas fa-cube"></i> النوع</label>
                <select id="typeFilter">
                    <option value="">الكل</option>
                    <option value="0">منتجات حقيقية</option>
                    <option value="1">منتجات رقمية</option>
                </select>
            </div>

            <div class="filter-group products-group">
                <label><i class="fas fa-dollar-sign"></i> السعر من</label>
                <input type="number" id="minPrice" placeholder="0">
            </div>

            <div class="filter-group products-group">
                <label><i class="fas fa-dollar-sign"></i> السعر إلى</label>
                <input type="number" id="maxPrice" placeholder="999999">
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

<script src="js/products.js"></script>

<?php include 'includes/footer.php'; ?>
