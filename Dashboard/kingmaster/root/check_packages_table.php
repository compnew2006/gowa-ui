<?php
require_once 'config/database.php';

try {
    $pdo = getDB();
    
    // Get table structure
    $stmt = $pdo->query("DESCRIBE packages");
    $columns = $stmt->fetchAll(PDO::FETCH_ASSOC);
    
    echo "<h2>بنية جدول packages:</h2>";
    echo "<pre>";
    print_r($columns);
    echo "</pre>";
    
} catch (PDOException $e) {
    echo "خطأ: " . $e->getMessage();
}
?>
