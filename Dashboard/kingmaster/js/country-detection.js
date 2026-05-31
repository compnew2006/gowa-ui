/**
 * King Master Dashboard - Country Detection from Phone Number
 * نظام اكتشاف الدولة من رقم الهاتف
 */

// قاعدة بيانات رموز الدول الهاتفية
const phoneCountryCodes = {
    // الشرق الأوسط وأفريقيا
    '20': { country: 'مصر', countryEn: 'Egypt', countryFr: 'Égypte', flag: '🇪🇬', timezone: 'UTC+02:00' },
    '966': { country: 'السعودية', countryEn: 'Saudi Arabia', countryFr: 'Arabie Saoudite', flag: '🇸🇦', timezone: 'UTC+03:00' },
    '971': { country: 'الإمارات', countryEn: 'United Arab Emirates', countryFr: 'Émirats arabes unis', flag: '🇦🇪', timezone: 'UTC+04:00' },
    '965': { country: 'الكويت', countryEn: 'Kuwait', countryFr: 'Koweït', flag: '🇰🇼', timezone: 'UTC+03:00' },
    '974': { country: 'قطر', countryEn: 'Qatar', countryFr: 'Qatar', flag: '🇶🇦', timezone: 'UTC+03:00' },
    '973': { country: 'البحرين', countryEn: 'Bahrain', countryFr: 'Bahreïn', flag: '🇧🇭', timezone: 'UTC+03:00' },
    '968': { country: 'عمان', countryEn: 'Oman', countryFr: 'Oman', flag: '🇴🇲', timezone: 'UTC+04:00' },
    '962': { country: 'الأردن', countryEn: 'Jordan', countryFr: 'Jordanie', flag: '🇯🇴', timezone: 'UTC+03:00' },
    '961': { country: 'لبنان', countryEn: 'Lebanon', countryFr: 'Liban', flag: '🇱🇧', timezone: 'UTC+02:00' },
    '963': { country: 'سوريا', countryEn: 'Syria', countryFr: 'Syrie', flag: '🇸🇾', timezone: 'UTC+03:00' },
    '964': { country: 'العراق', countryEn: 'Iraq', countryFr: 'Irak', flag: '🇮🇶', timezone: 'UTC+03:00' },
    '98': { country: 'إيران', countryEn: 'Iran', countryFr: 'Iran', flag: '🇮🇷', timezone: 'UTC+03:30' },
    '90': { country: 'تركيا', countryEn: 'Turkey', countryFr: 'Turquie', flag: '🇹🇷', timezone: 'UTC+03:00' },
    
    // شمال أفريقيا
    '212': { country: 'المغرب', countryEn: 'Morocco', countryFr: 'Maroc', flag: '🇲🇦', timezone: 'UTC+01:00' },
    '213': { country: 'الجزائر', countryEn: 'Algeria', countryFr: 'Algérie', flag: '🇩🇿', timezone: 'UTC+01:00' },
    '216': { country: 'تونس', countryEn: 'Tunisia', countryFr: 'Tunisie', flag: '🇹🇳', timezone: 'UTC+01:00' },
    '218': { country: 'ليبيا', countryEn: 'Libya', countryFr: 'Libye', flag: '🇱🇾', timezone: 'UTC+02:00' },
    '249': { country: 'السودان', countryEn: 'Sudan', countryFr: 'Soudan', flag: '🇸🇩', timezone: 'UTC+02:00' },
    
    // أوروبا
    '33': { country: 'فرنسا', countryEn: 'France', countryFr: 'France', flag: '🇫🇷', timezone: 'UTC+01:00' },
    '49': { country: 'ألمانيا', countryEn: 'Germany', countryFr: 'Allemagne', flag: '🇩🇪', timezone: 'UTC+01:00' },
    '44': { country: 'المملكة المتحدة', countryEn: 'United Kingdom', countryFr: 'Royaume-Uni', flag: '🇬🇧', timezone: 'UTC+00:00' },
    '39': { country: 'إيطاليا', countryEn: 'Italy', countryFr: 'Italie', flag: '🇮🇹', timezone: 'UTC+01:00' },
    '34': { country: 'إسبانيا', countryEn: 'Spain', countryFr: 'Espagne', flag: '🇪🇸', timezone: 'UTC+01:00' },
    '7': { country: 'روسيا', countryEn: 'Russia', countryFr: 'Russie', flag: '🇷🇺', timezone: 'UTC+03:00' },
    
    // أمريكا الشمالية
    '1': { country: 'الولايات المتحدة/كندا', countryEn: 'USA/Canada', countryFr: 'États-Unis/Canada', flag: '🇺🇸🇨🇦', timezone: 'UTC-05:00' },
    
    // آسيا
    '86': { country: 'الصين', countryEn: 'China', countryFr: 'Chine', flag: '🇨🇳', timezone: 'UTC+08:00' },
    '91': { country: 'الهند', countryEn: 'India', countryFr: 'Inde', flag: '🇮🇳', timezone: 'UTC+05:30' },
    '81': { country: 'اليابان', countryEn: 'Japan', countryFr: 'Japon', flag: '🇯🇵', timezone: 'UTC+09:00' },
    '82': { country: 'كوريا الجنوبية', countryEn: 'South Korea', countryFr: 'Corée du Sud', flag: '🇰🇷', timezone: 'UTC+09:00' },
    '92': { country: 'باكستان', countryEn: 'Pakistan', countryFr: 'Pakistan', flag: '🇵🇰', timezone: 'UTC+05:00' },
    '880': { country: 'بنغلاديش', countryEn: 'Bangladesh', countryFr: 'Bangladesh', flag: '🇧🇩', timezone: 'UTC+06:00' },
    '60': { country: 'ماليزيا', countryEn: 'Malaysia', countryFr: 'Malaisie', flag: '🇲🇾', timezone: 'UTC+08:00' },
    '65': { country: 'سنغافورة', countryEn: 'Singapore', countryFr: 'Singapour', flag: '🇸🇬', timezone: 'UTC+08:00' },
    '66': { country: 'تايلاند', countryEn: 'Thailand', countryFr: 'Thaïlande', flag: '🇹🇭', timezone: 'UTC+07:00' },
    '84': { country: 'فيتنام', countryEn: 'Vietnam', countryFr: 'Vietnam', flag: '🇻🇳', timezone: 'UTC+07:00' },
    
    // أستراليا وأوقيانوسيا
    '61': { country: 'أستراليا', countryEn: 'Australia', countryFr: 'Australie', flag: '🇦🇺', timezone: 'UTC+10:00' },
    '64': { country: 'نيوزيلندا', countryEn: 'New Zealand', countryFr: 'Nouvelle-Zélande', flag: '🇳🇿', timezone: 'UTC+12:00' },
    
    // أمريكا الجنوبية
    '55': { country: 'البرازيل', countryEn: 'Brazil', countryFr: 'Brésil', flag: '🇧🇷', timezone: 'UTC-03:00' },
    '54': { country: 'الأرجنتين', countryEn: 'Argentina', countryFr: 'Argentine', flag: '🇦🇷', timezone: 'UTC-03:00' },
    '57': { country: 'كولومبيا', countryEn: 'Colombia', countryFr: 'Colombie', flag: '🇨🇴', timezone: 'UTC-05:00' },
    '56': { country: 'تشيلي', countryEn: 'Chile', countryFr: 'Chili', flag: '🇨🇱', timezone: 'UTC-03:00' }
};

