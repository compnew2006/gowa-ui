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

// Password Strength Checker
const passwordInput = document.getElementById('password');
const strengthFill = document.getElementById('strengthFill');
const strengthText = document.getElementById('strengthText');

if (passwordInput) {
    passwordInput.addEventListener('input', function() {
        const password = this.value;
        let strength = 0;
        
        // Check length
        if (password.length >= 8) strength += 25;
        if (password.length >= 12) strength += 25;
        
        // Check for lowercase
        if (/[a-z]/.test(password)) strength += 12.5;
        
        // Check for uppercase
        if (/[A-Z]/.test(password)) strength += 12.5;
        
        // Check for numbers
        if (/[0-9]/.test(password)) strength += 12.5;
        
        // Check for special characters
        if (/[^a-zA-Z0-9]/.test(password)) strength += 12.5;
        
        // Update bar
        strengthFill.style.width = strength + '%';
        
        // Update color and text
        if (strength <= 25) {
            strengthFill.style.background = '#ef4444';
            strengthText.style.color = '#ef4444';
            strengthText.textContent = 'ضعيفة جداً';
        } else if (strength <= 50) {
            strengthFill.style.background = '#f59e0b';
            strengthText.style.color = '#f59e0b';
            strengthText.textContent = 'ضعيفة';
        } else if (strength <= 75) {
            strengthFill.style.background = '#fbbf24';
            strengthText.style.color = '#fbbf24';
            strengthText.textContent = 'متوسطة';
        } else if (strength < 100) {
            strengthFill.style.background = '#10b981';
            strengthText.style.color = '#10b981';
            strengthText.textContent = 'قوية';
        } else {
            strengthFill.style.background = '#059669';
            strengthText.style.color = '#059669';
            strengthText.textContent = 'قوية جداً';
        }
    });
}

// Password Match Checker
const confirmPasswordInput = document.getElementById('confirm_password');
const passwordMatch = document.getElementById('passwordMatch');

if (confirmPasswordInput) {
    confirmPasswordInput.addEventListener('input', function() {
        const password = passwordInput.value;
        const confirmPassword = this.value;
        
        if (confirmPassword.length === 0) {
            passwordMatch.textContent = '';
            passwordMatch.className = 'password-match';
        } else if (password === confirmPassword) {
            passwordMatch.textContent = '✓ كلمتا المرور متطابقتان';
            passwordMatch.className = 'password-match match';
        } else {
            passwordMatch.textContent = '✗ كلمتا المرور غير متطابقتين';
            passwordMatch.className = 'password-match no-match';
        }
    });
}

// Form Submission
const registerForm = document.getElementById('registerForm');
const errorMessage = document.getElementById('errorMessage');
const successMessage = document.getElementById('successMessage');

if (registerForm) {
    registerForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // Clear previous messages
        errorMessage.classList.remove('show');
        successMessage.classList.remove('show');
        
        // Get form data
        const formData = new FormData(registerForm);
        
        // Validate passwords match
        if (formData.get('password') !== formData.get('confirm_password')) {
            showError('كلمتا المرور غير متطابقتين');
            return;
        }
        
        // Validate terms agreement
        if (!formData.get('terms')) {
            showError('يجب الموافقة على الشروط والأحكام');
            return;
        }
        
        // Disable submit button
        const submitBtn = registerForm.querySelector('.btn-submit');
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> جاري التسجيل...';
        
        try {
            const response = await fetch('register_handler.php', {
                method: 'POST',
                body: formData
            });
            
            const data = await response.json();
            
            if (data.success) {
                showSuccess(data.message);
                
                // Show OTP for testing
                if (data.otp) {
                    console.log('رمز التحقق:', data.otp);
                }
                
                // Redirect after 2 seconds
                setTimeout(() => {
                    window.location.href = data.redirect;
                }, 2000);
            } else {
                showError(data.message);
                submitBtn.disabled = false;
                submitBtn.innerHTML = '<i class="fas fa-user-plus"></i> إنشاء حساب';
            }
        } catch (error) {
            showError('حدث خطأ أثناء الاتصال بالخادم');
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<i class="fas fa-user-plus"></i> إنشاء حساب';
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
