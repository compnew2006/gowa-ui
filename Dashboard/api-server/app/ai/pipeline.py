"""
LangGraph-based AI pipeline — PRD v2.0 compliant.

Pipeline nodes:
  detect_language → classify_intent → analyze_sentiment →
  check_rules → search_knowledge_base → guardrails →
  generate_reply → route_action

Confidence Threshold System:
  > 0.85  → Auto-reply
  0.60-0.85 → Reply + flag for admin review
  < 0.60  → Safe default reply + escalate
  = 0.0   → "سيتواصل معك فريقنا قريباً" + admin alert

Priority Routing:
  Angry/Refund/Legal → Priority 1 (high)
  Purchase/Price     → Priority 2 (normal)
  General/Compliment → Priority 3 (low)
"""
from __future__ import annotations
import time
import logging
import re
from typing import Any, Optional

from app.ai.llm import get_llm
from app.ai.dspy_optimizer import get_optimizer

logger = logging.getLogger(__name__)

MAX_COMMENT_LENGTH = 5000
TRUNCATE_TO_LENGTH = 2000

# Confidence thresholds (from PRD 4.8) - defaults, overridden by Settings
CONF_AUTO_REPLY = 0.85
CONF_FLAG_REVIEW = 0.60


def _get_thresholds() -> tuple[float, float]:
    """Get confidence thresholds from Settings, falling back to module constants."""
    try:
        from app.config import get_settings
        settings = get_settings()
        return settings.confidence_auto_reply, settings.confidence_flag_review
    except Exception:
        return CONF_AUTO_REPLY, CONF_FLAG_REVIEW

# Guardrails: topics that trigger immediate escalation
GUARDRAIL_PATTERNS = {
    "legal": ["محامي", "قانون", "مقاضاة", "شكوى رسمية", "lawyer", "legal", "sue", "court"],
    "medical": ["طبيب", "مستشفى", "مرض", "doctor", "hospital", "medical", "injury"],
    "refund": ["استرجاع", "استرداد", "فلوس", "refund", "money back", "chargeback"],
    "abuse": ["غبي", "احتيال", "نصاب", "كذاب", "scam", "fraud", "fake", "liar"],
    "payment": ["بنك", "فيزا", "تحويل", "bank", "visa", "transfer", "payment failed"],
}

INTENT_PRIORITY = {
    "complaint": "high",
    "refund": "high",
    "purchase": "normal",
    "price_inquiry": "normal",
    "compliment": "low",
    "general": "low",
}

# Default fallback replies (will be overridden by Settings from DB)
SAFE_DEFAULT_REPLY_AR = "شكراً لتواصلك معنا. سيتواصل معك أحد ممثلي خدمة العملاء في أقرب وقت ممكن."
SAFE_DEFAULT_REPLY_EN = "Thank you for reaching out. A customer service representative will contact you shortly."


# Regex for emoji detection
_EMOJI_RE = re.compile(
    "["
    "\U0001F600-\U0001F64F"  # emoticons
    "\U0001F300-\U0001F5FF"  # symbols & pictographs
    "\U0001F680-\U0001F6FF"  # transport & map symbols
    "\U0001F1E0-\U0001F1FF"  # flags
    "\U00002702-\U000027B0"  # dingbats
    "\U000024C2-\U0001F251"
    "\U0001f900-\U0001f9FF"  # supplemental symbols
    "\U0001fa00-\U0001fa6F"  # chess symbols
    "\U0001fa70-\U0001faFF"  # symbols extended-A
    "\U00002600-\U000026FF"  # misc symbols
    "\U0000FE00-\U0000FE0F"  # variation selectors
    "\U0000200D"             # zero-width joiner
    "]+",
    flags=re.UNICODE,
)


def _is_emoji_only(text: str) -> bool:
    """Return True if the text contains only emoji and whitespace characters."""
    stripped = _EMOJI_RE.sub("", text).strip()
    return len(stripped) == 0 and len(text.strip()) > 0


def _preprocess_comment(text: str, page_id: str = "") -> tuple:
    """
    Preprocess comment text, handling edge cases.
    Returns (processed_text, list_of_warnings).
    """
    warnings = []
    if not text or not text.strip():
        warnings.append("empty_comment")
        return "", warnings
    if len(text) > MAX_COMMENT_LENGTH:
        original_len = len(text)
        text = text[:TRUNCATE_TO_LENGTH] + f"\n\n[Comment truncated: {original_len} -> {TRUNCATE_TO_LENGTH} characters]"
        warnings.append(f"long_comment_truncated:{original_len}")
        logger.info("[Pipeline] Long comment truncated: %d -> %d chars, page=%s", original_len, TRUNCATE_TO_LENGTH, page_id)
    return text, warnings


