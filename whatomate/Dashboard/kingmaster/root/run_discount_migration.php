<?php
/**
 * تنفيذ migration لإضافة حقول الخصم
 */

require_once 'config/database.php';

header('Content-Type: application/json; charset=UTF-8');

try {
    $db = getDB();
    
    // قراءة محتوى ملف SQL
    $sql_content = file_get_contents('add_discount_fields.sql');
    
    if ($sql_content === false) {
        throw new Exception('فشل في قراءة ملف SQL');
    }
    
    // تنظيف النص وتقسيم الاستعلامات
    $sql_content = str_replace(["\r\n", "\r"], "\n", $sql_content);
    $sql_lines = explode("\n", $sql_content);
    
    $current_query = '';
    $queries = [];
    
    foreach ($sql_lines as $line) {
        $line = trim($line);
        if (empty($line) || strpos($line, '--') === 0) {
            continue;
        }
        
        $current_query .= $line . ' ';
        
        if (substr($line, -1) === ';') {
            $queries[] = trim($current_query);
            $current_query = '';
        }
    }
    
    $results = [];
    
    foreach ($queries as $query) {
        if (empty($query) || strpos(trim($query), '--') === 0) {
            continue;
        }
        
        try {
            $stmt = $db->exec($query);
            $results[] = [
                'query' => substr($query, 0, 100) . (strlen($query) > 100 ? '...' : ''),
                'success' => true,
                'affected_rows' => $stmt
            ];
        } catch (PDOException $e) {
            // تجاهل الأخطاء إذا كانت الأعمدة موجودة بالفعل
            if (strpos($e->getMessage(), 'Duplicate column name') !== false) {
                $results[] = [
                    'query' => substr($query, 0, 100) . '...',
                    'success' => true,
                    'message' => 'العمود موجود بالفعل - تم تجاهله'
                ];
            } else {
                $results[] = [
                    'query' => substr($query, 0, 100) . '...',
                    'success' => false,
                    'error' => $e->getMessage()
                ];
            }
        }
    }
    
    // التحقق من وجود الحقول الجديدة
    $stmt = $db->query("DESCRIBE packages");
    $columns = $stmt->fetchAll(PDO::FETCH_COLUMN);
    
    $discount_fields = ['monthly_discount', 'yearly_discount', 'monthly_discount_percentage', 'yearly_discount_percentage', 'has_discount'];
    $fields_exist = [];
    
    foreach ($discount_fields as $field) {
        $fields_exist[$field] = in_array($field, $columns);
    }
    
    echo json_encode([
        'success' => true,
        'message' => 'تم تنفيذ Migration بنجاح',
        'results' => $results,
        'discount_fields_exist' => $fields_exist,
        'timestamp' => date('Y-m-d H:i:s')
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    
} catch (Exception $e) {
    echo json_encode([
        'success' => false,
        'message' => 'فشل في تنفيذ Migration: ' . $e->getMessage(),
        'timestamp' => date('Y-m-d H:i:s')
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
}
?>