<?php
require_once '../includes/functions.php';

$id = $_GET['id'];
echo getContactCount($id);
