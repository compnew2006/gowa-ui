#!/usr/bin/env python3
"""
Knowledge-base-constrained reply generation for Hermes webhook agents.
"""

from __future__ import annotations

from dataclasses import dataclass
import re
from typing import Callable, Dict, List, Optional, Sequence


UNKNOWN_RESPONSES = {
    "en": "I don't have that specific information. Let me check with our team and get back to you shortly.",
    "ar": "لا أملك هذه المعلومة المحددة حالياً. دعني أتحقق مع فريقنا وأعود إليك قريباً.",
}

GENERIC_PATTERNS = [
    "thank you for reaching out",
    "happy to help",
    "we're here for you",
    "feel free to contact us",
    "feel free to reach out",
    "thanks for reaching out",
    "how can we help you today",
]

QUESTION_KEYWORDS = {
    "pricing": [
        "price",
        "pricing",
        "cost",
        "quote",
        "how much",
        "payment",
        "upfront",
        "delivery",
        "installment",
        "سعر",
        "الاسعار",
        "الأسعار",
        "كم",
        "بكم",
        "تكلفة",
        "دفع",
        "تقسيط",
    ],
    "location": [
        "where",
        "location",
        "address",
        "based",
        "branch",
        "office",
        "meet",
        "visit",
        "مكان",
        "فين",
        "عنوان",
        "موقع",
        "مقر",
        "فرع",
        "مكتب",
        "مقابلة",
        "اجتماع",
    ],
    "hours": [
        "hours",
        "open",
        "close",
        "time",
        "working hours",
        "business hours",
        "مفتوح",
        "مغلق",
        "ساعات",
        "دوام",
        "مواعيد",
        "وقت",
    ],
    "contact": [
        "contact",
        "reach",
        "message",
        "messenger",
        "whatsapp",
        "email",
        "phone",
        "number",
        "call",
        "تواصل",
        "اتصل",
        "راسل",
        "واتساب",
        "ايميل",
        "بريد",
        "هاتف",
        "رقم",
        "ماسنجر",
    ],
    "services": [
        "services",
        "service",
        "offer",
        "provide",
        "build",
        "develop",
        "website",
        "app",
        "application",
        "e-commerce",
        "ecommerce",
        "software",
        "platform",
        "solutions",
        "خدمات",
        "خدمة",
        "تقدم",
        "توفر",
        "تصميم",
        "تطوير",
        "موقع",
        "تطبيق",
        "متجر",
        "برمجيات",
        "حلول",
    ],
}

GREETING_PATTERNS = {
    "ar": [
        "سلام عليكم",
        "السلام عليكم",
        "السلام عليكم ورحمة الله وبركاته",
        "السلام عليكم ورحمه الله وبركاته",
        "اهلا",
        "أهلا",
        "اهلاً",
        "أهلاً",
        "مرحبا",
        "مرحباً",
        "مساء الخير",
        "صباح الخير",
    ],
    "en": [
        "hello",
        "hi",
        "hey",
        "good morning",
        "good evening",
        "good afternoon",
    ],
}

KNOWN_SERVICE_KEYWORDS = {
    "website": ["website", "site", "web", "موقع", "مواقع"],
    "mobile": ["mobile", "app", "application", "ios", "android", "تطبيق", "تطبيقات"],
    "ecommerce": ["e-commerce", "ecommerce", "store", "shop", "متجر", "متاجر"],
    "custom": ["custom", "software", "solution", "حلول", "برمجيات", "مخصص"],
}

GROUNDING_CONCEPT_GROUPS = [
    ("qena", "قنا"),
    ("egypt", "مصر"),
    ("facebook", "فيسبوك"),
    ("messenger", "ماسنجر"),
    ("whatsapp", "واتساب"),
    ("website", "web", "موقع", "مواقع"),
    ("mobile", "app", "application", "تطبيق", "تطبيقات"),
    ("e-commerce", "ecommerce", "store", "shop", "متجر", "متاجر"),
    ("software", "solution", "حلول", "برمجيات"),
    ("saturday", "السبت"),
    ("thursday", "الخميس"),
    ("friday", "الجمعة"),
]

