<?php
header('Content-Type: application/json');
header('Access-Control-Allow-Origin: *');

require_once '../config/database.php';
session_start();
$user_id  = $_SESSION['user_id'];;
$method = $_SERVER['REQUEST_METHOD'];
$conn = getDB();

if ($method === 'GET') {
    // جلب طلبات المستخدم مع البحث والفلاتر
    $search = isset($_GET['search']) ? trim($_GET['search']) : '';
    $status = isset($_GET['status']) ? trim($_GET['status']) : '';
    $date_from = isset($_GET['date_from']) ? trim($_GET['date_from']) : '';
    $date_to = isset($_GET['date_to']) ? trim($_GET['date_to']) : '';
    
    try {
        $sql = "
            SELECT o.*, p.name as product_name, p.is_digital 
            FROM orders o 
            JOIN products p ON o.product_id = p.id 
            WHERE o.user_id = :user_id
        ";
        $params = [':user_id' => $user_id];
        
        // البحث برقم الطلب أو رقم الهاتف
        if (!empty($search)) {
            $sql .= " AND (o.id = :search OR o.phone LIKE :search_like)";
            $params[':search'] = $search;
            $params[':search_like'] = "%$search%";
        }
        
        // فلترة بالحالة
        if (!empty($status)) {
            $sql .= " AND o.status = :status";
            $params[':status'] = $status;
        }
        
        // فلترة بالتاريخ
        if (!empty($date_from)) {
            $sql .= " AND DATE(o.created_at) >= :date_from";
            $params[':date_from'] = $date_from;
        }
        
        if (!empty($date_to)) {
            $sql .= " AND DATE(o.created_at) <= :date_to";
            $params[':date_to'] = $date_to;
        }
        
        $sql .= " ORDER BY o.created_at DESC";
        
        $stmt = $conn->prepare($sql);
        $stmt->execute($params);
        $orders = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        echo json_encode(['success' => true, 'orders' => $orders]);
    } catch (PDOException $e) {
        echo json_encode(['success' => false, 'message' => 'خطأ في جلب الطلبات: ' . $e->getMessage()]);
    }
} elseif ($method === 'POST') {
    // إنشاء طلب جديد
    $data = json_decode(file_get_contents('php://input'), true);
    
    $product_id = isset($data['product_id']) ? intval($data['product_id']) : 0;
    $quantity = isset($data['quantity']) ? intval($data['quantity']) : 1;
    $color = isset($data['color']) ? trim($data['color']) : null;
    $size = isset($data['size']) ? trim($data['size']) : null;
    $address = isset($data['address']) ? trim($data['address']) : null;
    $phone = isset($data['phone']) ? trim($data['phone']) : null;
    $payment_method = isset($data['payment_method']) ? trim($data['payment_method']) : 'balance';
    
    // التحقق من البيانات
    if ($product_id <= 0) {
        echo json_encode(['success' => false, 'message' => 'معرف المنتج غير صحيح']);
        exit;
    }
    
    if ($quantity <= 0) {
        echo json_encode(['success' => false, 'message' => 'الكمية غير صحيحة']);
        exit;
    }
    
    try {
        $conn->beginTransaction();
        
        // جلب معلومات المنتج
        $stmt = $conn->prepare("SELECT * FROM products WHERE id = :id FOR UPDATE");
        $stmt->execute([':id' => $product_id]);
        $product = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$product) {
            $conn->rollBack();
            echo json_encode(['success' => false, 'message' => 'المنتج غير موجود']);
            exit;
        }
        
        // التحقق من المخزون - استخدام stock_quantity إذا كان موجوداً
        $available_stock = isset($product['stock_quantity']) && $product['stock_quantity'] !== null ? $product['stock_quantity'] : $product['stock'];
        
        if ($available_stock < $quantity) {
            $conn->rollBack();
            echo json_encode(['success' => false, 'message' => 'الكمية المطلوبة غير متوفرة']);
            exit;
        }
        
        // حساب الإجمالي
        $price = floatval($product['price']);
        $total = $price * $quantity;
        $commission = floatval($product['commission']) * $quantity;
        
        // جلب محفظة المستخدم
        $stmt = $conn->prepare("SELECT * FROM users_wallet WHERE user_id = :user_id FOR UPDATE");
        $stmt->execute([':user_id' => $user_id]);
        $wallet = $stmt->fetch(PDO::FETCH_ASSOC);
        
        if (!$wallet) {
            $conn->rollBack();
            echo json_encode(['success' => false, 'message' => 'محفظة المستخدم غير موجودة']);
            exit;
        }
        
        // خصم الرصيد فقط إذا كانت طريقة الدفع من الرصيد
        $new_balance = $wallet['balance'];
        
        if ($payment_method === 'balance') {
            // التحقق من الرصيد
            if ($wallet['balance'] < $total) {
                $conn->rollBack();
                echo json_encode(['success' => false, 'message' => 'رصيدك غير كافٍ لإتمام عملية الشراء']);
                exit;
            }
            
            // خصم المبلغ من رصيد المستخدم
            $new_balance = $wallet['balance'] - $total;
            $stmt = $conn->prepare("UPDATE users_wallet SET balance = :balance WHERE user_id = :user_id");
            $stmt->execute([
                ':balance' => $new_balance,
                ':user_id' => $user_id
            ]);
            
            // تسجيل المعاملة في transactions
            $stmt = $conn->prepare("
                INSERT INTO transactions (user_id, amount, transaction_type, amount_type, created_at, to_email) 
                VALUES (:user_id, :amount, 'send', 'money', NOW(), :to_email)
            ");
            $stmt->execute([
                ':user_id' => $user_id,
                ':amount' => $total,
                ':to_email' => ' شراء منتج ' . $product['name'] // هنا بنمرر القيمة بأمان

            ]);
        }
        
        // إضافة العمولة للمسوق إذا كان هناك عمولة
        if ($commission > 0 && !empty($wallet['referrer_id'])) {
            // إضافة العمولة لرصيد المسوق
            $stmt = $conn->prepare("
                UPDATE users_wallet 
                SET balance = balance + :commission 
                WHERE user_id = :referrer_id
            ");
            $stmt->execute([
                ':commission' => $commission,
                ':referrer_id' => $wallet['referrer_id']
            ]);
            
            // تسجيل معاملة استقبال للمسوق
            $stmt = $conn->prepare("
                INSERT INTO transactions (user_id, amount, transaction_type, amount_type, created_at) 
                VALUES (:referrer_id, :amount, 'receive', 'money', NOW())
            ");
            $stmt->execute([
                ':referrer_id' => $wallet['referrer_id'],
                ':amount' => $commission
            ]);
        }
        
        // خصم الكمية من المخزون - تحديث stock_quantity إذا كان موجوداً
        if (isset($product['stock_quantity']) && $product['stock_quantity'] !== null) {
            $new_stock = $product['stock_quantity'] - $quantity;
            $stmt = $conn->prepare("UPDATE products SET stock_quantity = :stock WHERE id = :id");
            $stmt->execute([
                ':stock' => $new_stock,
                ':id' => $product_id
            ]);
        } else {
            $new_stock = $product['stock'] - $quantity;
            $stmt = $conn->prepare("UPDATE products SET stock = :stock WHERE id = :id");
            $stmt->execute([
                ':stock' => $new_stock,
                ':id' => $product_id
            ]);
        }
        
        // إنشاء الطلب
        $stmt = $conn->prepare("
            INSERT INTO orders (
                user_id, product_id, product_name, quantity, 
                color, size, price, total, commission, 
                address, phone, payment_method, status, created_at
            ) VALUES (
                :user_id, :product_id, :product_name, :quantity,
                :color, :size, :price, :total, :commission,
                :address, :phone, :payment_method, 'pending', NOW()
            )
        ");
        
        $stmt->execute([
            ':user_id' => $user_id,
            ':product_id' => $product_id,
            ':product_name' => $product['name'],
            ':quantity' => $quantity,
            ':color' => $color,
            ':size' => $size,
            ':price' => $price,
            ':total' => $total,
            ':commission' => $commission,
            ':address' => $address,
            ':phone' => $phone,
            ':payment_method' => $payment_method
        ]);
        
        $order_id = $conn->lastInsertId();
        
        $conn->commit();
        
        echo json_encode([
            'success' => true, 
            'message' => 'تم إنشاء الطلب بنجاح! رقم الطلب: ' . $order_id,
            'order_id' => $order_id,
            'new_balance' => $new_balance
        ]);
        
    } catch (PDOException $e) {
        $conn->rollBack();
        echo json_encode(['success' => false, 'message' => 'خطأ في إنشاء الطلب: ' . $e->getMessage()]);
    }
}
?>
