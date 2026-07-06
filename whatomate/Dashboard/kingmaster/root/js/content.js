// Content management
let contents = [];

// Load data from database on page load
document.addEventListener('DOMContentLoaded', function() {
  loadContents();
});

// Load contents from database
function loadContents() {
  fetch('api/content_api.php?action=get_all')
    .then(response => response.json())
    .then(data => {
      if(data.success) {
        contents = data.content;
        renderContents();
      } else {
        console.error('Error loading content:', data.message);
      }
    })
    .catch(error => {
      console.error('Error loading content:', error);
    });
}

// Open create modal
function openCreateModal() {
  document.getElementById('content-modal').classList.add('active');
  document.getElementById('modal-title').textContent = 'إنشاء محتوى جديد';
  document.getElementById('content-name').value = '';
  document.getElementById('content-text').value = '';
  document.getElementById('content-id').value = '';
  updateCharCount();
}

// Close content modal
function closeContentModal() {
  document.getElementById('content-modal').classList.remove('active');
}

// Toggle emoji picker
function toggleEmojiPicker() {
  const picker = document.getElementById('emoji-picker');
  picker.style.display = picker.style.display === 'none' ? 'block' : 'none';
}

// Insert emoji at cursor position
function insertEmoji(emoji) {
  const textarea = document.getElementById('content-text');
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const text = textarea.value;
  
  textarea.value = text.substring(0, start) + emoji + text.substring(end);
  textarea.selectionStart = textarea.selectionEnd = start + emoji.length;
  textarea.focus();
  
  updateCharCount();
  
  // Hide emoji picker after selection
  document.getElementById('emoji-picker').style.display = 'none';
}

// Update character and word count
function updateCharCount() {
  const text = document.getElementById('content-text').value;
  const charCount = text.length;
  const wordCount = text.trim() ? text.trim().split(/\s+/).length : 0;
  
  document.getElementById('char-count').textContent = charCount;
  document.getElementById('word-count').textContent = wordCount;
}

// Listen to textarea changes
document.addEventListener('DOMContentLoaded', function() {
  const textarea = document.getElementById('content-text');
  if (textarea) {
    textarea.addEventListener('input', updateCharCount);
  }
});

// Save content
function saveContent() {
  const name = document.getElementById('content-name').value.trim();
  const text = document.getElementById('content-text').value.trim();
  const contentId = document.getElementById('content-id').value;
  
  // Validate name
  if (!name || name.length === 0) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'يرجى إدخال اسم المحتوى',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    return;
  }
  
  // Validate text (at least 1 character)
  if (!text || text.length === 0) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'يرجى إدخال نص المحتوى (على الأقل حرف واحد)',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    return;
  }
  
  const formData = new FormData();
  formData.append('name', name);
  formData.append('text', text);
  
  if (contentId) {
    // Edit existing content
    formData.append('action', 'update');
    formData.append('id', contentId);
  } else {
    // Create new content
    formData.append('action', 'create');
  }
  
  fetch('api/content_api.php', {
    method: 'POST',
    body: formData
  })
  .then(response => response.json())
  .then(data => {
    if(data.success) {
      loadContents();
      closeContentModal();
      
      Swal.fire({
        icon: 'success',
        title: 'تم بنجاح!',
        text: data.message,
        timer: 2000,
        showConfirmButton: false,
        timerProgressBar: true,
        customClass: {
          container: 'swal-over-modal'
        }
      });
    } else {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: data.message,
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea',
        customClass: {
          container: 'swal-over-modal'
        }
      });
    }
  })
  .catch(error => {
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ أثناء حفظ المحتوى',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea',
      customClass: {
        container: 'swal-over-modal'
      }
    });
    console.error('Save error:', error);
  });
}

