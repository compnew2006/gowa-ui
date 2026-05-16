import pytest


@pytest.mark.parametrize(
    "message,expected",
    [
        ("How much for a website?", "Website design starts from 5,000 EGP."),
        ("كم سعر تصميم موقع؟", "تصميم المواقع يبدأ من 5,000 جنيه مصري."),
        ("Where are you located?", "We're based in Qena, Egypt (قنا، مصر)."),
        ("ما هو عنوانكم؟", "مكتبنا الرئيسي في قنا، صعيد مصر."),
        ("What are your working hours?", "Our hours are Saturday to Thursday, 9:00 AM to 9:00 PM. Friday is closed."),
        ("كيف أتواصل معكم؟", "يمكنك التواصل معنا عبر صفحة فيسبوك أو ماسنجر أو واتساب، وماسنجر متاح 24/7."),
        ("Do you build mobile apps?", "Yes, we provide mobile app development for iOS and Android."),
        ("هل تقدمون تطبيقات موبايل؟", "نعم، نقدم تطوير تطبيقات الموبايل لـ iOS وAndroid."),
    ],
)
def test_primary_category_answers(runtime_env, message, expected):
    result = runtime_env.responder.build_response(
        message,
        runtime_env.knowledge_text,
        "Ofuqalmadenah",
    )
    assert result.response == expected
    assert result.found_in_kb is True


@pytest.mark.parametrize(
    "message,expected",
    [
        (
            "Do you have a branch in Alexandria?",
            "I don't have that specific information. Let me check with our team and get back to you shortly.",
        ),
        (
            "هل لديكم فرع في الإسكندرية؟",
            "لا أملك هذه المعلومة المحددة حالياً. دعني أتحقق مع فريقنا وأعود إليك قريباً.",
        ),
    ],
)
def test_unknown_questions_return_exact_fallback(runtime_env, message, expected):
    result = runtime_env.responder.build_response(
        message,
        runtime_env.knowledge_text,
        "Ofuqalmadenah",
    )
    assert result.response == expected
    assert result.found_in_kb is False


def test_missing_knowledge_does_not_return_greeting(runtime_env):
    result = runtime_env.responder.build_response(
        "How much for a website?",
        "",
        "Ofuqalmadenah",
    )
    assert result.response == "I don't have that specific information. Let me check with our team and get back to you shortly."
    assert "welcome" not in result.response.lower()
    assert "thank you" not in result.response.lower()


def test_customer_interaction_section_is_ignored(runtime_env):
    polluted_knowledge = (
        runtime_env.knowledge_text
        + "\n\n## Customer Interaction\n\n- Website Design: Starting from 999 EGP\n"
    )
    result = runtime_env.responder.build_response(
        "How much for a website?",
        polluted_knowledge,
        "Ofuqalmadenah",
    )
    assert result.response == "Website design starts from 5,000 EGP."


def test_llm_unavailable_still_returns_grounded_faq_answer(runtime_env, monkeypatch):
    monkeypatch.setenv("HERMES_ENABLE_LLM_REPHRASE", "1")
    monkeypatch.setattr(runtime_env.app_module.shutil, "which", lambda _binary: None)

    result = runtime_env.app_module.build_reply_result(
        runtime_env.page_id,
        "Do you provide ongoing support?",
    )

    assert result.response == "Yes! We provide ongoing support after project delivery to ensure everything runs smoothly."
    assert result.found_in_kb is True


def test_response_time_uses_faq_not_hours_template(runtime_env):
    result = runtime_env.responder.build_response(
        "What is your response time?",
        runtime_env.knowledge_text,
        "Ofuqalmadenah",
    )
    assert result.response == "We typically respond within 2 hours during business hours (9 AM - 9 PM, Saturday - Thursday)."
    assert result.question_type == "general"
