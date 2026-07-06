<?php
header('Content-Type: application/json');
session_start();

require_once '../config/database.php';

try {
    $pdo = getDB();
    
    // استقبال البيانات
    $id = $_POST['id'] ?? 0;
    $name = $_POST['name'] ?? '';
    $description = $_POST['description'] ?? '';
    $price = $_POST['price'] ?? 0;
    $discount_percentage = $_POST['discount_percentage'] ?? 0;
    $stock_quantity = $_POST['stock_quantity'] ?? 0;
    $category = $_POST['category'] ?? 'other';
    $status = $_POST['status'] ?? 'active';
    $is_digital = isset($_POST['is_digital']) ? (int)$_POST['is_digital'] : 0;
    $is_new = isset($_POST['is_new']) ? (int)$_POST['is_new'] : 0;
    $is_featured = isset($_POST['is_featured']) ? (int)$_POST['is_featured'] : 0;
    $commission = $_POST['commission'] ?? 0;
    $colors = $_POST['colors'] ?? '';
    $sizes = $_POST['sizes'] ?? '';
    
    // رفع الصورษ
    $image_name = $_POST['existing_image'] ?? '';
    if (isset($_FILES['product_image']) && $_FILES['product_image']['error'] === 0) {
        $upload_dir = '../uploads/products/';
        $file_extension = pathinfo($_FILES['product_image']['name'], PATHINFO_EXTENSION);
        $file_name = uniqid('product_') . '.' . $file_extension;
        $upload_path = $upload_dir . $file_name;
        
        if (move_uploaded_file($_FILES['product_image']['tmp_name'], $upload_path)) {
            // حذف الصورة القديمة
            if (!empty($image_name) && file_exists('../uploads/products/' . $image_name)) {
                unlink('../uploads/products/' . $image_name);
            }
            $image_name = $file_name;
        }
    }
    
    // التحقق من البيانات المطلوبة
    if (empty($id) || empty($name) || empty($price)) {
        echo json_encode([
            'success' => false,
            'message' => 'المعرف والاسم والسعر مطلوبان'
        ]);
        exit;
    }
    
    // تحديث المنتج
    $stmt = $pdo->prepare("
        UPDATE products SET
            name = ?,
            description = ?,
            price = ?,
            discount_percentage = ?,
            stock_quantity = ?,
            category = ?,
            image = ?,
            commission = ?,
            status = ?,
            is_digital = ?,
            is_new = ?,
            is_featured = ?,
            colors = ?,
            sizes = ?
        WHERE id = ?
    ");
    
    $result = $stmt->execute([
        $name,
        $description,
        $price,
        $discount_percentage,
        $stock_quantity,
        $category,
        $image_name,
        $commission,
        $status,
        $is_digital,
        $is_new,
        $is_featured,
        $colors,
        $sizes,
        $id
    ]);
    
    if ($result) {
        echo json_encode([
            'success' => true,
            'message' => 'تم تحديث المنتج بنجاح'
        ]);
    } else {
        echo json_encode([
            'success' => false,
            'message' => 'فشل تحديث المنتج'
        ]);
    }
    
} catch (PDOException $e) {
    echo json_encode([
        'success' => false,
        'message' => 'خطأ في قاعدة البيانات: ' . $e->getMessage()
    ]);
}
?>
