<?php
/**
 * Add a new notification for a user
 * 
 * @param int $user_id User ID
 * @param string $title Notification title
 * @param string $message Notification message
 * @param string $type Notification type (info, success, warning, error)
 * @return bool Success status
 */






function addNotification($user_id, $title, $message, $type = 'info') {
    require_once __DIR__ . '/../config/database.php';
    $conn = getDB();
    
    try {
        $stmt = $conn->prepare("
            INSERT INTO notifications (user_id, title, message, type, is_read, created_at)
            VALUES (:user_id, :title, :message, :type, 0, NOW())
        ");
        
        $stmt->execute([
            ':user_id' => $user_id,
            ':title' => $title,
            ':message' => $message,
            ':type' => $type
        ]);
        
        return true;
    } catch (Exception $e) {
        error_log("Error adding notification: " . $e->getMessage());
        return false;
    }
}

/**
 * Add notification to multiple users
 * 
 * @param array $user_ids Array of user IDs
 * @param string $title Notification title
 * @param string $message Notification message
 * @param string $type Notification type
 * @return bool Success status
 */
function addNotificationToMultipleUsers($user_ids, $title, $message, $type = 'info') {
    foreach ($user_ids as $user_id) {
        addNotification($user_id, $title, $message, $type);
    }
    return true;
}
