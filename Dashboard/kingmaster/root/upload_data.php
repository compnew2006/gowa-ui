<?php
set_time_limit(0);
ini_set('memory_limit', '1024M');

$host = "localhost";
$db   = "kingmaster";
$user = "kingmaster";
$pass = "kingmaster";

$conn = new mysqli($host, $user, $pass, $db);
if ($conn->connect_error) die("Connection failed: " . $conn->connect_error);

$conn->set_charset("utf8mb4"); // مهم

if (isset($_FILES['csv_file']['tmp_name'])) {
    $file = $_FILES['csv_file']['tmp_name'];

    if (($handle = fopen($file, "r")) !== FALSE) {
        $row = 0;
        $stmt = $conn->prepare("INSERT INTO `data_fb` (`fb_id`, `name`, `mobile_phone`, `gender`, `birthday`, `location`, `relationship`, `email`, `work`, `education`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)");
        if(!$stmt) die("Error in preparing statement: " . $conn->error);

        while (($data = fgetcsv($handle, 1000, ",")) !== FALSE) {
            if ($row == 0) { $row++; continue; }

            if(count($data) >= 10){
                // تحويل كل البيانات ل UTF-8
                for ($i = 0; $i < count($data); $i++) {
                    $data[$i] = mb_convert_encoding($data[$i], 'UTF-8', 'auto');
                }

                $stmt->bind_param(
                    "ssssssssss", 
                    $data[0], $data[1], $data[2], $data[3], $data[4], $data[5], $data[6], $data[7], $data[8], $data[9]
                );
                $stmt->execute();
            }
            $row++;
        }

        fclose($handle);
        $stmt->close();
        $conn->close();
        echo "تم رفع الملف وإضافة البيانات بنجاح!";
    } else {
        echo "خطأ في فتح الملف.";
    }
} else {
    echo '<form action="" method="post" enctype="multipart/form-data">
        اختر ملف CSV: <input type="file" name="csv_file" required>
        <input type="submit" value="رفع الملف">
    </form>';
}
?>
