// تحميل الحسابات عند فتح الصفحة
document.addEventListener('DOMContentLoaded', () => {
    loadAccounts();
    loadAccountLimit();
});

let currentPlatform = null;
let allAccounts = []; // تخزين جميع الحسابات للفلترة

// جلب الحسابات
async function loadAccounts() {
    try {
        const response = await fetch('api/accounts.php');
        const data = await response.json();
        
        if (data.success) {
            allAccounts = data.accounts; // حفظ جميع الحسابات
            applyFilters(); // تطبيق الفلاتر
        } else {
            console.error('خطأ في جلب الحسابات:', data.message);
        }
    } catch (error) {
        console.error('خطأ:', error);
    }
}

// عرض الحسابات
function renderAccounts(accounts) {
    const grid = document.getElementById('accountsGrid');
    
    if (accounts.length === 0) {
        grid.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-user-plus"></i>
                <h3>لا توجد حسابات</h3>
                <p>ابدأ بإضافة حساباتك على المنصات المختلفة</p>
            </div>
        `;
        return;
    }
    
    grid.innerHTML = accounts.map(account => createAccountCard(account)).join('');
}

// إنشاء بطاقة حساب
function createAccountCard(account) {
    const platformInfo = getPlatformInfo(account.channel);
    
    // تحديد ما إذا كان يحتاج زر إعادة ربط
    const needsReconnect = ['instagram', 'whatsapp', 'telegram'].includes(account.channel) || 
                          (account.channel === 'facebook' && account.method === 'data');
    
    return `
        <div class="account-card">
            <i class="${platformInfo.icon} account-platform-icon platform-${account.channel}"></i>
            <div class="account-name">${account.name || platformInfo.name}</div>
            <div class="account-uid">ID: ${account.account_uid || 'غير محدد'}</div>
            <span class="account-status status-${account.status}">
                <i class="fas fa-circle"></i>
                ${getStatusLabel(account.status)}
            </span>
            <div class="account-actions" style="${needsReconnect ? 'grid-template-columns: 1fr 1fr 1fr 1fr;' : ''}">
                <button class="action-btn edit-btn" onclick="editAccount(${account.id})">
                    <i class="fas fa-edit"></i> تعديل
                </button>
                <button class="action-btn verify-btn" onclick="verifyAccount(${account.id}, '${account.status}')">
                    <i class="fas fa-check-circle"></i> تحقق
                </button>
                ${needsReconnect ? `
                    <button class="action-btn reconnect-btn" onclick="reconnectAccount(${account.id}, '${account.channel}')">
                        <i class="fas fa-sync-alt"></i> إعادة ربط
                    </button>
                ` : ''}
                <button class="action-btn delete-btn" onclick="deleteAccount(${account.id})">
                    <i class="fas fa-trash"></i> حذف
                </button>
            </div>
        </div>
    `;
}

// معلومات المنصات
function getPlatformInfo(platform) {
    const platforms = {
        'facebook': { name: 'Facebook', icon: 'fab fa-facebook' },
        'whatsapp': { name: 'WhatsApp', icon: 'fab fa-whatsapp' },
        'instagram': { name: 'Instagram', icon: 'fab fa-instagram' },
        'telegram': { name: 'Telegram', icon: 'fab fa-telegram' },
        'email': { name: 'Email', icon: 'fas fa-envelope' }
    };
    return platforms[platform] || { name: platform, icon: 'fas fa-user' };
}

// تسميات الحالة
function getStatusLabel(status) {
    const labels = {
        'active': 'نشط',
        'inactive': 'غير نشط',
        'pending': 'قيد الانتظار',
        'closed': 'مغلق'
    };
    return labels[status] || status;
}

// ===== نظام الفلترة والبحث =====
function applyFilters() {
    const platformFilter = document.getElementById('platformFilter')?.value || 'all';
    const statusFilter = document.getElementById('statusFilter')?.value || 'all';
    const searchQuery = document.getElementById('searchInput')?.value.toLowerCase().trim() || '';
    
    let filteredAccounts = allAccounts;
    
    // فلتر حسب المنصة
    if (platformFilter !== 'all') {
        filteredAccounts = filteredAccounts.filter(acc => acc.channel === platformFilter);
    }
    
    // فلتر حسب الحالة
    if (statusFilter !== 'all') {
        filteredAccounts = filteredAccounts.filter(acc => acc.status === statusFilter);
    }
    
    // بحث بالاسم
    if (searchQuery) {
        filteredAccounts = filteredAccounts.filter(acc => {
            const name = (acc.name || '').toLowerCase();
            const uid = (acc.account_uid || '').toLowerCase();
            const platform = (acc.channel || '').toLowerCase();
            return name.includes(searchQuery) || uid.includes(searchQuery) || platform.includes(searchQuery);
        });
    }
    
    renderAccounts(filteredAccounts);
}

// فتح نافذة اختيار المنصة
function openPlatformSelector() {
    openModal('platformModal');
}

// اختيار المنصة
function selectPlatform(platform) {
    currentPlatform = platform;
    closeModal('platformModal');
    
    // فتح النافذة المناسبة حسب المنصة
    switch(platform) {
        case 'facebook':
            openModal('facebookMethodModal');
            break;
        case 'whatsapp':
            openModal('whatsappModal');
            startWhatsAppQR();
            break;
        case 'instagram':
            openModal('instagramModal');
            break;
        case 'telegram':
            openModal('telegramModal');
            break;
        case 'email':
            openModal('emailModal');
            break;
    }
}

// فتح/إغلاق النوافذ المنبثقة
function openModal(modalId) {
    document.getElementById(modalId).style.display = 'block';
}

// إغلاق النافذة عند الضغط خارجها
window.onclick = function(event) {
    if (event.target.classList.contains('modal')) {
        event.target.style.display = 'none';
    }
}

// حذف حساب
async function deleteAccount(accountId) {
    const result = await Swal.fire({
        title: 'تأكيد الحذف',
        text: 'هل أنت متأكد من حذف هذا الحساب؟',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'نعم، احذف',
        cancelButtonText: 'إلغاء',
        confirmButtonColor: '#ff6b6b'
    });
    
    if (!result.isConfirmed) return;
    
    try {
        const response = await fetch('api/accounts.php', {
            method: 'DELETE',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ account_id: accountId })
        });
        
        const data = await response.json();
        
        if (data.success) {
            Swal.fire('تم الحذف!', data.message, 'success');
            loadAccounts();
            loadAccountLimit(); // تحديث عداد الحسابات
        } else {
            Swal.fire('خطأ!', data.message, 'error');
        }
    } catch (error) {
        console.error('خطأ:', error);
        Swal.fire('خطأ!', 'حدث خطأ في حذف الحساب', 'error');
    }
}

// إضافة حساب عام
async function addAccount(accountData) {
    try {
        const response = await fetch('api/accounts.php', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(accountData)
        });
        
        const data = await response.json();
        
        if (data.success) {
            Swal.fire('نجح!', data.message, 'success');
            loadAccounts();
            // إغلاق جميع النوافذ المفتوحة
            document.querySelectorAll('.modal').forEach(modal => {
                modal.style.display = 'none';
            });
        } else {
            Swal.fire('خطأ!', data.message, 'error');
        }
    } catch (error) {
        console.error('خطأ:', error);
        Swal.fire('خطأ!', 'حدث خطأ في إضافة الحساب', 'error');
    }
}

// إضافة أو تحديث حساب (مع التحقق)
async function addOrUpdateAccount(accountData) {
    try {
        const response = await fetch('api/add_or_update_account.php', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(accountData)
        });
        
        const data = await response.json();
        
        if (data.success) {
            const actionText = data.action === 'updated' ? 'تم تحديث الحساب بنجاح' : 'تم إضافة الحساب بنجاح';
            Swal.fire('نجح!', actionText, 'success');
            loadAccounts();
            loadAccountLimit(); // تحديث عداد الحسابات
            // إغلاق جميع النوافذ المفتوحة
            document.querySelectorAll('.modal').forEach(modal => {
                modal.style.display = 'none';
            });
            // إعادة تعيين وضع إعادة الربط
            isReconnectMode = false;
        } else {
            // عرض رسالة خطأ محددة عند الوصول للحد الأقصى
            if (data.current_count !== undefined && data.max_count !== undefined) {
                Swal.fire({
                    icon: 'error',
                    title: 'الحد الأقصى للحسابات',
                    html: `
                        <div style="text-align: center; padding: 20px;">
                            <i class="fas fa-exclamation-triangle" style="font-size: 70px; color: #e74c3c; margin-bottom: 20px;"></i>
                            <p style="font-size: 18px; font-weight: 600; color: #2c3e50; margin-bottom: 15px;">
                                ${data.message}
                            </p>
                            <div style="background: #f8f9fa; padding: 15px; border-radius: 10px; margin: 20px 0;">
                                <p style="font-size: 16px; color: #7f8c8d; margin: 5px 0;">
                                    <i class="fas fa-info-circle" style="color: #3498db; margin-left: 8px;"></i>
                                    الحسابات الحالية: <strong style="color: #e74c3c;">${data.current_count}</strong> / <strong style="color: #27ae60;">${data.max_count}</strong>
                                </p>
                            </div>
                            <p style="font-size: 14px; color: #95a5a6; margin-top: 15px;">
                                <i class="fas fa-lightbulb" style="color: #f39c12; margin-left: 5px;"></i>
                                يرجى حذف حساب قديم أو ترقية باقتك لإضافة المزيد
                            </p>
                        </div>
                    `,
                    confirmButtonText: 'حسناً',
                    confirmButtonColor: '#e74c3c',
                    allowOutsideClick: false
                });
            } else {
                Swal.fire('خطأ!', data.message, 'error');
            }
        }
    } catch (error) {
        console.error('خطأ:', error);
        Swal.fire('خطأ!', 'حدث خطأ في العملية', 'error');
    }
}

// ===== Facebook Methods =====
function selectFacebookMethod(method) {
    closeModal('facebookMethodModal');
    if (method === 'credentials') {
        openModal('facebookCredentialsModal');
    } else if (method === 'cookies') {
        openModal('facebookCookiesTypeModal');
    }
}

function selectFacebookCookiesType(type) {
    closeModal('facebookCookiesTypeModal');
    if (type === 'single') {
        openModal('facebookSingleCookieModal');
    } else if (type === 'multiple') {
        openModal('facebookMultipleCookiesModal');
    }
}

async function submitFacebookCredentials() {
    const username = document.getElementById('fbUsername').value;
    const password = document.getElementById('fbPassword').value;
    const twoFa = document.getElementById('fbTwoFa').value;
    
    if (!username || !password) {
        Swal.fire('خطأ!', 'يرجى ملء جميع الحقول المطلوبة', 'warning');
        return;
    }
    
    await addAccount({
        channel: 'facebook',
        method: 'data',
        name: username,
        account_uid: username,
        data: { username, password, two_fa: twoFa }
    });
}

async function submitFacebookSingleCookie() {
    const cookies = document.getElementById('fbSingleCookie').value;
    
    if (!cookies) {
        Swal.fire('خطأ!', 'يرجى إدخال الكوكيز', 'warning');
        return;
    }
    
    await addAccount({
        channel: 'facebook',
        method: 'cookies',
        name: 'Facebook Account',
        cookies_text: cookies
    });
}

function handleFacebookMultipleCookies() {
    const fileInput = document.getElementById('fbMultipleCookiesFile');
    const file = fileInput.files[0];
    
    if (!file) {
        Swal.fire('خطأ!', 'يرجى اختيار ملف', 'warning');
        return;
    }
    
    const reader = new FileReader();
    reader.onload = async function(e) {
        const content = e.target.result;
        const accounts = content.split('|').filter(acc => acc.trim());
        
        for (const cookieText of accounts) {
            await addAccount({
                channel: 'facebook',
                method: 'cookies',
                name: 'Facebook Account',
                cookies_text: cookieText.trim()
            });
        }
        
        Swal.fire('نجح!', `تم إضافة ${accounts.length} حساب`, 'success');
        loadAccounts();
        closeModal('facebookMultipleCookiesModal');
    };
    reader.readAsText(file);
}

// ===== WhatsApp QR via WPPConnect API =====
let whatsappTimer = null;
let whatsappCheckInterval = null;
let currentInstanceId = null; // مستخدم في إعادة الربط القديمة (لو موجودة)
let currentWppSession = null; // اسم الجلسة في WPPConnect
let isReconnectMode = false; // لتحديد إذا كان في وضع إعادة الربط

const WPP_API = 'https://kingmaster.info/js/proxy.php';

async function startWhatsAppQR() {
    try {
        isReconnectMode = false;
        document.getElementById('whatsappQRSection').style.display = 'block';
        document.getElementById('whatsappReloadSection').style.display = 'none';
        document.querySelector('#whatsappQRSection .qr-code img').src = '';
        document.getElementById('whatsappTimer').textContent = 'جاري إنشاء الجلسة...';

        // اطلب اسم الجلسة من المستخدم أو أنشئ واحدًا افتراضيًا
        const { value: sess } = await Swal.fire({
            title: 'اكتب اسم الجلسة',
            input: 'text',
            inputValue: 'wa_' + Date.now(),
            inputLabel: 'سيتم إنشاء الجلسة بهذا الاسم',
            showCancelButton: true,
            confirmButtonText: 'متابعة',
            cancelButtonText: 'إلغاء'
        });
        if (!sess) { return; }
        currentWppSession = sess.trim();

        // start_session عبر AJAX مع البروكسي
        await new Promise((resolve, reject) => {
            $.ajax({
                url: '/js/proxy.php?action=start_session&session=' + encodeURIComponent(currentWppSession),
                method: 'POST',
                contentType: 'application/json',
                data: JSON.stringify({
                    webhook: 'https://apis.kingmaster.info/webhook.php',
                    waitQrCode: true
                }),
                dataType: 'json',
                success: function(response) {
                    resolve(response);
                },
                error: function(xhr, status, error) {
                    reject(error);
                }
            });
        });

        // اجلب QR وأعرضه
        await fetchQRCodeWPP();
        // استمر بالتحديث كل 15 ثانية
        whatsappCheckInterval = setInterval(fetchQRCodeWPP, 15000);
        // ابدأ التحقق من الاتصال كل 3 ثوانٍ
        pollConnectionStatus();

    } catch (error) {
        console.error('Error:', error);
        Swal.fire({
            icon: 'error',
            title: 'خطأ',
            text: 'تعذر إنشاء الجلسة/QR',
            confirmButtonText: 'حسناً'
        });
        document.getElementById('whatsappQRSection').style.display = 'none';
        document.getElementById('whatsappReloadSection').style.display = 'block';
    }
}


async function fetchQRCodeWPP(accountId = null) {
    try {
        if (accountId) {
            const account = allAccounts.find(acc => acc.id === accountId);
            currentWppSession = account.account_uid;
        }
        const j = await new Promise((resolve, reject) => {
            $.ajax({
                url: '/js/proxy.php?action=get_qr&session=' + encodeURIComponent(currentWppSession),
                method: 'POST',
                contentType: 'application/json',
                dataType: 'json',
                success: function(response) {
                    resolve(response);
                },
                error: function(xhr, status, error) {
                    reject(error);
                }
            });
        });
        
        // التحقق من الحالة "open" باستخدام j كما في الـ JSON الخاص بك
        const state = j.data && j.data.instance && j.data.instance.state;
        
        if (state === 'open') {
            // 1. الوصول للحاوية (الديزاين) الذي يحتوي على الصورة وإخفاؤه تماماً
            const qrSection = document.querySelector('#whatsappQRSection .qr-code');
            if (qrSection) {
                qrSection.style.display = 'none'; // هذا سيلغي شكل المربع والأيقونة المكسورة وكل شيء
            }
        
            // 2. تحديث الرسالة للمستخدم
            document.getElementById('whatsappTimer').textContent = 'الحساب متصل بالفعل ✅';
            document.getElementById('whatsappTimer').style.color = '#25D366';
            
            // 3. إيقاف التحديث الدوري وتفريغ الـ src للأمان
            document.querySelector('#whatsappQRSection .qr-code img').src = '';
            if (whatsappCheckInterval) clearInterval(whatsappCheckInterval);
            
            return; 
        }

        const qrcode = j.qrcode || (j.data && j.data.qrcode);
        const urlcode = j.urlcode || (j.data && j.data.urlcode);

        if (qrcode) {
            const imgEl = document.querySelector('#whatsappQRSection .qr-code img');
            const src = (typeof qrcode === 'string' && qrcode.startsWith('data:')) 
                ? qrcode 
                : ('data:image/png;base64,' + qrcode);
            imgEl.src = src;
            document.getElementById('whatsappTimer').textContent = 'امسح الكود الآن';
            document.getElementById('whatsappTimer').style.color = '#25D366';
        } else if (urlcode) {
            // لو تم إرجاع رابط فقط
            document.querySelector('#whatsappQRSection .qr-code img').src = '';
            document.getElementById('whatsappTimer').innerHTML = `<a href="${urlcode}" target="_blank">فتح QR</a>`;
        } else {
            document.getElementById('whatsappTimer').textContent = 'في انتظار QR...';
        }
    } catch (e) {
        console.error('fetchQRCodeWPP error', e);
    }
}


async function pollConnectionStatus() {
    let tries = 0;
    const maxTries = 120; // ~6 دقائق
    const tick = async () => {
        tries++;
        try {
            const j = await new Promise((resolve, reject) => {
                $.ajax({
                    url: '/js/proxy.php?action=check_connection&session=' + encodeURIComponent(currentWppSession),
                    method: 'POST',
                    contentType: 'application/json',
                    dataType: 'json',
                    success: function(response) {
                        resolve(response);
                    },
                    error: function(xhr, status, error) {
                        reject(error);
                    }
                });
            });
            
            const connected =
              (j && (j.connected === true)) ||
              (j && typeof j.status === 'string' && j.status.toUpperCase() === 'CONNECTED') ||
              (j && typeof j.state === 'string' && j.state.toLowerCase() === 'open') ||
              (j && j.data && typeof j.data.state === 'string' && j.data.state.toLowerCase() === 'open') ||
              (j && j.data && j.data.instance && typeof j.data.instance.state === 'string' && j.data.instance.state.toLowerCase() === 'open');
            //const rawVal = (j && (j.connected !== undefined ? j.connected : (j.data && j.data.connected)));
            //const connected = (rawVal === true) || (typeof rawVal === 'string' && rawVal.toLowerCase() === 'connected');
            if (connected) {
                // أوقف تحديث QR وأكمل الإعداد
                if (whatsappCheckInterval) clearInterval(whatsappCheckInterval);
                document.getElementById('whatsappTimer').textContent = '✅ Connected';
                await onWhatsappConnected();
                return;
            }
        } catch (e) {
            console.error('pollConnectionStatus error', e);
        }

        if (tries < maxTries) setTimeout(tick, 3000);
    };

    tick();
}


async function onWhatsappConnected() {
    try {
        // حاول جلب رقم الهاتف لعرضه كاسم
        let phone = null;
        try {
            const j = await new Promise((resolve, reject) => {
                $.ajax({
                    url: '/js/proxy.php?action=get_api__session__get_phone_number&session=' + encodeURIComponent(currentWppSession),
                    method: 'POST',
                    contentType: 'application/json',
                    dataType: 'json',
                    success: function(response) {
                        resolve(response);
                    },
                    error: function(xhr, status, error) {
                        reject(error);
                    }
                });
            });

            phone = (j.phoneNumber ? j.phoneNumber : '') + (j.pushname ? ' - ' + j.pushname : '') || null;
        } catch (e) {
            console.warn('Failed to get phone number:', e);
        }

        Swal.fire({
            icon: 'success',
            title: 'تم الربط!',
            text: 'تم ربط WhatsApp بنجاح',
            timer: 2000,
            showConfirmButton: false
        });

        await addOrUpdateAccount({
            channel: 'whatsapp',
            method: 'qr',
            name: phone || currentWppSession,
            account_uid: currentWppSession,
            is_reconnect: isReconnectMode,
            data: { session: currentWppSession }
        });

        closeModal('whatsappModal');

    } catch (e) {
        console.error('onWhatsappConnected error', e);
    }
}


function reloadWhatsAppQR() {
    if (whatsappCheckInterval) clearInterval(whatsappCheckInterval);
    if (currentWppSession) {
        document.getElementById('whatsappQRSection').style.display = 'block';
        document.getElementById('whatsappReloadSection').style.display = 'none';
        fetchQRCodeWPP();
        whatsappCheckInterval = setInterval(fetchQRCodeWPP, 15000);
        pollConnectionStatus();
    } else {
        startWhatsAppQR();
    }
}

// تنظيف عند إغلاق الـ Modal
function closeModal(modalId) {
    if (modalId === 'whatsappModal') {
        if (whatsappCheckInterval) clearInterval(whatsappCheckInterval);
        currentInstanceId = null;
        currentWppSession = null;
    }
    document.getElementById(modalId).style.display = 'none';
}

// ===== Instagram =====
async function submitInstagram() {
    //const username = document.getElementById('instaUsername').value;
    //const password = document.getElementById('instaPassword').value;
    //if (!username || !password) {
        //Swal.fire('خطأ!', 'يرجى ملء جميع الحقول', 'warning');
        //return;
    //}
    const cookie =  document.getElementById('instaSingleCookie').value;
    
    if (!cookie) {
        Swal.fire('خطأ!', 'يرجى ملء جميع الحقول', 'warning');
        return;
    }
    
    await addAccount({
        channel: 'instagram',
        method: 'cookies',
        name: 'Instagram Account',
        cookies_text: cookie.trim()
    });
}

// ===== Telegram =====
// تسجيل الدخول - يظهر قسم الكود
function loginTelegram() {
    const username = document.getElementById('telegramUsername').value;
    const code = document.getElementById('telegramCode').value;
    const phone = document.getElementById('telegramPhone').value;
    
    if (!phone) {
        Swal.fire('خطأ!', 'يرجى إدخال رقم الهاتف', 'warning');
        return;
    }
    
    // إظهار قسم رمز التحقق
    document.getElementById('telegramVerificationSection').style.display = 'block';
    
    Swal.fire({
        icon: 'info',
        title: 'الخطوة التالية',
        text: 'الآن أدخل الكود واضغط إرسال الرمز',
        confirmButtonText: 'حسناً'
    });
}

// إرسال رمز التحقق
async function sendTelegramVerificationCode() {
    const username = document.getElementById('telegramUsername').value;
    const code = document.getElementById('telegramCode').value;
    const phone = document.getElementById('telegramPhone').value;
    const verificationCode = document.getElementById('telegramVerificationCode').value;
    
    if (!verificationCode) {
        Swal.fire('خطأ!', 'يرجى إدخال الكود', 'warning');
        return;
    }
    
    // إضافة الحساب
    await addAccount({
        channel: 'telegram',
        method: 'data',
        name: username || phone,
        account_uid: phone,
        data: { 
            username, 
            code, 
            phone, 
            verification_code: verificationCode 
        }
    });
}

// ===== Email =====
async function submitEmail() {
    const email = document.getElementById('emailAddress').value;
    const password = document.getElementById('emailPassword').value;
    
    if (!email || !password) {
        Swal.fire('خطأ!', 'يرجى ملء جميع الحقول', 'warning');
        return;
    }
    
    await addAccount({
        channel: 'email',
        method: 'data',
        name: email,
        account_uid: email,
        data: { email, password }
    });
}

// ===== تعديل حساب =====
async function editAccount(accountId) {
    try {
        const response = await fetch('api/accounts.php');
        const data = await response.json();
        
        if (data.success) {
            const account = data.accounts.find(acc => acc.id == accountId);
            
            if (!account) {
                Swal.fire('خطأ!', 'الحساب غير موجود', 'error');
                return;
            }
            
            // عرض نموذج التعديل حسب نوع المنصة
            const accountData = account.data ? JSON.parse(account.data) : {};
            
            const platformInfo = getPlatformInfo(account.channel);
            
            const { value: formValues } = await Swal.fire({
                title: `<i class="${platformInfo.icon}" style="font-size: 32px; margin-left: 10px;"></i> تعديل حساب ${platformInfo.name}`,
                html: `
                    <div style="text-align: right; padding: 10px;">
                        <div style="margin-bottom: 20px;">
                            <label>
                                <i class="fas fa-user" style="margin-left: 8px;"></i>
                                الاسم
                            </label>
                            <input id="edit-name" class="swal2-input" style="width: 100%; margin: 0;" value="${account.name || ''}" placeholder="أدخل الاسم">
                        </div>
                        <div style="margin-bottom: 20px;">
                            <label>
                                <i class="fas fa-id-card" style="margin-left: 8px;"></i>
                                ID
                            </label>
                            <input id="edit-uid" class="swal2-input" style="width: 100%; margin: 0;" value="${account.account_uid || ''}" placeholder="أدخل ID">
                        </div>
                        ${account.method === 'cookies' ? `
                            <div style="margin-bottom: 20px;">
                                <label>
                                    <i class="fas fa-cookie" style="margin-left: 8px;"></i>
                                    الكوكيز
                                </label>
                                <textarea id="edit-cookies" class="swal2-textarea" style="width: 100%; height: 120px; margin: 0;" placeholder="لصق الكوكيز هنا">${account.cookies_text || ''}</textarea>
                            </div>
                        ` : ''}
                        ${accountData.username !== undefined ? `
                            <div style="margin-bottom: 20px;">
                                <label>
                                    <i class="fas fa-at" style="margin-left: 8px;"></i>
                                    اسم المستخدم
                                </label>
                                <input id="edit-username" class="swal2-input" style="width: 100%; margin: 0;" value="${accountData.username || ''}" placeholder="أدخل اسم المستخدم">
                            </div>
                        ` : ''}
                        ${accountData.password !== undefined ? `
                            <div style="margin-bottom: 20px;">
                                <label>
                                    <i class="fas fa-lock" style="margin-left: 8px;"></i>
                                    كلمة السر
                                </label>
                                <input id="edit-password" type="password" class="swal2-input" style="width: 100%; margin: 0;" value="${accountData.password || ''}" placeholder="أدخل كلمة السر">
                            </div>
                        ` : ''}
                    </div>
                `,
                width: '600px',
                focusConfirm: false,
                showCancelButton: true,
                confirmButtonText: 'حفظ',
                cancelButtonText: 'إلغاء',
                preConfirm: () => {
                    const updatedData = {
                        name: document.getElementById('edit-name').value,
                        account_uid: document.getElementById('edit-uid').value
                    };
                    
                    if (account.method === 'cookies') {
                        updatedData.cookies_text = document.getElementById('edit-cookies').value;
                    }
                    
                    if (accountData.username !== undefined) {
                        updatedData.data = {
                            ...accountData,
                            username: document.getElementById('edit-username').value,
                            password: document.getElementById('edit-password')?.value || accountData.password
                        };
                    }
                    
                    return updatedData;
                }
            });
            
            if (formValues) {
                // تحديث الحساب في قاعدة البيانات
                const updateResponse = await fetch('api/accounts.php', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        account_id: accountId,
                        ...formValues
                    })
                });
                
                const updateData = await updateResponse.json();
                
                if (updateData.success) {
                    Swal.fire('نجح!', 'تم تحديث الحساب بنجاح', 'success');
                    loadAccounts();
                } else {
                    Swal.fire('خطأ!', updateData.message, 'error');
                }
            }
        }
    } catch (error) {
        console.error('خطأ:', error);
        Swal.fire('خطأ!', 'حدث خطأ في تحميل بيانات الحساب', 'error');
    }
}

// ===== التحقق من حالة الحساب =====
async function verifyAccount(accountId, status) {
    // البحث عن الحساب في القائمة
    const account = allAccounts.find(acc => acc.id === accountId);
    
    if (!account) {
        Swal.fire('خطأ!', 'لم يتم العثور على الحساب', 'error');
        return;
    }
    
    // إذا كان الحساب من نوع WhatsApp، نتحقق من حالته عبر API
    if (account.channel === 'whatsapp' && account.account_uid) {
        // عرض رسالة انتظار
        Swal.fire({
            title: 'جاري التحقق...',
            html: '<i class="fas fa-spinner fa-spin" style="font-size: 50px; color: #667eea;"></i>',
            showConfirmButton: false,
            allowOutsideClick: false
        });
        
        try {
            // التحقق من حالة الجلسة عبر API
            //const response = await fetch(`api/verify_whatsapp.php?account_uid=${account.account_uid}`);
            const response = await fetch(`api/verify_whatsapp.php?account_uid=${encodeURIComponent(account.account_uid)}`);
            const data = await response.json();
            const phone = data.phone || data.phoneNumber || '';
            const isConnected =
                              data?.success === true &&
                              (
                                data?.status_QR === 'success' ||                 // old style
                                String(data?.state || '').toLowerCase() === 'open' ||
                                String(data?.status || '').toUpperCase() === 'CONNECTED' ||
                                String(data?.connection || '').toUpperCase() === 'CONNECTED'
                              );
            
            if (isConnected) {
                // تحديث حالة الحساب في قاعدة البيانات
               
                await updateAccountStatus(accountId, 'active', phone);
                
                Swal.fire({
                    title: '<i class="fas fa-check-circle" style="color: #25D366; animation: checkPulse 1s ease-in-out infinite;"></i> تم التحقق بنجاح!',
                    html: `
                        <style>
                            @keyframes checkPulse {
                                0%, 100% { transform: scale(1); }
                                50% { transform: scale(1.1); }
                            }
                            @keyframes whatsappBounce {
                                0%, 100% { transform: scale(1) rotate(0deg); }
                                25% { transform: scale(1.1) rotate(-5deg); }
                                75% { transform: scale(1.1) rotate(5deg); }
                            }
                        </style>
                        <div style="font-size: 70px; margin: 20px 0;">
                            <i class="fab fa-whatsapp" style="color: #25D366; animation: whatsappBounce 2s ease-in-out infinite;"></i>
                        </div>
                        <p style="font-size: 18px; font-weight: 600; margin: 10px 0; color: #25D366;">
                            الحساب نشط ومتصل بنجاح!
                        </p>
                        <p style="font-size: 16px; color: #636e72;">
                            <i class="fas fa-phone" style="color: #25D366; margin-left: 8px;"></i>
                            رقم الهاتف: <strong dir="ltr">${phone}</strong>
                        </p>
                    `,
                    icon: 'success',
                    confirmButtonText: 'حسناً',
                    confirmButtonColor: '#25D366'
                });
                
                // إعادة تحميل الحسابات لتحديث الواجهة
                loadAccounts();
                
            } else {
                // الجلسة غير نشطة
                await updateAccountStatus(accountId, 'inactive');
                
                Swal.fire({
                    title: '<i class="fas fa-exclamation-triangle" style="color: #f39c12; animation: warningShake 1.5s ease-in-out infinite;"></i> الحساب غير نشط',
                    html: `
                        <style>
                            @keyframes warningShake {
                                0%, 100% { transform: rotate(0deg); }
                                10%, 30%, 50%, 70%, 90% { transform: rotate(-5deg); }
                                20%, 40%, 60%, 80% { transform: rotate(5deg); }
                            }
                            @keyframes disconnectPulse {
                                0%, 100% { transform: scale(1); opacity: 0.7; }
                                50% { transform: scale(1.15); opacity: 1; }
                            }
                        </style>
                        <div style="font-size: 70px; margin: 20px 0;">
                            <i class="fas fa-unlink" style="color: #f39c12; animation: disconnectPulse 2s ease-in-out infinite;"></i>
                        </div>
                        <p style="font-size: 18px; font-weight: 600; color: #f39c12;">
                            <i class="fas fa-times-circle" style="margin-left: 8px;"></i>
                            ${data.message || 'الجلسة غير نشطة أو منتهية الصلاحية'}
                        </p>
                        <p style="font-size: 16px; color: #636e72; margin-top: 15px;">
                            <i class="fas fa-sync-alt" style="color: #f39c12; margin-left: 8px;"></i>
                            يرجى إعادة الربط لاستخدام الحساب
                        </p>
                    `,
                    icon: 'warning',
                    confirmButtonText: 'حسناً',
                    confirmButtonColor: '#f39c12'
                });
                
                loadAccounts();
            }
            
        } catch (error) {
            console.error('خطأ في التحقق:', error);
            Swal.fire({
                title: '<i class="fas fa-exclamation-circle" style="color: #e74c3c; animation: errorShake 0.5s ease-in-out infinite;"></i> خطأ في الاتصال',
                html: `
                    <style>
                        @keyframes errorShake {
                            0%, 100% { transform: translateX(0); }
                            25% { transform: translateX(-5px); }
                            75% { transform: translateX(5px); }
                        }
                        @keyframes errorPulse {
                            0%, 100% { transform: scale(1); }
                            50% { transform: scale(1.1); }
                        }
                    </style>
                    <div style="font-size: 70px; margin: 20px 0;">
                        <i class="fas fa-wifi" style="color: #e74c3c; animation: errorPulse 2s ease-in-out infinite; transform: rotate(45deg); opacity: 0.7;"></i>
                        <i class="fas fa-slash" style="position: absolute; font-size: 80px; color: #e74c3c; margin-right: -75px; margin-top: -5px;"></i>
                    </div>
                    <p style="font-size: 18px; font-weight: 600; color: #e74c3c;">
                        <i class="fas fa-times-circle" style="margin-left: 8px;"></i>
                        حدث خطأ أثناء التحقق من حالة الحساب
                    </p>
                    <p style="font-size: 14px; color: #636e72; margin-top: 10px;">
                        <i class="fas fa-info-circle" style="color: #95a5a6; margin-left: 5px;"></i>
                        ${error.message}
                    </p>
                `,
                icon: 'error',
                confirmButtonText: 'حسناً',
                confirmButtonColor: '#e74c3c'
            });
        }
        return;
    }
    
    // للمنصات الأخرى، عرض الحالة الحالية
    let message = '';
    let icon = '';
    let iconColor = '';
    
    switch(status) {
        case 'active':
            message = 'الحساب نشط ويعمل بشكل جيد! ✅';
            icon = 'success';
            iconColor = '#27ae60';
            break;
        case 'inactive':
            message = 'الحساب غير نشط. يرجى مراجعة بيانات التسجيل. ⚠️';
            icon = 'warning';
            iconColor = '#f39c12';
            break;
        case 'closed':
            message = 'الحساب مغلق. يجب إعادة تسجيل الدخول. ❌';
            icon = 'error';
            iconColor = '#e74c3c';
            break;
        default:
            message = 'حالة الحساب: ' + getStatusLabel(status);
            icon = 'info';
            iconColor = '#3498db';
    }
    
    Swal.fire({
        title: 'حالة الحساب',
        html: `
            <style>
                @keyframes statusPulse {
                    0%, 100% { transform: scale(1); }
                    50% { transform: scale(1.15); }
                }
                @keyframes statusRotate {
                    0% { transform: rotate(0deg); }
                    100% { transform: rotate(360deg); }
                }
            </style>
            <div style="font-size: 70px; margin: 20px 0;">
                ${status === 'active' ? 
                    '<i class="fas fa-check-circle" style="color: #27ae60; animation: statusPulse 2s ease-in-out infinite;"></i>' : 
                  status === 'inactive' ? 
                    '<i class="fas fa-exclamation-triangle" style="color: #f39c12; animation: statusPulse 2s ease-in-out infinite;"></i>' : 
                    '<i class="fas fa-times-circle" style="color: #e74c3c; animation: statusPulse 2s ease-in-out infinite;"></i>'}
            </div>
            <p style="font-size: 18px; font-weight: 600; color: ${iconColor};">
                ${status === 'active' ? '<i class="fas fa-thumbs-up" style="margin-left: 8px;"></i>' : 
                  status === 'inactive' ? '<i class="fas fa-info-circle" style="margin-left: 8px;"></i>' : 
                  '<i class="fas fa-ban" style="margin-left: 8px;"></i>'}
                ${message}
            </p>
        `,
        icon: icon,
        confirmButtonText: 'حسناً',
        confirmButtonColor: iconColor
    });
}

// دالة مساعدة لتحديث حالة الحساب
async function updateAccountStatus(accountId, newStatus, phone = null) {
    try {
        const updateData = {
            account_id: accountId,
            status: newStatus
        };
        
        // إذا كان هناك رقم هاتف، نحدث الاسم أيضاً
        if (phone) {
            updateData.name = phone;
        }
        
        const response = await fetch('api/accounts.php', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(updateData)
        });
        
        const data = await response.json();
        
        if (!data.success) {
            console.error('فشل تحديث حالة الحساب:', data.message);
        }
    } catch (error) {
        console.error('خطأ في تحديث حالة الحساب:', error);
    }
}

// ===== إعادة ربط الحساب =====
async function reconnectAccount(accountId, channel) {
    // إذا كان الحساب WhatsApp، نفتح modal الQR
    if (channel === 'whatsapp') {
        // البحث عن الحساب للحصول على account_uid
        const account = allAccounts.find(acc => acc.id === accountId);
        
        if (!account || !account.account_uid) {
            Swal.fire({
                icon: 'error',
                title: 'خطأ',
                text: 'لم يتم العثور على معرف الحساب',
                confirmButtonColor: '#e74c3c'
            });
            return;
        }
        
        // تعيين currentInstanceId للمعرف الحالي
        currentInstanceId = account.account_uid;
        
        // فتح modal الواتساب
        openModal('whatsappModal');
        
        // إعادة تحميل QR code
        await reloadWhatsAppQRForReconnect(accountId);
        
    } else {
        // للمنصات الأخرى، عرض رسالة بسيطة
        const platformInfo = getPlatformInfo(channel);
        
        Swal.fire({
            position: 'center',
            icon: 'info',
            title: `إعادة ربط ${platformInfo.name}`,
            html: `
                <div style="text-align: center; padding: 10px;">
                    <i class="fas fa-sync-alt" style="font-size: 50px; color: #3498db; margin-bottom: 15px; animation: statusRotate 2s linear infinite;"></i>
                    <p style="font-size: 16px; color: #95a5a6;">يرجى مراجعة إعدادات الحساب لإعادة الربط</p>
                </div>
            `,
            showConfirmButton: true,
            confirmButtonText: 'حسناً',
            confirmButtonColor: '#3498db'
        });
    }
}

// دالة إعادة تحميل QR للإعادة الربط
async function reloadWhatsAppQRForReconnect(accountId = null) {
    try {
        // تفعيل وضع إعادة الربط
        isReconnectMode = true;
        
        // إظهار loading
        document.getElementById('whatsappQRSection').style.display = 'block';
        document.getElementById('whatsappReloadSection').style.display = 'none';
        document.querySelector('#whatsappQRSection .qr-code img').src = '';
        document.getElementById('whatsappTimer').textContent = 'جاري إعادة الاتصال...';
        
        // جلب QR Code باستخدام نفس instance_id
        await fetchQRCodeWPP(accountId);
        
        // بدء التحديث كل 15 ثانية
        if (whatsappCheckInterval) {
            clearInterval(whatsappCheckInterval);
        }
        
        whatsappCheckInterval = setInterval(() => fetchQRCodeWPP(accountId), 15000);
   
    } catch (error) {
        console.error('Error:', error);
    
        document.getElementById('whatsappQRSection').style.display = 'none';
        document.getElementById('whatsappReloadSection').style.display = 'block';
    }
}


// ===== تحميل عداد الحسابات =====
async function loadAccountLimit() {
    try {
        const response = await fetch('api/get_account_limit.php');
        const data = await response.json();
        
        if (data.success) {
            // إذا كان يوجد عنصر لعرض العداد في الصفحة
            const limitElement = document.getElementById('accountLimitInfo');
            if (limitElement) {
                const percentage = (data.current_count / data.max_count) * 100;
                const statusColor = percentage >= 100 ? '#e74c3c' : percentage >= 80 ? '#f39c12' : '#27ae60';
                
                limitElement.innerHTML = `
                    <div style="display: flex; align-items: center; gap: 10px; padding: 10px 15px; background: rgba(102, 126, 234, 0.1); border-radius: 10px; border: 1px solid rgba(102, 126, 234, 0.3);">
                        <i class="fas fa-info-circle" style="color: #667eea; font-size: 18px;"></i>
                        <span style="font-size: 14px; color: var(--text-primary); font-weight: 600;">
                            الحسابات: 
                            <strong style="color: ${statusColor};">${data.current_count}</strong> / 
                            <strong style="color: #27ae60;">${data.max_count}</strong>
                        </span>
                        ${!data.can_add ? '<span style="color: #e74c3c; font-size: 12px; font-weight: 600;">(وصلت للحد الأقصى)</span>' : ''}
                    </div>
                `;
            }
            
            // تعطيل زر الإضافة إذا وصل للحد الأقصى
            const addButton = document.querySelector('.add-account-btn');
            if (addButton && !data.can_add) {
                addButton.style.opacity = '0.6';
                addButton.style.cursor = 'not-allowed';
                addButton.onclick = function(e) {
                    e.preventDefault();
                    Swal.fire({
                        icon: 'error',
                        title: 'الحد الأقصى للحسابات',
                        html: `
                            <p>لقد وصلت للحد الأقصى لعدد الحسابات في باقتك</p>
                            <p style="margin-top: 10px;"><strong>${data.current_count} / ${data.max_count}</strong></p>
                        `,
                        confirmButtonText: 'حسناً'
                    });
                };
            }
        }
    } catch (error) {
        console.error('خطأ في تحميل عداد الحسابات:', error);
    }
}
