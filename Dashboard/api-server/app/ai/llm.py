from __future__ import annotations
import os
from typing import Any
from app.config import get_settings


class MockMessage:
    def __init__(self, content: str):
        self.content = content


class MockLLM:
    """Fallback LLM using keyword heuristics when no API key is configured."""

    INTENT_KEYWORDS = {
        "price_inquiry": ["سعر", "كم", "بكام", "price", "cost", "how much", "تكلفة", "ثمن"],
        "purchase": ["اشتري", "عايز", "أريد", "buy", "order", "اطلب", "شراء"],
        "complaint": ["مشكلة", "خطأ", "سيئ", "complaint", "bad", "wrong", "problem", "شكوى"],
        "refund": ["استرجاع", "رجوع", "refund", "return", "money back"],
        "compliment": ["شكرا", "ممتاز", "رائع", "thanks", "great", "excellent", "جميل"],
    }

    SENTIMENT_KEYWORDS = {
        "positive": ["شكرا", "ممتاز", "رائع", "thanks", "great", "good", "جيد", "ممتاز"],
        "negative": ["سيئ", "مشكلة", "خطأ", "bad", "wrong", "problem", "لا يعمل"],
        "angry": ["غاضب", "فضيحة", "مستاء", "angry", "terrible", "awful", "احتيال", "كذب"],
        "neutral": [],
    }

    async def ainvoke(self, prompt: str) -> MockMessage:
        prompt_lower = prompt.lower()
        if "intent" in prompt_lower or "classify" in prompt_lower:
            return MockMessage(self._classify_intent(prompt))
        if "sentiment" in prompt_lower:
            return MockMessage(self._classify_sentiment(prompt))
        return MockMessage("أشكرك على تواصلك معنا. سيقوم فريقنا بالرد عليك في أقرب وقت.")

    def _classify_intent(self, text: str) -> str:
        text_lower = text.lower()
        for intent, keywords in self.INTENT_KEYWORDS.items():
            if any(kw in text_lower for kw in keywords):
                return intent
        return "general"

    def _classify_sentiment(self, text: str) -> str:
        text_lower = text.lower()
        for sentiment in ("angry", "negative", "positive"):
            if any(kw in text_lower for kw in self.SENTIMENT_KEYWORDS[sentiment]):
                return sentiment
        return "neutral"