STOPWORDS = {
    "a",
    "an",
    "and",
    "are",
    "as",
    "at",
    "be",
    "for",
    "from",
    "have",
    "in",
    "is",
    "it",
    "of",
    "on",
    "or",
    "our",
    "the",
    "to",
    "we",
    "with",
    "you",
    "your",
    "عن",
    "في",
    "من",
    "على",
    "الى",
    "إلى",
    "مع",
    "هل",
    "ما",
    "هو",
    "هي",
    "يمكن",
    "نحن",
}


@dataclass
class SectionMatch:
    title: str
    content: str


@dataclass
class FaqEntry:
    question: str
    answer: str


@dataclass
class ResponseResult:
    response: str
    language: str
    question_type: str
    found_in_kb: bool
    source_section: str
    kb_snippet: str = ""


def detect_language(message):
    return "ar" if contains_arabic(message) else "en"


def contains_arabic(text):
    return any("\u0600" <= char <= "\u06FF" for char in text)


def count_script_chars(text):
    arabic = sum(1 for char in text if "\u0600" <= char <= "\u06FF")
    latin = sum(1 for char in text if ("a" <= char.lower() <= "z"))
    return arabic, latin


def normalize_text(text):
    return re.sub(r"\s+", " ", text.strip().lower())


def tokenize(text):
    return re.findall(r"[A-Za-z0-9\u0600-\u06FF']+", text.lower())


def is_arabic_token(text):
    return any("\u0600" <= char <= "\u06FF" for char in text)


def clean_markdown_text(text):
    cleaned = text.strip()
    cleaned = re.sub(r"^\s*[-*]\s*", "", cleaned)
    cleaned = cleaned.replace("**", "")
    cleaned = re.sub(r"\s+", " ", cleaned)
    return cleaned.strip()


def split_sentences(text):
    compact = re.sub(r"\s+", " ", text.strip())
    if not compact:
        return []
    parts = re.split(r"(?<=[\.\!\?؟])\s+|\n+", compact)
    return [part.strip() for part in parts if part.strip()]


def limit_words(text, limit=40):
    words = text.split()
    if len(words) <= limit:
        return text.strip()
    return " ".join(words[:limit]).strip().rstrip(".,;:") + "."


def best_answer_excerpt(text):
    sentences = split_sentences(text)
    if not sentences:
        return ""
    excerpt = sentences[0]
    if len(excerpt.split()) <= 2 and len(sentences) > 1:
        excerpt = "{first} {second}".format(first=excerpt, second=sentences[1])
    return limit_words(excerpt)


def canonical_section_name(title):
    lowered = title.strip().lower()
    if lowered.startswith("location"):
        return "location"
    if lowered == "pricing":
        return "pricing"
    if lowered.startswith("business hours"):
        return "hours"
    if lowered.startswith("contact"):
        return "contact"
    if lowered.startswith("services"):
        return "services"
    if lowered.startswith("common questions"):
        return "faq"
    return None


def parse_authoritative_sections(knowledge):
    sections = {}
    current_title = None
    current_lines = []

    for raw_line in knowledge.splitlines():
        if raw_line.startswith("## "):
            if current_title is not None:
                store_section(sections, current_title, current_lines)
            current_title = raw_line[3:].strip()
            current_lines = []
            continue
        if current_title is not None:
            current_lines.append(raw_line.rstrip())

    if current_title is not None:
        store_section(sections, current_title, current_lines)

    return sections


def store_section(sections, title, lines):
    canonical_name = canonical_section_name(title)
    if not canonical_name:
        return
    content = "\n".join(lines).strip()
    if content:
        sections[canonical_name] = SectionMatch(title=title, content=content)


def parse_faq_entries(faq_text):
    entries = []
    current_question = None
    current_answer_lines = []

    for raw_line in faq_text.splitlines():
        line = raw_line.strip()
        if not line:
            continue

        question_match = re.match(r"^\*\*Q:\s*(.+?)\*\*\s*$", line)
        if question_match:
            if current_question and current_answer_lines:
                entries.append(
                    FaqEntry(
                        question=current_question,
                        answer="\n".join(current_answer_lines).strip(),
                    )
                )
            current_question = question_match.group(1).strip()
            current_answer_lines = []
            continue

        if current_question is None:
            continue

        if line.startswith("## "):
            break

        if line.startswith("A:"):
            current_answer_lines.append(line[2:].strip())
        else:
            current_answer_lines.append(line)

    if current_question and current_answer_lines:
        entries.append(FaqEntry(question=current_question, answer="\n".join(current_answer_lines).strip()))

    return entries


