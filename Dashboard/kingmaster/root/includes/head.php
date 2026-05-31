<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title><?php echo isset($page_title) ? $page_title : 'Kingmaster Dashboard'; ?></title>
  <link rel="icon" type="image/png" href="https://cdn-icons-png.flaticon.com/512/3135/3135715.png">
  <link rel="apple-touch-icon" href="https://cdn-icons-png.flaticon.com/512/3135/3135715.png">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Cairo:wght@400;500;600;700&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
  <link rel="stylesheet" href="/css/navbar_styles.css">
  <link rel="stylesheet" href="/css/styles.css">
  <link rel="stylesheet" href="/css/rtl-ltr.css">
  <!-- SweetAlert2 CSS -->
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/sweetalert2@11/dist/sweetalert2.min.css">
  
  <!-- Chart.js (loaded only on pages that need it) -->
  <?php if (!isset($skip_chartjs)): ?>
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
  <?php endif; ?>
  
  <!-- Translations -->
  <script src="js/translations.js"></script>
  <?php if (function_exists('csrfToken')): ?>
  <meta name="csrf-token" content="<?= htmlspecialchars(csrfToken(), ENT_QUOTES, 'UTF-8') ?>">
  <script>window.KM_CSRF_TOKEN = document.querySelector('meta[name="csrf-token"]')?.content || '';</script>
  <?php endif; ?>
  <script src="js/script.js"></script>
  <?php if (!empty($page_css)): ?>
    <?php foreach ($page_css as $css): ?>
        <link rel="stylesheet" href="<?= $css ?>">
    <?php endforeach; ?>
<?php endif; ?>
  <link rel="stylesheet" href="/css/enterprise-polish.css">
</head>
<body>
  <div class="bg-gradient"></div>
