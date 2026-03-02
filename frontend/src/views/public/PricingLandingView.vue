<script setup lang="ts">
import { computed, markRaw, ref } from 'vue'
import type { Component } from 'vue'
import {
  ArrowRight,
  BarChart3,
  Bot,
  Check,
  Clock,
  Database,
  FileText,
  Globe,
  HardDrive,
  Key,
  Layers,
  Link2,
  Lock,
  Megaphone,
  MessageSquare,
  Server,
  Settings,
  Shield,
  Sparkles,
  Star,
  Tags,
  Users,
  Workflow,
  Zap,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import ThemeSwitcher from '@/components/layout/ThemeSwitcher.vue'

type Lang = 'en' | 'ar'
type BiText = { en: string; ar: string }

type NavItem = {
  href: string
  label: BiText
}

type StatItem = {
  value: string
  label: BiText
  hint: BiText
}

type FeatureGroup = {
  icon: Component
  title: BiText
  summary: BiText
  bullets: BiText[]
}

type PlanItem = {
  key: string
  badge?: BiText
  highlight?: boolean
  title: BiText
  subtitle: BiText
  price: string
  unit: BiText
  setupFee: BiText
  target: BiText
  features: BiText[]
  cta: BiText
}

type ComparisonRow = {
  label: BiText
  values: [BiText, BiText, BiText, BiText]
}

type StepItem = {
  icon: Component
  title: BiText
  body: BiText
}

type FaqItem = {
  q: BiText
  a: BiText
}

const initialLang: Lang = typeof navigator !== 'undefined' && navigator.language.toLowerCase().startsWith('ar')
  ? 'ar'
  : 'en'
const language = ref<Lang>(initialLang)

const isArabic = computed(() => language.value === 'ar')
const pageDir = computed(() => (isArabic.value ? 'rtl' : 'ltr'))
const year = new Date().getFullYear()

const text = (value: BiText) => (isArabic.value ? value.ar : value.en)

const ui = {
  brand: { en: 'Whatomate', ar: 'وتوميت' },
  navFeatures: { en: 'Features', ar: 'المزايا' },
  navPlans: { en: 'Plans', ar: 'الباقات' },
  navCompare: { en: 'Compare', ar: 'المقارنة' },
  navFaq: { en: 'FAQ', ar: 'الأسئلة' },
  langLabel: { en: 'Language', ar: 'اللغة' },
  themeLabel: { en: 'Theme', ar: 'المظهر' },
  heroBadge: {
    en: 'Arabic + English WhatsApp Operations Platform',
    ar: 'منصة تشغيل واتساب بالعربية والإنجليزية',
  },
  heroTitle: {
    en: 'Sell a complete WhatsApp business system, not just chat replies.',
    ar: 'بع نظامًا كاملًا لإدارة واتساب للأعمال، وليس مجرد ردود دردشة.',
  },
  heroSubtitle: {
    en: 'Show clients one page that explains the full value: inbox, team management, automation, campaigns, analytics, security, and dedicated deployments.',
    ar: 'اعرض على العميل صفحة واحدة تشرح القيمة كاملة: صندوق الوارد، إدارة الفريق، الأتمتة، الحملات، التحليلات، الأمان، والبيئات المخصصة.',
  },
  primaryCta: { en: 'Start Trial / Demo', ar: 'ابدأ تجربة / عرضًا' },
  secondaryCta: { en: 'See Pricing Plans', ar: 'شاهد الباقات والأسعار' },
  tertiaryCta: { en: 'Login', ar: 'تسجيل الدخول' },
  heroFootnote: {
    en: 'Suggested prices below are optimized for agency-style reselling and managed onboarding.',
    ar: 'الأسعار المقترحة أدناه مناسبة للبيع كخدمة مُدارة مع تهيئة وتشغيل للعملاء.',
  },
  painLabel: { en: 'Why clients buy', ar: 'لماذا يشتري العميل' },
  painTitle: {
    en: 'You are selling speed, visibility, and control over customer conversations.',
    ar: 'أنت تبيع السرعة والوضوح والسيطرة على محادثات العملاء.',
  },
  painBody: {
    en: 'Clients usually lose leads because messages are missed, agents reply inconsistently, and there is no reporting. Whatomate fixes that with one operational system.',
    ar: 'غالبًا يخسر العملاء الفرص لأن الرسائل تضيع، والردود غير موحدة، ولا توجد تقارير واضحة. وتوميت يعالج ذلك بنظام تشغيل واحد.',
  },
  featuresBadge: { en: 'Feature Coverage', ar: 'تغطية المزايا' },
  featuresTitle: {
    en: 'Everything your client expects in a modern WhatsApp operations platform',
    ar: 'كل ما يتوقعه العميل في منصة حديثة لتشغيل واتساب للأعمال',
  },
  featuresSubtitle: {
    en: 'Use this page in sales calls to explain scope clearly and justify pricing.',
    ar: 'استخدم هذه الصفحة في مكالمات البيع لشرح النطاق بوضوح وتبرير السعر.',
  },
  inventoryTitle: { en: 'Feature Inventory (quick scan)', ar: 'قائمة المزايا (نظرة سريعة)' },
  plansBadge: { en: 'Suggested Pricing', ar: 'أسعار مقترحة' },
  plansTitle: {
    en: 'Suggested plans you can sell today',
    ar: 'باقات مقترحة يمكنك بيعها اليوم',
  },
  plansSubtitle: {
    en: 'Position “Dedicated Business” as the main offer for serious clients. Use Starter only to reduce objections.',
    ar: 'قدّم “البيئة الخاصة للأعمال” كالعرض الرئيسي للعملاء الجادين. واستخدم باقة البداية فقط لتقليل الاعتراضات.',
  },
  monthlySuffix: { en: '/ month', ar: '/ شهريًا' },
  setupFeeLabel: { en: 'Setup', ar: 'رسوم التهيئة' },
  compareBadge: { en: 'Plan Comparison', ar: 'مقارنة الباقات' },
  compareTitle: { en: 'What changes between plans', ar: 'ما الذي يختلف بين الباقات' },
  compareSubtitle: {
    en: 'Keep the conversation simple: start with use case, then show the right tier.',
    ar: 'اجعل الحديث بسيطًا: ابدأ بحالة الاستخدام ثم اعرض المستوى المناسب.',
  },
  onboardingBadge: { en: 'Sales Delivery Flow', ar: 'خط سير التسليم' },
  onboardingTitle: { en: 'How to onboard a client in 4 steps', ar: 'كيف تُشغّل العميل في 4 خطوات' },
  onboardingSubtitle: {
    en: 'This helps close deals because the customer sees a clear implementation process.',
    ar: 'هذا يساعد على إغلاق الصفقة لأن العميل يرى خطوات تنفيذ واضحة.',
  },
  faqBadge: { en: 'FAQ', ar: 'الأسئلة الشائعة' },
  faqTitle: { en: 'Questions clients ask before buying', ar: 'أسئلة يسألها العملاء قبل الشراء' },
  finalTitle: {
    en: 'Package it as a managed service and clients will buy outcomes, not software.',
    ar: 'قدّمه كخدمة مُدارة وسيشتري العميل النتائج، لا مجرد البرنامج.',
  },
  finalBody: {
    en: 'Offer setup, training, message templates, automation design, and monthly optimization reports to increase recurring revenue.',
    ar: 'قدّم التهيئة والتدريب وقوالب الرسائل وتصميم الأتمتة وتقارير التحسين الشهرية لزيادة الإيراد المتكرر.',
  },
  finalPrimary: { en: 'Use Dedicated Business Offer', ar: 'استخدم عرض البيئة الخاصة للأعمال' },
  finalSecondary: { en: 'Contact / Custom Quote', ar: 'تواصل / عرض سعر مخصص' },
  footerText: {
    en: 'Built for agencies and operators selling WhatsApp growth systems.',
    ar: 'مصممة للوكالات والمشغلين الذين يبيعون أنظمة نمو واتساب.',
  },
} as const

const navItems: NavItem[] = [
  { href: '#features', label: ui.navFeatures },
  { href: '#plans', label: ui.navPlans },
  { href: '#compare', label: ui.navCompare },
  { href: '#faq', label: ui.navFaq },
]

const stats: StatItem[] = [
  {
    value: '24/7',
    label: { en: 'Conversation coverage', ar: 'تغطية المحادثات' },
    hint: { en: 'Human agents + automation + routing', ar: 'موظفون + أتمتة + توجيه' },
  },
  {
    value: '1 Page',
    label: { en: 'Sales explanation', ar: 'شرح البيع' },
    hint: { en: 'Features, plans, pricing, onboarding', ar: 'مزايا، باقات، أسعار، تشغيل' },
  },
  {
    value: 'Dedicated',
    label: { en: 'Client environment option', ar: 'خيار بيئة مخصصة' },
    hint: { en: 'Separate DB, users, uploads, port, domain', ar: 'قاعدة بيانات ومستخدمون وملفات ومجال منفصل' },
  },
  {
    value: 'ROI',
    label: { en: 'Outcome-focused positioning', ar: 'بيع على أساس النتائج' },
    hint: { en: 'Faster replies, fewer missed leads, reporting', ar: 'رد أسرع، فرص أقل ضياعًا، تقارير' },
  },
]

const featureGroups: FeatureGroup[] = [
  {
    icon: markRaw(MessageSquare),
    title: { en: 'Team Inbox & Chat Operations', ar: 'صندوق الوارد وإدارة المحادثات' },
    summary: {
      en: 'Multi-agent inbox with assignment, claim, close/reopen, notes, and WhatsApp-optimized message sending.',
      ar: 'صندوق وارد متعدد الموظفين مع التعيين والاستلام والإغلاق/إعادة الفتح والملاحظات وإرسال رسائل متوافق مع واتساب.',
    },
    bullets: [
      { en: 'Assigned / pending / closed chat workflows', ar: 'مسارات المحادثات: مُعيّنة / معلقة / مغلقة' },
      { en: 'Agent replies, media sending, rich-text helpers, canned responses', ar: 'ردود الموظفين وإرسال الوسائط ومحرر منسق وردود جاهزة' },
      { en: 'Conversation notes and contact-side operational metadata', ar: 'ملاحظات المحادثة وبيانات تشغيلية مرتبطة بالعميل' },
    ],
  },
  {
    icon: markRaw(Users),
    title: { en: 'Users, Teams, Roles & Restrictions', ar: 'المستخدمون والفرق والأدوار والقيود' },
    summary: {
      en: 'Control who can send, what accounts they can use, and what parts of the app they can access.',
      ar: 'تحكّم بمن يمكنه الإرسال، وأي حسابات يمكنه استخدامها، وما هي أجزاء النظام التي يراها.',
    },
    bullets: [
      { en: 'Users, teams, custom roles, permission matrix', ar: 'مستخدمون، فرق، أدوار مخصصة، مصفوفة صلاحيات' },
      { en: 'Send restrictions per user (including allowed WhatsApp instances)', ar: 'قيود إرسال لكل مستخدم (بما فيها الحسابات المسموحة)' },
      { en: 'Agent name prefix behavior and operational controls', ar: 'خيارات اسم الموظف داخل الرسائل وضوابط التشغيل' },
    ],
  },
  {
    icon: markRaw(Server),
    title: { en: 'WhatsApp Accounts / Instances', ar: 'حسابات واتساب / المثيلات' },
    summary: {
      en: 'Manage WhatsApp instances, health, status, and access visibility across organizations or dedicated tenants.',
      ar: 'إدارة مثيلات واتساب وحالتها وصحتها وصلاحيات رؤيتها داخل المنظمات أو البيئات المخصصة.',
    },
    bullets: [
      { en: 'Whatsmeow instance management + health views', ar: 'إدارة مثيلات Whatsmeow + لوحات الحالة الصحية' },
      { en: 'Meta account mode support (where used)', ar: 'دعم نمط Meta للحسابات (عند استخدامه)' },
      { en: 'Per-user and per-tenant instance scoping', ar: 'تحديد نطاق الحسابات لكل مستخدم ولكل عميل' },
    ],
  },
  {
    icon: markRaw(Bot),
    title: { en: 'Automation, Chatbot & Flow Builder', ar: 'الأتمتة والشات بوت وبناء التدفقات' },
    summary: {
      en: 'Build chatbot flows, keyword triggers, AI contexts, and route conversations to agents when needed.',
      ar: 'ابنِ تدفقات شات بوت ومحفزات بالكلمات المفتاحية وسياقات AI وحوّل المحادثة للموظف عند الحاجة.',
    },
    bullets: [
      { en: 'Visual chatbot flow builder and previews', ar: 'بناء تدفقات بصري مع معاينة' },
      { en: 'Keywords, AI contexts, and transfer rules', ar: 'كلمات مفتاحية وسياقات AI وقواعد تحويل' },
      { en: 'Agent transfer workflows and chatbot settings', ar: 'مسارات تحويل للموظفين وإعدادات الشات بوت' },
    ],
  },
  {
    icon: markRaw(Megaphone),
    title: { en: 'Campaigns, Templates & Content Ops', ar: 'الحملات والقوالب وإدارة المحتوى' },
    summary: {
      en: 'Run outbound campaigns, use templates, and standardize replies with reusable text and media assets.',
      ar: 'شغّل حملات إرسال واستخدم القوالب ووحّد الردود بنصوص ووسائط قابلة لإعادة الاستخدام.',
    },
    bullets: [
      { en: 'Campaigns and flow-linked automation journeys', ar: 'حملات وربطها بتدفقات أتمتة' },
      { en: 'Templates, canned responses, and media attachments', ar: 'قوالب وردود جاهزة ووسائط مرفقة' },
      { en: 'Tag-based organization and reusable messaging assets', ar: 'تنظيم بالوسوم وأصول رسائل قابلة لإعادة الاستخدام' },
    ],
  },
  {
    icon: markRaw(BarChart3),
    title: { en: 'Analytics, Ratings & Activity Logs', ar: 'التحليلات والتقييمات وسجل النشاط' },
    summary: {
      en: 'Give decision-makers visibility into agent performance, ratings, and operational events.',
      ar: 'امنح الإدارة رؤية واضحة لأداء الموظفين والتقييمات والأحداث التشغيلية.',
    },
    bullets: [
      { en: 'Agent analytics, averages, and performance tracking', ar: 'تحليلات الموظفين والمتوسطات وتتبع الأداء' },
      { en: 'Chat rating capture and follow-up comments', ar: 'التقاط التقييم وتعليقات ما بعد الإغلاق' },
      { en: 'Activity logs and audit-friendly event trails', ar: 'سجل نشاط ومسارات أحداث مناسبة للتدقيق' },
    ],
  },
  {
    icon: markRaw(Link2),
    title: { en: 'Integrations, API & Extensibility', ar: 'التكاملات وواجهات API والتوسعة' },
    summary: {
      en: 'Connect external systems and automate internal actions using API keys, webhooks, and custom actions.',
      ar: 'اربط الأنظمة الخارجية وأتمت الإجراءات الداخلية باستخدام API والمشغلات والخطوات المخصصة.',
    },
    bullets: [
      { en: 'API keys, webhooks, custom actions', ar: 'مفاتيح API وWebhooks وإجراءات مخصصة' },
      { en: 'SSO settings for enterprise deployments', ar: 'إعدادات SSO للبيئات المؤسسية' },
      { en: 'Migration and admin settings for managed deployments', ar: 'إعدادات الترحيل والإدارة للنشر المُدار' },
    ],
  },
  {
    icon: markRaw(Shield),
    title: { en: 'Dedicated Deployment & Security Controls', ar: 'النشر المخصص وضوابط الأمان' },
    summary: {
      en: 'Sell isolated tenant environments with separate database, uploads, service, domain, and SSL.',
      ar: 'بع بيئات معزولة لكل عميل بقاعدة بيانات وملفات وخدمة ومجال وSSL منفصل.',
    },
    bullets: [
      { en: 'Per-client config, database, uploads path, and service', ar: 'إعدادات وقاعدة بيانات وملفات وخدمة مستقلة لكل عميل' },
      { en: 'Per-client subdomain and SSL certificate', ar: 'نطاق فرعي وشهادة SSL لكل عميل' },
      { en: 'Shared binary deploy for fast updates across tenants', ar: 'تحديث موحد عبر ملف تنفيذي واحد لكل العملاء' },
    ],
  },
]

const inventoryChips: BiText[] = [
  { en: 'Inbox', ar: 'صندوق الوارد' },
  { en: 'Assigned / Pending / Closed Chats', ar: 'المحادثات المعيّنة / المعلقة / المغلقة' },
  { en: 'Contacts', ar: 'جهات الاتصال' },
  { en: 'Tags', ar: 'الوسوم' },
  { en: 'Users', ar: 'المستخدمون' },
  { en: 'Teams', ar: 'الفرق' },
  { en: 'Roles & Permissions', ar: 'الأدوار والصلاحيات' },
  { en: 'Instances / Accounts', ar: 'المثيلات / الحسابات' },
  { en: 'Templates', ar: 'القوالب' },
  { en: 'Canned Responses', ar: 'الردود الجاهزة' },
  { en: 'Campaigns', ar: 'الحملات' },
  { en: 'Chatbot Flows', ar: 'تدفقات الشات بوت' },
  { en: 'Keywords', ar: 'الكلمات المفتاحية' },
  { en: 'AI Contexts', ar: 'سياقات الذكاء الاصطناعي' },
  { en: 'Agent Transfers', ar: 'تحويلات الموظفين' },
  { en: 'Agent Analytics', ar: 'تحليلات الموظفين' },
  { en: 'Meta Insights', ar: 'تحليلات Meta' },
  { en: 'API Keys', ar: 'مفاتيح API' },
  { en: 'Webhooks', ar: 'الويب هوكس' },
  { en: 'SSO', ar: 'الدخول الموحد' },
  { en: 'Custom Actions', ar: 'إجراءات مخصصة' },
  { en: 'Activity Logs', ar: 'سجل النشاط' },
]

const plans: PlanItem[] = [
  {
    key: 'starter',
    badge: { en: 'Entry Offer', ar: 'عرض دخول' },
    title: { en: 'Starter', ar: 'البداية' },
    subtitle: { en: 'Shared SaaS', ar: 'سحابي مشترك' },
    price: '799 SAR',
    unit: ui.monthlySuffix,
    setupFee: { en: 'Setup: 1,500 SAR', ar: 'التهيئة: 1,500 ريال' },
    target: {
      en: 'For small teams that want basic inbox + reporting fast.',
      ar: 'للشركات الصغيرة التي تريد صندوق وارد وتقارير أساسية بسرعة.',
    },
    features: [
      { en: '1 organization / 1 admin', ar: 'منظمة واحدة / مدير واحد' },
      { en: 'Up to 3 users', ar: 'حتى 3 مستخدمين' },
      { en: '1 WhatsApp instance', ar: 'حساب واتساب واحد' },
      { en: 'Chat inbox, tags, contacts, canned responses', ar: 'صندوق وارد، وسوم، جهات اتصال، ردود جاهزة' },
      { en: 'Basic analytics + monthly support', ar: 'تحليلات أساسية + دعم شهري' },
    ],
    cta: { en: 'Use as low-friction entry plan', ar: 'استخدمها كباقة دخول منخفضة الاعتراض' },
  },
  {
    key: 'growth',
    badge: { en: 'Most Popular', ar: 'الأكثر طلبًا' },
    highlight: true,
    title: { en: 'Growth', ar: 'النمو' },
    subtitle: { en: 'Managed Business Plan', ar: 'خطة أعمال مُدارة' },
    price: '1,990 SAR',
    unit: ui.monthlySuffix,
    setupFee: { en: 'Setup: 2,500 SAR', ar: 'التهيئة: 2,500 ريال' },
    target: {
      en: 'For sales and support teams that need automation + campaigns.',
      ar: 'لفرق المبيعات والدعم التي تحتاج الأتمتة والحملات.',
    },
    features: [
      { en: 'Up to 10 users + teams + roles', ar: 'حتى 10 مستخدمين + فرق + أدوار' },
      { en: 'Up to 3 WhatsApp instances', ar: 'حتى 3 حسابات واتساب' },
      { en: 'Chatbot flows, keywords, transfer rules', ar: 'تدفقات الشات بوت والكلمات المفتاحية والتحويل' },
      { en: 'Campaigns, templates, media-rich canned responses', ar: 'حملات وقوالب وردود جاهزة مع وسائط' },
      { en: 'API/webhooks + priority support', ar: 'API / Webhooks + دعم أولوية' },
    ],
    cta: { en: 'Best balance for most clients', ar: 'أفضل توازن لمعظم العملاء' },
  },
  {
    key: 'dedicated',
    badge: { en: 'Recommended Offer', ar: 'العرض الموصى به' },
    title: { en: 'Dedicated Business', ar: 'بيئة خاصة للأعمال' },
    subtitle: { en: 'Isolated Client Environment', ar: 'بيئة عميل معزولة' },
    price: '4,900 SAR',
    unit: ui.monthlySuffix,
    setupFee: { en: 'Setup: 4,900 SAR', ar: 'التهيئة: 4,900 ريال' },
    target: {
      en: 'For serious companies that require isolated environment, higher reliability, and cleaner governance.',
      ar: 'للشركات الجادة التي تحتاج بيئة معزولة وموثوقية أعلى وحوكمة أوضح.',
    },
    features: [
      { en: 'Dedicated app instance + dedicated DB + dedicated uploads', ar: 'بيئة تطبيق + قاعدة بيانات + ملفات مستقلة' },
      { en: 'Dedicated subdomain + SSL certificate', ar: 'نطاق فرعي + SSL مستقل' },
      { en: 'Up to 25 users / 5 WhatsApp instances', ar: 'حتى 25 مستخدمًا / 5 حسابات واتساب' },
      { en: 'Advanced restrictions, analytics, and managed onboarding', ar: 'قيود متقدمة وتحليلات وتهيئة مُدارة' },
      { en: 'Faster support SLA + backup policy', ar: 'دعم أسرع + سياسة نسخ احتياطي' },
    ],
    cta: { en: 'Best plan to maximize margin and trust', ar: 'أفضل باقة لرفع الربح والثقة' },
  },
  {
    key: 'enterprise',
    badge: { en: 'Custom', ar: 'مخصص' },
    title: { en: 'Enterprise', ar: 'مؤسسي' },
    subtitle: { en: 'Dedicated Server / Private Hosting', ar: 'خادم مخصص / استضافة خاصة' },
    price: 'From 9,900 SAR',
    unit: ui.monthlySuffix,
    setupFee: { en: 'Setup: Custom quote', ar: 'التهيئة: عرض سعر' },
    target: {
      en: 'For high-volume operations, compliance needs, and custom integrations.',
      ar: 'للتشغيل عالي الحجم ومتطلبات الامتثال والتكاملات الخاصة.',
    },
    features: [
      { en: 'Private VPS or client-owned infrastructure', ar: 'VPS خاص أو بنية تحتية يملكها العميل' },
      { en: 'SSO, custom integrations, custom workflows', ar: 'SSO وتكاملات ومسارات عمل مخصصة' },
      { en: 'Larger user/instance limits', ar: 'حدود أعلى للمستخدمين والحسابات' },
      { en: 'Operational reviews + optimization reports', ar: 'مراجعات تشغيلية وتقارير تحسين' },
      { en: 'Custom SLA and support model', ar: 'SLA ونموذج دعم مخصص' },
    ],
    cta: { en: 'Quote based on volume and requirements', ar: 'سعّر حسب الحجم والمتطلبات' },
  },
]

const comparisonRows: ComparisonRow[] = [
  {
    label: { en: 'Deployment model', ar: 'نوع النشر' },
    values: [
      { en: 'Shared SaaS', ar: 'سحابي مشترك' },
      { en: 'Shared SaaS (managed)', ar: 'سحابي مشترك (مُدار)' },
      { en: 'Dedicated instance', ar: 'بيئة مخصصة' },
      { en: 'Dedicated server', ar: 'خادم مخصص' },
    ],
  },
  {
    label: { en: 'Client isolation', ar: 'عزل العميل' },
    values: [
      { en: 'Logical', ar: 'منطقي' },
      { en: 'Logical + controls', ar: 'منطقي + ضوابط' },
      { en: 'Separate app/DB/uploads', ar: 'تطبيق/قاعدة/ملفات منفصلة' },
      { en: 'Full infrastructure isolation', ar: 'عزل كامل للبنية' },
    ],
  },
  {
    label: { en: 'Users included', ar: 'المستخدمون' },
    values: [
      { en: 'Up to 3', ar: 'حتى 3' },
      { en: 'Up to 10', ar: 'حتى 10' },
      { en: 'Up to 25', ar: 'حتى 25' },
      { en: 'Custom', ar: 'مخصص' },
    ],
  },
  {
    label: { en: 'WhatsApp instances', ar: 'حسابات واتساب' },
    values: [
      { en: '1', ar: '1' },
      { en: 'Up to 3', ar: 'حتى 3' },
      { en: 'Up to 5', ar: 'حتى 5' },
      { en: 'Custom', ar: 'مخصص' },
    ],
  },
  {
    label: { en: 'Chatbot / Flows / Keywords', ar: 'شات بوت / تدفقات / كلمات مفتاحية' },
    values: [
      { en: 'Optional add-on', ar: 'إضافة اختيارية' },
      { en: 'Included', ar: 'مضمنة' },
      { en: 'Included', ar: 'مضمنة' },
      { en: 'Included + custom', ar: 'مضمنة + تخصيص' },
    ],
  },
  {
    label: { en: 'Campaigns & templates', ar: 'الحملات والقوالب' },
    values: [
      { en: 'Basic', ar: 'أساسية' },
      { en: 'Included', ar: 'مضمنة' },
      { en: 'Included', ar: 'مضمنة' },
      { en: 'Included + advanced', ar: 'مضمنة + متقدمة' },
    ],
  },
  {
    label: { en: 'API / Webhooks / Custom actions', ar: 'API / Webhooks / إجراءات مخصصة' },
    values: [
      { en: 'Limited', ar: 'محدود' },
      { en: 'Included', ar: 'مضمن' },
      { en: 'Included', ar: 'مضمن' },
      { en: 'Included + custom integration', ar: 'مضمن + تكامل مخصص' },
    ],
  },
  {
    label: { en: 'SSO', ar: 'الدخول الموحد SSO' },
    values: [
      { en: 'No', ar: 'لا' },
      { en: 'No', ar: 'لا' },
      { en: 'Optional', ar: 'اختياري' },
      { en: 'Yes', ar: 'نعم' },
    ],
  },
  {
    label: { en: 'Support & SLA', ar: 'الدعم وSLA' },
    values: [
      { en: 'Business hours', ar: 'ساعات العمل' },
      { en: 'Priority business hours', ar: 'أولوية خلال ساعات العمل' },
      { en: 'Priority + faster response', ar: 'أولوية + استجابة أسرع' },
      { en: 'Custom SLA', ar: 'SLA مخصص' },
    ],
  },
  {
    label: { en: 'Best for', ar: 'مناسبة لـ' },
    values: [
      { en: 'Testing / small team', ar: 'تجربة / فريق صغير' },
      { en: 'Growing sales/support', ar: 'فرق مبيعات/دعم نامية' },
      { en: 'Serious SMB / agency clients', ar: 'شركات جادة / عملاء الوكالات' },
      { en: 'High volume / enterprise', ar: 'حجم عالٍ / مؤسسي' },
    ],
  },
]

const onboardingSteps: StepItem[] = [
  {
    icon: markRaw(Settings),
    title: { en: '1. Discovery & Offer Fit', ar: '1. اكتشاف الاحتياج وتحديد الباقة' },
    body: {
      en: 'Identify use case (sales, support, booking, follow-up), user count, WhatsApp numbers, and required integrations.',
      ar: 'حدد حالة الاستخدام (مبيعات/دعم/حجوزات/متابعة) وعدد المستخدمين وأرقام واتساب والتكاملات المطلوبة.',
    },
  },
  {
    icon: markRaw(Server),
    title: { en: '2. Provision & Secure', ar: '2. التهيئة والتأمين' },
    body: {
      en: 'Create tenant environment (or shared org), admin account, domain/subdomain, SSL, database, and backups.',
      ar: 'أنشئ بيئة العميل (أو المنظمة المشتركة)، حساب المدير، النطاق/النطاق الفرعي، SSL، قاعدة البيانات، والنسخ الاحتياطي.',
    },
  },
  {
    icon: markRaw(Workflow),
    title: { en: '3. Build Operations Flow', ar: '3. بناء مسار التشغيل' },
    body: {
      en: 'Configure users, roles, teams, routing rules, canned responses, chatbot flows, tags, and templates.',
      ar: 'اضبط المستخدمين والأدوار والفرق وقواعد التوجيه والردود الجاهزة وتدفقات الشات بوت والوسوم والقوالب.',
    },
  },
  {
    icon: markRaw(BarChart3),
    title: { en: '4. Train, Launch, Optimize', ar: '4. التدريب والإطلاق والتحسين' },
    body: {
      en: 'Train the client team, launch, then review analytics/ratings monthly and optimize response quality.',
      ar: 'درّب فريق العميل ثم أطلق الخدمة وراجع التحليلات والتقييمات شهريًا لتحسين جودة الردود.',
    },
  },
]

const proofPoints = [
  {
    icon: markRaw(Clock),
    title: { en: 'Faster first response', ar: 'استجابة أولى أسرع' },
    body: {
      en: 'Assignment, visibility, and automation reduce missed or delayed conversations.',
      ar: 'التعيين والوضوح والأتمتة تقلل الرسائل الضائعة أو المتأخرة.',
    },
  },
  {
    icon: markRaw(Shield),
    title: { en: 'Operational control', ar: 'تحكم تشغيلي' },
    body: {
      en: 'Permissions, roles, send restrictions, and activity logs give management confidence.',
      ar: 'الصلاحيات والأدوار وقيود الإرسال وسجلات النشاط تعطي الإدارة ثقة أعلى.',
    },
  },
  {
    icon: markRaw(Star),
    title: { en: 'Measured quality', ar: 'جودة قابلة للقياس' },
    body: {
      en: 'Agent analytics and ratings help prove value and justify renewals.',
      ar: 'تحليلات الموظفين والتقييمات تساعد على إثبات القيمة وتجديد الاشتراك.',
    },
  },
]

const faqItems: FaqItem[] = [
  {
    q: {
      en: 'Can each client have a separate admin, users, and WhatsApp accounts?',
      ar: 'هل يمكن لكل عميل أن يملك مديره ومستخدميه وحسابات واتساب الخاصة به؟',
    },
    a: {
      en: 'Yes. In the Dedicated Business plan, each client runs in an isolated environment with a separate database, service, uploads folder, and domain/subdomain.',
      ar: 'نعم. في باقة البيئة الخاصة للأعمال يعمل كل عميل في بيئة معزولة بقاعدة بيانات وخدمة وملفات ونطاق مستقل.',
    },
  },
  {
    q: {
      en: 'How do updates get applied to all clients later?',
      ar: 'كيف يتم تطبيق التحديثات على كل العملاء لاحقًا؟',
    },
    a: {
      en: 'All tenants can share one binary. You build once, replace the shared binary, and restart tenant services one by one (or all together after testing).',
      ar: 'يمكن لكل العملاء مشاركة ملف تنفيذي واحد. تبني مرة واحدة ثم تستبدل الملف التنفيذي وتعيد تشغيل خدمات العملاء بالتدريج أو دفعة واحدة بعد الاختبار.',
    },
  },
  {
    q: {
      en: 'Can I offer Arabic and English interfaces to clients?',
      ar: 'هل يمكنني تقديم الواجهة بالعربية والإنجليزية للعملاء؟',
    },
    a: {
      en: 'Yes. The app supports multilingual UI, and this sales page itself supports Arabic/English with light and dark themes.',
      ar: 'نعم. التطبيق يدعم واجهة متعددة اللغات، وهذه الصفحة التسويقية نفسها تدعم العربية والإنجليزية مع الوضعين الفاتح والداكن.',
    },
  },
  {
    q: {
      en: 'Should I sell shared plans or dedicated plans?',
      ar: 'هل أبيع باقات مشتركة أم باقات مخصصة؟',
    },
    a: {
      en: 'Use shared plans to remove price objections, but position Dedicated Business as the main plan for companies that care about isolation, governance, and long-term reliability.',
      ar: 'استخدم الباقات المشتركة لتقليل اعتراضات السعر، لكن قدّم البيئة الخاصة للأعمال كالباقة الرئيسية للشركات التي تهتم بالعزل والحوكمة والاعتمادية طويلة المدى.',
    },
  },
]

const planColumnHeaders = [
  { en: 'Starter', ar: 'البداية' },
  { en: 'Growth', ar: 'النمو' },
  { en: 'Dedicated Business', ar: 'بيئة خاصة' },
  { en: 'Enterprise', ar: 'مؤسسي' },
]

const accentChecklist = [
  { icon: markRaw(Database), label: { en: 'Separate database per dedicated client', ar: 'قاعدة بيانات منفصلة لكل عميل مخصص' } },
  { icon: markRaw(HardDrive), label: { en: 'Separate uploads storage path', ar: 'مسار ملفات منفصل للوسائط' } },
  { icon: markRaw(Lock), label: { en: 'SSL + isolated subdomain', ar: 'SSL + نطاق فرعي معزول' } },
  { icon: markRaw(Key), label: { en: 'Per-client admin and users', ar: 'مدير ومستخدمون مستقلون لكل عميل' } },
]
</script>

<template>
  <div
    class="relative min-h-screen overflow-x-clip bg-background text-foreground"
    :dir="pageDir"
    :lang="language"
  >
    <div class="pointer-events-none absolute inset-0 opacity-80">
      <div class="orb orb-a" />
      <div class="orb orb-b" />
      <div class="orb orb-c" />
      <div class="grid-overlay" />
    </div>

    <header class="sticky top-0 z-40 border-b border-border/60 bg-background/70 backdrop-blur-xl">
      <div class="mx-auto flex w-full max-w-7xl items-center justify-between gap-4 px-4 py-3 md:px-6">
        <div class="flex items-center gap-3">
          <div class="flex size-9 items-center justify-center rounded-xl border border-emerald-500/30 bg-emerald-500/10 text-emerald-400">
            <Sparkles class="size-4" />
          </div>
          <div>
            <div class="text-sm font-semibold tracking-wide text-foreground">{{ text(ui.brand) }}</div>
            <div class="text-xs text-muted-foreground">
              {{ isArabic ? 'صفحة بيع الباقات والمزايا' : 'Features + Pricing Sales Page' }}
            </div>
          </div>
        </div>

        <nav class="hidden items-center gap-5 md:flex">
          <a
            v-for="item in navItems"
            :key="item.href"
            :href="item.href"
            class="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {{ text(item.label) }}
          </a>
        </nav>

        <div class="flex items-center gap-2">
          <div class="hidden items-center gap-1 rounded-xl border border-border/70 bg-card/80 p-1 sm:flex">
            <button
              type="button"
              class="rounded-lg px-2.5 py-1.5 text-xs font-medium transition-colors"
              :class="language === 'en' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
              @click="language = 'en'"
            >
              EN
            </button>
            <button
              type="button"
              class="rounded-lg px-2.5 py-1.5 text-xs font-medium transition-colors"
              :class="language === 'ar' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
              @click="language = 'ar'"
            >
              العربية
            </button>
          </div>

          <div class="hidden rounded-xl border border-border/70 bg-card/80 sm:block">
            <ThemeSwitcher />
          </div>

          <Button as-child size="sm" variant="outline" class="hidden md:inline-flex">
            <RouterLink to="/login">{{ text(ui.tertiaryCta) }}</RouterLink>
          </Button>
          <Button as-child size="sm" class="shadow-emerald-500/30">
            <a href="#plans">{{ text(ui.secondaryCta) }}</a>
          </Button>
        </div>
      </div>
    </header>

    <main class="relative z-10">
      <section class="mx-auto w-full max-w-7xl px-4 pb-14 pt-10 md:px-6 md:pb-20 md:pt-16">
        <div class="grid items-start gap-8 lg:grid-cols-[1.08fr_0.92fr]">
          <div class="animate-rise space-y-7">
            <div class="inline-flex items-center gap-2 rounded-full border border-emerald-500/25 bg-emerald-500/10 px-4 py-2 text-xs font-medium text-emerald-300">
              <Globe class="size-3.5" />
              <span>{{ text(ui.heroBadge) }}</span>
            </div>

            <div class="space-y-4">
              <h1 class="max-w-3xl text-3xl font-semibold leading-tight tracking-tight text-foreground sm:text-4xl md:text-5xl">
                {{ text(ui.heroTitle) }}
              </h1>
              <p class="max-w-2xl text-sm leading-7 text-muted-foreground sm:text-base">
                {{ text(ui.heroSubtitle) }}
              </p>
            </div>

            <div class="flex flex-col gap-3 sm:flex-row">
              <Button as-child size="lg" class="group">
                <RouterLink to="/register">
                  <span>{{ text(ui.primaryCta) }}</span>
                  <ArrowRight class="size-4 transition-transform group-hover:translate-x-0.5" />
                </RouterLink>
              </Button>
              <Button as-child size="lg" variant="outline">
                <a href="#plans">{{ text(ui.secondaryCta) }}</a>
              </Button>
            </div>

            <p class="text-xs text-muted-foreground">
              {{ text(ui.heroFootnote) }}
            </p>

            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <div
                v-for="item in stats"
                :key="item.value + text(item.label)"
                class="rounded-2xl border border-border/70 bg-card/70 p-4 shadow-[0_10px_30px_-18px_rgba(16,185,129,0.35)] backdrop-blur-sm"
              >
                <div class="text-sm font-semibold text-foreground">{{ item.value }}</div>
                <div class="mt-1 text-xs font-medium text-emerald-400">{{ text(item.label) }}</div>
                <div class="mt-1.5 text-xs leading-5 text-muted-foreground">{{ text(item.hint) }}</div>
              </div>
            </div>
          </div>

          <div class="animate-rise-delay space-y-4">
            <div class="rounded-3xl border border-border/70 bg-card/75 p-5 shadow-[0_24px_60px_-24px_rgba(0,0,0,0.45)] backdrop-blur-xl">
              <div class="mb-4 flex items-center justify-between gap-2">
                <div>
                  <div class="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                    {{ isArabic ? 'لوحة عرض البيع' : 'Sales Snapshot' }}
                  </div>
                  <div class="mt-1 text-lg font-semibold">{{ isArabic ? 'ماذا يشتري العميل؟' : 'What the client buys' }}</div>
                </div>
                <div class="rounded-xl border border-emerald-500/25 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-300">
                  {{ isArabic ? 'خدمة مُدارة' : 'Managed Service' }}
                </div>
              </div>

              <div class="space-y-3">
                <div class="rounded-2xl border border-border/60 bg-background/50 p-4">
                  <div class="flex items-center gap-3">
                    <div class="flex size-10 items-center justify-center rounded-xl bg-primary/15 text-primary">
                      <Layers class="size-4" />
                    </div>
                    <div class="min-w-0">
                      <div class="text-sm font-medium">{{ isArabic ? 'تشغيل المحادثات' : 'Conversation Operations' }}</div>
                      <div class="text-xs text-muted-foreground">
                        {{ isArabic ? 'صندوق وارد + فرق + صلاحيات + تتبع' : 'Inbox + teams + permissions + audit visibility' }}
                      </div>
                    </div>
                  </div>
                </div>

                <div class="rounded-2xl border border-border/60 bg-background/50 p-4">
                  <div class="grid gap-3 sm:grid-cols-2">
                    <div class="rounded-xl border border-border/60 bg-card/70 p-3">
                      <div class="text-xs text-muted-foreground">{{ isArabic ? 'الأتمتة' : 'Automation' }}</div>
                      <div class="mt-1 flex items-center gap-2 text-sm font-medium">
                        <Bot class="size-4 text-emerald-400" />
                        {{ isArabic ? 'تدفقات + كلمات + تحويلات' : 'Flows + keywords + transfers' }}
                      </div>
                    </div>
                    <div class="rounded-xl border border-border/60 bg-card/70 p-3">
                      <div class="text-xs text-muted-foreground">{{ isArabic ? 'القياس' : 'Measurement' }}</div>
                      <div class="mt-1 flex items-center gap-2 text-sm font-medium">
                        <BarChart3 class="size-4 text-emerald-400" />
                        {{ isArabic ? 'تحليلات + تقييمات + تقارير' : 'Analytics + ratings + reports' }}
                      </div>
                    </div>
                  </div>
                </div>

                <div class="rounded-2xl border border-emerald-500/20 bg-gradient-to-br from-emerald-500/10 via-transparent to-green-500/5 p-4">
                  <div class="mb-3 text-xs uppercase tracking-[0.16em] text-emerald-300/90">
                    {{ isArabic ? 'عرض موصى به للبيع' : 'Recommended Offer to Sell' }}
                  </div>
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div class="text-base font-semibold">
                        {{ isArabic ? 'بيئة خاصة للأعمال' : 'Dedicated Business' }}
                      </div>
                      <div class="text-xs text-muted-foreground">
                        {{ isArabic ? 'بيئة معزولة + SSL + قاعدة بيانات مستقلة' : 'Isolated app + SSL + separate database' }}
                      </div>
                    </div>
                    <div class="text-right" :class="isArabic ? 'text-left' : 'text-right'">
                      <div class="text-lg font-semibold text-emerald-300">4,900 SAR</div>
                      <div class="text-xs text-muted-foreground">{{ text(ui.monthlySuffix) }}</div>
                    </div>
                  </div>
                  <div class="mt-4 grid gap-2">
                    <div
                      v-for="entry in accentChecklist"
                      :key="text(entry.label)"
                      class="flex items-start gap-2 text-xs text-foreground/90"
                    >
                      <component :is="entry.icon" class="mt-0.5 size-3.5 shrink-0 text-emerald-400" />
                      <span>{{ text(entry.label) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="mx-auto w-full max-w-7xl px-4 pb-8 md:px-6">
        <div class="rounded-3xl border border-border/70 bg-card/70 p-5 md:p-7">
          <div class="mb-4 inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/70 px-3 py-1.5 text-xs text-muted-foreground">
            <Zap class="size-3.5 text-emerald-400" />
            {{ text(ui.painLabel) }}
          </div>
          <div class="grid gap-5 lg:grid-cols-[1.2fr_0.8fr]">
            <div>
              <h2 class="text-xl font-semibold md:text-2xl">{{ text(ui.painTitle) }}</h2>
              <p class="mt-3 max-w-3xl text-sm leading-7 text-muted-foreground">
                {{ text(ui.painBody) }}
              </p>
            </div>
            <div class="grid gap-3">
              <div
                v-for="point in proofPoints"
                :key="text(point.title)"
                class="rounded-2xl border border-border/70 bg-background/70 p-4"
              >
                <div class="flex items-start gap-3">
                  <div class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-emerald-500/25 bg-emerald-500/10 text-emerald-400">
                    <component :is="point.icon" class="size-4" />
                  </div>
                  <div>
                    <div class="text-sm font-medium">{{ text(point.title) }}</div>
                    <div class="mt-1 text-xs leading-5 text-muted-foreground">{{ text(point.body) }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="features" class="scroll-mt-24 mx-auto w-full max-w-7xl px-4 py-12 md:px-6 md:py-16">
        <div class="mb-8 space-y-3">
          <div class="section-pill">
            <Sparkles class="size-3.5" />
            <span>{{ text(ui.featuresBadge) }}</span>
          </div>
          <h2 class="text-2xl font-semibold tracking-tight md:text-3xl">
            {{ text(ui.featuresTitle) }}
          </h2>
          <p class="max-w-3xl text-sm leading-7 text-muted-foreground">
            {{ text(ui.featuresSubtitle) }}
          </p>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <article
            v-for="group in featureGroups"
            :key="text(group.title)"
            class="group rounded-3xl border border-border/70 bg-card/70 p-5 transition-all duration-300 hover:-translate-y-0.5 hover:border-emerald-500/25 hover:shadow-[0_20px_45px_-28px_rgba(16,185,129,0.35)]"
          >
            <div class="flex items-start gap-4">
              <div class="flex size-11 shrink-0 items-center justify-center rounded-2xl border border-emerald-500/25 bg-emerald-500/10 text-emerald-400">
                <component :is="group.icon" class="size-5" />
              </div>
              <div class="min-w-0">
                <h3 class="text-base font-semibold">{{ text(group.title) }}</h3>
                <p class="mt-2 text-xs leading-6 text-muted-foreground">{{ text(group.summary) }}</p>
                <ul class="mt-4 space-y-2.5">
                  <li
                    v-for="bullet in group.bullets"
                    :key="text(bullet)"
                    class="flex items-start gap-2 text-xs text-foreground/90"
                  >
                    <Check class="mt-0.5 size-3.5 shrink-0 text-emerald-400" />
                    <span class="leading-5">{{ text(bullet) }}</span>
                  </li>
                </ul>
              </div>
            </div>
          </article>
        </div>

        <div class="mt-8 rounded-3xl border border-border/70 bg-card/65 p-5 md:p-6">
          <div class="mb-4 flex items-center gap-2 text-sm font-medium">
            <FileText class="size-4 text-emerald-400" />
            <span>{{ text(ui.inventoryTitle) }}</span>
          </div>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="chip in inventoryChips"
              :key="text(chip)"
              class="inline-flex items-center rounded-full border border-border/70 bg-background/80 px-3 py-1.5 text-xs text-muted-foreground"
            >
              {{ text(chip) }}
            </span>
          </div>
        </div>
      </section>

      <section id="plans" class="scroll-mt-24 mx-auto w-full max-w-7xl px-4 py-12 md:px-6 md:py-16">
        <div class="mb-8 space-y-3">
          <div class="section-pill">
            <Tags class="size-3.5" />
            <span>{{ text(ui.plansBadge) }}</span>
          </div>
          <h2 class="text-2xl font-semibold tracking-tight md:text-3xl">{{ text(ui.plansTitle) }}</h2>
          <p class="max-w-3xl text-sm leading-7 text-muted-foreground">{{ text(ui.plansSubtitle) }}</p>
        </div>

        <div class="grid gap-4 xl:grid-cols-4 md:grid-cols-2">
          <article
            v-for="plan in plans"
            :key="plan.key"
            class="relative rounded-3xl border p-5 transition-all duration-300"
            :class="plan.highlight
              ? 'border-emerald-500/35 bg-gradient-to-b from-emerald-500/10 via-card/75 to-card/70 shadow-[0_25px_65px_-30px_rgba(16,185,129,0.45)]'
              : 'border-border/70 bg-card/70 hover:border-emerald-500/20 hover:shadow-[0_20px_50px_-35px_rgba(16,185,129,0.35)]'"
          >
            <div
              v-if="plan.badge"
              class="mb-3 inline-flex rounded-full border px-3 py-1 text-[11px] font-medium"
              :class="plan.highlight
                ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-300'
                : 'border-border/70 bg-background/70 text-muted-foreground'"
            >
              {{ text(plan.badge) }}
            </div>

            <div class="space-y-2">
              <h3 class="text-lg font-semibold">{{ text(plan.title) }}</h3>
              <p class="text-xs text-muted-foreground">{{ text(plan.subtitle) }}</p>
            </div>

            <div class="mt-4">
              <div class="flex items-end gap-1.5">
                <span class="text-2xl font-semibold tracking-tight">{{ plan.price }}</span>
                <span class="pb-0.5 text-xs text-muted-foreground">{{ text(plan.unit) }}</span>
              </div>
              <div class="mt-1 text-xs text-muted-foreground">{{ text(plan.setupFee) }}</div>
            </div>

            <p class="mt-4 text-xs leading-6 text-muted-foreground">{{ text(plan.target) }}</p>

            <ul class="mt-4 space-y-2.5">
              <li
                v-for="item in plan.features"
                :key="text(item)"
                class="flex items-start gap-2 text-xs"
              >
                <Check class="mt-0.5 size-3.5 shrink-0 text-emerald-400" />
                <span class="leading-5">{{ text(item) }}</span>
              </li>
            </ul>

            <div class="mt-5 border-t border-border/70 pt-4">
              <div class="mb-3 text-xs text-muted-foreground">{{ text(plan.cta) }}</div>
              <Button as-child class="w-full" :variant="plan.highlight ? 'default' : 'outline'">
                <a href="#compare">{{ isArabic ? 'قارن الباقة' : 'Compare This Plan' }}</a>
              </Button>
            </div>
          </article>
        </div>
      </section>

      <section id="compare" class="scroll-mt-24 mx-auto w-full max-w-7xl px-4 py-12 md:px-6 md:py-16">
        <div class="mb-8 space-y-3">
          <div class="section-pill">
            <Layers class="size-3.5" />
            <span>{{ text(ui.compareBadge) }}</span>
          </div>
          <h2 class="text-2xl font-semibold tracking-tight md:text-3xl">{{ text(ui.compareTitle) }}</h2>
          <p class="max-w-3xl text-sm leading-7 text-muted-foreground">{{ text(ui.compareSubtitle) }}</p>
        </div>

        <div class="overflow-x-auto rounded-3xl border border-border/70 bg-card/70">
          <table class="min-w-[860px] w-full border-collapse text-left" :class="isArabic ? 'text-right' : 'text-left'">
            <thead>
              <tr class="border-b border-border/70 bg-background/40">
                <th class="px-4 py-4 text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                  {{ isArabic ? 'العنصر' : 'Item' }}
                </th>
                <th
                  v-for="header in planColumnHeaders"
                  :key="text(header)"
                  class="px-4 py-4 text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground"
                >
                  {{ text(header) }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, rowIndex) in comparisonRows"
                :key="text(row.label)"
                class="border-b border-border/60 last:border-b-0"
                :class="rowIndex % 2 === 0 ? 'bg-card/40' : 'bg-background/20'"
              >
                <td class="px-4 py-4 align-top text-sm font-medium text-foreground">
                  {{ text(row.label) }}
                </td>
                <td
                  v-for="(cell, cellIndex) in row.values"
                  :key="cellIndex + text(cell)"
                  class="px-4 py-4 align-top text-xs leading-6 text-muted-foreground"
                >
                  {{ text(cell) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="mx-auto w-full max-w-7xl px-4 py-12 md:px-6 md:py-16">
        <div class="mb-8 space-y-3">
          <div class="section-pill">
            <Workflow class="size-3.5" />
            <span>{{ text(ui.onboardingBadge) }}</span>
          </div>
          <h2 class="text-2xl font-semibold tracking-tight md:text-3xl">{{ text(ui.onboardingTitle) }}</h2>
          <p class="max-w-3xl text-sm leading-7 text-muted-foreground">{{ text(ui.onboardingSubtitle) }}</p>
        </div>

        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <article
            v-for="step in onboardingSteps"
            :key="text(step.title)"
            class="relative overflow-hidden rounded-3xl border border-border/70 bg-card/70 p-5"
          >
            <div class="pointer-events-none absolute -right-8 top-3 size-20 rounded-full bg-emerald-500/10 blur-2xl" />
            <div class="relative">
              <div class="mb-3 flex size-10 items-center justify-center rounded-xl border border-emerald-500/25 bg-emerald-500/10 text-emerald-400">
                <component :is="step.icon" class="size-4" />
              </div>
              <h3 class="text-sm font-semibold">{{ text(step.title) }}</h3>
              <p class="mt-2 text-xs leading-6 text-muted-foreground">{{ text(step.body) }}</p>
            </div>
          </article>
        </div>
      </section>

      <section id="faq" class="scroll-mt-24 mx-auto w-full max-w-7xl px-4 py-12 md:px-6 md:py-16">
        <div class="mb-8 space-y-3">
          <div class="section-pill">
            <Sparkles class="size-3.5" />
            <span>{{ text(ui.faqBadge) }}</span>
          </div>
          <h2 class="text-2xl font-semibold tracking-tight md:text-3xl">{{ text(ui.faqTitle) }}</h2>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <article
            v-for="item in faqItems"
            :key="text(item.q)"
            class="rounded-3xl border border-border/70 bg-card/70 p-5"
          >
            <h3 class="text-sm font-semibold leading-6">{{ text(item.q) }}</h3>
            <p class="mt-2 text-xs leading-6 text-muted-foreground">{{ text(item.a) }}</p>
          </article>
        </div>
      </section>

      <section class="mx-auto w-full max-w-7xl px-4 pb-14 pt-4 md:px-6 md:pb-20">
        <div class="relative overflow-hidden rounded-3xl border border-emerald-500/25 bg-gradient-to-br from-emerald-500/12 via-card/85 to-card/70 p-6 md:p-8">
          <div class="pointer-events-none absolute inset-0 opacity-70">
            <div class="absolute left-[10%] top-[20%] h-28 w-28 rounded-full bg-emerald-500/10 blur-3xl" />
            <div class="absolute right-[12%] top-[35%] h-24 w-24 rounded-full bg-green-400/10 blur-3xl" />
          </div>

          <div class="relative grid gap-6 lg:grid-cols-[1.25fr_0.75fr] lg:items-center">
            <div>
              <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-emerald-400/20 bg-emerald-500/10 px-3 py-1.5 text-xs font-medium text-emerald-300">
                <Sparkles class="size-3.5" />
                <span>{{ isArabic ? 'رسالة بيع مقترحة' : 'Recommended Sales Positioning' }}</span>
              </div>
              <h2 class="text-2xl font-semibold tracking-tight md:text-3xl">{{ text(ui.finalTitle) }}</h2>
              <p class="mt-3 max-w-3xl text-sm leading-7 text-muted-foreground">{{ text(ui.finalBody) }}</p>
            </div>

            <div class="space-y-3 rounded-2xl border border-border/70 bg-background/60 p-4 backdrop-blur-sm">
              <Button as-child class="w-full justify-center">
                <a href="#plans">{{ text(ui.finalPrimary) }}</a>
              </Button>
              <Button as-child variant="outline" class="w-full justify-center">
                <a href="mailto:sales@ofuqalmadenah.com">{{ text(ui.finalSecondary) }}</a>
              </Button>
              <div class="text-center text-xs text-muted-foreground">
                {{ isArabic ? 'يمكنك تعديل الأسعار حسب السوق ونطاق الخدمة.' : 'Adjust prices by market, SLA, and onboarding scope.' }}
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="relative z-10 border-t border-border/70 bg-background/80">
      <div class="mx-auto flex w-full max-w-7xl flex-col gap-3 px-4 py-6 text-xs text-muted-foreground md:flex-row md:items-center md:justify-between md:px-6">
        <div class="flex items-center gap-2">
          <Sparkles class="size-3.5 text-emerald-400" />
          <span>{{ text(ui.footerText) }}</span>
        </div>
        <div>
          © {{ year }} {{ text(ui.brand) }} · {{ isArabic ? 'صفحة تسويقية للباقات والمزايا' : 'Marketing page for features & pricing' }}
        </div>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.orb {
  position: absolute;
  border-radius: 9999px;
  filter: blur(48px);
  opacity: 0.6;
  animation: floatOrb 18s ease-in-out infinite;
}

.orb-a {
  top: 4rem;
  left: 8%;
  width: 16rem;
  height: 16rem;
  background: radial-gradient(circle, rgba(16, 185, 129, 0.22), transparent 70%);
}

.orb-b {
  top: 18rem;
  right: 10%;
  width: 18rem;
  height: 18rem;
  background: radial-gradient(circle, rgba(34, 197, 94, 0.12), transparent 70%);
  animation-delay: -6s;
}

.orb-c {
  top: 45rem;
  left: 35%;
  width: 14rem;
  height: 14rem;
  background: radial-gradient(circle, rgba(5, 150, 105, 0.14), transparent 70%);
  animation-delay: -11s;
}

.grid-overlay {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(to right, rgba(255, 255, 255, 0.02) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: radial-gradient(circle at 50% 20%, black 35%, transparent 90%);
}

.light .grid-overlay {
  background-image:
    linear-gradient(to right, rgba(0, 0, 0, 0.03) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(0, 0, 0, 0.03) 1px, transparent 1px);
}

.section-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 9999px;
  border: 1px solid hsl(var(--border) / 0.8);
  background: hsl(var(--card) / 0.65);
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
}

.animate-rise {
  animation: riseIn 0.55s ease-out;
}

.animate-rise-delay {
  animation: riseIn 0.65s ease-out;
}

@keyframes riseIn {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes floatOrb {
  0%,
  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
  50% {
    transform: translate3d(0, -14px, 0) scale(1.04);
  }
}
</style>
