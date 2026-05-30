"""
Prompt Versioning Router — track DSPy optimization history.

Endpoints:
  GET /prompt-versions        — List all versions with accuracy metrics
  GET /prompt-versions/latest — Get current active version
"""
from fastapi import APIRouter

router = APIRouter(tags=["prompt-versions"])


@router.get("/prompt-versions")
async def list_prompt_versions():
    """List all DSPy prompt optimization versions with performance metrics."""
    from app.ai.dspy_optimizer import get_optimizer
    optimizer = get_optimizer()
    versions = optimizer.get_versions()
    return {
        "total_versions": len(versions),
        "versions": sorted(versions, key=lambda v: v.get("version", 0), reverse=True),
        "current_version": optimizer._current_version,
    }


@router.get("/prompt-versions/latest")
async def get_latest_version():
    """Get the currently active prompt version."""
    from app.ai.dspy_optimizer import get_optimizer
    optimizer = get_optimizer()
    versions = optimizer.get_versions()
    if not versions:
        return {
            "version": 0,
            "message": "No optimization runs yet — using base prompts",
            "intent_accuracy": None,
            "sentiment_accuracy": None,
        }
    latest = max(versions, key=lambda v: v.get("version", 0))
    return latest
