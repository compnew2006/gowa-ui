<?php
header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';

$conn = getDB();

// التحقق من تسجيل الدخول
 
$method = $_SERVER['REQUEST_METHOD'];

if ($method === 'GET') {
    // جلب المنتجات
    $search = isset($_GET['search']) ? $_GET['search'] : '';
    $category = isset($_GET['category']) ? $_GET['category'] : '';
    $is_digital = isset($_GET['is_digital']) ? $_GET['is_digital'] : '';
    $min_price = isset($_GET['min_price']) ? floatval($_GET['min_price']) : 0;
    $max_price = isset($_GET['max_price']) ? floatval($_GET['max_price']) : 999999;
    
    try {
        $sql = "SELECT * FROM products WHERE 1=1";
        $params = [];
        
        if (!empty($search)) {
            $sql .= " AND (name LIKE :search OR description LIKE :search)";
            $params[':search'] = "%$search%";
        }
        
        if (!empty($category)) {
            $sql .= " AND category = :category";
            $params[':category'] = $category;
        }
        
        if ($is_digital !== '') {
            $sql .= " AND is_digital = :is_digital";
            $params[':is_digital'] = intval($is_digital);
        }
        
        $sql .= " AND price BETWEEN :min_price AND :max_price";
        $params[':min_price'] = $min_price;
        $params[':max_price'] = $max_price;
        
        $sql .= " ORDER BY created_at DESC";
        
        $stmt = $conn->prepare($sql);
        $stmt->execute($params);
        $products = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        // تحويل النصوص إلى مصفوفات
        foreach ($products as &$product) {
            $product['colors_array'] = !empty($product['colors']) ? explode(',', $product['colors']) : [];
            $product['sizes_array'] = !empty($product['sizes']) ? explode(',', $product['sizes']) : [];
            $product['is_digital'] = (bool)$product['is_digital'];
            // Use stock_quantity if available, fallback to stock
            if (isset($product['stock_quantity']) && $product['stock_quantity'] !== null) {
                $product['display_stock'] = $product['stock_quantity'];
            } else {
                $product['display_stock'] = isset($product['stock']) ? $product['stock'] : 0;
            }
        }
        
        echo json_encode(['success' => true, 'products' => $products]);
    } catch (PDOException $e) {
        echo json_encode(['success' => false, 'message' => 'خطأ في جلب المنتجات: ' . $e->getMessage()]);
    }
} elseif ($method === 'POST') {
    // جلب تفاصيل منتج واحد
    $data = json_decode(file_get_contents('php://input'), true);
    $product_id = isset($data['product_id']) ? intval($data['product_id']) : 0;
    
    if ($product_id <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المنتج غير صحيح']);
        exit;
    }
    
    try {
        $stmt = $conn->prepare("SELECT * FROM products WHERE id = :id");
        $stmt->execute([':id' => $product_id]);
        $product = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$product) {
            echo json_encode(['success' => false, 'message' => 'المنتج غير موجود']);
            exit;
        }
        
        // تحويل النصوص إلى مصفوفات
        $product['colors_array'] = !empty($product['colors']) ? explode(',', $product['colors']) : [];
        $product['sizes_array'] = !empty($product['sizes']) ? explode(',', $product['sizes']) : [];
        $product['is_digital'] = (bool)$product['is_digital'];
        // Use stock_quantity if available, fallback to stock
        if (isset($product['stock_quantity']) && $product['stock_quantity'] !== null) {
            $product['display_stock'] = $product['stock_quantity'];
        } else {
            $product['display_stock'] = isset($product['stock']) ? $product['stock'] : 0;
        }
        
        echo json_encode(['success' => true, 'product' => $product]);
    } catch (PDOException $e) {
        echo json_encode(['success' => false, 'message' => 'خطأ في جلب المنتج: ' . $e->getMessage()]);
    }
}
?>