try:
    from langgraph.graph import StateGraph, END
    from typing import TypedDict

    class CommentState(TypedDict):
        comment_text: str
        page_id: str
        language: str
        intent: Optional[str]
        sentiment: Optional[str]
        urgency: str                      # normal | urgent
        confidence: float
        kb_results: list[dict]
        guardrail_triggered: bool
        guardrail_reason: Optional[str]
        ai_reply: Optional[str]
        action: str                       # reply | escalate | flag_review
        priority: str                     # high | normal | low
        escalation_reason: Optional[str]
        matched_rule_id: Optional[str]
        processing_time_ms: int
        _safe_reply: str
        _default_language: str
        _warnings: list[str]

    # ─────────────────────────────────────────────
    # Node 1: Language Detection
    # ─────────────────────────────────────────────
    async def detect_language(state: CommentState) -> CommentState:
        text = state["comment_text"]
        default_lang = state.get("_default_language", "ar")

        if not text or not text.strip():
            state["language"] = default_lang
            return state

        if _is_emoji_only(text):
            state["language"] = default_lang
            logger.info("[Pipeline] Emoji-only comment, defaulting to page language: %s", default_lang)
            return state

        arabic_chars = sum(1 for c in text if "\u0600" <= c <= "\u06ff")
        total = max(len(text), 1)
        ratio = arabic_chars / total

        if ratio > 0.2:
            state["language"] = "ar"
        else:
            latin_chars = sum(1 for c in text if ("a" <= c.lower() <= "z"))
            if latin_chars / total > 0.3:
                state["language"] = "en"
            else:
                state["language"] = "other"
                logger.info("[Pipeline] Non-Arabic/Non-English text, using English fallback")
        return state

    # ─────────────────────────────────────────────
    # Node 2: Intent Classification
    # ─────────────────────────────────────────────
    async def classify_intent(state: CommentState) -> CommentState:
        optimizer = get_optimizer()
        dspy_intent = optimizer.get_optimized_intent(state["comment_text"])
        valid_intents = {"price_inquiry", "purchase", "complaint", "refund", "compliment", "general"}

        if dspy_intent and dspy_intent in valid_intents:
            state["intent"] = dspy_intent
            state["priority"] = INTENT_PRIORITY.get(dspy_intent, "normal")
            return state

        llm = await get_llm()
        prompt = (
            "Classify the intent of this comment into exactly one of: "
            "price_inquiry, purchase, complaint, refund, compliment, general.\n"
            f"Comment: {state['comment_text']}\n"
            "Respond with just the intent label, nothing else."
        )
        try:
            resp = await llm.ainvoke(prompt)
            raw = resp.content.strip().lower()
            state["intent"] = raw if raw in valid_intents else "general"
        except Exception:
            state["intent"] = "general"

        state["priority"] = INTENT_PRIORITY.get(state["intent"], "normal")
        return state

    # ─────────────────────────────────────────────
    # Node 3: Sentiment Analysis (independent layer)
    # ─────────────────────────────────────────────
    async def analyze_sentiment(state: CommentState) -> CommentState:
        optimizer = get_optimizer()
        dspy_sentiment, dspy_conf = optimizer.get_optimized_sentiment(state["comment_text"])
        valid = {"positive", "neutral", "negative", "angry", "urgent"}

        if dspy_sentiment and dspy_sentiment in valid:
            state["sentiment"] = dspy_sentiment
            state["confidence"] = max(state["confidence"], dspy_conf)
        else:
            llm = await get_llm()
            prompt = (
                "Analyze the sentiment of this customer comment. "
                "Choose exactly one of: positive, neutral, negative, angry, urgent.\n"
                "- urgent: time-sensitive issue requiring immediate attention\n"
                "- angry: frustrated or using harsh language\n"
                f"Comment: {state['comment_text']}\n"
                "Respond with just the sentiment label."
            )
            try:
                resp = await llm.ainvoke(prompt)
                raw = resp.content.strip().lower()
                state["sentiment"] = raw if raw in valid else "neutral"
            except Exception:
                state["sentiment"] = "neutral"

        # Determine urgency
        state["urgency"] = "urgent" if state["sentiment"] in ("urgent", "angry") else "normal"

        # Escalate immediately for angry — boost priority
        if state["sentiment"] == "angry":
            state["priority"] = "high"

        return state

    # Node 4: Rules Engine Check
    # ─────────────────────────────────────────────
    async def check_rules(state: CommentState, db: Any = None) -> CommentState:
        from app.services.rules_engine import evaluate_rules_from_db, apply_rule_to_pipeline
        context = {
            "intent": state.get("intent"),
            "sentiment": state.get("sentiment"),
            "confidence": state.get("confidence"),
            "language": state.get("language"),
            "comment_text": state.get("comment_text", ""),
            "customer_churn_risk": None,
            "urgency": state.get("urgency"),
            "priority": state.get("priority"),
            "guardrail_triggered": state.get("guardrail_triggered"),
        }
        rule_match = await evaluate_rules_from_db(
            page_id=state.get("page_id", ""),
            context=context,
            db=db,
        )
        if rule_match:
            ai_result = dict(state)
            ai_result = await apply_rule_to_pipeline(rule_match, ai_result)
            state["action"] = ai_result.get("action", state["action"])
            state["priority"] = ai_result.get("priority", state["priority"])
            state["escalation_reason"] = ai_result.get("escalation_reason")
            state["matched_rule_id"] = ai_result.get("matched_rule_id")
            if ai_result.get("ai_reply") is not None:
                state["ai_reply"] = ai_result["ai_reply"]
        return state

        # ─────────────────────────────────────────────
    # Node 5: RAG — Knowledge Base Search
    # ─────────────────────────────────────────────
    async def search_knowledge_base(state: CommentState, db: Any = None) -> CommentState:
        state["kb_empty"] = False
        if db is None:
            state["kb_results"] = []
            state["kb_empty"] = True
            return state
        try:
            from sqlalchemy import select, or_
            from app.db import KnowledgeBase
            
            # Search for page-specific knowledge OR global knowledge (page_id is null)
            # Prioritize page-specific knowledge
            stmt = (
                select(KnowledgeBase)
                .where(
                    KnowledgeBase.is_active == True,
                    or_(KnowledgeBase.page_id == state["page_id"], KnowledgeBase.page_id == None)
                )
            )
            
            # If intent is known, prioritize matching categories
            if state.get("intent"):
                # Map intents to categories
                intent_map = {
                    "price_inquiry": ["pricing", "services"],
                    "purchase": ["pricing", "services", "location"],
                    "complaint": ["faq", "hours"],
                }
                target_categories = intent_map.get(state["intent"], [])
                if target_categories:
                    # Boost target categories by ordering them first
                    from sqlalchemy import case
                    stmt = stmt.order_by(
                        case(
                            (KnowledgeBase.category.in_(target_categories), 0),
                            else_=1
                        ),
                        KnowledgeBase.usage_count.desc()
                    )
                else:
                    stmt = stmt.order_by(KnowledgeBase.usage_count.desc())
            else:
                stmt = stmt.order_by(KnowledgeBase.usage_count.desc())

            result = await db.execute(stmt.limit(5))
            entries = result.scalars().all()
            state["kb_results"] = [
                {"question": e.question, "answer": e.answer, "category": e.category}
                for e in entries
            ]
        except Exception:
            state["kb_results"] = []
        return state

    # ─────────────────────────────────────────────
    # Node 6: Guardrails Layer
    # ─────────────────────────────────────────────
    async def check_guardrails(state: CommentState) -> CommentState:
        text_lower = state["comment_text"].lower()
        state["guardrail_triggered"] = False
        state["guardrail_reason"] = None

        for category, keywords in GUARDRAIL_PATTERNS.items():
            if any(kw in text_lower for kw in keywords):
                state["guardrail_triggered"] = True
                state["guardrail_reason"] = f"Guardrail: {category} topic detected"
                state["priority"] = "high"
                logger.info(
                    "[Guardrails] Triggered: %s | page=%s",
                    category, state["page_id"]
                )
                break

        return state

    # ─────────────────────────────────────────────
    # Node 7: Response Generation
    # ─────────────────────────────────────────────
    async def generate_reply(state: CommentState, settings: Any = None) -> CommentState:
        # If guardrail triggered -- use safe default, don't call LLM
        if state.get("guardrail_triggered"):
            lang = state.get("language", "ar")
            if lang == "ar":
                safe_reply = SAFE_DEFAULT_REPLY_AR
            elif settings:
                safe_reply = settings.safe_reply_en
            else:
                safe_reply = SAFE_DEFAULT_REPLY_EN
            state["ai_reply"] = safe_reply
            state["confidence"] = 0.0
            return state

        # If a rule already set a skip action, bypass generation
        if state.get("action") == "skip":
            state["ai_reply"] = None
            state["confidence"] = 0.0
            return state

        # If a rule already provided a custom reply, use it
        if state.get("ai_reply") and state.get("matched_rule_id"):
            return state

        llm = await get_llm()
        context = "\n".join(
            f"Q: {r['question']}\nA: {r['answer']}" for r in state["kb_results"][:3]
        )

        # Handle empty KB
        if state.get("kb_empty") or not context or len(context.strip()) < 10:
            context = "No specific knowledge base entries available. Use general customer service best practices."
            logger.info("[Pipeline] No KB context available, using generic responses")

        lang = state.get("language", "ar")
        if lang == "ar":
            lang_instruction = "Arabic (formal Egyptian dialect)"
        elif lang == "other":
            lang_instruction = "English (fallback -- customer language not supported)"
        else:
            lang_instruction = "English"
        sentiment = state.get("sentiment", "neutral")
        intent = state.get("intent", "general")

        tone_guide = {
            "angry": "Be extra empathetic and apologetic. Acknowledge their frustration first.",
            "negative": "Be understanding and reassuring. Offer to help resolve the issue.",
            "urgent": "Be prompt and direct. Prioritize resolving their concern immediately.",
            "positive": "Be warm and encourage further engagement.",
        }.get(sentiment, "Be professional and helpful.")

        prompt = (
            f"You are a professional customer service agent for an Arabic social media business.\n"
            f"Reply in {lang_instruction}.\n"
            f"Customer intent: {intent} | Sentiment: {sentiment}\n"
            f"Tone guide: {tone_guide}\n\n"
            f"Knowledge base:\n{context}\n\n"
            f"Customer comment: {state['comment_text']}\n\n"
            f"Write an extremely concise and brief reply (exactly 1 sentence maximum, keep it short, friendly, and sweet). Do not mention AI or automation."
        )
        try:
            # Try optimized reply first (Learning Loop)
            optimizer = get_optimizer()
            optimized_reply = optimizer.get_optimized_reply(
                comment=state["comment_text"],
                context=context,
                intent=intent,
                sentiment=sentiment
            )
            
            if optimized_reply:
                state["ai_reply"] = optimized_reply
                state["confidence"] = 0.95 # Higher confidence for optimized replies
                logger.info("[Pipeline] Used DSPy optimized reply")
                return state

            # Fallback to standard LLM prompt
            resp = await llm.ainvoke(prompt)
            reply = resp.content.strip()
            state["ai_reply"] = reply
            # Confidence based on KB context availability
            if context and len(context) > 50:
                state["confidence"] = 0.88 if state["kb_results"] else 0.65
            else:
                state["confidence"] = 0.62
        except Exception as e:
            logger.error(
                "[Pipeline] LLM generation failed: %s | page=%s | comment='%s'",
                e, state.get("page_id"), state.get("comment_text", "")[:100],
            )
            state["ai_reply"] = None
            state["confidence"] = 0.0

            # Record failure metric
            try:
                from app.metrics import PROMETHEUS_AVAILABLE
                if PROMETHEUS_AVAILABLE:
                    from app.metrics import ai_action_total
                    ai_action_total.labels(action="llm_failure").inc()
            except Exception:
                pass

            # Admin notification
            try:
                from app.services.notifications import notify_admin
                import asyncio
                asyncio.create_task(notify_admin(
                    title="[CRITICAL] AI Pipeline LLM Failure",
                    message=(
                        f"All LLM providers failed to generate a reply.\n"
                        f"Page: {state.get('page_id', 'unknown')}\n"
                        f"Error: {str(e)[:200]}\n"
                        f"Comment: {state.get('comment_text', '')[:100]}"
                    ),
                    priority="critical",
                ))
            except Exception:
                pass

        return state

    # ─────────────────────────────────────────────
    # Node 8: Route Action (Confidence Threshold System)
    # ─────────────────────────────────────────────
    async def route_action(state: CommentState) -> CommentState:
        conf_auto_reply, conf_flag_review = _get_thresholds()
        sentiment = state.get("sentiment", "neutral")
        confidence = state.get("confidence", 0.0)
        guardrail = state.get("guardrail_triggered", False)

        # If a rule already set skip, respect it
        if state.get("action") == "skip":
            return state

        # Immediate escalation conditions (PRD §4.12)
        if guardrail:
            state["action"] = "escalate"
            state["escalation_reason"] = state.get("guardrail_reason", "Guardrail triggered")

        elif sentiment == "angry":
            state["action"] = "escalate"
            state["escalation_reason"] = "Angry customer — immediate escalation required"

        elif state.get("intent") in ("refund",):
            state["action"] = "escalate"
            state["escalation_reason"] = "Refund request requires human review"

        elif confidence == 0.0 or not state.get("ai_reply"):
            # All LLMs failed — use safe fallback
            lang = state.get("language", "ar")
            safe_reply = state.get("_safe_reply", SAFE_DEFAULT_REPLY_AR if lang in ("ar", "other") else SAFE_DEFAULT_REPLY_EN)
            state["ai_reply"] = safe_reply
            state["action"] = "escalate"
            state["escalation_reason"] = "LLM unavailable — safe default reply used"

        elif confidence < conf_flag_review:
            # Low confidence → escalate
            state["action"] = "escalate"
            state["escalation_reason"] = f"Low confidence score ({confidence:.0%} < {conf_flag_review:.0%})"

        elif confidence < conf_auto_reply:
            # Medium confidence → reply but flag for admin review
            state["action"] = "flag_review"
            state["escalation_reason"] = None

        else:
            # High confidence → auto-reply
            state["action"] = "reply"
            state["escalation_reason"] = None

        return state

    # ─────────────────────────────────────────────
    # Build the graph
    # ─────────────────────────────────────────────
    def _build_pipeline():
        graph: StateGraph = StateGraph(CommentState)
        graph.add_node("detect_language", detect_language)
        graph.add_node("classify_intent", classify_intent)
        graph.add_node("analyze_sentiment", analyze_sentiment)
        graph.add_node("check_rules", check_rules)
        graph.add_node("search_knowledge_base", search_knowledge_base)
        graph.add_node("check_guardrails", check_guardrails)
        graph.add_node("generate_reply", generate_reply)
        graph.add_node("route_action", route_action)

        graph.set_entry_point("detect_language")
        graph.add_edge("detect_language", "classify_intent")
        graph.add_edge("classify_intent", "analyze_sentiment")
        graph.add_edge("analyze_sentiment", "check_rules")
        graph.add_edge("check_rules", "search_knowledge_base")
        graph.add_edge("search_knowledge_base", "check_guardrails")
        graph.add_edge("check_guardrails", "generate_reply")
        graph.add_edge("generate_reply", "route_action")
        graph.add_edge("route_action", END)

        return graph.compile()

    _pipeline = _build_pipeline()
    _LANGGRAPH_AVAILABLE = True
    logger.info("[Pipeline] LangGraph pipeline built (8 nodes)")