def select_answer_language(answer_text, language):
    lines = [clean_markdown_text(line) for line in answer_text.splitlines() if clean_markdown_text(line)]
    if not lines:
        return ""

    if language == "ar":
        arabic_lines = [line for line in lines if count_script_chars(line)[0] > count_script_chars(line)[1]]
        if arabic_lines:
            return " ".join(arabic_lines).strip()
    else:
        english_lines = [line for line in lines if count_script_chars(line)[1] >= count_script_chars(line)[0]]
        if english_lines:
            return " ".join(english_lines).strip()

    return " ".join(lines).strip()


def classify_question(message):
    normalized = normalize_text(message)
    tokens = tokenize(normalized)
    scores = {}

    for question_type, keywords in QUESTION_KEYWORDS.items():
        score = 0
        for keyword in keywords:
            if keyword_matches(keyword, normalized, tokens):
                score += len(keyword.split())
        if score:
            scores[question_type] = score

    if not scores:
        return "greeting" if is_greeting_message(message) else "general"

    ordered_types = ["pricing", "location", "hours", "contact", "services"]
    best_type = max(
        ordered_types,
        key=lambda question_type: (scores.get(question_type, 0), -ordered_types.index(question_type)),
    )
    if scores.get(best_type, 0) == 0:
        return "greeting" if is_greeting_message(message) else "general"
    return best_type


def is_greeting_message(message):
    normalized = normalize_text(message)
    if not normalized:
        return False

    stripped = re.sub(r"[^\w\u0600-\u06FF\s]", " ", normalized)
    compact = " ".join(stripped.split())

    for patterns in GREETING_PATTERNS.values():
        for pattern in patterns:
            normalized_pattern = normalize_text(pattern)
            if compact == normalized_pattern:
                return True

    greeting_hits = 0
    for patterns in GREETING_PATTERNS.values():
        for pattern in patterns:
            if normalize_text(pattern) in compact:
                greeting_hits += 1
                break
    return greeting_hits > 0 and len(tokenize(compact)) <= 6


def keyword_matches(keyword, normalized_text, tokens):
    if " " in keyword:
        return keyword in normalized_text

    if is_arabic_token(keyword):
        if len(keyword) <= 2:
            return keyword in tokens
        return any(keyword in token for token in tokens)

    return keyword in tokens


def find_best_faq_match(message, faq_entries):
    message_tokens = [token for token in tokenize(message) if token not in STOPWORDS]
    if not message_tokens:
        return None

    best_entry = None
    best_score = 0.0

    for entry in faq_entries:
        candidate_tokens = set(token for token in tokenize(entry.question + " " + entry.answer) if token not in STOPWORDS)
        if not candidate_tokens:
            continue

        overlap = sum(1 for token in message_tokens if token in candidate_tokens)
        if overlap == 0:
            continue

        score = float(overlap) / max(len(set(message_tokens)), 1)
        normalized_question = normalize_text(entry.question)
        if normalize_text(message) in normalized_question or normalized_question in normalize_text(message):
            score += 0.5

        if score > best_score:
            best_score = score
            best_entry = entry

    if best_score < 0.25:
        return None
    return best_entry


def unknown_response(language):
    return UNKNOWN_RESPONSES["ar" if language == "ar" else "en"]


def find_line(section_content, keywords):
    for raw_line in section_content.splitlines():
        cleaned = clean_markdown_text(raw_line)
        if not cleaned:
            continue
        lowered = cleaned.lower()
        for keyword in keywords:
            if keyword in lowered:
                return cleaned
    return ""


def find_service_match(message, services_section):
    lines = [clean_markdown_text(line) for line in services_section.splitlines() if clean_markdown_text(line)]
    lowered_message = normalize_text(message)

    for line in lines:
        lowered_line = line.lower()
        if any(keyword in lowered_message for keyword in KNOWN_SERVICE_KEYWORDS["mobile"]) and "mobile" in lowered_line:
            return line
        if any(keyword in lowered_message for keyword in KNOWN_SERVICE_KEYWORDS["ecommerce"]) and (
            "commerce" in lowered_line or "store" in lowered_line
        ):
            return line
        if any(keyword in lowered_message for keyword in KNOWN_SERVICE_KEYWORDS["website"]) and (
            "website" in lowered_line or "web" in lowered_line
        ):
            return line
        if any(keyword in lowered_message for keyword in KNOWN_SERVICE_KEYWORDS["custom"]) and (
            "custom" in lowered_line or "software" in lowered_line or "solution" in lowered_line
        ):
            return line

    return ""


