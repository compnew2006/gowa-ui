/**
 * King Master Dashboard - Timezones Utility
 * دالة المناطق الزمنية لجميع أنحاء العالم
 */

// قاعدة بيانات المناطق الزمنية متعددة اللغات
const timezonesDatabase = {
    'UTC-12:00': { ar: 'خط التاريخ الدولي الغربي', en: 'International Date Line West', fr: 'Ligne de changement de date internationale ouest' },
    'UTC-11:00': { ar: 'جزر ساموا', en: 'Samoa', fr: 'Samoa' },
    'UTC-10:00': { ar: 'هاواي', en: 'Hawaii', fr: 'Hawaï' },
    'UTC-09:30': { ar: 'جزر ماركيساس', en: 'Marquesas Islands', fr: 'Îles Marquises' },
    'UTC-09:00': { ar: 'ألاسكا', en: 'Alaska', fr: 'Alaska' },
    'UTC-08:00': { ar: 'المحيط الهادئ (الولايات المتحدة وكندا)', en: 'Pacific Time (US & Canada)', fr: 'Heure du Pacifique (États-Unis et Canada)' },
    'UTC-07:00': { ar: 'الجبلية (الولايات المتحدة وكندا)', en: 'Mountain Time (US & Canada)', fr: 'Heure des Rocheuses (États-Unis et Canada)' },
    'UTC-06:00': { ar: 'الوسطى (الولايات المتحدة وكندا)', en: 'Central Time (US & Canada)', fr: 'Heure centrale (États-Unis et Canada)' },
    'UTC-05:00': { ar: 'الشرقية (الولايات المتحدة وكندا)', en: 'Eastern Time (US & Canada)', fr: 'Heure de l\'Est (États-Unis et Canada)' },
    'UTC-04:00': { ar: 'الأطلسي (كندا)', en: 'Atlantic Time (Canada)', fr: 'Heure de l\'Atlantique (Canada)' },
    'UTC-03:30': { ar: 'نيوفاوندلاند', en: 'Newfoundland', fr: 'Terre-Neuve' },
    'UTC-03:00': { ar: 'البرازيل، الأرجنتين', en: 'Brazil, Argentina', fr: 'Brésil, Argentine' },
    'UTC-02:00': { ar: 'جرينلاند', en: 'Greenland', fr: 'Groenland' },
    'UTC-01:00': { ar: 'جزر الأزور', en: 'Azores', fr: 'Açores' },
    'UTC+00:00': { ar: 'توقيت غرينتش - لندن، دبلن', en: 'GMT - London, Dublin', fr: 'GMT - Londres, Dublin' },
    'UTC+01:00': { ar: 'أوروبا الوسطى - باريس، برلين', en: 'Central Europe - Paris, Berlin', fr: 'Europe centrale - Paris, Berlin' },
    'UTC+02:00': { ar: 'أوروبا الشرقية - القاهرة، هلسنكي', en: 'Eastern Europe - Cairo, Helsinki', fr: 'Europe de l\'Est - Le Caire, Helsinki' },
    'UTC+03:00': { ar: 'موسكو، الرياض، الكويت', en: 'Moscow, Riyadh, Kuwait', fr: 'Moscou, Riyad, Koweït' },
    'UTC+03:30': { ar: 'إيران - طهران', en: 'Iran - Tehran', fr: 'Iran - Téhéran' },
    'UTC+04:00': { ar: 'الإمارات، سلطنة عمان', en: 'UAE, Oman', fr: 'EAU, Oman' },
    'UTC+04:30': { ar: 'أفغانستان', en: 'Afghanistan', fr: 'Afghanistan' },
    'UTC+05:00': { ar: 'باكستان، كازاخستان', en: 'Pakistan, Kazakhstan', fr: 'Pakistan, Kazakhstan' },
    'UTC+05:30': { ar: 'الهند، سريلانكا', en: 'India, Sri Lanka', fr: 'Inde, Sri Lanka' },
    'UTC+05:45': { ar: 'نيبال', en: 'Nepal', fr: 'Népal' },
    'UTC+06:00': { ar: 'بنغلاديش، بوتان', en: 'Bangladesh, Bhutan', fr: 'Bangladesh, Bhoutan' },
    'UTC+06:30': { ar: 'جزر كوكوس', en: 'Cocos Islands', fr: 'Îles Cocos' },
    'UTC+07:00': { ar: 'تايلاند، فيتنام', en: 'Thailand, Vietnam', fr: 'Thaïlande, Vietnam' },
    'UTC+08:00': { ar: 'الصين، ماليزيا، سنغافورة', en: 'China, Malaysia, Singapore', fr: 'Chine, Malaisie, Singapour' },
    'UTC+08:45': { ar: 'أستراليا الغربية', en: 'Western Australia', fr: 'Australie-Occidentale' },
    'UTC+09:00': { ar: 'اليابان، كوريا الجنوبية', en: 'Japan, South Korea', fr: 'Japon, Corée du Sud' },
    'UTC+09:30': { ar: 'أستراليا الوسطى', en: 'Central Australia', fr: 'Australie centrale' },
    'UTC+10:00': { ar: 'أستراليا الشرقية', en: 'Eastern Australia', fr: 'Australie orientale' },
    'UTC+10:30': { ar: 'جزيرة لورد هاو', en: 'Lord Howe Island', fr: 'Île Lord Howe' },
    'UTC+11:00': { ar: 'جزر سليمان', en: 'Solomon Islands', fr: 'Îles Salomon' },
    'UTC+11:30': { ar: 'جزيرة نورفولك', en: 'Norfolk Island', fr: 'Île Norfolk' },
    'UTC+12:00': { ar: 'نيوزيلندا', en: 'New Zealand', fr: 'Nouvelle-Zélande' },
    'UTC+12:45': { ar: 'جزر شاتام', en: 'Chatham Islands', fr: 'Îles Chatham' },
    'UTC+13:00': { ar: 'تونغا', en: 'Tonga', fr: 'Tonga' },
    'UTC+14:00': { ar: 'جزر لاين', en: 'Line Islands', fr: 'Îles de la Ligne' }
};

