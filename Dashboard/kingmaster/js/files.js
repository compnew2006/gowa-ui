// Storage management
let totalStorage = 100; // 100 MB
let usedStorage = 0; // MB
let files = [];
let selectedFile = null;

// Load saved data from database
function loadData() {
  // Load files
  fetch('api/files_api.php?action=get_all')
    .then(response => response.json())
    .then(data => {
      if(data.success) {
        files = data.files;
        renderFiles();
      }
    })
    .catch(error => console.error('Error loading files:', error));
  
  // Load storage info
  fetch('api/files_api.php?action=get_storage')
    .then(response => response.json())
    .then(data => {
      if(data.success) {
        usedStorage = data.used_mb;
        totalStorage = 100;
        updateStorageBar();
      }
    })
    .catch(error => console.error('Error loading storage:', error));
}

// Update storage bar
function updateStorageBar() {
  const percentage = (usedStorage / totalStorage) * 100;
  document.getElementById('storage-fill').style.width = percentage + '%';
  document.getElementById('storage-text').textContent = usedStorage.toFixed(2) + ' MB / ' + totalStorage + ' MB';
  document.getElementById('remaining-storage').textContent = (totalStorage - usedStorage).toFixed(2);
  
  // Change color based on usage
  const fillEl = document.getElementById('storage-fill');
  if (percentage > 90) {
    fillEl.style.background = 'linear-gradient(90deg, #ef4444, #dc2626)';
  } else if (percentage > 70) {
    fillEl.style.background = 'linear-gradient(90deg, #f59e0b, #d97706)';
  } else {
    fillEl.style.background = 'linear-gradient(90deg, var(--primary-color), var(--secondary-color))';
  }
}

// Open upload modal
function openUploadModal() {
  document.getElementById('upload-modal').classList.add('active');
  resetUploadForm();
}

// Close upload modal
function closeUploadModal() {
  document.getElementById('upload-modal').classList.remove('active');
  resetUploadForm();
}

// Reset upload form
function resetUploadForm() {
  document.getElementById('file-input').value = '';
  document.getElementById('file-name-input').value = '';
  document.getElementById('preview-area').style.display = 'none';
  document.getElementById('file-info').style.display = 'none';
  document.getElementById('upload-btn').disabled = true;
  document.getElementById('upload-area').style.display = 'block';
  selectedFile = null;
}

// Handle file selection
function handleFileSelect(event) {
  const file = event.target.files[0];
  if (!file) return;
  
  const fileSizeMB = file.size / (1024 * 1024);
  
  // Check if file size exceeds remaining storage
  if (fileSizeMB + usedStorage > totalStorage) {
    Swal.fire({
      icon: 'error',
      title: 'مساحة غير كافية',
      text: 'الملف أكبر من المساحة المتبقية. يرجى حذف بعض الملفات أولاً',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    return;
  }
  
  selectedFile = file;
  
  // Hide upload area, show preview
  document.getElementById('upload-area').style.display = 'none';
  document.getElementById('preview-area').style.display = 'block';
  document.getElementById('file-info').style.display = 'block';
  document.getElementById('upload-btn').disabled = false;
  
  // Set file info
  document.getElementById('file-size').textContent = fileSizeMB.toFixed(2) + ' MB';
  document.getElementById('file-type').textContent = file.type || 'غير معروف';
  document.getElementById('file-name-input').value = file.name;
  
  // Show preview
  const previewContent = document.getElementById('preview-content');
  previewContent.innerHTML = '';
  
  if (file.type.startsWith('image/')) {
    const img = document.createElement('img');
    img.src = URL.createObjectURL(file);
    previewContent.appendChild(img);
  } else if (file.type.startsWith('video/')) {
    const video = document.createElement('video');
    video.src = URL.createObjectURL(file);
    video.controls = true;
    previewContent.appendChild(video);
  } else if (file.type === 'application/pdf') {
    const iframe = document.createElement('iframe');
    iframe.src = URL.createObjectURL(file);
    previewContent.appendChild(iframe);
  }
}

// Upload file
function uploadFile() {
  if (!selectedFile) return;
  
  const fileName = document.getElementById('file-name-input').value || selectedFile.name;
  
  // Create FormData to send file
  const formData = new FormData();
  formData.append('action', 'upload');
  formData.append('file', selectedFile);
  formData.append('name', fileName);
  
  // Show loading
  Swal.fire({
    title: 'جاري الرفع...',
    text: 'يرجى الانتظار',
    allowOutsideClick: false,
    didOpen: () => {
      Swal.showLoading();
    }
  });
  
  // Upload to server
  fetch('api/files_api.php', {
    method: 'POST',
    body: formData
  })
  .then(response => response.json())
  .then(data => {
    Swal.close();
    
    if(data.success) {
      loadData(); // Reload files and storage
      closeUploadModal();
      
      Swal.fire({
        icon: 'success',
        title: 'تم بنجاح!',
        text: 'تم رفع الملف بنجاح',
        timer: 2000,
        showConfirmButton: false,
        timerProgressBar: true
      });
    } else {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: data.message || 'فشل رفع الملف',
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea'
      });
    }
  })
  .catch(error => {
    Swal.close();
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ أثناء رفع الملف',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    console.error('Upload error:', error);
  });
}

