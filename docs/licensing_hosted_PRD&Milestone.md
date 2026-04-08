Product Requirements Document
Whatomate License System — Hosted & Self-Hosted
Version: 1.0 Status: Final Draft Date: 2026-04-08

Executive Summary
Whatomate هو برنامج دردشة تجاري يحتاج إلى نظام ترخيص محكم يدعم نموذجَي نشر مختلفَين تماماً في حدود الثقة والمتطلبات التقنية. الهدف هو بناء بنية ترخيص قابلة للتطوير تحمي الإيرادات دون أن تُعيق تجربة العميل أو تكسر المسار القائم حالياً.

Problem Statement
النظام الحالي يعتمد على تحقق محلي كامل بـ Ed25519 + HWID matching داخل license.go وservice.go وtoken.go. هذا يعمل جيداً للـ self-hosted، لكنه لا يوفر:

تفعيلاً تلقائياً للعملاء الـ hosted
حماية من إساءة استخدام الـ trials
رؤية كاملة على التراخيص الصادرة
آلية لإيقاف أو تعليق hosted instances من الـ control plane
Goals
الهدف الأول هو الحفاظ على المسار القائم للـ self-hosted بدون أي كسر في الواجهات الموجودة، مع إضافة ضوابط تشغيلية تمنع إساءة الاستخدام اليدوي.

الهدف الثاني هو بناء مسار hosted منفصل تماماً يُتيح تفعيلاً تلقائياً للـ trials بدون تدخل بشري، مع حماية كاملة من abuse.

الهدف الثالث هو عزل المفتاح الخاص في خدمة داخلية وحيدة لا تلمسها أي خدمة عامة أو binary عميل.

Non-Goals
لا تعديل على منطق التحقق الأساسي في token.go أو service.go
لا heartbeat أو revocation إلزامي في self-hosted offline
لا TPM أو hardware binding في هذه المرحلة
لا دعم multi-region في هذا الإصدار
لا payment processing مدمج في هذا الإصدار
User Personas
العميل Self-Hosted هو شركة أو فرد يشغّل Whatomate على بنيته التحتية الخاصة، يتوقع تفعيلاً يدوياً بسيطاً ولا يريد اعتماداً إلزامياً على اتصال خارجي دائم.

العميل Hosted هو شركة تشترك في Whatomate كـ SaaS، تتوقع تجربة onboarding فورية بدون خطوات يدوية، وتريد trial مجانية فورية.

Vendor Operations Team هم الفريق الداخلي الذي يُصدر ويراقب ويُلغي التراخيص، يحتاجون رؤية كاملة وأدوات واضحة بدون الوصول المباشر للمفتاح الخاص.

Functional Requirements
Self-Hosted Path
يجب أن يبقى /api/license/bootstrap عاماً ويعيد hwid_full بدون تغيير في السلوك الحالي. يجب أن يبقى /api/license/activate يقبل الـ token ويُطبّق التحقق المحلي الكامل كما هو. يجب إضافة rate limiting على /api/license/activate بحد أقصى 5 محاولات في الساعة لكل IP مع تسجيل كل فشل في audit log يحتوي على hash(token) لا الـ token الخام.

يجب وجود SOP رسمي موثّق يُلزم كل طلب إصدار أو إعادة إصدار يدوي بـ customer_id موجود في قاعدة البيانات مسبقاً، وبريد إلكتروني مسجّل، وسبب موثّق، واسم الموافق، مع عدّاد لإعادة الإصدار يُنبّه عند تجاوز حد معين.

Hosted Path
يجب إضافة endpoint داخلي جديد /internal/license/bootstrap منفصل عن المسار اليدوي، محمي بـ mTLS أو signed internal JWT، يعيد instance_id وhwid_full وhwid_hash وbootstrap_nonce. الـ bootstrap_nonce يكون أحادي الاستخدام بـ TTL لا يتجاوز 5 دقائق ومرتبطاً بـ deployment_id محدد.

يجب أن يدعم الـ Issuer idempotency صريح عبر request_id بحيث يُعيد نفس النتيجة عند تكرار الطلب بنفس الـ request_id خلال 24 ساعة دون إصدار ترخيص ثانٍ.

يجب أن يحمل كل token صادر للـ hosted مسار deployment_id صريح، والـ instance يرفض أي token لا يطابق deployment_id الخاص به.

Private Issuer Service
يجب أن يكون الـ Issuer خدمة داخلية معزولة لا تقبل اتصالات إلا من Provisioning Service عبر network policy صارمة. يجب أن يحمل كل token صادر kid صريح يُحدد المفتاح المُستخدم في التوقيع. يجب تطبيق rate limiting على مستوى الـ Issuer نفسه بحدود واضحة لكل دقيقة ولكل ساعة مع alert فوري عند التجاوز.

يجب تسجيل كل عملية إصدار في external append-only audit log يحتوي على customer_id لا البريد المباشر، وIP مقطوعة، وhwid_hash مع pepper إضافي، مع تشفير at rest.

Key Rotation
يجب أن يدعم النظام current + previous kid في آنٍ واحد. المفتاح القديم يدخل حالة verify-only لمدة grace window ثابتة لا تتجاوز 90 يوماً بغض النظر عن TTL أي token. بعد الـ grace window يتوقف الـ Issuer عن قبول tokens موقّعة بالمفتاح القديم في المسار الـ hosted. في self-hosted offline لا يوجد ضمان تقني بإيقاف binary قديم لا يزال يثق بالمفتاح القديم — هذا قيد معترف به وموثّق.

