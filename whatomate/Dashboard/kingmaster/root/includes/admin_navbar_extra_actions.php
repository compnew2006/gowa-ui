<!-- Extra Actions (Language & Theme) -->
<div class="nav-extra-actions">
  <!-- Language Selector -->
  <div class="option-icon" onclick="toggleLanguageDropdown()">
    <i class="fas fa-globe"></i>
    <div class="language-dropdown" id="language-dropdown">
      <div class="language-item" onclick="changeLanguage('ar')">
        <img src="https://flagcdn.com/w40/eg.png" alt="العربية">
        <span>العربية</span>
      </div>
      <div class="language-item" onclick="changeLanguage('en')">
        <img src="https://flagcdn.com/w40/us.png" alt="English">
        <span>English</span>
      </div>
      <div class="language-item" onclick="changeLanguage('fr')">
        <img src="https://flagcdn.com/w40/fr.png" alt="Français">
        <span>Français</span>
      </div>
    </div>
  </div>
  
  <!-- Theme Toggle -->
  <div class="option-icon" onclick="toggleTheme()" id="theme-toggle">
    <i class="fas fa-moon" id="theme-icon"></i>
  </div>
</div>
