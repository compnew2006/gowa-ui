import uuid
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update, delete
from datetime import datetime, timezone
from app.deps import get_db
from app.db import CustomAIModel
from app.schemas import CustomAIModelOut, CustomAIModelCreate, CustomAIModelUpdate
from app.services.token_service import encrypt_token, decrypt_token
from app.ai.llm import invalidate_llm_cache

router = APIRouter(tags=["ai-models"])


def _mask_key(encrypted_key: str) -> str:
    """Decrypt the API key, mask it for safe API output, and return."""
    try:
        decrypted = decrypt_token(encrypted_key)
        if len(decrypted) > 7:
            return decrypted[:3] + "..." + decrypted[-4:]
        return "****"
    except Exception:
        return "****"


def _to_out_dict(model: CustomAIModel) -> dict:
    """Format model into the response dictionary with masked API key."""
    return {
        "id": model.id,
        "name": model.name,
        "provider": model.provider,
        "model_name": model.model_name,
        "api_key_masked": _mask_key(model.api_key_encrypted),
        "api_base": model.api_base,
        "is_active": model.is_active,
        "created_at": model.created_at,
        "updated_at": model.updated_at,
    }


@router.get("/ai-models", response_model=list[CustomAIModelOut])
async def list_ai_models(db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(CustomAIModel).order_by(CustomAIModel.created_at.desc()))
    models = result.scalars().all()
    return [_to_out_dict(m) for m in models]


@router.post("/ai-models", response_model=CustomAIModelOut, status_code=201)
async def create_ai_model(body: CustomAIModelCreate, db: AsyncSession = Depends(get_db)):
    # If this model is activated, deactivate all other models
    if body.is_active:
        await db.execute(update(CustomAIModel).values(is_active=False))
        invalidate_llm_cache()

    model = CustomAIModel(
        id=str(uuid.uuid4()),
        name=body.name,
        provider=body.provider,
        model_name=body.model_name,
        api_key_encrypted=encrypt_token(body.api_key),
        api_base=body.api_base,
        is_active=body.is_active,
    )
    db.add(model)
    await db.commit()
    await db.refresh(model)
    return _to_out_dict(model)


@router.patch("/ai-models/{model_id}", response_model=CustomAIModelOut)
async def update_ai_model(model_id: str, body: CustomAIModelUpdate, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(CustomAIModel).where(CustomAIModel.id == model_id))
    model = result.scalar_one_or_none()
    if not model:
        raise HTTPException(status_code=404, detail="AI Model not found")

    # If active state is being toggled to True, deactivate all others
    if body.is_active is True:
        await db.execute(update(CustomAIModel).values(is_active=False))
        invalidate_llm_cache()
    elif body.is_active is False and model.is_active:
        invalidate_llm_cache()

    updates = body.model_dump(exclude_none=True)
    if "api_key" in updates:
        model.api_key_encrypted = encrypt_token(updates.pop("api_key"))

    for field, value in updates.items():
        setattr(model, field, value)

    model.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(model)
    
    # Invalidate cache again just to be 100% safe
    invalidate_llm_cache()
    return _to_out_dict(model)


@router.delete("/ai-models/{model_id}", status_code=204)
async def delete_ai_model(model_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(CustomAIModel).where(CustomAIModel.id == model_id))
    model = result.scalar_one_or_none()
    if not model:
        raise HTTPException(status_code=404, detail="AI Model not found")

    if model.is_active:
        invalidate_llm_cache()

    await db.delete(model)
    await db.commit()


@router.post("/ai-models/{model_id}/test")
async def test_ai_model(model_id: str, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(CustomAIModel).where(CustomAIModel.id == model_id))
    model = result.scalar_one_or_none()
    if not model:
        raise HTTPException(status_code=404, detail="AI Model not found")

    try:
        decrypted_key = decrypt_token(model.api_key_encrypted)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Failed to decrypt API Key: {str(e)}")

    provider = model.provider.lower()
    model_name = model.model_name
    api_base = model.api_base

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

    try:
        import litellm
        # Set short timeout for testing connection
        resp = await litellm.acompletion(
            model=litellm_model,
            messages=[{"role": "user", "content": "ping"}],
            api_key=decrypted_key,
            api_base=api_base,
            max_tokens=10,
            timeout=10
        )
        content = resp.choices[0].message.content or ""
        return {"status": "success", "response": content.strip()}
    except Exception as e:
        return {"status": "error", "message": str(e)}
