# خطة دمج RuVector للتعلّم السلوكي وتوجيه المحادثات للموظف المطلوب

## الملخص
سنضيف طبقة توجيه ذكي داخل `Whatomate` تعتمد على `RuVector` كخدمة مستقلة (sidecar) مع Embeddings محلية، لتتعلّم من:
1. المحادثات (incoming/outgoing).
2. أحداث التحويل بين الموظفين.
3. تغيّر توفر الموظفين.

وعند طلب العميل موظفًا محددًا بالاسم:
1. نكتشف الاسم تلقائيًا.
2. نؤكد المطابقة (داخليًا إذا كانت فريدة، أو برسالة تأكيد إذا كانت ملتبسة).
3. نطبّق سياسة افتراضية: `تفضيل ذكي + fallback بعد 5 دقائق`.
4. إذا لم يتوفر الموظف المطلوب بعد المهلة، نُسند لأفضل بديل.

الإعدادات ستكون قابلة للتعديل من النظام، مع القيم الافتراضية التي اخترتها:
- `mode = preferred_with_fallback`
- `fallback_minutes = 5`
- `confidence_threshold = 0.75`
- `retention_days = 180`
- `rollout = all_teams`
- `embeddings = local`
- `learning_signals = messages + transfers + availability`
- `employee matching = full_name + aliases`

## تغييرات Public APIs / Interfaces / Types
### 1) توسيع API إعدادات المؤسسة
نوسّع `GET/PUT /api/org/settings` بإضافة الحقول التالية:
- `smart_routing_enabled: bool`
- `smart_routing_mode: "preferred_with_fallback" | "strict" | "dynamic"`
- `smart_routing_fallback_minutes: int`
- `smart_routing_confidence_threshold: float`
- `smart_routing_detect_requested_agent: bool`
- `smart_routing_retention_days: int`
- `smart_routing_learning_signals: "messages_transfers_availability"`

### 2) API إدارة أسماء الموظفين البديلة
إضافة endpoints:
- `GET /api/users/{id}/routing-aliases`
- `POST /api/users/{id}/routing-aliases`
- `DELETE /api/users/{id}/routing-aliases/{alias_id}`

صيغة الإنشاء:
```json
{
  "alias": "أ. أحمد",
  "is_primary": false
}
```

### 3) API فحص قرار التوجيه (تشخيصي للإدارة)
إضافة:
- `POST /api/chatbot/routing/preview`

الطلب:
```json
{
  "contact_id": "uuid",
  "message_text": "ابغى أكلم أحمد",
  "whatsapp_account": "sales"
}
```

الاستجابة:
```json
{
  "requested_agent": {"user_id": "uuid", "confidence": 0.92, "match_type": "alias_exact"},
  "recommended_agent": {"user_id": "uuid", "confidence": 0.88, "source": "ruvector"},
  "decision": {"mode": "preferred_with_fallback", "assigned_agent_id": "uuid", "fallback_at": "RFC3339"}
}
```

### 4) تغييرات Types/Models
- إضافة `TransferSource` جديدة:
  - `requested_agent`
  - `smart_routing`
- إضافة نماذج جديدة:
  - `AgentAlias`
  - `ContactAgentPreference`
  - `RoutingDecisionAudit`
- إضافة حقول توجيه إلى `AgentTransfer`:
  - `PreferredAgentID`
  - `PreferredUntil`
  - `RoutingMode`
  - `RoutingConfidence`
  - `RoutingReason`

## المعمارية التنفيذية
### 1) خدمة RuVector مستقلة
خدمة `ruvector-router` تعمل عبر HTTP داخل Docker network وتقدّم:
- `POST /v1/ingest/events`
- `POST /v1/route/agent`
- `POST /v1/preferences/upsert`
- `POST /v1/maintenance/prune`
- `GET /health`

### 2) Embeddings محلية
خدمة `embeddings` محلية (model multilingual مناسب للعربية مثل `bge-m3`) يتصل بها `ruvector-router`.
لا يوجد إرسال نصوص خام لخدمة خارجية.

### 3) عميل Go داخل Whatomate
إضافة `RuvectorClient` داخل الـbackend للتواصل مع sidecar بمهلة قصيرة وإعادة محاولات محدودة.

## تدفق البيانات والتعلّم
### 1) Event Ingestion (غير متزامن)
نضيف stream/consumer خاص بالتوجيه الذكي (منفصل عن campaign queue) لتجنّب التأثير على زمن الرد.

### 2) مصادر الإشارات
- عند حفظ incoming message.
- عند إرسال outgoing message من موظف.
- عند create/assign/pick/resume transfer.
- عند تغيير `user.is_available`.