// دالة إرجاع جميع المناطق الزمنية بناءً على اللغة
function getWorldTimezones(language = 'ar') {
    const selectText = {
        ar: 'اختر المنطقة الزمنية',
        en: 'Select timezone',
        fr: 'Sélectionner le fuseau horaire'
    };
    
    const timezones = [{ value: '', text: selectText[language] || selectText.ar, disabled: true }];
    
    Object.keys(timezonesDatabase).forEach(utc => {
        const regionText = timezonesDatabase[utc][language] || timezonesDatabase[utc].ar;
        timezones.push({
            value: utc,
            text: `(${utc}) ${regionText}`,
            searchText: `${utc} ${regionText}`.toLowerCase()
        });
    });
    
    return timezones;
}

// دالة إنشاء خيارات المناطق الزمنية في عنصر select
function populateTimezoneSelect(selectId, options = {}) {
    const selectElement = document.getElementById(selectId);
    if (!selectElement) {
        console.error(`Element with ID '${selectId}' not found`);
        return;
    }

    const timezones = getWorldTimezones();
    const defaultSelected = options.defaultSelected || 'UTC+03:00';
    const autoDetect = options.autoDetect !== false; // Default true

    // مسح الخيارات الموجودة
    selectElement.innerHTML = '';

    // إضافة كل منطقة زمنية
    timezones.forEach(timezone => {
        const option = document.createElement('option');
        option.value = timezone.value;
        option.textContent = timezone.text;

        if (timezone.disabled) {
            option.disabled = true;
        }

        if (timezone.selected || timezone.value === defaultSelected) {
            option.selected = true;
        }

        selectElement.appendChild(option);
    });

    // اكتشاف المنطقة الزمنية تلقائياً
    if (autoDetect) {
        autoDetectUserTimezone(selectId);
    }
}

// دالة اكتشاف المنطقة الزمنية للمستخدم تلقائياً
function autoDetectUserTimezone(selectId) {
    try {
        const selectElement = document.getElementById(selectId);
        if (!selectElement) return;

        const timezoneOffset = new Date().getTimezoneOffset() / -60;
        const offsetString = (timezoneOffset >= 0 ? '+' : '') + 
            String(Math.floor(Math.abs(timezoneOffset))).padStart(2, '0') + ':' + 
            String(Math.abs(timezoneOffset) % 1 * 60).padStart(2, '0');
        
        const option = selectElement.querySelector(`option[value="UTC${offsetString}"]`);
        if (option) {
            option.selected = true;
        }
    } catch (error) {
        console.log('Could not auto-detect timezone:', error);
    }
}

// دالة الحصول على معلومات المنطقة الزمنية بواسطة القيمة
function getTimezoneInfo(utcValue) {
    const timezones = getWorldTimezones();
    return timezones.find(tz => tz.value === utcValue);
}

// دالة تحويل الوقت بين المناطق الزمنية
function convertTime(date, fromTimezone, toTimezone) {
    try {
        // استخراج الإزاحة من النص
        const parseOffset = (tzString) => {
            const match = tzString.match(/UTC([+-])(\d{2}):(\d{2})/);
            if (!match) return 0;
            const sign = match[1] === '+' ? 1 : -1;
            const hours = parseInt(match[2]);
            const minutes = parseInt(match[3]);
            return sign * (hours * 60 + minutes);
        };

        const fromOffset = parseOffset(fromTimezone);
        const toOffset = parseOffset(toTimezone);
        
        const utcTime = new Date(date.getTime() - fromOffset * 60000);
        const targetTime = new Date(utcTime.getTime() + toOffset * 60000);
        
        return targetTime;
    } catch (error) {
        console.error('Error converting time:', error);
        return date;
    }
}

// تصدير الدوال للاستخدام في ملفات أخرى
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        getWorldTimezones,
        populateTimezoneSelect,
        autoDetectUserTimezone,
        getTimezoneInfo,
        convertTime
    };
}