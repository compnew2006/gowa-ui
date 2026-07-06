<!-- Modal for Edit User Settings -->
<div class="modal" id="editSettingsModal">
    <div class="modal-content" style="max-width: 700px;">
        <div class="modal-header">
            <h2 class="modal-title">تعديل إعدادات المستخدم</h2>
            <button class="close-modal" onclick="closeSettingsModal()">&times;</button>
        </div>
        
        <div style="padding: 20px 0;">
            <!-- اسم المستخدم -->
            <div class="form-group">
                <label class="form-label">اسم المستخدم</label>
                <input type="text" class="form-input" id="settingsUserName" readonly>
                <input type="hidden" id="settingsUserId">
            </div>

            <!-- الصلاحية (مستخدم/مسؤول) -->
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-user-shield"></i> الصلاحية
                </label>
                <div style="display: flex; align-items: center; gap: 15px;">
                    <label class="toggle-switch">
                        <input type="checkbox" id="settingsIsAdmin">
                        <span class="toggle-slider"></span>
                    </label>
                    <span id="settingsAdminLabel" style="font-weight: 600; color: var(--text-primary); font-size: 16px;">مستخدم</span>
                    <button type="button" class="btn-icon btn-edit" onclick="updateUserRole()" title="تحديث الصلاحية" style="width: 45px; height: 45px; margin-right: auto;">
                        <i class="fas fa-check"></i>
                    </button>
                </div>
            </div>

            <hr style="border: 1px solid var(--border-color); margin: 25px 0;">

            <!-- المنطقة الزمنية -->
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-globe-americas"></i> المنطقة الزمنية
                </label>
                <div style="display: flex; gap: 10px;">
                    <select class="form-select" id="timezoneSelect" style="flex: 1;">
                        <option value="">اختر المنطقة الزمنية...</option>
                    </select>
                    <button type="button" class="btn-icon btn-edit" onclick="updateTimezone()" title="تحديث المنطقة الزمنية" style="width: 45px; height: 45px;">
                        <i class="fas fa-check"></i>
                    </button>
                </div>
                <small style="color: var(--text-secondary); font-size: 13px; display: block; margin-top: 5px;">
                    المنطقة الزمنية الحالية: <strong id="currentTimezone">غير محددة</strong>
                </small>
            </div>

            <hr style="border: 1px solid var(--border-color); margin: 25px 0;">

            <!-- تحويل نقود -->
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-money-bill-transfer"></i> تحويل نقود
                </label>
                <div style="display: flex; gap: 10px;">
                    <input type="number" class="form-input" id="moneyAmount" placeholder="ادخل المبلغ..." min="0" step="0.01" style="flex: 1;">
                    <button type="button" class="btn-icon btn-activate" onclick="transferMoney()" title="تحويل النقود" style="width: 45px; height: 45px;">
                        <i class="fas fa-paper-plane"></i>
                    </button>
                </div>
                <small style="color: var(--text-secondary); font-size: 13px; display: block; margin-top: 5px;">
                    <i class="fas fa-info-circle"></i> سيتم إضافة المبلغ إلى رصيد المستخدم
                </small>
            </div>

            <hr style="border: 1px solid var(--border-color); margin: 25px 0;">

            <!-- إعادة تعيين -->
            <div class="form-group">
                <label class="form-label">
                    <i class="fas fa-redo"></i> إعادة التعيين
                </label>
                <div style="display: flex; gap: 15px; flex-wrap: wrap;">
                    <button type="button" class="btn-reset btn-reset-accounts" onclick="resetAccounts()">
                        <i class="fas fa-user-slash"></i>
                        <span>إعادة تعيين الحسابات</span>
                     </button>
                    <button type="button" class="btn-reset btn-reset-messages" onclick="resetMessages()">
                        <i class="fas fa-envelope-open-text"></i>
                        <span>إعادة تعيين الرسائل</span>
                     </button>
                </div>
            </div>
        </div>

        <div class="modal-footer">
            <button type="button" class="btn-cancel" onclick="closeSettingsModal()">إغلاق</button>
        </div>
    </div>
