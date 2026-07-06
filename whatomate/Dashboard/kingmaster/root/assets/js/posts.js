// Modal Elements
const modal = document.getElementById('postModal');
const addPostBtn = document.getElementById('addPostBtn');
const closeModal = document.getElementById('closeModal');
const cancelBtn = document.getElementById('cancelBtn');
const postForm = document.getElementById('postForm');
const errorMessage = document.getElementById('errorMessage');
const successMessage = document.getElementById('successMessage');

// فتح Modal لإضافة نشرة جديدة
addPostBtn.addEventListener('click', () => {
    openModal();
    document.getElementById('modalTitle').innerHTML = '<i class="fas fa-plus-circle"></i> إضافة نشرة جديدة';
    postForm.reset();
    document.getElementById('postId').value = '';
    clearMessages();
});

// إغلاق Modal
closeModal.addEventListener('click', closeModalHandler);
cancelBtn.addEventListener('click', closeModalHandler);

// إغلاق Modal عند الضغط خارجها
modal.addEventListener('click', (e) => {
    if (e.target === modal) {
        closeModalHandler();
    }
});

function openModal() {
    modal.classList.add('show');
    document.body.style.overflow = 'hidden';
}

function closeModalHandler() {
    modal.classList.remove('show');
    document.body.style.overflow = 'auto';
    clearMessages();
}

function clearMessages() {
    errorMessage.classList.remove('show');
    successMessage.classList.remove('show');
}

function showError(message) {
    errorMessage.textContent = message;
    errorMessage.classList.add('show');
    successMessage.classList.remove('show');
}

function showSuccess(message) {
    successMessage.textContent = message;
    successMessage.classList.add('show');
    errorMessage.classList.remove('show');
}

// إرسال النموذج
postForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearMessages();
    
    const formData = new FormData(postForm);
    const submitBtn = postForm.querySelector('.btn-save');
    
    // تعطيل الزر
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> جاري الحفظ...';
    
    try {
        const response = await fetch('posts_handler.php', {
            method: 'POST',
            body: formData
        });
        
        const data = await response.json();
        
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'نجاح',
                text: data.message,
                timer: 2000,
                showConfirmButton: false
            });
            postForm.reset();
            closeModalHandler();
            loadPosts();
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message,
                confirmButtonText: 'حسناً'
            });
        }
    } catch (error) {
        Swal.fire({
            icon: 'error',
            title: 'خطأ في الاتصال',
            text: 'حدث خطأ أثناء الاتصال بالخادم',
            confirmButtonText: 'حسناً'
        });
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = '<i class="fas fa-save"></i> حفظ النشرة';
    }
});

// تحميل النشرات
async function loadPosts() {
    const postsGrid = document.getElementById('postsGrid');
    postsGrid.innerHTML = '<div class="loading"><i class="fas fa-spinner fa-spin"></i>جاري التحميل...</div>';
    
    try {
        const response = await fetch('get_posts.php');
        const data = await response.json();
        
        if (data.success) {
            if (data.posts.length === 0) {
                postsGrid.innerHTML = `
                    <div class="loading">
                        <i class="fas fa-inbox"></i>
                        لا توجد نشرات حتى الآن. أضف أول نشرة!
                    </div>
                `;
            } else {
                postsGrid.innerHTML = data.posts.map(post => createPostCard(post)).join('');
            }
        } else {
            postsGrid.innerHTML = `
                <div class="loading">
                    <i class="fas fa-exclamation-circle"></i>
                    ${data.message}
                </div>
            `;
        }
    } catch (error) {
        postsGrid.innerHTML = `
            <div class="loading">
                <i class="fas fa-exclamation-triangle"></i>
                حدث خطأ أثناء تحميل النشرات
            </div>
        `;
    }
}

// إنشاء بطاقة نشرة
function createPostCard(post) {
    const date = new Date(post.created_at).toLocaleDateString('ar-EG', {
        year: 'numeric',
        month: 'long',
        day: 'numeric'
    });
    
    // اختصار الوصف إلى 150 حرف
    const content = post.content || '';
    const shortContent = content.length > 150 ? content.substring(0, 150) + '...' : content;
    
    // تحديد نوع المنشور
    const typeLabels = {
        'New Feature': 'ميزة جديدة',
        'System Update': 'تحديث نظام',
        'Maintenance': 'صيانة'
    };
    
    const typeClasses = {
        'New Feature': 'type-new-feature',
        'System Update': 'type-system-update',
        'Maintenance': 'type-maintenance'
    };
    
    const typeLabel = typeLabels[post.type] || post.type;
    const typeClass = typeClasses[post.type] || 'type-new-feature';
    
    return `
        <div class="post-card">
            <h3>${post.title}</h3>
            <span class="post-type ${typeClass}">${typeLabel}</span>
            <p>${shortContent}</p>
            <div class="post-footer">
                <span class="post-date">
                    <i class="fas fa-calendar"></i>
                    ${date}
                </span>
                <div class="post-actions">
                    <button class="btn-delete" onclick="deletePost(${post.id})">
                        <i class="fas fa-trash"></i> حذف
                    </button>
                </div>
            </div>
        </div>
    `;
}

// حذف نشرة
async function deletePost(postId) {
    const result = await Swal.fire({
        title: 'هل أنت متأكد؟',
        text: 'هل تريد حذف هذه النشرة؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#dc2626',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء'
    });
    
    if (!result.isConfirmed) return;
    
    try {
        const formData = new FormData();
        formData.append('post_id', postId);
        
        const response = await fetch('delete_post.php', {
            method: 'POST',
            body: formData
        });
        
        const data = await response.json();
        
        if (data.success) {
            Swal.fire({
                icon: 'success',
                title: 'تم الحذف',
                text: 'تم حذف النشرة بنجاح',
                timer: 2000,
                showConfirmButton: false
            });
            loadPosts(); // إعادة تحميل النشرات
        } else {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: data.message || 'فشل حذف النشرة',
                confirmButtonText: 'حسناً'
            });
        }
    } catch (error) {
        console.error('Error:', error);
        Swal.fire({
            icon: 'error',
            title: 'خطأ في الاتصال',
            text: 'حدث خطأ أثناء حذف النشرة',
            confirmButtonText: 'حسناً'
        });
    }
}

// تحميل النشرات عند فتح الصفحة
document.addEventListener('DOMContentLoaded', loadPosts);
