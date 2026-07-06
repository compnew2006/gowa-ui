// إنشاء النجوم في الخلفية
function createStars() {
    const starsContainer = document.getElementById('stars-bg');
    if (!starsContainer) return;
    
    const numberOfStars = 150;
    
    for (let i = 0; i < numberOfStars; i++) {
        const star = document.createElement('div');
        star.className = 'star-auth';
        
        const size = Math.random() * 3 + 1;
        star.style.width = size + 'px';
        star.style.height = size + 'px';
        
        star.style.left = Math.random() * 100 + '%';
        star.style.top = Math.random() * 100 + '%';
        
        const duration = Math.random() * 3 + 2;
        star.style.animationDuration = duration + 's';
        
        const delay = Math.random() * 3;
        star.style.animationDelay = delay + 's';
        
        starsContainer.appendChild(star);
    }
}

// إنشاء النجوم عند تحميل الصفحة
document.addEventListener('DOMContentLoaded', createStars);

// Toggle Password Visibility
function togglePassword(inputId) {
    const input = document.getElementById(inputId);
    const icon = input.nextElementSibling;
    
    if (input.type === 'password') {
        input.type = 'text';
        icon.classList.remove('fa-eye');
        icon.classList.add('fa-eye-slash');
    } else {
        input.type = 'password';
        icon.classList.remove('fa-eye-slash');
        icon.classList.add('fa-eye');
    }
}

// Form Submission
const loginForm = document.getElementById('loginForm');
const errorMessage = document.getElementById('errorMessage');
const successMessage = document.getElementById('successMessage');

if (loginForm) {
    loginForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // Clear previous messages
        errorMessage.classList.remove('show');
        successMessage.classList.remove('show');
        
        // Get form data
        const formData = new FormData(loginForm);
        
        // Validate terms agreement
        if (!formData.get('terms')) {
            showError('يجب الموافقة على الشروط والأحكام');
            return;
        }
        
        // Disable submit button
        const submitBtn = loginForm.querySelector('.btn-submit');
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> جاري تسجيل الدخول...';
        
        try {
            const response = await fetch('login_handler.php', {
                method: 'POST',
                body: formData
            });
            
            const data = await response.json();
            
            if (data.success) {
                showSuccess(data.message);
                
                // Redirect after 1 second
                setTimeout(() => {
                    window.location.href = data.redirect;
                }, 1000);
            } else {
                showError(data.message);
                submitBtn.disabled = false;
                submitBtn.innerHTML = '<i class="fas fa-sign-in-alt"></i> تسجيل الدخول';
            }
        } catch (error) {
            showError('حدث خطأ أثناء الاتصال بالخادم');
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<i class="fas fa-sign-in-alt"></i> تسجيل الدخول';
        }
    });
}

// Forgot Password Modal Logic
const forgotLink = document.getElementById('openForgotModal');
const forgotOverlay = document.getElementById('forgotModalOverlay');
const forgotClose = document.getElementById('closeForgotModal');
const forgotCancel = document.getElementById('cancelForgot');
const forgotSubmit = document.getElementById('submitForgot');
const forgotEmail = document.getElementById('forgotEmail');
const forgotInlineMsg = document.getElementById('forgotInlineMsg');

function openForgot() {
    if (!forgotOverlay) return;
    forgotOverlay.style.display = 'flex';
    forgotOverlay.setAttribute('aria-hidden', 'false');
    if (forgotEmail) { forgotEmail.focus(); }
    if (forgotInlineMsg) { forgotInlineMsg.textContent = ''; forgotInlineMsg.className = 'inline-feedback'; }
}
function closeForgot() {
    if (!forgotOverlay) return;
    forgotOverlay.style.display = 'none';
    forgotOverlay.setAttribute('aria-hidden', 'true');
}

if (forgotLink) forgotLink.addEventListener('click', openForgot);
if (forgotClose) forgotClose.addEventListener('click', closeForgot);
if (forgotCancel) forgotCancel.addEventListener('click', closeForgot);
if (forgotOverlay) forgotOverlay.addEventListener('click', (e) => { if (e.target === forgotOverlay) closeForgot(); });

if (forgotSubmit) {
    forgotSubmit.addEventListener('click', async () => {
        if (!forgotEmail || !forgotEmail.value) {
            if (forgotInlineMsg) { forgotInlineMsg.textContent = 'يرجى إدخال البريد الإلكتروني'; forgotInlineMsg.className = 'inline-feedback error'; }
            return;
        }
        const btnText = forgotSubmit.innerHTML;
        forgotSubmit.disabled = true;
        forgotSubmit.innerHTML = '<i class="fas fa-spinner fa-spin"></i> جاري الإرسال...';
        if (forgotInlineMsg) { forgotInlineMsg.textContent = ''; forgotInlineMsg.className = 'inline-feedback'; }
        try {
            const res = await fetch('api/reset_password_login.php', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email: forgotEmail.value.trim() })
            });
            const data = await res.json();
            if (data.success) {
                if (forgotInlineMsg) { forgotInlineMsg.textContent = data.message || 'تمت العملية بنجاح'; forgotInlineMsg.className = 'inline-feedback success'; }
                setTimeout(closeForgot, 1500);
            } else {
                if (forgotInlineMsg) { forgotInlineMsg.textContent = data.message || 'حدث خطأ'; forgotInlineMsg.className = 'inline-feedback error'; }
            }
        } catch (e) {
            if (forgotInlineMsg) { forgotInlineMsg.textContent = 'تعذر الاتصال بالخادم'; forgotInlineMsg.className = 'inline-feedback error'; }
        } finally {
            forgotSubmit.disabled = false;
            forgotSubmit.innerHTML = btnText;
        }
    });
}

function showError(message) {
    errorMessage.textContent = message;
    errorMessage.classList.add('show');
    
    // Scroll to error
    errorMessage.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function showSuccess(message) {
    successMessage.textContent = message;
    successMessage.classList.add('show');
}