// Render files
function renderFiles() {
  const grid = document.getElementById('files-grid');
  
  if (files.length === 0) {
    grid.innerHTML = `
      <div class="empty-state">
        <i class="fas fa-folder-open" style="font-size: 4rem; color: var(--text-gray); opacity: 0.3;"></i>
        <p style="color: var(--text-gray); margin-top: 1rem;">لا توجد ملفات حتى الآن</p>
        <button onclick="openUploadModal()" class="upload-btn" style="margin-top: 1rem;">
          <i class="fas fa-plus"></i>
          رفع أول ملف
        </button>
      </div>
    `;
    return;
  }
  
  grid.innerHTML = files.map(file => {
    let previewHTML = '';
    const fileSizeMB = (file.file_size / (1024 * 1024)).toFixed(2);
    
    if (file.file_type === 'image') {
      previewHTML = `<img src="${file.file_path}" alt="${file.name}">`;
    } else if (file.file_type === 'video') {
      previewHTML = `<video src="${file.file_path}" controls></video>`;
    } else if (file.file_type === 'pdf') {
      previewHTML = `<i class="fas fa-file-pdf fa-beat-fade" style="color: #ef4444;"></i>`;
    } else {
      previewHTML = `<i class="fas fa-file fa-beat-fade" style="color: #6b7280;"></i>`;
    }
    
    return `
      <div class="file-card">
        <div class="file-preview">${previewHTML}</div>
        <div class="file-info-text">
          <div class="file-name" title="${file.name}">${file.name}</div>
          <div class="file-size">${fileSizeMB} MB</div>
        </div>
        <div class="file-actions">
          <button class="btn-edit" onclick="openEditModal(${file.id})">
            <i class="fas fa-edit fa-flip" style="--fa-animation-duration: 2s;"></i>
            تعديل
          </button>
          <button class="btn-delete" onclick="deleteFile(${file.id})">
            <i class="fas fa-trash fa-shake" style="--fa-animation-duration: 3s;"></i>
            حذف
          </button>
        </div>
      </div>
    `;
  }).join('');
}

// Open edit modal
function openEditModal(fileId) {
  const file = files.find(f => f.id === fileId);
  if (!file) return;
  
  document.getElementById('edit-file-name').value = file.name;
  document.getElementById('edit-file-id').value = fileId;
  document.getElementById('edit-modal').classList.add('active');
}

// Close edit modal
function closeEditModal() {
  document.getElementById('edit-modal').classList.remove('active');
}

// Save edit
function saveEdit() {
  const fileId = parseInt(document.getElementById('edit-file-id').value);
  const newName = document.getElementById('edit-file-name').value;
  
  if (!newName) {
    Swal.fire({
      icon: 'warning',
      title: 'تنبيه',
      text: 'يرجى إدخال اسم الملف',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    return;
  }
  
  const formData = new FormData();
  formData.append('action', 'update');
  formData.append('id', fileId);
  formData.append('name', newName);
  
  fetch('api/files_api.php', {
    method: 'POST',
    body: formData
  })
  .then(response => response.json())
  .then(data => {
    if(data.success) {
      loadData();
      closeEditModal();
      Swal.fire({
        icon: 'success',
        title: 'تم التحديث!',
        text: 'تم تحديث اسم الملف بنجاح',
        timer: 2000,
        showConfirmButton: false,
        timerProgressBar: true
      });
    } else {
      Swal.fire({
        icon: 'error',
        title: 'خطأ',
        text: data.message || 'فشل تحديث الملف',
        confirmButtonText: 'حسناً',
        confirmButtonColor: '#667eea'
      });
    }
  })
  .catch(error => {
    Swal.fire({
      icon: 'error',
      title: 'خطأ',
      text: 'حدث خطأ أثناء التحديث',
      confirmButtonText: 'حسناً',
      confirmButtonColor: '#667eea'
    });
    console.error('Update error:', error);
  });
}

// Delete file
function deleteFile(fileId) {
  Swal.fire({
    title: 'هل أنت متأكد؟',
    text: 'لن تتمكن من استرجاع هذا الملف!',
    icon: 'warning',
    showCancelButton: true,
    confirmButtonColor: '#ef4444',
    cancelButtonColor: '#667eea',
    confirmButtonText: 'نعم، احذف!',
    cancelButtonText: 'إلغاء'
  }).then((result) => {
    if (result.isConfirmed) {
      const formData = new FormData();
      formData.append('action', 'delete');
      formData.append('id', fileId);
      
      fetch('api/files_api.php', {
        method: 'POST',
        body: formData
      })
      .then(response => response.json())
      .then(data => {
        if(data.success) {
          loadData();
          Swal.fire({
            icon: 'success',
            title: 'تم الحذف!',
            text: 'تم حذف الملف بنجاح',
            timer: 2000,
            showConfirmButton: false,
            timerProgressBar: true
          });
        } else {
          Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: data.message || 'فشل حذف الملف',
            confirmButtonText: 'حسناً',
            confirmButtonColor: '#667eea'
          });
        }
      })
      .catch(error => {
        Swal.fire({
          icon: 'error',
          title: 'خطأ',
          text: 'حدث خطأ أثناء حذف الملف',
          confirmButtonText: 'حسناً',
          confirmButtonColor: '#667eea'
        });
        console.error('Delete error:', error);
      });
    }
  });
}

// Close modals on outside click
window.onclick = function(event) {
  const uploadModal = document.getElementById('upload-modal');
  const editModal = document.getElementById('edit-modal');
  
  if (event.target === uploadModal) {
    closeUploadModal();
  }
  if (event.target === editModal) {
    closeEditModal();
  }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
  loadData();
});
