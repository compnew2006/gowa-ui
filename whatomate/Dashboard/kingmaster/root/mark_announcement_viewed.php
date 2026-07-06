<?php
session_start();
if (!isset($_SESSION['user_id'])) {
    http_response_code(401);
    echo json_encode(['success' => false, 'message' => 'Unauthorized']);
    exit;
}

require_once 'includes/functions.php';

$user_id = $_SESSION['user_id'];

// Get JSON input
$input = file_get_contents('php://input');
$data = json_decode($input, true);

if (!isset($data['announcement_id'])) {
    http_response_code(400);
    echo json_encode(['success' => false, 'message' => 'Missing announcement_id']);
    exit;
}

$announcement_id = intval($data['announcement_id']);

// Mark as viewed
$result = markAnnouncementAsViewed($user_id, $announcement_id);

if ($result) {
    echo json_encode(['success' => true]);
} else {
    http_response_code(500);
    echo json_encode(['success' => false, 'message' => 'Failed to mark as viewed']);
}
?>