async def _build_litellm_router():
    """Build a LiteLLM Router with Hermes → GLM fallback chain and active DB models."""
    try:
        from litellm import Router
        settings = get_settings()

        model_list = []

        # 1. Try to load dynamic custom active model from the database
        try:
            from app.db import AsyncSessionLocal, CustomAIModel
            from sqlalchemy import select
            from app.services.token_service import decrypt_token

            async with AsyncSessionLocal() as db:
                result = await db.execute(
                    select(CustomAIModel).where(CustomAIModel.is_active == True)
                )
                active_model = result.scalar_one_or_none()

            if active_model:
                try:
                    decrypted_key = decrypt_token(active_model.api_key_encrypted)
                except Exception as dec_err:
                    print(f"[LLM] Decryption of custom model API key failed: {dec_err}")
                    decrypted_key = "dummy"

                provider = active_model.provider.lower()
                model_name = active_model.model_name
                api_base = active_model.api_base

                if api_base:
                    # Any provider with a custom api_base is treated as an OpenAI-compatible custom gateway in LiteLLM
                    litellm_model = f"openai/{model_name}"
                elif provider == "openai":
                    litellm_model = f"openai/{model_name}"
                elif provider == "zhipuai":
                    # Native LiteLLM prefix for ZhipuAI is glm/
                    litellm_model = f"glm/{model_name}"
                elif provider == "anthropic":
                    litellm_model = f"anthropic/{model_name}"
                else:
                    litellm_model = f"openai/{model_name}"

                params = {
                    "model": litellm_model,
                    "api_key": decrypted_key,
                }
                if api_base:
                    params["api_base"] = api_base

                model_list.append({
                    "model_name": "primary",
                    "litellm_params": params,
                    "model_info": {"id": f"custom-{active_model.id}"},
                })
                print(f"[LLM] Loaded active custom model: {active_model.name} ({litellm_model})")
        except Exception as db_err:
            print(f"[LLM] Error loading custom model from database: {db_err}")

        # 2. Add Hermes / settings environment fallbacks
        # Primary: Hermes (via LiteLLM proxy or OpenAI-compatible endpoint)
        if settings.litellm_proxy_url:
            model_list.append({
                "model_name": "primary",
                "litellm_params": {
                    "model": f"openai/{settings.litellm_primary_model}",
                    "api_base": settings.litellm_proxy_url,
                    "api_key": settings.litellm_proxy_key or "dummy",
                },
                "model_info": {"id": "hermes-primary"},
            })

        # Fallback 1: OpenAI
        if settings.openai_api_key:
            model_list.append({
                "model_name": "primary",
                "litellm_params": {
                    "model": "gpt-4o-mini",
                    "api_key": settings.openai_api_key,
                },
                "model_info": {"id": "openai-fallback"},
            })

        # Primary: GLM-4 via z.ai (or fallback ZhipuAI)
        if settings.glm_api_key:
            glm_base = os.environ.get("GLM_BASE_URL", "")
            glm_params = {
                "api_key": settings.glm_api_key,
            }
            if glm_base:
                glm_params["model"] = "openai/" + settings.primary_llm_model
                glm_params["api_base"] = glm_base
            else:
                glm_params["model"] = f"zhipuai/{settings.primary_llm_model}"
            model_list.insert(0, {
                "model_name": "primary",
                "litellm_params": glm_params,
                "model_info": {"id": "glm-primary"},
            })

        if not model_list:
            return None

        router = Router(
            model_list=model_list,
            fallbacks=[{"primary": ["primary"]}],
            num_retries=2,
            retry_after=5,
            timeout=30,
            routing_strategy="least-busy",
        )
        return router
    except Exception as e:
        print(f"[LiteLLM] Router build failed: {e}")
        return None


class LiteLLMWrapper:
    """Thin async wrapper around the LiteLLM router."""

    def __init__(self, router):
        self._router = router

    async def ainvoke(self, prompt: str) -> MockMessage:
        try:
            resp = await self._router.acompletion(
                model="primary",
                messages=[{"role": "user", "content": prompt}],
                temperature=0.3,
                max_tokens=4096,
            )
            msg = resp.choices[0].message
            content = msg.content or ""
            # GLM-5 thinking models put chain-of-thought in reasoning_content
            if not content.strip():
                pass  # reasoning_content is thinking chain, not the actual reply
            return MockMessage(content.strip())
        except Exception as e:
            print(f"[LiteLLM] Completion failed: {e}")
            raise


_llm_instance = None


async def get_llm():
    global _llm_instance
    if _llm_instance is not None:
        return _llm_instance

    settings = get_settings()

    # Try LiteLLM router (Hermes → GLM fallback chain + dynamic custom AI model)
    router = await _build_litellm_router()
    if router:
        _llm_instance = LiteLLMWrapper(router)
        print("[LLM] Using LiteLLM Router (Hermes → GLM fallback + dynamic models)")
        return _llm_instance

    # Direct OpenAI if available
    if settings.openai_api_key:
        try:
            from langchain_openai import ChatOpenAI
            _llm_instance = ChatOpenAI(
                model="gpt-4o-mini",
                api_key=settings.openai_api_key,
                temperature=0.3,
            )
            print("[LLM] Using direct OpenAI (gpt-4o-mini)")
            return _llm_instance
        except Exception:
            pass

    # Final fallback: keyword heuristics
    _llm_instance = MockLLM()
    print("[LLM] Using MockLLM (no API keys configured)")
    return _llm_instance


def invalidate_llm_cache():
    """Invalidate the cached LLM instance, forcing a rebuild on next use."""
    global _llm_instance
    _llm_instance = None
    print("[LLM] Cache invalidated successfully")
