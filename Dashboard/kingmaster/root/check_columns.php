<?php
require_once 'config/database.php';

$pdo = getDB();
$stmt = $pdo->query("SHOW COLUMNS FROM products");
$columns = $stmt->fetchAll(PDO::FETCH_ASSOC);

echo "Columns in products table:\n\n";
foreach ($columns as $column) {
    echo $column['Field'] . " - " . $column['Type'] . "\n";
}

echo "\n\n";
echo "Sample product data:\n\n";
$stmt = $pdo->query("SELECT * FROM products WHERE id = 8 LIMIT 1");
$product = $stmt->fetch(PDO::FETCH_ASSOC);

if ($product) {
    foreach ($product as $key => $value) {
        echo "$key: $value\n";
    }
} else {
    echo "Product not found\n";
}
?>
