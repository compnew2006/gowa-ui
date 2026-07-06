<?php
ini_set('display_errors', 1);
error_reporting(E_ALL);

echo "RUNNING FILE: " . __FILE__ . "<br>";

$DB_HOST = "localhost";
$DB_NAME = "kingmaster";
$DB_USER = "kingmaster";
$DB_PASS = "kingmaster";

$conn = new mysqli($DB_HOST, $DB_USER, $DB_PASS, $DB_NAME);
if ($conn->connect_error) {
  die("DB Failed: " . $conn->connect_error);
}

$res = $conn->query("SELECT COUNT(*) as c FROM data_fb");
$row = $res->fetch_assoc();

echo "DB OK ✅ , Rows = " . $row['c'];