// Render contents
function renderContents() {
  const grid = document.getElementById('content-grid');
  
  if (contents.length === 0) {
    grid.innerHTML = `
      <div class="empty-state">
        <i class="fas fa-file-alt fa-beat-fade" style="font-size: 4rem; color: #667eea; opacity: 0.6; --fa-animation-duration: 2s;"></i>
        <p style="color: var(--text-gray); margin-top: 1rem;">لا يوجد محتوى حتى الآن</p>
        <button onclick="openCreateModal()" class="create-btn" style="margin-top: 1rem;">
          <i class="fas fa-plus-circle fa-bounce" style="--fa-animation-duration: 1s;"></i>
          إنشاء أول محتوى
        </button>
      </div>
    `;
    return;
  }
  
  grid.innerHTML = contents.map(content => `
    <div class="content-card-item">
      <div class="content-card-title">
        <i class="fas fa-file-alt fa-beat" style="color: #667eea; --fa-animation-duration: 1.5s;"></i>
        ${content.name}
      </div>
      <div class="content-card-text">${escapeHtml(content.text)}</div>
      <div class="content-stats">
        <span>
          <i class="fas fa-font fa-fade" style="color: #10b981; --fa-animation-duration: 3s;"></i>
          <strong>${content.char_count}</strong> حرف
        </span>
        <span>
          <i class="fas fa-text-width fa-fade" style="color: #3b82f6; --fa-animation-duration: 3s;"></i>
          <strong>${content.word_count}</strong> كلمة
        </span>
      </div>
      <div class="content-actions">
        <button class="btn-edit" onclick="editContent(${content.id})">
          <i class="fas fa-edit fa-flip" style="color: #667eea; --fa-animation-duration: 2s;"></i>
          تعديل
        </button>
        <button class="btn-delete" onclick="deleteContent(${content.id})">
          <i class="fas fa-trash fa-shake" style="color: #ef4444; --fa-animation-duration: 3s;"></i>
          حذف
        </button>
      </div>
    </div>
  `).join('');
}

// Edit content
function editContent(contentId) {
  const content = contents.find(c => c.id === contentId);
  if (!content) return;
  
  document.getElementById('content-modal').classList.add('active');
  document.getElementById('modal-title').textContent = 'تعديل المحتوى';
  document.getElementById('content-name').value = content.name;
  document.getElementById('content-text').value = content.text;
  document.getElementById('content-id').value = content.id;
  updateCharCount();
}

// Delete content
function deleteContent(contentId) {
  Swal.fire({
    title: 'هل أنت متأكد؟',
    text: 'لن تتمكن من استرجاع هذا المحتوى!',
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#ef4444',
    cancelButtonColor: '#667eea',
    confirmButtonText: 'نعم، احذف!',
    cancelButtonText: 'إلغاء',
    customClass: {
      container: 'swal-over-modal'
    }
  }).then((result) => {
    if (result.isConfirmed) {
      const formData = new FormData();
      formData.append('action', 'delete');
      formData.append('id', contentId);
      
      fetch('api/content_api.php', {
        method: 'POST',
        body: formData
      })
      .then(response => response.json())
      .then(data => {
        if(data.success) {
          loadContents();
          
          Swal.fire({
            icon: 'success',
            title: 'تم الحذف!',
            text: data.message,
            timer: 2000,
            showConfirmButton: false,
            timerProgressBar: true,
            customClass: {
              container: 'swal-over-modal'
            }
          });
        } else {
          Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: data.message,
            confirmButtonText: 'حسناً',
            confirmButtonColor: '#667eea',
            customClass: {
              container: 'swal-over-modal'
            }
          });
        }
      })
      .catch(error => {
        Swal.fire({
          icon: 'error',
          title: 'خطأ',
          text: 'حدث خطأ أثناء حذف المحتوى',
          confirmButtonText: 'حسناً',
          confirmButtonColor: '#667eea',
          customClass: {
            container: 'swal-over-modal'
          }
        });
        console.error('Delete error:', error);
      });
    }
  });
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
  const map = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;'
  };
  return text.replace(/[&<>"']/g, m => map[m]);
}

// Close modal on outside click
window.onclick = function(event) {
  const modal = document.getElementById('content-modal');
  if (event.target === modal) {
    closeContentModal();
  }
}