</div>

<style>
    .btn-reset {
        background: rgba(239, 68, 68, 0.1);
        color: #ef4444;
        border: 2px solid rgba(239, 68, 68, 0.3);
        padding: 15px 20px;
        border-radius: 12px;
        cursor: pointer;
        font-weight: 600;
        font-family: 'Cairo', sans-serif;
        transition: all 0.3s ease;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        min-width: 200px;
    }

    .btn-reset i {
        font-size: 24px;
    }

    .btn-reset span {
        font-size: 15px;
    }

    .btn-reset small {
        font-size: 12px;
        opacity: 0.7;
        font-weight: 400;
    }

    .btn-reset:hover {
        background: #ef4444;
        color: white;
        border-color: #ef4444;
        transform: translateY(-3px);
        box-shadow: 0 8px 20px rgba(239, 68, 68, 0.3);
    }

    .btn-reset-accounts {
        background: rgba(59, 130, 246, 0.1);
        color: #3b82f6;
        border-color: rgba(59, 130, 246, 0.3);
    }

    .btn-reset-accounts:hover {
        background: #3b82f6;
        color: white;
        border-color: #3b82f6;
        box-shadow: 0 8px 20px rgba(59, 130, 246, 0.3);
    }

    .btn-reset-messages {
        background: rgba(251, 191, 36, 0.1);
        color: #f59e0b;
        border-color: rgba(251, 191, 36, 0.3);
    }

    .btn-reset-messages:hover {
        background: #f59e0b;
        color: white;
        border-color: #f59e0b;
        box-shadow: 0 8px 20px rgba(251, 191, 36, 0.3);
    }
</style>

<script>
// قائمة المناطق الزمنية مرتبة أبجدياً
const timezones = [
    "Africa/Abidjan",
    "Africa/Accra",
    "Africa/Addis_Ababa",
    "Africa/Algiers",
    "Africa/Cairo",
    "Africa/Casablanca",
    "Africa/Johannesburg",
    "Africa/Lagos",
    "Africa/Nairobi",
    "Africa/Tunis",
    "America/Anchorage",
    "America/Argentina/Buenos_Aires",
    "America/Bogota",
    "America/Caracas",
    "America/Chicago",
    "America/Denver",
    "America/Lima",
    "America/Los_Angeles",
    "America/Mexico_City",
    "America/New_York",
    "America/Sao_Paulo",
    "America/Toronto",
    "Asia/Baghdad",
    "Asia/Baku",
    "Asia/Bangkok",
    "Asia/Beirut",
    "Asia/Colombo",
    "Asia/Damascus",
    "Asia/Dhaka",
    "Asia/Dubai",
    "Asia/Hong_Kong",
    "Asia/Jakarta",
    "Asia/Jerusalem",
    "Asia/Karachi",
    "Asia/Kolkata",
    "Asia/Kuwait",
    "Asia/Manila",
    "Asia/Muscat",
    "Asia/Riyadh",
    "Asia/Seoul",
    "Asia/Shanghai",
    "Asia/Singapore",
    "Asia/Taipei",
    "Asia/Tehran",
    "Asia/Tokyo",
    "Atlantic/Reykjavik",
    "Australia/Melbourne",
    "Australia/Perth",
    "Australia/Sydney",
    "Europe/Amsterdam",
    "Europe/Athens",
    "Europe/Berlin",
    "Europe/Brussels",
    "Europe/Budapest",
    "Europe/Dublin",
    "Europe/Istanbul",
    "Europe/Lisbon",
    "Europe/London",
    "Europe/Madrid",
    "Europe/Moscow",
    "Europe/Paris",
    "Europe/Prague",
    "Europe/Rome",
    "Europe/Stockholm",
    "Europe/Vienna",
    "Europe/Warsaw",
    "Europe/Zurich",
    "Pacific/Auckland",
    "Pacific/Fiji",
    "Pacific/Honolulu",
    "UTC"
].sort();

