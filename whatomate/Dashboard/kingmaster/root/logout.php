<?php
// بدء الجلسة
session_start();

// حذف جميع بيانات الجلسة
$_SESSION = [];

// حذف الكوكيز الخاصة بالجلسة (اختياري)
if (ini_get("session.use_cookies")) {
    $params = session_get_cookie_params();
    setcookie(session_name(), '', time() - 42000,
        $params["path"], $params["domain"],
        $params["secure"], $params["httponly"]
    );
}

// تدمير الجلسة تمامًا
session_destroy();

// إعادة التوجيه إلى صفحة تسجيل الدخول أو الرئيسية
header("Location: landing.php");
exit;
?>
