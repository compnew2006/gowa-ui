"""Add composite performance indexes for common query patterns.

Revision ID: 0002_performance_indexes
Revises: 0001_initial_schema
Create Date: 2026-05-15
"""
from __future__ import annotations

from alembic import op

revision = "0002_performance_indexes"
down_revision = "0001_initial_schema"
branch_labels = None
depends_on = None


def upgrade() -> None:
    # Conversations: filtered by page, status, and sorted by recency
    op.execute(
        "CREATE INDEX IF NOT EXISTS "
        "idx_conversations_page_status_created "
        "ON conversations(page_id, status, created_at DESC)"
    )

    # Conversations: customer lookup (partial index for non-null only)
    op.execute(
        "CREATE INDEX IF NOT EXISTS "
        "idx_conversations_customer "
        "ON conversations(customer_id) WHERE customer_id IS NOT NULL"
    )

    # Customers: CRM queries filtered by page and churn risk
    op.execute(
        "CREATE INDEX IF NOT EXISTS "
        "idx_customers_page_churn "
        "ON customers(page_id, churn_risk)"
    )

    # Customers: conversion tracking per page
    op.execute(
        "CREATE INDEX IF NOT EXISTS "
        "idx_customers_page_conversion "
        "ON customers(page_id, conversion_status)"
    )

    # Escalations: open escalations sorted by recency
    op.execute(
        "CREATE INDEX IF NOT EXISTS "
        "idx_escalations_status_created "
        "ON escalations(status, created_at DESC)"
    )

    # Audit logs: entity-scoped audit trail
    op.execute(
        "CREATE INDEX IF NOT EXISTS "
        "idx_audit_logs_entity_created "
        "ON audit_logs(entity_type, entity_id, created_at DESC)"
    )

    # Knowledge base: active entries per page
    op.execute(
        "CREATE INDEX IF NOT EXISTS "
        "idx_knowledge_base_page_active "
        "ON knowledge_base(page_id, is_active) WHERE is_active = true"
    )


def downgrade() -> None:
    op.execute("DROP INDEX IF EXISTS idx_conversations_page_status_created")
    op.execute("DROP INDEX IF EXISTS idx_conversations_customer")
    op.execute("DROP INDEX IF EXISTS idx_customers_page_churn")
    op.execute("DROP INDEX IF EXISTS idx_customers_page_conversion")
    op.execute("DROP INDEX IF EXISTS idx_escalations_status_created")
    op.execute("DROP INDEX IF EXISTS idx_audit_logs_entity_created")
    op.execute("DROP INDEX IF EXISTS idx_knowledge_base_page_active")
