<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

$pdo = getDB();

try {
    // للإدارة - جلب جميع المنتجات بغض النظر عن الحالة
    $isAdmin = isset($_GET['admin']) && $_GET['admin'] === 'true';
    $category = isset($_GET['category']) ? $_GET['category'] : 'all';
    
    // بناء الاستعلام
    if ($isAdmin) {
        $sql = "SELECT * FROM products";
    } else {
        $sql = "SELECT * FROM products WHERE status = 'active'";
        
        if ($category === 'new') {
            $sql .= " AND is_new = 1";
        } elseif ($category === 'featured') {
            $sql .= " AND is_featured = 1";
        } elseif ($category === 'sale') {
            $sql .= " AND discount_percentage > 0";
        }
    }
    
    $sql .= " ORDER BY created_at DESC";
    
    $stmt = $pdo->query($sql);
    $products = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    // جلب الألوان والمقاسات لكل منتج
    foreach ($products as &$product) {
        // الألوان
        $stmt = $pdo->prepare("SELECT color_name, color_hex FROM product_colors WHERE product_id = ?");
        $stmt->execute([$product['id']]);
        $product['colors'] = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        // المقاسات
        $stmt = $pdo->prepare("SELECT size_name FROM product_sizes WHERE product_id = ?");
        $stmt->execute([$product['id']]);
        $sizes = $stmt->fetchAll(PDO::FETCH_ASSOC);
        $product['sizes'] = array_column($sizes, 'size_name');
        
        // حساب السعر النهائي
        $discount = isset($product['discount_percentage']) ? $product['discount_percentage'] : 0;
        $product['discount_percentage'] = $discount;
        $product['final_price'] = $product['price'] - ($product['price'] * $discount / 100);
        
        // إضافة قيم افتراضية للحقول المفقودة
        $product['rating'] = isset($product['rating']) ? $product['rating'] : 0;
        $product['reviews_count'] = isset($product['reviews_count']) ? $product['reviews_count'] : 0;
        $product['stock_quantity'] = isset($product['stock_quantity']) ? $product['stock_quantity'] : (isset($product['stock']) ? $product['stock'] : 0);
        $product['image_url'] = isset($product['image_url']) ? $product['image_url'] : (isset($product['image']) ? $product['image'] : '');
        $product['status'] = isset($product['status']) ? $product['status'] : 'active';
        $product['is_new'] = isset($product['is_new']) ? $product['is_new'] : 0;
        $product['is_featured'] = isset($product['is_featured']) ? $product['is_featured'] : 0;
        $product['description'] = isset($product['description']) ? $product['description'] : '';
    }
    
    echo json_encode([
        'success' => true,
        'products' => $products
    ], JSON_UNESCAPED_UNICODE);
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