### 3) التطبيع والخصوصية
قبل الإرسال لـRuVector:
- إخفاء/تنظيف PII من نص الرسائل (أرقام، بريد، معرفات حساسة).
- الاعتماد على `contact_id/user_id/org_id` بدل البيانات الحساسة المباشرة.
- الاحتفاظ ببيانات التعلم 180 يوم ثم pruning دوري.

## منطق اكتشاف الموظف المطلوب والإسناد
### 1) اكتشاف الطلب
- مطابقة `full_name + aliases`.
- ترتيب المطابقة: exact alias ثم exact name ثم fuzzy.
- إذا النتيجة فريدة وعالية الثقة: تأكيد داخلي مباشر.
- إذا ملتبسة: إرسال رسالة تأكيد بأزرار لاختيار الموظف.

### 2) قرار الإسناد
عند `confidence >= 0.75`:
- إذا الموظف المطلوب متاح: إسناد مباشر.
- إذا غير متاح: إنشاء transfer مع `PreferredAgentID` و`PreferredUntil=now+5m`.
- بعد 5 دقائق: fallback تلقائي لأفضل بديل (RuVector recommendation ثم team strategy الحالية).

عند `confidence < 0.75`:
- لا إسناد قسري.
- نستخدم التوجيه الذكي العام أو fallback للآلية الحالية.

### 3) حدود السلوك
- لا نعيد توجيه محادثة نشطة بالفعل مع موظف آخر إلا بسياسة صريحة.
- نحافظ على SLA الحالية، مع تسجيل سبب التوجيه في audit.

## التنفيذ حسب الملفات (Decision-Complete)
- [cmd/whatomate/main.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/cmd/whatomate/main.go): تهيئة `RuvectorClient` وتشغيل workers الخاصة بالتوجيه والـprune.
- [internal/config/config.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/config/config.go): إضافة `RuvectorConfig`.
- [config.example.toml](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/config.example.toml): قسم `[ruvector]`.
- [internal/models/chatbot.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models/chatbot.go): حقول التوجيه في `AgentTransfer`.
- [internal/models/constants.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models/constants.go): مصادر تحويل جديدة.
- [internal/models](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/models): ملفات نماذج `AgentAlias`, `ContactAgentPreference`, `RoutingDecisionAudit`.
- [internal/database/postgres.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/database/postgres.go): تسجيل models الجديدة + indexes.
- [internal/handlers/organization.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/organization.go): حقول إعدادات smart routing قراءة/تحديث.
- [internal/handlers/chatbot_processor.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/chatbot_processor.go): detection + confirmation + route trigger.
- [internal/handlers/agent_transfers.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/agent_transfers.go): إنشاء transfer ذكي + fallback handling.
- [internal/handlers/users.go](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/users.go): endpoints إدارة aliases.
- [docker/docker-compose.yml](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/docker/docker-compose.yml): إضافة `ruvector-router` + `embeddings` + volumes.
- [internal/queue](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/queue): stream/consumer جديد للتوجيه الذكي.

## الاختبارات (Test Cases & Scenarios)
1. `requested agent` متاح: إسناد فوري صحيح وتسجيل source=`requested_agent`.
2. `requested agent` غير متاح: انتظار 5 دقائق ثم fallback تلقائي.
3. اسم ملتبس: إرسال تأكيد، ثم إسناد بناءً على اختيار العميل.
4. ثقة أقل من 0.75: لا فرض، والانتقال للخوارزمية الاعتيادية.
5. تعلّم السلوك: زيادة score لموظف بعد نجاحات متكررة مع نفس العميل.
6. حماية الخصوصية: عدم خروج PII الخام إلى خدمة RuVector.
7. انقطاع `ruvector-router`: النظام يعود تلقائيًا إلى `assignToTeam/queue` بدون توقف الخدمة.
8. pruning بعد 180 يوم يعمل ويقلّص البيانات دون كسر التوجيه.
9. تغطية عربية: تطبيع النص العربي والهمزات/التشكيل في مطابقة الأسماء.

## معايير القبول
- توجيه المحادثة للموظف المطلوب بدقة أعلى من الآلية الحالية مع trace واضح لكل قرار.
- fallback يعمل خلال 5 دقائق وفق الإعداد الافتراضي.
- جميع الفرق تعمل على النظام من أول إطلاق.
- لا تدهور ملحوظ في زمن معالجة الرسائل الواردة.
- إمكانية تعديل السياسة من الإعدادات بدون إعادة نشر.

## الافتراضات والافتراضات الافتراضية المثبتة
- RuVector يُدار كخدمة مستقلة داخل نفس بيئة Docker.
- Embeddings محلية بالكامل (بدون مزود خارجي).
- الإطلاق شامل لكل الفرق مباشرة.
- البيانات المتعلمة محفوظة 180 يوم.
- الاعتماد على `full_name + aliases` كمرجع أسماء الموظفين.
- السياسة الافتراضية `preferred_with_fallback` مع `fallback=5m` و`threshold=0.75`.
