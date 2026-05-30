from pydantic_settings import BaseSettings
from functools import lru_cache
from urllib.parse import urlparse, parse_qs, urlencode, urlunparse
import os


class Settings(BaseSettings):
    # ── Database ──────────────────────────────────────────────
    database_url: str = os.environ.get("DATABASE_URL", "")
    redis_url: str = os.environ.get("REDIS_URL", "redis://localhost:6379/0")

    # ── LLM Keys ─────────────────────────────────────────────
    openai_api_key: str = os.environ.get("OPENAI_API_KEY", "")
    glm_api_key: str = os.environ.get("GLM_API_KEY", "")

    # ── Meta / Facebook ───────────────────────────────────────
    meta_app_secret: str = os.environ.get("META_APP_SECRET", "")
    meta_app_id: str = os.environ.get("META_APP_ID", "")

    # ── LiteLLM Proxy ─────────────────────────────────────────
    litellm_proxy_url: str = os.environ.get("LITELLM_PROXY_URL", "")
    litellm_proxy_key: str = os.environ.get("LITELLM_PROXY_KEY", "")
    litellm_primary_model: str = os.environ.get("LITELLM_PRIMARY_MODEL", "hermes-3-llama-3.1-8b")

    # ── Token Encryption ──────────────────────────────────────
    token_encryption_key: str = os.environ.get("TOKEN_ENCRYPTION_KEY", "")

    # ── Sentry ────────────────────────────────────────────────
    sentry_dsn: str = os.environ.get("SENTRY_DSN", "")
    environment: str = os.environ.get("ENVIRONMENT", "development")

    # ── Email Notifications ───────────────────────────────────
    smtp_host: str = os.environ.get("SMTP_HOST", "")
    smtp_port: int = int(os.environ.get("SMTP_PORT", "587"))
    smtp_user: str = os.environ.get("SMTP_USER", "")
    smtp_password: str = os.environ.get("SMTP_PASSWORD", "")
    admin_email: str = os.environ.get("ADMIN_EMAIL", "")

    # ── Rate Limiting ─────────────────────────────────────────
    rate_limit_rpm: int = int(os.environ.get("RATE_LIMIT_RPM", "60"))
    rate_limit_mutation_rpm: int = int(os.environ.get("RATE_LIMIT_MUTATION_RPM", "30"))
    webhook_rate_limit_rpm: int = int(os.environ.get("WEBHOOK_RATE_LIMIT_RPM", "200"))
    rate_limit_warning_threshold: float = float(os.environ.get("RATE_LIMIT_WARNING_THRESHOLD", "0.8"))
    auto_apply_schema_patches: bool = os.environ.get("AUTO_APPLY_SCHEMA_PATCHES", "false").lower() in {"1", "true", "yes"}

    # ── JWT Authentication ────────────────────────────────────
    jwt_secret: str = os.environ.get("JWT_SECRET", "")
    jwt_algorithm: str = os.environ.get("JWT_ALGORITHM", "HS256")
    jwt_expire_minutes: int = int(os.environ.get("JWT_EXPIRE_MINUTES", "60"))

    # ── AI Pipeline Defaults ──────────────────────────────────
    primary_llm_model: str = os.environ.get("PRIMARY_LLM_MODEL", "glm-4")
    fallback_llm_model: str = os.environ.get("FALLBACK_LLM_MODEL", "")
    default_language: str = os.environ.get("DEFAULT_LANGUAGE", "ar")
    max_retries: int = int(os.environ.get("MAX_RETRIES", "3"))
    warmup_mode: bool = os.environ.get("WARMUP_MODE", "false").lower() in {"1", "true", "yes"}

    # ── Automation Settings ───────────────────────────────────
    auto_escalate_angry: bool = os.environ.get("AUTO_ESCALATE_ANGRY", "true").lower() in {"1", "true", "yes"}
    auto_reply_start_date: str = os.environ.get("AUTO_REPLY_START_DATE", "")
    auto_reply_end_date: str = os.environ.get("AUTO_REPLY_END_DATE", "")
    confidence_threshold: float = float(os.environ.get("CONFIDENCE_THRESHOLD", "0.85"))
    reply_mode: str = os.environ.get("REPLY_MODE", "public_and_private")

    # ── Safe Fallback Replies ─────────────────────────────────
    safe_reply_ar: str = os.environ.get("SAFE_REPLY_AR", "شكراً لتواصلك معنا. سيتم الرد عليك في أقرب وقت.")
    safe_reply_en: str = os.environ.get("SAFE_REPLY_EN", "Thank you for reaching out. We will get back to you shortly.")
    public_reply_message_ar: str = os.environ.get("PUBLIC_REPLY_MESSAGE_AR", "")
    public_reply_message_en: str = os.environ.get("PUBLIC_REPLY_MESSAGE_EN", "")

    # ── Integrations ──────────────────────────────────────────
    telegram_bot_token: str = os.environ.get("TELEGRAM_BOT_TOKEN", "")
    telegram_chat_id: str = os.environ.get("TELEGRAM_CHAT_ID", "")
    webhook_verify_token: str = os.environ.get("WEBHOOK_VERIFY_TOKEN", "")

    # ── CORS ──────────────────────────────────────────────────
    allowed_origins: str = os.environ.get("ALLOWED_ORIGINS", "http://localhost:3000")

    # ── Database Pool ─────────────────────────────────────────
    db_pool_size: int = int(os.environ.get("DB_POOL_SIZE", "20"))
    db_max_overflow: int = int(os.environ.get("DB_MAX_OVERFLOW", "30"))
    db_pool_recycle: int = int(os.environ.get("DB_POOL_RECYCLE", "1800"))
    db_pool_timeout: int = int(os.environ.get("DB_POOL_TIMEOUT", "30"))

    # ── AI Pipeline Thresholds ──────────────────────────────
    confidence_auto_reply: float = float(os.environ.get("CONFIDENCE_AUTO_REPLY", "0.85"))
    confidence_flag_review: float = float(os.environ.get("CONFIDENCE_FLAG_REVIEW", "0.60"))

    model_config = {"env_file": ".env", "extra": "ignore"}

    @property
    def async_database_url(self) -> str:
        url = self.database_url
        if url.startswith("postgresql://"):
            url = url.replace("postgresql://", "postgresql+asyncpg://", 1)
        elif url.startswith("postgres://"):
            url = url.replace("postgres://", "postgresql+asyncpg://", 1)

        parsed = urlparse(url)
        params = parse_qs(parsed.query)
        # asyncpg doesn't support these SSL params in the URL
        for key in ("sslmode", "sslcert", "sslkey", "sslrootcert"):
            params.pop(key, None)
        new_query = urlencode({k: v[0] for k, v in params.items()})
        return urlunparse(parsed._replace(query=new_query))

    @property
    def ssl_required(self) -> bool:
        return "sslmode=require" in self.database_url or "sslmode=verify" in self.database_url


@lru_cache
def get_settings() -> Settings:
    return Settings()
