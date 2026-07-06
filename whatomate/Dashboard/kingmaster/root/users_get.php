<?php
// db.php - الاتصال بقاعدة البيانات
$host = "localhost";    // اسم السيرفر
$user = "old";         // اسم المستخدم
$pass = "old";             // الباسورد
$db   = "old"; // اسم قاعدة البيانات

$conn = new mysqli($host, $user, $pass, $db);
if ($conn->connect_error) {
    die("Connection failed: " . $conn->connect_error);
}

// إذا تم إرسال النموذج
$resultData = null;
if (isset($_POST['search_email'])) {
    $email = $_POST['search_email'];

    // البحث في جدول users
    $stmt = $conn->prepare("SELECT id FROM users WHERE User_Email = ?");
    $stmt->bind_param("s", $email);
    $stmt->execute();
    $stmt->bind_result($userId);
    if ($stmt->fetch()) {
        $stmt->close();

        // البحث في جدول userskey1
        $stmt2 = $conn->prepare("SELECT Points, Date_End FROM userskey1 WHERE us_id = ?");
        $stmt2->bind_param("i", $userId);
        $stmt2->execute();
        $stmt2->bind_result($col5, $col9);
        if ($stmt2->fetch()) {
            $resultData = [
                'Points' => $col5,
                'Date_End' => $col9
            ];
        } else {
            $resultData = "لا توجد بيانات في userskey1 لهذا الـ ID.";
        }
        $stmt2->close();
    } else {
        $resultData = "الايميل غير موجود في جدول users.";
        $stmt->close();
    }
}

$conn->close();
?>

<!DOCTYPE html>
<html lang="ar">
<head>
    <meta charset="UTF-8">
    <title>بحث في قاعدة البيانات</title>
</head>
<body>
    <h2>بحث عن بيانات المستخدم</h2>
    <form method="POST">
        <input type="email" name="search_email" placeholder="ادخل الايميل" required>
        <button type="submit">بحث</button>
    </form>

    <?php if ($resultData): ?>
        <h3>النتيجة:</h3>
        <?php if (is_array($resultData)): ?>
            <p>Points: <?= htmlspecialchars($resultData['Points']) ?></p>
            <p>Date_End: <?= htmlspecialchars($resultData['Date_End']) ?></p>
        <?php else: ?>
            <p><?= htmlspecialchars($resultData) ?></p>
        <?php endif; ?>
    <?php endif; ?>
</body>
</html>
