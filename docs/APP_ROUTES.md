Listed directory router
Viewed index.ts:1-505
Searched for "(api\.(get|post|put|delete)|axios\.)"
Searched for "useSocket|websocket|WebSocket"

بناءً على تحليل ملف الـ Router (`frontend/src/router/index.ts`) والـ Components المرتبطة به في المجلد `views`، قمت باستخراج الجدول التالي لجميع مسارات (Routes) لوحة تحكم Whatomate:

| مسار الصفحة (`path`) | اسم الصفحة (`name`) | ملف الـ `component` 📝 | الصلاحية / `permission` 🔐 | عام / دخول؟ 🚪 | أبرز الـ API Endpoints 🔗 | WebSockets؟ 🔌 |
|-------------------|----------------|---------------------------|--------------------------|------------------|------------------------------|--------------|
| `/login` | `login` | `auth/LoginView.vue` | لا يوجد | 🌍 عام | `/api/auth/login` | ❌ |
| `/register` | `register` | `auth/RegisterView.vue` | لا يوجد | 🌍 عام | `/api/auth/register` | ❌ |
| `/auth/sso/callback` | `sso-callback` | `auth/SSOCallbackView.vue` | لا يوجد | 🌍 عام | `/api/auth/sso/*/callback` | ❌ |
| `/activate` | `activate` | `public/ActivateLicenseView.vue`| لا يوجد | 🌍 عام | `/api/license/activate` | ❌ |
| `/pricing` | `marketing-redirect`| `public/MarketingRedirectView.vue`| لا يوجد | 🌍 عام | لا يوجد | ❌ |
| `/license-cleanup`| `license-cleanup` | `settings/LicenseCleanupView.vue`| لا يوجد | 👤 يتطلب دخول | `/api/contacts`, `/api/tags` | ❌ |
| `/dashboard` | `dashboard` | `dashboard/DashboardView.vue` | `analytics` | 👤 يتطلب دخول | `/api/analytics/dashboard` | ❌ |
| `/chat` <br> `/chat/:contactId` | `chat` <br> `chat-conversation`| `chat/ChatView.vue` | `chat` | 👤 يتطلب دخول | `/api/contacts/*`, `/api/chats/*`, `.../messages` | ✅ (مباشر) |
| `/profile` | `profile` | `profile/ProfileView.vue` | متاح لكافة الأدوار | 👤 يتطلب دخول | `/api/me`, `.../settings`, `.../password` | ❌ |
| `/templates` | `templates` | `settings/TemplatesView.vue`| `templates` (Meta Only)| 👤 يتطلب دخول | `/api/templates/*` | ❌ |
| `/flows` | `flows` | `settings/FlowsView.vue`| `flows.whatsapp` | 👤 يتطلب دخول | `/api/flows/*` | ❌ |
| `/campaigns` | `campaigns` | `settings/CampaignsView.vue`| `campaigns` | 👤 يتطلب دخول | `/api/campaigns/*` | ✅ (استماع مباشر للتقدم) |
| `/chatbot` | `chatbot` | `chatbot/ChatbotView.vue` | `settings.chatbot` | 👤 يتطلب دخول | `/api/chatbot/settings` | ❌ |
| `/chatbot/keywords`| `chatbot-keywords` | `chatbot/KeywordsView.vue` | `chatbot.keywords` | 👤 يتطلب دخول | `/api/chatbot/keywords/*` | ❌ |
| `/chatbot/flows` | `chatbot-flows` | `chatbot/ChatbotFlowsView.vue` | `flows.chatbot` | 👤 يتطلب دخول | `/api/chatbot/flows/*` | ❌ |
| `/chatbot/flows/new`<br>`.../:id/edit`| `chatbot-flow-new`<br>`chatbot-flow-edit`| `chatbot/ChatbotFlowBuilderView.vue`| `flows.chatbot` | 👤 يتطلب دخول | `/api/chatbot/flows/*` | ❌ |
| `/chatbot/ai` | `chatbot-ai` | `chatbot/AIContextsView.vue`| `chatbot.ai` | 👤 يتطلب دخول | `/api/chatbot/ai-contexts/*` | ❌ |
| `/chatbot/transfers`| `chatbot-transfers`| `chatbot/AgentTransfersView.vue`| `transfers` | 👤 يتطلب دخول | `/api/chatbot/transfers/*` | ✅ (لاستقبال التحويلات فورا) |
| `/analytics/agents`| `agent-analytics` | `analytics/AgentAnalyticsView.vue`| `analytics.agents` | 👤 يتطلب دخول | `/api/analytics/agents/*` | ❌ |
| `/analytics/meta-insights`| `meta-insights` | `analytics/MetaInsightsView.vue`| `analytics` (Meta Only)| 👤 يتطلب دخول | `/api/analytics/meta/*` | ❌ |
| `/settings` | `settings` | `settings/SettingsView.vue` | `settings.general` أو `uploads_cleanup` | 👤 يتطلب دخول | `/api/org/settings`, `org/uploads-cleanup` | ❌ |
| `/settings/chatbot`| `chatbot-settings`| `settings/ChatbotSettingsView.vue`| `settings.chatbot` | 👤 يتطلب دخول | `/api/chatbot/settings` | ❌ |
| `/settings/accounts`| `accounts` | `settings/AccountsView.vue` | `accounts` (Meta Only)| 👤 يتطلب دخول | `/api/accounts/*` | ❌ |
| `/settings/instances`| `instances` | `settings/InstancesView.vue`| `accounts` | 👤 يتطلب دخول | `/api/instances/*` | ✅ (تحديث الحالة) |
| `/settings/instances/health`| `instances-health` | `settings/InstanceHealthView.vue`| `accounts` | 👤 يتطلب دخول | `/api/instances/{id}/health`| ❌ |
| `/settings/canned-responses`| `canned-responses`| `settings/CannedResponsesView.vue`| `canned_responses` | 👤 يتطلب دخول | `/api/canned-responses/*` | ❌ |
| `/settings/contacts`| `contacts` | `settings/ContactsView.vue` | `contacts` | 👤 يتطلب دخول | `/api/contacts/*` | ❌ |
| `/settings/closed-chats`| `closed-chats` | `settings/ClosedChatsView.vue`| `chat` | 👤 يتطلب دخول | `/api/chats?status=closed` | ❌ |
| `/settings/tags` | `tags` | `settings/TagsView.vue` | `tags` | 👤 يتطلب دخول | `/api/tags/*` | ❌ |
| `/settings/users` | `users` | `settings/UsersView.vue` | `users` | 👤 يتطلب دخول | `/api/users/*`, `/api/roles`| ❌ |
| `/settings/roles` | `roles` | `settings/RolesView.vue` | `roles` | 👤 يتطلب دخول | `/api/roles/*`, `/api/permissions`| ❌ |
| `/settings/teams` | `teams` | `settings/TeamsView.vue` | `teams` | 👤 يتطلب دخول | `/api/teams/*` | ❌ |
| `/settings/api-keys`| `api-keys` | `settings/APIKeysView.vue` | `api_keys` | 👤 يتطلب دخول | `/api/api-keys/*` | ❌ |
| `/settings/webhooks`| `webhooks` | `settings/WebhooksView.vue` | `webhooks` | 👤 يتطلب دخول | `/api/webhooks/*` | ❌ |
| `/settings/sso` | `sso-settings` | `settings/SSOSettingsView.vue`| `settings.sso` | 👤 يتطلب دخول | `/api/settings/sso/*` | ❌ |
| `/settings/license`| `license-settings`| `settings/LicenseSettingsView.vue`| `adminOnly` 👑 | 👤 يتطلب دخول | `/api/license/*` | ❌ |
| `/settings/custom-actions`| `custom-actions`| `settings/CustomActionsView.vue` | `custom_actions` | 👤 يتطلب دخول | `/api/custom-actions/*` | ❌ |
| `/:pathMatch(.*)*` | `not-found` | `NotFoundView.vue` | لا يوجد | 🌍 (متاح لأي مسار خاطئ) | لا يوجد | ❌ |

**ملاحظة:** إضافةً إلى صفحات الـ View المذكورة التي تستخدم `wsService` مباشرةً، يقوم الغلاف الأساسي للتطبيق `AppLayout.vue` بفتح وتوصيل اتصال الـ WebSockets تلقائياً فور دخول المستخدم (بشرط تسجيل الدخول)، لضمان الإشعارات الفورية على مستوى النظام كاملاً.