Non-Functional Requirements
الأمان يتطلب أن يبقى private.key حصرياً في الـ Issuer Service، ولا يُمرَّر أو يُخزَّن في أي خدمة أخرى أو binary عميل أو متغيرات بيئة عامة.

الموثوقية تتطلب أن يتحمل الـ Provisioning Service فشل الـ bootstrap وإعادة المحاولة بدون إصدار تراخيص مكررة، عبر idempotency صريح.

القابلية للمراقبة تتطلب رؤية كاملة على كل ترخيص صادر، ومحاولات تفعيل فاشلة، وأنماط abuse محتملة عبر audit log خارجي.

عدم التأثير على الحالي يتطلب أن تبقى جميع الـ endpoints الموجودة في main.go بنفس سلوكها الحالي بدون أي تغيير breaking.

Constraints & Accepted Limitations
VM Cloning في self-hosted offline لا يمكن منعه تقنياً بدون TPM أو online attestation. install_id يُضاف كطبقة إضافية مستقبلاً لكنه لا يمنع clone بعد التفعيل. الحماية الفعلية هنا تعاقدية لا تقنية.

Key rotation في self-hosted offline لا يمكن فرضها فورياً على binary موجود. الـ grace window سياسة Issuer تشغيلية، ليست ضماناً تقنياً على النسخ القديمة المنشورة.

Revocation و Heartbeat ميزات hosted control plane حصرياً ولا تُضاف كمتطلب لنواة Whatomate.

Success Metrics
النجاح في هذا المشروع يُقاس بثلاثة محاور: الأول هو صفر كسر في مسار self-hosted القائم بعد كل milestone. الثاني هو وصول زمن onboarding للـ hosted trial إلى أقل من دقيقتين من التسجيل حتى الـ instance الجاهز. الثالث هو رؤية 100% على كل ترخيص صادر في audit log قبل إطلاق الـ hosted path.

Milestones
Milestone 1 — Operational Hardening (Self-Hosted)
الهدف: تقوية المسار القائم بدون أي تغيير breaking المدة المقدّرة: أسبوعان

هذا الـ milestone لا يلمس الكود الأساسي — يضيف طبقات حماية فوقه.

النتائج المتوقعة عند الانتهاء:

Rate limiting فعّال على /api/license/activate مع audit log لكل محاولة فاشلة
SOP رسمي موثّق ومُطبَّق على كل طلبات الإصدار اليدوي
Audit log خارجي يستقبل كل أحداث الترخيص مع privacy hardening كامل
لا تغيير في أي endpoint موجود أو سلوك تحقق
Milestone 2 — Private Issuer Service
الهدف: عزل المفتاح الخاص في خدمة داخلية مستقلة المدة المقدّرة: ثلاثة أسابيع

هذا الـ milestone يبني الـ trust anchor للنظام كله.

النتائج المتوقعة عند الانتهاء:

Issuer Service منشورة ومعزولة بـ network policy
توثيق بين الخدمات الداخلية عبر mTLS أو signed internal JWT
Rate limiting على مستوى الـ Issuer مع alerting
kid مدمج في كل token صادر
Idempotency كامل عبر request_id مع 24h cache
الـ private.key لا يوجد في أي مكان آخر
Milestone 3 — Hosted Provisioning Path
الهدف: تفعيل تلقائي كامل للـ hosted trials المدة المقدّرة: ثلاثة أسابيع

هذا الـ milestone يبني المسار الجديد كاملاً بدون تعديل النواة.

النتائج المتوقعة عند الانتهاء:

/internal/license/bootstrap منشور ومحمي ومنفصل عن المسار اليدوي
bootstrap_nonce lifecycle كامل: CREATED → CONSUMED → EXPIRED
deployment_id مدمج في كل token hosted
Provisioning Service تُكمل onboarding كامل بدون تدخل بشري
زمن onboarding أقل من دقيقتين end-to-end
Milestone 4 — Abuse Controls & Observability
الهدف: منع استغلال الـ trials وتوفير رؤية كاملة المدة المقدّرة: أسبوعان

النتائج المتوقعة عند الانتهاء:

Velocity checks فعّالة على email + IP + ASN + domain
Cooldown period مُطبَّق بين trials لنفس الكيان
Admin dashboard داخلي يعرض التراخيص الصادرة وأنماط الـ abuse
Revoke/reissue workflow واضح من الـ dashboard
Trial counts وsuspicious patterns مرئية للـ operations team
Milestone 5 — Hosted Control Plane
الهدف: إدارة دورة حياة الـ hosted instances من الـ control plane المدة المقدّرة: أربعة أسابيع

النتائج المتوقعة عند الانتهاء:

Outbound heartbeat mechanism من الـ instance إلى الـ control plane كل 24 ساعة مع grace period 48 ساعة
Suspension وrevocation فوريان من الـ control plane للـ hosted instances
Key rotation policy مُطبَّقة: current + previous kid مع grace window 90 يوم
Forced renewal للـ hosted instances خلال الـ grace window
توثيق رسمي لقيود self-hosted offline فيما يخص rotation وrevocation
Milestone 6 — Hardening & Future-Proofing
الهدف: إغلاق الثغرات المتبقية وتجهيز النظام للتطوير المستقبلي المدة المقدّرة: أسبوعان

النتائج المتوقعة عند الانتهاء:

install_id مُضاف كطبقة إضافية في self-hosted مع توثيق صريح لحدوده
Key rotation tested end-to-end على كلا المسارين
Runbook كامل لسيناريوهات: تسريب المفتاح، إلغاء اشتراك عميل hosted، طلب HWID migration في self-hosted
Security review نهائي على كل السطح المُضاف
توثيق تعاقدي لقيود VM cloning في self-hosted