// دالة استخراج رمز الدولة من رقم الهاتف
function extractCountryCodeFromPhone(phoneNumber) {
    // تنظيف رقم الهاتف من الرموز والمسافات
    const cleanPhone = phoneNumber.replace(/[^\d]/g, '');
    
    // إزالة أي أصفار أولية أو علامات +
    let normalizedPhone = cleanPhone.replace(/^00/, '').replace(/^\+/, '');
    
    // البحث عن أطول رمز دولة مطابق (ابتداءً من الأطول)
    const sortedCodes = Object.keys(phoneCountryCodes).sort((a, b) => b.length - a.length);
    
    for (const code of sortedCodes) {
        if (normalizedPhone.startsWith(code)) {
            return {
                countryCode: code,
                countryInfo: phoneCountryCodes[code],
                remainingNumber: normalizedPhone.substring(code.length)
            };
        }
    }
    
    return null;
}

// دالة اكتشاف الدولة من رقم الهاتف
function detectCountryFromPhone(phoneNumber, language = 'ar') {
    const result = extractCountryCodeFromPhone(phoneNumber);
    
    if (!result) {
        return {
            success: false,
            message: t('country_not_detected', language),
            countryCode: null,
            country: null,
            timezone: null,
            flag: null
        };
    }
    
    const { countryCode, countryInfo } = result;
    let countryName;
    
    switch (language) {
        case 'en':
            countryName = countryInfo.countryEn;
            break;
        case 'fr':
            countryName = countryInfo.countryFr;
            break;
        case 'ar':
        default:
            countryName = countryInfo.country;
            break;
    }
    
    return {
        success: true,
        message: t('country_detected', language),
        countryCode: '+' + countryCode,
        country: countryName,
        timezone: countryInfo.timezone,
        flag: countryInfo.flag,
        fullInfo: countryInfo
    };
}