except ImportError:
    _LANGGRAPH_AVAILABLE = False
    _pipeline = None
    logger.warning("[Pipeline] LangGraph not available — using fallback")


async def process_comment(comment: str, page_id: str, db: Any = None, settings: Any = None) -> dict:
    """Process a comment through the AI pipeline. Handles all edge cases."""
    start = time.time()

    # Edge case: empty/whitespace-only comment
    processed_comment, warnings = _preprocess_comment(comment, page_id)

    if not processed_comment:
        logger.warning("[Pipeline] Skipping empty comment, page=%s", page_id)
        return {
            "comment_text": comment, "page_id": page_id, "language": "ar",
            "intent": None, "sentiment": None, "urgency": "normal",
            "confidence": 0.0, "kb_results": [], "kb_empty": True,
            "guardrail_triggered": False, "guardrail_reason": None,
            "ai_reply": None, "action": "skip", "priority": "normal",
            "escalation_reason": "Empty comment -- skipped",
            "matched_rule_id": None, "processing_time_ms": 0,
            "_warnings": warnings,
        }

    safe_reply = ""
    default_language = "ar"
    if settings:
        safe_reply = settings.safe_reply_ar or SAFE_DEFAULT_REPLY_AR
        default_language = getattr(settings, "default_language", "ar") or "ar"

    if _LANGGRAPH_AVAILABLE and _pipeline is not None:
        initial: CommentState = {
            "comment_text": processed_comment,
            "page_id": page_id,
            "language": default_language,
            "intent": None,
            "sentiment": None,
            "urgency": "normal",
            "confidence": 0.0,
            "kb_results": [],
            "kb_empty": False,
            "guardrail_triggered": False,
            "guardrail_reason": None,
            "ai_reply": None,
            "action": "reply",
            "priority": "normal",
            "escalation_reason": None,
            "matched_rule_id": None,
            "processing_time_ms": 0,
            "_safe_reply": safe_reply,
            "_default_language": default_language,
            "_warnings": warnings,
        }
        try:
            result = dict(await _pipeline.ainvoke(initial))
        except Exception as e:
            logger.error("[Pipeline] Execution error: %s", e)
            result = await _fallback_process(processed_comment, page_id, db, settings)
    else:
        result = await _fallback_process(processed_comment, page_id, db, settings)

    result["processing_time_ms"] = int((time.time() - start) * 1000)
    if warnings:
        result["_warnings"] = warnings
    return result


