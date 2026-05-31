<?php
require_once 'config/database.php';

try {
    $conn = getDB();
    
    // التحقق من وجود جدول transactions
    $stmt = $conn->query("SHOW TABLES LIKE 'transactions'");
    $tableExists = $stmt->rowCount() > 0;
    
    echo "<h3>التحقق من جدول transactions:</h3>";
    
    if ($tableExists) {
        echo "✅ الجدول موجود<br><br>";
        
        // عرض بنية الجدول
        $columns = $conn->query("DESCRIBE transactions");
        echo "<h4>بنية الجدول:</h4>";
        echo "<table border='1' cellpadding='5' style='border-collapse: collapse; font-family: Cairo, sans-serif;'>";
        echo "<tr><th>Field</th><th>Type</th><th>Null</th><th>Key</th><th>Default</th></tr>";
        
        while ($row = $columns->fetch(PDO::FETCH_ASSOC)) {
            echo "<tr>";
            echo "<td>{$row['Field']}</td>";
            echo "<td>{$row['Type']}</td>";
            echo "<td>{$row['Null']}</td>";
            echo "<td>{$row['Key']}</td>";
            echo "<td>{$row['Default']}</td>";
            echo "</tr>";
        }
        echo "</table>";
        
    } else {
        echo "❌ الجدول غير موجود<br>";
        echo "<a href='install_users_wallet.php'>إنشاء الجدول</a>";
    }
    
} catch (PDOException $e) {
    echo "❌ خطأ: " . $e->getMessage();
}
?>