// تحميل المناطق الزمنية عند فتح الصفحة
document.addEventListener('DOMContentLoaded', function() {
    const select = document.getElementById('timezoneSelect');
    timezones.forEach(tz => {
        const option = document.createElement('option');
        option.value = tz;
        option.textContent = tz.replace(/_/g, ' ');
        select.appendChild(option);
    });
});

// فتح مودال الإعدادات
function openSettingsModal(user) {
    document.getElementById('settingsUserId').value = user.id;
    document.getElementById('settingsUserName').value = `${user.first_name} ${user.last_name}`;
    
    // تعيين الصلاحية
    const isAdmin = user.is_admin == 1;
    document.getElementById('settingsIsAdmin').checked = isAdmin;
    document.getElementById('settingsAdminLabel').textContent = isAdmin ? 'مسؤول' : 'مستخدم';
    
    // تعيين المنطقة الزمنية
    document.getElementById('timezoneSelect').value = user.timezone || '';
    document.getElementById('currentTimezone').textContent = user.timezone || 'غير محددة';
    
    document.getElementById('editSettingsModal').classList.add('active');
}

// تحديث العنوان عند تغيير Toggle
document.addEventListener('DOMContentLoaded', function() {
    const toggleAdmin = document.getElementById('settingsIsAdmin');
    if (toggleAdmin) {
        toggleAdmin.addEventListener('change', function() {
            document.getElementById('settingsAdminLabel').textContent = this.checked ? 'مسؤول' : 'مستخدم';
        });
    }
});

// إغلاق المودال
function closeSettingsModal() {
    document.getElementById('editSettingsModal').classList.remove('active');
}

// تحويل نقود للمستخدم
function transferMoney() {
    const userId = document.getElementById('settingsUserId').value;
    const userName = document.getElementById('settingsUserName').value;
    const amount = document.getElementById('moneyAmount').value;

    if (!amount || amount <= 0) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'يرجى إدخال مبلغ صحيح',
            confirmButtonText: 'حسناً'
        });
        return;
    }

    Swal.fire({
        title: 'هل أنت متأكد؟',
        html: `سيتم تحويل <strong style="color: #10b981;">${amount} جنيه</strong> إلى حساب <strong>${userName}</strong>`,
        icon: 'question',
        showCancelButton: true,
        confirmButtonColor: '#10b981',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، حوّل',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('transfer_money.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `user_id=${userId}&amount=${amount}`
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم التحويل!',
                        text: data.message,
                        timer: 2000,
                        showConfirmButton: false
                    });
                    document.getElementById('moneyAmount').value = '';
                    loadUsers();
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        confirmButtonText: 'حسناً'
                    });
                }
            })
            .catch(error => {
                console.error('Error:', error);
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ في الاتصال',
                    text: 'حدث خطأ أثناء التحويل',
                    confirmButtonText: 'حسناً'
                });
            });
        }
    });
}

// تحديث الصلاحية (مستخدم/مسؤول)
function updateUserRole() {
    const userId = document.getElementById('settingsUserId').value;
    const isAdmin = document.getElementById('settingsIsAdmin').checked ? 1 : 0;
    const roleText = isAdmin ? 'مسؤول' : 'مستخدم';

    Swal.fire({
        title: 'هل أنت متأكد؟',
        text: `سيتم تغيير الصلاحية إلى: ${roleText}`,
        icon: 'question',
        showCancelButton: true,
        confirmButtonColor: '#3b82f6',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، حدّث',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('update_user_role.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `user_id=${userId}&is_admin=${isAdmin}`
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم التحديث!',
                        text: 'تم تحديث الصلاحية بنجاح',
                        timer: 2000,
                        showConfirmButton: false
                    });
                    loadUsers();
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        confirmButtonText: 'حسناً'
                    });
                }
            })
            .catch(error => {
                console.error('Error:', error);
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ في الاتصال',
                    text: 'حدث خطأ أثناء التحديث',
                    confirmButtonText: 'حسناً'
                });
            });
        }
    });
}