def contains_branch_request(message):
    normalized = normalize_text(message)
    branch_keywords = ["branch", "office", "فرع", "مكتب"]
    return any(keyword in normalized for keyword in branch_keywords)


def contains_specific_contact_number_request(message):
    normalized = normalize_text(message)
    number_keywords = ["phone", "number", "mobile number", "رقم", "هاتف", "تليفون"]
    return any(keyword in normalized for keyword in number_keywords)


def contains_specific_unlisted_service(message):
    normalized = normalize_text(message)
    direct_request_keywords = ["seo", "marketing", "hosting", "سيو", "تسويق", "استضافة"]
    return any(keyword in normalized for keyword in direct_request_keywords)


def build_location_response(message, sections, language, faq_entry):
    if contains_branch_request(message):
        return None

    if faq_entry:
        localized = select_answer_language(faq_entry.answer, language)
        if localized:
            sentence = best_answer_excerpt(localized)
            return ResponseResult(
                response=sentence,
                language=language,
                question_type="location",
                found_in_kb=True,
                source_section="Common Questions & Answers",
                kb_snippet=localized,
            )

    section = sections.get("location")
    if not section:
        return None

    normalized = normalize_text(message)
    if any(keyword in normalized for keyword in ["address", "عنوان", "office", "مكتب"]):
        if language == "ar":
            response = "عنواننا في قنا، صعيد مصر، جمهورية مصر العربية، والاجتماعات الشخصية متاحة في قنا بموعد مسبق."
        else:
            response = "Our address is Qena, Upper Egypt, Arab Republic of Egypt, and in-person meetings in Qena are available by appointment."
    else:
        if language == "ar":
            response = "نحن موجودون في قنا، مصر، ونعمل عن بُعد مع العملاء في جميع أنحاء مصر والعالم."
        else:
            response = "We're based in Qena, Egypt, and we work remotely with clients across Egypt and worldwide."

    return ResponseResult(
        response=limit_words(response),
        language=language,
        question_type="location",
        found_in_kb=True,
        source_section=section.title,
        kb_snippet=section.content,
    )


def build_pricing_response(message, sections, language, faq_entry):
    section = sections.get("pricing")
    if not section:
        return None

    normalized = normalize_text(message)

    if any(keyword in normalized for keyword in ["payment", "upfront", "delivery", "تقسيط", "دفع"]):
        if faq_entry:
            localized = select_answer_language(faq_entry.answer, language)
            if localized:
                sentence = best_answer_excerpt(localized)
                return ResponseResult(
                    response=sentence,
                    language=language,
                    question_type="pricing",
                    found_in_kb=True,
                    source_section="Common Questions & Answers",
                    kb_snippet=localized,
                )

        if language == "ar":
            response = "طريقة الدفع هي 50% مقدماً و50% عند التسليم."
        else:
            response = "Payment is 50% upfront and 50% on delivery."
        return ResponseResult(
            response=response,
            language=language,
            question_type="pricing",
            found_in_kb=True,
            source_section=section.title,
            kb_snippet=section.content,
        )

    if any(keyword in normalized for keyword in ["e-commerce", "ecommerce", "store", "shop", "متجر"]):
        if language == "ar":
            response = "المتاجر الإلكترونية تبدأ من 15,000 جنيه مصري."
        else:
            response = "E-commerce starts from 15,000 EGP."
    elif any(keyword in normalized for keyword in ["website", "site", "web", "موقع"]):
        if language == "ar":
            response = "تصميم المواقع يبدأ من 5,000 جنيه مصري."
        else:
            response = "Website design starts from 5,000 EGP."
    else:
        if language == "ar":
            response = "تصميم المواقع يبدأ من 5,000 جنيه مصري، والمتاجر الإلكترونية تبدأ من 15,000 جنيه مصري."
        else:
            response = "Website design starts from 5,000 EGP, and e-commerce starts from 15,000 EGP."

    return ResponseResult(
        response=limit_words(response),
        language=language,
        question_type="pricing",
        found_in_kb=True,
        source_section=section.title,
        kb_snippet=section.content,
    )