// دالة التحقق من صحة تنسيق رقم الهاتف
function validatePhoneNumber(phoneNumber) {
    // تنظيف رقم الهاتف
    const cleanPhone = phoneNumber.replace(/[^\d]/g, '');
    
    // يجب أن يكون طول الرقم بين 7 و 15 رقماً (معايير ITU-T E.164)
    if (cleanPhone.length < 7 || cleanPhone.length > 15) {
        return false;
    }
    
    // محاولة استخراج رمز الدولة
    const result = extractCountryCodeFromPhone(phoneNumber);
    return result !== null;
}

// دالة تنسيق رقم الهاتف
function formatPhoneNumber(phoneNumber, countryCode) {
    const cleanPhone = phoneNumber.replace(/[^\d]/g, '');
    const result = extractCountryCodeFromPhone(cleanPhone);
    
    if (!result) {
        return phoneNumber;
    }
    
    const { remainingNumber } = result;
    return `+${result.countryCode} ${remainingNumber}`;
}

// دالة ربط اكتشاف الدولة مع حقل رقم الهاتف
function setupPhoneCountryDetection(phoneInputId, countryInputId, timezoneSelectId = null) {
    const phoneInput = document.getElementById(phoneInputId);
    const countryInput = document.getElementById(countryInputId);
    
    if (!phoneInput || !countryInput) {
        console.error('Phone or country input elements not found');
        return;
    }
    
    // إضافة مستمع للأحداث
    phoneInput.addEventListener('input', function(e) {
        const phoneNumber = e.target.value;
        
        // مسح الحقل إذا كان فارغاً
        if (!phoneNumber.trim()) {
            countryInput.value = '';
            countryInput.placeholder = t('auto_detect_country');
            return;
        }
        
        // إظهار رسالة "جاري الاكتشاف"
        countryInput.placeholder = t('detecting_country');
        
        // تأخير صغير لتجنب الاستدعاءات المتكررة
        clearTimeout(phoneInput.detectTimeout);
        phoneInput.detectTimeout = setTimeout(() => {
            const currentLanguage = getCurrentLanguage();
            const detection = detectCountryFromPhone(phoneNumber, currentLanguage);
            
            if (detection.success) {
                countryInput.value = `${detection.flag} ${detection.country}`;
                countryInput.style.color = 'var(--success)';
                
                // تحديث المنطقة الزمنية إذا كان العنصر متاحاً
                if (timezoneSelectId && detection.timezone) {
                    const timezoneSelect = document.getElementById(timezoneSelectId);
                    if (timezoneSelect) {
                        const timezoneOption = timezoneSelect.querySelector(`option[value="${detection.timezone}"]`);
                        if (timezoneOption) {
                            timezoneOption.selected = true;
                        }
                    }
                }
            } else {
                countryInput.value = '';
                countryInput.placeholder = t('country_not_detected');
                countryInput.style.color = 'var(--danger)';
            }
        }, 300);
    });
    
    // التحقق من الرقم عند فقدان التركيز
    phoneInput.addEventListener('blur', function(e) {
        const phoneNumber = e.target.value;
        
        if (phoneNumber.trim() && !validatePhoneNumber(phoneNumber)) {
            e.target.style.borderColor = 'var(--danger)';
            // يمكن إضافة رسالة خطأ هنا
        } else {
            e.target.style.borderColor = '';
        }
    });
}

// تصدير الدوال للاستخدام العام
if (typeof window !== 'undefined') {
    window.detectCountryFromPhone = detectCountryFromPhone;
    window.setupPhoneCountryDetection = setupPhoneCountryDetection;
    window.validatePhoneNumber = validatePhoneNumber;
    window.formatPhoneNumber = formatPhoneNumber;
    window.extractCountryCodeFromPhone = extractCountryCodeFromPhone;
}