// تحديث المنطقة الزمنية
function updateTimezone() {
    const userId = document.getElementById('settingsUserId').value;
    const timezone = document.getElementById('timezoneSelect').value;
    
    if (!timezone) {
        Swal.fire({
            icon: 'warning',
            title: 'تنبيه',
            text: 'يرجى اختيار منطقة زمنية',
            confirmButtonText: 'حسناً'
        });
        return;
    }

    Swal.fire({
        title: 'هل أنت متأكد؟',
        text: `سيتم تحديث المنطقة الزمنية إلى: ${timezone}`,
        icon: 'question',
        showCancelButton: true,
        confirmButtonColor: '#3b82f6',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، حدّث',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('update_user_timezone.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `user_id=${userId}&timezone=${encodeURIComponent(timezone)}`
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم التحديث!',
                        text: 'تم تحديث المنطقة الزمنية بنجاح',
                        timer: 2000,
                        showConfirmButton: false
                    });
                    document.getElementById('currentTimezone').textContent = timezone;
                    loadUsers();
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        confirmButtonText: 'حسناً'
                    });
                }
            })
            .catch(error => {
                console.error('Error:', error);
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ في الاتصال',
                    text: 'حدث خطأ أثناء التحديث',
                    confirmButtonText: 'حسناً'
                });
            });
        }
    });
}

// إعادة تعيين الحسابات
function resetAccounts() {
    const userId = document.getElementById('settingsUserId').value;
    const userName = document.getElementById('settingsUserName').value;

    Swal.fire({
        title: 'هل أنت متأكد؟',
        html: `سيتم إعادة تعيين عدد الحسابات للمستخدم <strong>${userName}</strong><br><span style="color: #3b82f6; font-weight: 600;">account = 0</span>`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#3b82f6',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، أعد التعيين',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('reset_user_count.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `user_id=${userId}&type=accounts`
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم إعادة التعيين!',
                        text: 'تم إعادة تعيين عدد الحسابات بنجاح',
                        timer: 2000,
                        showConfirmButton: false
                    });
                    loadUsers();
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        confirmButtonText: 'حسناً'
                    });
                }
            })
            .catch(error => {
                console.error('Error:', error);
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ في الاتصال',
                    text: 'حدث خطأ أثناء إعادة التعيين',
                    confirmButtonText: 'حسناً'
                });
            });
        }
    });
}

// إعادة تعيين الرسائل
function resetMessages() {
    const userId = document.getElementById('settingsUserId').value;
    const userName = document.getElementById('settingsUserName').value;

    Swal.fire({
        title: 'هل أنت متأكد؟',
        html: `سيتم إعادة تعيين عدد الرسائل للمستخدم <strong>${userName}</strong><br><span style="color: #f59e0b; font-weight: 600;">Messages = 0</span>`,
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#f59e0b',
        cancelButtonColor: '#6b7280',
        confirmButtonText: 'نعم، أعد التعيين',
        cancelButtonText: 'إلغاء'
    }).then((result) => {
        if (result.isConfirmed) {
            fetch('reset_user_count.php', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `user_id=${userId}&type=messages`
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    Swal.fire({
                        icon: 'success',
                        title: 'تم إعادة التعيين!',
                        text: 'تم إعادة تعيين عدد الرسائل بنجاح',
                        timer: 2000,
                        showConfirmButton: false
                    });
                    loadUsers();
                } else {
                    Swal.fire({
                        icon: 'error',
                        title: 'خطأ',
                        text: data.message,
                        confirmButtonText: 'حسناً'
                    });
                }
            })
            .catch(error => {
                console.error('Error:', error);
                Swal.fire({
                    icon: 'error',
                    title: 'خطأ في الاتصال',
                    text: 'حدث خطأ أثناء إعادة التعيين',
                    confirmButtonText: 'حسناً'
                });
            });
        }
    });
}
</script>