def build_hours_response(sections, language):
    section = sections.get("hours")
    if not section:
        return None

    if language == "ar":
        response = "ساعات العمل من السبت إلى الخميس من 9 صباحاً إلى 9 مساءً، والجمعة مغلق."
    else:
        response = "Our hours are Saturday to Thursday, 9:00 AM to 9:00 PM. Friday is closed."

    return ResponseResult(
        response=limit_words(response),
        language=language,
        question_type="hours",
        found_in_kb=True,
        source_section=section.title,
        kb_snippet=section.content,
    )


def build_contact_response(message, sections, language):
    section = sections.get("contact")
    if not section:
        return None

    if contains_specific_contact_number_request(message):
        return None

    if language == "ar":
        response = "يمكنك التواصل معنا عبر صفحة فيسبوك أو ماسنجر أو واتساب، وماسنجر متاح 24/7."
    else:
        response = "You can reach us via our Facebook page, Messenger, or WhatsApp. Messenger is available 24/7."

    return ResponseResult(
        response=limit_words(response),
        language=language,
        question_type="contact",
        found_in_kb=True,
        source_section=section.title,
        kb_snippet=section.content,
    )


def build_services_response(message, sections, language):
    section = sections.get("services")
    if not section:
        return None

    if contains_specific_unlisted_service(message):
        return None

    matched_service = find_service_match(message, section.content)
    if matched_service:
        lowered = matched_service.lower()
        if "mobile" in lowered:
            response = (
                "نعم، نقدم تطوير تطبيقات الموبايل لـ iOS وAndroid."
                if language == "ar"
                else "Yes, we provide mobile app development for iOS and Android."
            )
        elif "commerce" in lowered or "store" in lowered:
            response = (
                "نعم، نقدم تطوير المتاجر الإلكترونية."
                if language == "ar"
                else "Yes, we provide e-commerce platform development."
            )
        elif "website" in lowered or "web" in lowered:
            response = (
                "نعم، نقدم تصميم وتطوير المواقع."
                if language == "ar"
                else "Yes, we provide website design and development."
            )
        else:
            response = (
                "نعم، نقدم حلول برمجية مخصصة."
                if language == "ar"
                else "Yes, we provide custom software solutions."
            )
    else:
        response = (
            "نقدم تصميم المواقع، تطبيقات الموبايل، المتاجر الإلكترونية، والحلول البرمجية المخصصة."
            if language == "ar"
            else "We offer website design, mobile apps, e-commerce platforms, and custom software solutions."
        )

    return ResponseResult(
        response=limit_words(response),
        language=language,
        question_type="services",
        found_in_kb=True,
        source_section=section.title,
        kb_snippet=section.content,
    )


def token_overlap_ratio(response, kb_snippet):
    response_tokens = set(token for token in tokenize(response) if token not in STOPWORDS)
    snippet_tokens = set(token for token in tokenize(kb_snippet) if token not in STOPWORDS)
    if not response_tokens or not snippet_tokens:
        return 0.0
    return float(len(response_tokens & snippet_tokens)) / float(len(response_tokens))


def has_shared_digits(response, kb_snippet):
    response_digits = set(re.findall(r"\d+", response))
    snippet_digits = set(re.findall(r"\d+", kb_snippet))
    return bool(response_digits & snippet_digits)


def has_grounding_concept_overlap(response, kb_snippet):
    normalized_response = normalize_text(response)
    normalized_snippet = normalize_text(kb_snippet)

    for concept_group in GROUNDING_CONCEPT_GROUPS:
        if any(term in normalized_response for term in concept_group) and any(
            term in normalized_snippet for term in concept_group
        ):
            return True
    return False


def is_generic_response(response):
    normalized = normalize_text(response)
    return any(pattern in normalized for pattern in GENERIC_PATTERNS)


def validate_response(candidate, kb_snippet):
    if not candidate.strip():
        return False
    if len(candidate.split()) > 40:
        return False
    if is_generic_response(candidate):
        return False
    if candidate in UNKNOWN_RESPONSES.values():
        return False
    if kb_snippet and token_overlap_ratio(candidate, kb_snippet) == 0.0:
        if not has_shared_digits(candidate, kb_snippet) and not has_grounding_concept_overlap(candidate, kb_snippet):
            return False
    return True


