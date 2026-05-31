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

// OTP Input Auto-focus
const otpInputs = document.querySelectorAll('.otp-input');

otpInputs.forEach((input, index) => {
    input.addEventListener('input', function(e) {
        // Only allow numbers
        this.value = this.value.replace(/[^0-9]/g, '');
        
        // Move to next input
        if (this.value.length === 1 && index < otpInputs.length - 1) {
            otpInputs[index + 1].focus();
        }
    });
    
    input.addEventListener('keydown', function(e) {
        // Move to previous input on backspace
        if (e.key === 'Backspace' && this.value === '' && index > 0) {
            otpInputs[index - 1].focus();
        }
    });
    
    // Handle paste
    input.addEventListener('paste', function(e) {
        e.preventDefault();
        const pastedData = e.clipboardData.getData('text').replace(/[^0-9]/g, '');
        
        // Fill inputs with pasted data
        for (let i = 0; i < pastedData.length && index + i < otpInputs.length; i++) {
            otpInputs[index + i].value = pastedData[i];
        }
        
        // Focus on last filled input
        const lastIndex = Math.min(index + pastedData.length, otpInputs.length - 1);
        otpInputs[lastIndex].focus();
    });
});

// OTP Form Submission
const otpForm = document.getElementById('otpForm');
const errorMessage = document.getElementById('errorMessage');
const successMessage = document.getElementById('successMessage');

if (otpForm) {
    otpForm.addEventListener('submit', async function(e) {
        e.preventDefault();
        
        // Clear previous messages
        errorMessage.classList.remove('show');
        successMessage.classList.remove('show');
        
        // Get OTP value
        let otp = '';
        otpInputs.forEach(input => {
            otp += input.value;
        });
        
        // Validate OTP length
        if (otp.length !== 6) {
            showError('يرجى إدخال رمز التحقق كاملاً');
            return;
        }
        
        // Disable submit button
        const submitBtn = otpForm.querySelector('.btn-submit');
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> جاري التحقق...';
        
        try {
            const formData = new FormData();
            formData.append('otp', otp);
            
            const response = await fetch('verify_otp_handler.php', {
                method: 'POST',
                body: formData
            });
            
            const data = await response.json();
            
            if (data.success) {
                showSuccess(data.message);
                
                // Redirect after 2 seconds
                setTimeout(() => {
                    window.location.href = data.redirect;
                }, 2000);
            } else {
                showError(data.message);
                
                // Clear OTP inputs
                otpInputs.forEach(input => input.value = '');
                otpInputs[0].focus();
                
                submitBtn.disabled = false;
                submitBtn.innerHTML = '<i class="fas fa-check-circle"></i> تحقق من الرمز';
            }
        } catch (error) {
            showError('حدث خطأ أثناء الاتصال بالخادم');
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<i class="fas fa-check-circle"></i> تحقق من الرمز';
        }
    });
}

// Resend OTP with Countdown
const resendBtn = document.getElementById('resendBtn');
const countdownEl = document.getElementById('countdown');
const resendTextEl = document.getElementById('resendText');

let countdownInterval;
let timeLeft = 60;

// بدء العداد التنازلي عند تحميل الصفحة
function startCountdown() {
    resendBtn.disabled = true;
    countdownEl.style.display = 'inline';
    
    countdownInterval = setInterval(() => {
        timeLeft--;
        countdownEl.textContent = `(${timeLeft})`;
        
        if (timeLeft <= 0) {
            clearInterval(countdownInterval);
            resendBtn.disabled = false;
            countdownEl.style.display = 'none';
            timeLeft = 60;
        }
    }, 1000);
}

// بدء العد عند تحميل الصفحة
if (resendBtn) {
    startCountdown();
    
    resendBtn.addEventListener('click', async function() {
        // حفظ النص الأصلي
        const originalHTML = resendTextEl.innerHTML;
        
        this.disabled = true;
        resendTextEl.innerHTML = 'جاري الإرسال...';
        const icon = this.querySelector('i');
        icon.className = 'fas fa-spinner fa-spin';
        countdownEl.style.display = 'none';
        
        try {
            const response = await fetch('resend_otp.php', {
                method: 'POST'
            });
            
            const data = await response.json();
            
            if (data.success) {
                showSuccess('تم إرسال رمز جديد بنجاح');
                
                // Show OTP for testing
                if (data.otp) {
                    console.log('رمز التحقق الجديد:', data.otp);
                }
                
                // إعادة النص والأيقونة
                resendTextEl.innerHTML = originalHTML;
                icon.className = 'fas fa-redo';
                
                // بدء عداد تنازلي جديد
                timeLeft = 60;
                startCountdown();
            } else {
                showError(data.message);
                this.disabled = false;
                resendTextEl.innerHTML = originalHTML;
                icon.className = 'fas fa-redo';
            }
        } catch (error) {
            showError('حدث خطأ أثناء إرسال الرمز');
            this.disabled = false;
            resendTextEl.innerHTML = originalHTML;
            icon.className = 'fas fa-redo';
        }
    });
}

function showError(message) {
    errorMessage.textContent = message;
    errorMessage.classList.add('show');
}

function showSuccess(message) {
    successMessage.textContent = message;
    successMessage.classList.add('show');
}
