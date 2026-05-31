<?php
require_once 'config/database.php';

$pdo = getDB();

echo "<h2>Testing Colors and Sizes Tables</h2>";

// Check if tables exist
echo "<h3>Tables Check:</h3>";
$tables = $pdo->query("SHOW TABLES LIKE 'product_%'")->fetchAll(PDO::FETCH_COLUMN);
echo "<pre>";
print_r($tables);
echo "</pre>";

// Check product_colors structure
echo "<h3>product_colors structure:</h3>";
try {
    $stmt = $pdo->query("DESCRIBE product_colors");
    echo "<pre>";
    print_r($stmt->fetchAll(PDO::FETCH_ASSOC));
    echo "</pre>";
} catch (PDOException $e) {
    echo "Table doesn't exist or error: " . $e->getMessage();
}

// Check product_sizes structure
echo "<h3>product_sizes structure:</h3>";
try {
    $stmt = $pdo->query("DESCRIBE product_sizes");
    echo "<pre>";
    print_r($stmt->fetchAll(PDO::FETCH_ASSOC));
    echo "</pre>";
} catch (PDOException $e) {
    echo "Table doesn't exist or error: " . $e->getMessage();
}

// Check existing data
echo "<h3>Existing Colors:</h3>";
try {
    $stmt = $pdo->query("SELECT * FROM product_colors ORDER BY id DESC LIMIT 10");
    echo "<pre>";
    print_r($stmt->fetchAll(PDO::FETCH_ASSOC));
    echo "</pre>";
} catch (PDOException $e) {
    echo "Error: " . $e->getMessage();
}

echo "<h3>Existing Sizes:</h3>";
try {
    $stmt = $pdo->query("SELECT * FROM product_sizes ORDER BY id DESC LIMIT 10");
    echo "<pre>";
    print_r($stmt->fetchAll(PDO::FETCH_ASSOC));
    echo "</pre>";
} catch (PDOException $e) {
    echo "Error: " . $e->getMessage();
}

// Test insert
echo "<h3>Test Insert:</h3>";
try {
    // Get last product ID
    $stmt = $pdo->query("SELECT id FROM products ORDER BY id DESC LIMIT 1");
    $lastProduct = $stmt->fetch(PDO::FETCH_ASSOC);
    
    if ($lastProduct) {
        $productId = $lastProduct['id'];
        echo "Testing with product ID: $productId<br>";
        
        // Try to insert a test color
        $stmt = $pdo->prepare("INSERT INTO product_colors (product_id, color_name) VALUES (?, ?)");
        $result = $stmt->execute([$productId, 'Test Color']);
        echo "Color insert result: " . ($result ? 'Success' : 'Failed') . "<br>";
        
        // Try to insert a test size
        $stmt = $pdo->prepare("INSERT INTO product_sizes (product_id, size_name) VALUES (?, ?)");
        $result = $stmt->execute([$productId, 'Test Size']);
        echo "Size insert result: " . ($result ? 'Success' : 'Failed') . "<br>";
    }
} catch (PDOException $e) {
    echo "Error: " . $e->getMessage();
}
?>