def build_llm_prompt(page_name, language, customer_message, kb_snippet):
    fallback = unknown_response(language)
    return (
        "You are a customer service agent for {page_name}. "
        "Answer using ONLY the provided knowledge snippet.\n\n"
        "Rules:\n"
        "1. Search the snippet first, then answer.\n"
        "2. Do not add facts that are not in the snippet.\n"
        "3. Keep the answer under 40 words.\n"
        "4. Match the customer's language.\n"
        "5. If the snippet does not answer the question exactly, reply with: {fallback}\n"
        "6. Do not output analysis, markdown headings, or generic filler.\n\n"
        "Knowledge snippet:\n"
        "{kb_snippet}\n\n"
        "Customer message: {customer_message}\n"
        "Customer-facing response:"
    ).format(
        page_name=page_name,
        fallback=fallback,
        kb_snippet=kb_snippet,
        customer_message=customer_message,
    )


def build_general_response(message, page_name, language, faq_entry, llm_runner, enable_llm_rephrase):
    if not faq_entry:
        return None

    localized_answer = select_answer_language(faq_entry.answer, language)
    if not localized_answer:
        return None

    fallback_sentence = best_answer_excerpt(localized_answer)
    result = ResponseResult(
        response=fallback_sentence,
        language=language,
        question_type="general",
        found_in_kb=True,
        source_section="Common Questions & Answers",
        kb_snippet=localized_answer,
    )

    if not enable_llm_rephrase or llm_runner is None:
        return result

    prompt = build_llm_prompt(page_name, language, message, localized_answer)
    candidate = llm_runner(prompt, message)
    if candidate:
        cleaned_candidate = limit_words(clean_markdown_text(candidate))
        if validate_response(cleaned_candidate, localized_answer):
            result.response = cleaned_candidate
    return result


def should_prefer_faq_response(message, question_type, faq_entry, sections):
    if not faq_entry:
        return False

    normalized = normalize_text(message)
    if question_type == "services":
        services_section = sections.get("services")
        if not services_section or not find_service_match(message, services_section.content):
            return True

    if question_type == "hours" and any(keyword in normalized for keyword in ["response time", "respond", "رد", "الرد"]):
        return True

    return False


def build_response(message, knowledge, page_name, llm_runner=None, enable_llm_rephrase=False):
    language = detect_language(message)
    question_type = classify_question(message)

    if not knowledge.strip():
        return ResponseResult(
            response=unknown_response(language),
            language=language,
            question_type=question_type,
            found_in_kb=False,
            source_section="",
        )

    sections = parse_authoritative_sections(knowledge)
    faq_entries = parse_faq_entries(sections.get("faq").content if sections.get("faq") else "")
    faq_entry = find_best_faq_match(message, faq_entries)

    if should_prefer_faq_response(message, question_type, faq_entry, sections):
        result = build_general_response(
            message,
            page_name,
            language,
            faq_entry,
            llm_runner,
            enable_llm_rephrase,
        )
        if result is not None:
            return result

    if question_type == "location":
        result = build_location_response(message, sections, language, faq_entry)
    elif question_type == "pricing":
        result = build_pricing_response(message, sections, language, faq_entry)
    elif question_type == "hours":
        result = build_hours_response(sections, language)
    elif question_type == "contact":
        result = build_contact_response(message, sections, language)
    elif question_type == "services":
        result = build_services_response(message, sections, language)
    else:
        result = build_general_response(
            message,
            page_name,
            language,
            faq_entry,
            llm_runner,
            enable_llm_rephrase,
        )

    if result is None:
        return ResponseResult(
            response=unknown_response(language),
            language=language,
            question_type=question_type,
            found_in_kb=False,
            source_section="",
        )

    if not validate_response(result.response, result.kb_snippet):
        if result.found_in_kb and result.kb_snippet:
            deterministic_fallback = best_answer_excerpt(result.kb_snippet)
            if validate_response(deterministic_fallback, result.kb_snippet):
                result.response = deterministic_fallback
                return result
        return ResponseResult(
            response=unknown_response(language),
            language=language,
            question_type=question_type,
            found_in_kb=False,
            source_section=result.source_section,
            kb_snippet=result.kb_snippet,
        )

    return result
