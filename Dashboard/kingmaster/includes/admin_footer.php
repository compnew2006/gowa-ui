  <!-- SweetAlert2 JS -->
  <script src="https://cdn.jsdelivr.net/npm/sweetalert2@11"></script>
  
  <script>
    // Toggle Language Dropdown
    function toggleLanguageDropdown() {
      const dropdown = document.getElementById('language-dropdown');
      dropdown.classList.toggle('active');
    }

    // Change Language
    function changeLanguage(lang) {
      console.log('Language changed to: ' + lang);
      
      const currentLang = localStorage.getItem('language');
      
      // إذا كانت اللغة هي نفسها، لا تفعل شيئًا
      if (currentLang === lang) {
        console.log('Language is already set to: ' + lang);
        return;
      }
      
      localStorage.setItem('language', lang);
      localStorage.setItem('languageJustChanged', 'true');
      
      // إغلاق القائمة
      const dropdown = document.getElementById('language-dropdown');
      if (dropdown) {
        dropdown.classList.remove('active');
      }
      
      // إظهار رسالة وإعادة تحميل الصفحة
      if (typeof Swal !== 'undefined') {
        const langNames = {
          'ar': 'العربية',
          'en': 'English',
          'fr': 'Français'
        };
        
        Swal.fire({
          icon: 'success',
          title: 'تم تغيير اللغة!',
          text: langNames[lang],
          timer: 1000,
          showConfirmButton: false,
          position: 'center',
          didClose: () => {
            window.location.reload();
          }
        });
      } else {
        window.location.reload();
      }
    }

    // Toggle Theme (Dark/Light)
    function toggleTheme() {
      const body = document.body;
      const themeIcon = document.getElementById('theme-icon');
      
      body.classList.toggle('light-theme');
      
      if (body.classList.contains('light-theme')) {
        themeIcon.classList.remove('fa-moon');
        themeIcon.classList.add('fa-sun');
        localStorage.setItem('theme', 'light');
      } else {
        themeIcon.classList.remove('fa-sun');
        themeIcon.classList.add('fa-moon');
        localStorage.setItem('theme', 'dark');
      }
    }

    // Load saved theme and language on page load
    document.addEventListener('DOMContentLoaded', function() {
      // Load theme
      const savedTheme = localStorage.getItem('theme');
      if (savedTheme === 'light') {
        document.body.classList.add('light-theme');
        const themeIcon = document.getElementById('theme-icon');
        if (themeIcon) {
          themeIcon.classList.replace('fa-moon', 'fa-sun');
        }
      }
      
      // Check if language was just changed
      const languageJustChanged = localStorage.getItem('languageJustChanged');
      if (languageJustChanged === 'true') {
        // Remove the flag to prevent infinite loop
        localStorage.removeItem('languageJustChanged');
        
        // Show success message
        const savedLang = localStorage.getItem('language');
        const langNames = {
          'ar': 'العربية',
          'en': 'English',
          'fr': 'Français'
        };
        
        if (typeof Swal !== 'undefined' && savedLang) {
          Swal.fire({
            icon: 'success',
            title: 'تم تغيير اللغة!',
            text: langNames[savedLang] || savedLang,
            timer: 1500,
            showConfirmButton: false,
            position: 'top-end',
            toast: true
          });
        }
      }
      
      // Load and apply saved language automatically
      const savedLang = localStorage.getItem('language');
      if (savedLang && savedLang !== 'ar') {
        // Apply language without reload (since we just loaded the page)
        applyLanguageWithoutReload(savedLang);
      }
    });
    
    // Apply language without reload (for initial page load)
    function applyLanguageWithoutReload(lang) {
      // Try to use the changeLanguage from script.js if available
      if (typeof window.changeLanguage === 'function') {
        // Temporarily override to prevent reload loop
        const originalChangeLanguage = window.changeLanguage;
        window.changeLanguage = function(l) {
          currentLanguage = l;
          document.documentElement.setAttribute('lang', l);
          
          // Update all elements with data-i18n
          const elements = document.querySelectorAll('[data-i18n]');
          elements.forEach(element => {
            const key = element.getAttribute('data-i18n');
            if (translations && translations[l] && translations[l][key]) {
              element.textContent = translations[l][key];
            }
          });
          
          // Update placeholders
          const placeholders = document.querySelectorAll('[data-i18n-placeholder]');
          placeholders.forEach(element => {
            const key = element.getAttribute('data-i18n-placeholder');
            if (translations && translations[l] && translations[l][key]) {
              element.setAttribute('placeholder', translations[l][key]);
            }
          });
          
          // Update current language text
          const currentLanguageText = document.getElementById('current-language');
          if (currentLanguageText && translations && translations[l]) {
            currentLanguageText.textContent = translations[l].current_language_display || l.toUpperCase();
          }
          
          // Re-init charts if function exists
          if (typeof initCharts === 'function') {
            initCharts();
          }
        };
        
        window.changeLanguage(lang);
        
        // Restore original function
        window.changeLanguage = originalChangeLanguage;
      }
    }

    // Toggle Dropdown
    function toggleDropdown(type) {
      const dropdownId = type + '-dropdown';
      const dropdown = document.getElementById(dropdownId);
      const allDropdowns = document.querySelectorAll('.dropdown');
      
      // Close all other dropdowns
      allDropdowns.forEach(d => {
        if (d.id !== dropdownId) {
          d.classList.remove('active');
        }
      });
      
      // Toggle current dropdown
      dropdown.classList.toggle('active');
    }
    
    // Close dropdowns when clicking outside
    document.addEventListener('click', function(event) {
      if (!event.target.closest('.action-icon')) {
        document.querySelectorAll('.dropdown').forEach(d => {
          d.classList.remove('active');
        });
      }
    });

    // Hide/Show navbar and actions on scroll
    let lastScrollTop = 0;
    const navbar = document.querySelector('.top-navbar');
    const actions = document.querySelector('.nav-actions-container');
    const extraActions = document.querySelector('.nav-extra-actions');
    
    window.addEventListener('scroll', function() {
      const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
      
      if (scrollTop > 100) {
        // Scrolling down - hide
        navbar.classList.add('hidden');
        actions.classList.add('hidden');
        if(extraActions) extraActions.classList.add('hidden');
      } else {
        // At top - show
        navbar.classList.remove('hidden');
        actions.classList.remove('hidden');
        if(extraActions) extraActions.classList.remove('hidden');
      }
      
      lastScrollTop = scrollTop;
    });

    // Platforms Doughnut Chart (Modern Design with Theme Support)
    const platformsCtx = document.getElementById('platformsChart');
    if (platformsCtx) {
      // Detect theme
      const isLightTheme = document.body.classList.contains('light-theme');
      const textColor = isLightTheme ? '#2d3436' : '#e5e7eb';
      const borderColor = isLightTheme ? '#ffffff' : '#1e293b';
      
      new Chart(platformsCtx, {
        type: 'doughnut',
        data: {
          labels: ['فيسبوك', 'واتساب', 'تليجرام', 'إنستجرام', 'تيك توك', 'جوجل', 'أعمال'],
          datasets: [{
            data: [25, 20, 15, 18, 10, 8, 4],
            backgroundColor: [
              '#1877f2',  // Facebook Blue
              '#25D366',  // WhatsApp Green
              '#0088cc',  // Telegram Blue
              '#E1306C',  // Instagram Gradient (Pink)
              '#010101',  // TikTok Black
              '#4285F4',  // Google Blue
              '#667eea'   // Business Purple
            ],
            borderColor: borderColor,
            borderWidth: 4,
            hoverOffset: 18,
            hoverBorderColor: '#ffffff',
            hoverBorderWidth: 4,
            spacing: 3,
            borderRadius: 6
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: true,
          aspectRatio: 1.5,
          cutout: '60%',
          plugins: {
            legend: {
              position: 'right',
              rtl: true,
              labels: {
                color: textColor,
                padding: 15,
                font: {
                  size: 13,
                  weight: '600',
                  family: 'Cairo'
                },
                usePointStyle: true,
                pointStyle: 'rectRounded',
                boxWidth: 14,
                boxHeight: 14,
                generateLabels: function(chart) {
                  const data = chart.data;
                  return data.labels.map((label, i) => ({
                    text: label + ' (' + data.datasets[0].data[i] + '%)',
                    fillStyle: data.datasets[0].backgroundColor[i],
                    hidden: false,
                    index: i,
                    fontColor: textColor
                  }));
                }
              }
            },
            tooltip: {
              backgroundColor: isLightTheme ? 'rgba(255, 255, 255, 0.95)' : 'rgba(30, 41, 59, 0.95)',
              titleColor: textColor,
              bodyColor: textColor,
              borderColor: 'rgba(102, 126, 234, 0.5)',
              borderWidth: 2,
              padding: 14,
              boxPadding: 8,
              usePointStyle: true,
              titleFont: {
                size: 14,
                weight: 'bold'
              },
              bodyFont: {
                size: 13
              },
              callbacks: {
                label: function(context) {
                  return ' ' + context.label + ': ' + context.parsed + '%';
                }
              }
            }
          },
          animation: {
            animateRotate: true,
            animateScale: true,
            duration: 1000,
            easing: 'easeInOutQuart'
          }
        }
      });
    }

    // Sales Line Chart
    const ctx = document.getElementById('salesChart');
    if (ctx) {
      new Chart(ctx, {
        type: 'line',
        data: {
          labels: ['السبت', 'الأحد', 'الأثنين', 'الثلاثاء', 'الأربعاء', 'الخميس', 'الجمعة'],
          datasets: [{
            label: 'الرسائل',
            data: [12, 19, 15, 25, 22, 30, 28],
            borderColor: '#667eea',
            backgroundColor: 'rgba(102, 126, 234, 0.1)',
            tension: 0.4,
            fill: true,
            pointBackgroundColor: '#667eea',
            pointBorderColor: '#fff',
            pointBorderWidth: 2,
            pointRadius: 4,
            pointHoverRadius: 6
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: true,
          aspectRatio: 1.5,
          plugins: {
            legend: {
              display: false
            },
            tooltip: {
              backgroundColor: 'rgba(30, 41, 59, 0.95)',
              titleColor: '#e5e7eb',
              bodyColor: '#e5e7eb',
              borderColor: 'rgba(102, 126, 234, 0.5)',
              borderWidth: 1,
              padding: 10,
              displayColors: false
            }
          },
          scales: {
            y: {
              beginAtZero: true,
              ticks: {
                color: '#9ca3af',
                font: {
                  size: 11
                }
              },
              grid: {
                color: 'rgba(255, 255, 255, 0.1)'
              }
            },
            x: {
              ticks: {
                color: '#9ca3af',
                font: {
                  size: 10
                }
              },
              grid: {
                display: false
              }
            }
          }
        }
      });
    }

    // Yearly Column Chart (12 Months)
    const monthlyCtx = document.getElementById('monthlyChart');
    if (monthlyCtx) {
      new Chart(monthlyCtx, {
        type: 'bar',
        data: {
          labels: ['يناير', 'فبراير', 'مارس', 'أبريل', 'مايو', 'يونيو', 'يوليو', 'أغسطس', 'سبتمبر', 'أكتوبر', 'نوفمبر', 'ديسمبر'],
          datasets: [{
            label: 'الاستخراجات',
            data: [1200, 1900, 1500, 2200, 1800, 2400, 2100, 2600, 2300, 2800, 2500, 3000],
            backgroundColor: [
              'rgba(59, 130, 246, 0.8)',
              'rgba(16, 185, 129, 0.8)',
              'rgba(245, 158, 11, 0.8)',
              'rgba(139, 92, 246, 0.8)',
              'rgba(236, 72, 153, 0.8)',
              'rgba(14, 165, 233, 0.8)',
              'rgba(234, 179, 8, 0.8)',
              'rgba(239, 68, 68, 0.8)',
              'rgba(168, 85, 247, 0.8)',
              'rgba(34, 197, 94, 0.8)',
              'rgba(249, 115, 22, 0.8)',
              'rgba(99, 102, 241, 0.8)'
            ],
            borderColor: [
              '#3b82f6',
              '#10b981',
              '#f59e0b',
              '#8b5cf6',
              '#ec4899',
              '#0ea5e9',
              '#eab308',
              '#ef4444',
              '#a855f7',
              '#22c55e',
              '#f97316',
              '#6366f1'
            ],
            borderWidth: 2,
            borderRadius: 6
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: true,
          aspectRatio: 2.2,
          plugins: {
            legend: {
              display: false
            },
            tooltip: {
              backgroundColor: 'rgba(30, 41, 59, 0.95)',
              titleColor: '#e5e7eb',
              bodyColor: '#e5e7eb',
              borderColor: 'rgba(102, 126, 234, 0.5)',
              borderWidth: 2,
              padding: 12,
              titleFont: {
                size: 13,
                weight: 'bold'
              },
              bodyFont: {
                size: 12
              }
            }
          },
          scales: {
            y: {
              beginAtZero: true,
              ticks: {
                color: '#9ca3af',
                font: {
                  size: 11
                }
              },
              grid: {
                color: 'rgba(255, 255, 255, 0.1)'
              }
            },
            x: {
              ticks: {
                color: '#9ca3af',
                font: {
                  size: 10
                },
                maxRotation: 45,
                minRotation: 45
              },
              grid: {
                display: false
              }
            }
          }
        }
      });
    }

  </script>
</body>
</html>