async def _fallback_process(comment: str, page_id: str, db: Any = None, settings: Any = None) -> dict:
    from app.ai.llm import MockLLM
    llm = MockLLM()

    safe_reply_ar = SAFE_DEFAULT_REPLY_AR
    safe_reply_en = SAFE_DEFAULT_REPLY_EN
    default_language = "ar"
    if settings:
        safe_reply_ar = getattr(settings, "safe_reply_ar", None) or SAFE_DEFAULT_REPLY_AR
        safe_reply_en = getattr(settings, "safe_reply_en", None) or SAFE_DEFAULT_REPLY_EN
        default_language = getattr(settings, "default_language", "ar") or "ar"

    if not comment or not comment.strip():
        language = default_language
    elif _is_emoji_only(comment):
        language = default_language
    else:
        arabic = sum(1 for c in comment if "\u0600" <= c <= "\u06ff")
        language = "ar" if len(comment) > 0 and arabic / len(comment) > 0.2 else "en"

    intent_resp = await llm.ainvoke(f"classify intent: {comment}")
    sentiment_resp = await llm.ainvoke(f"analyze sentiment: {comment}")
    intent = intent_resp.content
    sentiment = sentiment_resp.content

    # Check guardrails even in fallback
    text_lower = comment.lower()
    guardrail_triggered = False
    guardrail_reason = None
    for category, keywords in GUARDRAIL_PATTERNS.items():
        if any(kw in text_lower for kw in keywords):
            guardrail_triggered = True
            guardrail_reason = f"Guardrail: {category}"
            break

    kb_results = []
    if db:
        try:
            from sqlalchemy import select
            from app.db import KnowledgeBase
            r = await db.execute(
                select(KnowledgeBase).where(KnowledgeBase.is_active == True).limit(2)
            )
            kb_results = [{"question": e.question, "answer": e.answer} for e in r.scalars()]
        except Exception:
            pass

    if guardrail_triggered:
        ai_reply = SAFE_DEFAULT_REPLY_AR if language == "ar" else SAFE_DEFAULT_REPLY_EN
        confidence = 0.0
    else:
        try:
            reply_resp = await llm.ainvoke(f"generate reply: {comment}")
            ai_reply = reply_resp.content
            confidence = 0.75
        except Exception as e:
            logger.error("[Pipeline-Fallback] LLM failed: %s", e)
            ai_reply = None
            confidence = 0.0

    _conf_auto, _conf_flag = _get_thresholds()
    action = "reply"
    escalation_reason = None
    priority = INTENT_PRIORITY.get(intent, "normal")

    if guardrail_triggered:
        action = "escalate"
        escalation_reason = guardrail_reason
        priority = "high"
    elif sentiment == "angry":
        action = "escalate"
        escalation_reason = "Angry customer sentiment"
        priority = "high"
    elif intent == "refund":
        action = "escalate"
        escalation_reason = "Refund request"
        priority = "high"
    elif confidence == 0.0 or not ai_reply:
        action = "escalate"
        escalation_reason = "LLM unavailable -- safe default reply used"
        ai_reply = safe_reply_ar if language in ("ar", "other") else safe_reply_en
    elif confidence < _conf_flag:
        action = "escalate"
        escalation_reason = f"Low confidence ({confidence:.0%})"
    elif confidence < _conf_auto:
        action = "flag_review"

    return {
        "comment_text": comment,
        "page_id": page_id,
        "language": language,
        "intent": intent,
        "sentiment": sentiment,
        "urgency": "urgent" if sentiment in ("angry", "urgent") else "normal",
        "confidence": confidence,
        "kb_results": kb_results,
        "guardrail_triggered": guardrail_triggered,
        "guardrail_reason": guardrail_reason,
        "ai_reply": ai_reply,
        "action": action,
        "priority": priority,
        "escalation_reason": escalation_reason,
        "matched_rule_id": None,
    }
