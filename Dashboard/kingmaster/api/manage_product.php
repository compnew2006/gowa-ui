<?php
header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: POST, PUT, DELETE');
header('Access-Control-Allow-Headers: Content-Type');

require_once '../config/database.php';

$pdo = getDB();
$input = json_decode(file_get_contents('php://input'), true);
$action = isset($input['action']) ? $input['action'] : '';

try {
    switch ($action) {
        case 'add':
            // إضافة منتج جديد
            $stmt = $pdo->prepare("
                INSERT INTO products (name, description, price, discount_percentage, stock_quantity, 
                                    image_url, is_digital, is_new, is_featured, category, status)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ");
            
            $stmt->execute([
                $input['name'],
                $input['description'],
                $input['price'],
                $input['discount_percentage'] ?? 0,
                $input['stock_quantity'] ?? 0,
                $input['image_url'] ?? '',
                $input['is_digital'] ?? 0,
                $input['is_new'] ?? 0,
                $input['is_featured'] ?? 0,
                $input['category'] ?? '',
                'active'
            ]);
            
            $product_id = $pdo->lastInsertId();
            
            // إضافة الألوان
            if (!empty($input['colors']) && !$input['is_digital']) {
                foreach ($input['colors'] as $color) {
                    $stmt = $pdo->prepare("INSERT INTO product_colors (product_id, color_name, color_hex) VALUES (?, ?, ?)");
                    $stmt->execute([$product_id, $color['name'], $color['hex']]);
                }
            }
            
            // إضافة المقاسات
            if (!empty($input['sizes']) && !$input['is_digital']) {
                foreach ($input['sizes'] as $size) {
                    $stmt = $pdo->prepare("INSERT INTO product_sizes (product_id, size_name) VALUES (?, ?)");
                    $stmt->execute([$product_id, $size]);
                }
            }
            
            echo json_encode(['success' => true, 'message' => 'تم إضافة المنتج بنجاح', 'product_id' => $product_id], JSON_UNESCAPED_UNICODE);
            break;
            
        case 'update':
            // تحديث منتج
            $stmt = $pdo->prepare("
                UPDATE products 
                SET name = ?, description = ?, price = ?, discount_percentage = ?, 
                    stock_quantity = ?, image_url = ?, is_digital = ?, is_new = ?, 
                    is_featured = ?, category = ?, status = ?
                WHERE id = ?
            ");
            
            $stmt->execute([
                $input['name'],
                $input['description'],
                $input['price'],
                $input['discount_percentage'] ?? 0,
                $input['stock_quantity'] ?? 0,
                $input['image_url'] ?? '',
                $input['is_digital'] ?? 0,
                $input['is_new'] ?? 0,
                $input['is_featured'] ?? 0,
                $input['category'] ?? '',
                $input['status'] ?? 'active',
                $input['id']
            ]);
            
            // حذف الألوان والمقاسات القديمة
            $pdo->prepare("DELETE FROM product_colors WHERE product_id = ?")->execute([$input['id']]);
            $pdo->prepare("DELETE FROM product_sizes WHERE product_id = ?")->execute([$input['id']]);
            
            // إضافة الألوان الجديدة
            if (!empty($input['colors']) && !$input['is_digital']) {
                foreach ($input['colors'] as $color) {
                    $stmt = $pdo->prepare("INSERT INTO product_colors (product_id, color_name, color_hex) VALUES (?, ?, ?)");
                    $stmt->execute([$input['id'], $color['name'], $color['hex']]);
                }
            }
            
            // إضافة المقاسات الجديدة
            if (!empty($input['sizes']) && !$input['is_digital']) {
                foreach ($input['sizes'] as $size) {
                    $stmt = $pdo->prepare("INSERT INTO product_sizes (product_id, size_name) VALUES (?, ?)");
                    $stmt->execute([$input['id'], $size]);
                }
            }
            
            echo json_encode(['success' => true, 'message' => 'تم تحديث المنتج بنجاح'], JSON_UNESCAPED_UNICODE);
            break;
            
        case 'delete':
            // حذف منتج
            $stmt = $pdo->prepare("DELETE FROM products WHERE id = ?");
            $stmt->execute([$input['id']]);
            
            echo json_encode(['success' => true, 'message' => 'تم حذف المنتج بنجاح'], JSON_UNESCAPED_UNICODE);
            break;
            
        case 'get':
            // جلب منتج واحد
            $stmt = $pdo->prepare("SELECT * FROM products WHERE id = ?");
            $stmt->execute([$input['id']]);
            $product = $stmt->fetch(PDO::FETCH_ASSOC);
            
            if ($product) {
                // جلب الألوان
                $stmt = $pdo->prepare("SELECT color_name, color_hex FROM product_colors WHERE product_id = ?");
                $stmt->execute([$product['id']]);
                $product['colors'] = $stmt->fetchAll(PDO::FETCH_ASSOC);
                
                // جلب المقاسات
                $stmt = $pdo->prepare("SELECT size_name FROM product_sizes WHERE product_id = ?");
                $stmt->execute([$product['id']]);
                $sizes = $stmt->fetchAll(PDO::FETCH_ASSOC);
                $product['sizes'] = array_column($sizes, 'size_name');
                
                echo json_encode(['success' => true, 'product' => $product], JSON_UNESCAPED_UNICODE);
            } else {
                echo json_encode(['success' => false, 'message' => 'المنتج غير موجود'], JSON_UNESCAPED_UNICODE);
            }
            break;
            
        default:
            echo json_encode(['success' => false, 'message' => 'عملية غير صحيحة'], JSON_UNESCAPED_UNICODE);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ], JSON_UNESCAPED_UNICODE);
}
