"""
Customer service layer.

Business logic extracted from the customers router.
Handles CRUD, scoring, notes, and customer-conversation lookups.
"""
import uuid
from datetime import datetime, timezone

from sqlalchemy import select, func, cast, String
from sqlalchemy.ext.asyncio import AsyncSession

from app.db import Customer, Conversation, Page


async def _attach_page_info(db: AsyncSession, customer: Customer | None) -> Customer | None:
    if not customer:
        return None
    
    # Pre-populate platform based on customer fields
    if customer.whatsapp_id:
        customer.platform = "whatsapp"
    elif customer.instagram_id:
        customer.platform = "instagram"
    else:
        customer.platform = "facebook"
        
    if customer.page_id:
        page_res = await db.execute(select(Page.name, Page.platform).where(cast(Page.id, String) == customer.page_id))
        row = page_res.first()
        if row:
            customer.page_name = row[0]
            if not customer.whatsapp_id and not customer.instagram_id:
                customer.platform = row[1] or "facebook"
    return customer


async def list_customers(
    db: AsyncSession,
    *,
    page: int = 1,
    limit: int = 20,
    page_id: str | None = None,
    purchase_intent: str | None = None,
    conversion_status: str | None = None,
    churn_risk: str | None = None,
    search: str | None = None,
) -> tuple[list, int]:
    """Return (customers, total_count) matching the given filters."""
    q = select(Customer).where(Customer.gdpr_deleted == False)
    if page_id:
        q = q.where(Customer.page_id == page_id)
    if purchase_intent:
        q = q.where(Customer.purchase_intent == purchase_intent)
    if conversion_status:
        q = q.where(Customer.conversion_status == conversion_status)
    if churn_risk:
        q = q.where(Customer.churn_risk == churn_risk)
    if search:
        q = q.where(
            Customer.full_name.ilike(f"%{search}%")
            | Customer.username.ilike(f"%{search}%")
        )

    count_q = select(func.count()).select_from(q.subquery())
    total_result = await db.execute(count_q)
    total = total_result.scalar_one()

    # Now select Customer joined with Page for page_name and platform
    q_data = select(Customer, Page.name, Page.platform).outerjoin(Page, Customer.page_id == cast(Page.id, String)).where(Customer.gdpr_deleted == False)
    if page_id:
        q_data = q_data.where(Customer.page_id == page_id)
    if purchase_intent:
        q_data = q_data.where(Customer.purchase_intent == purchase_intent)
    if conversion_status:
        q_data = q_data.where(Customer.conversion_status == conversion_status)
    if churn_risk:
        q_data = q_data.where(Customer.churn_risk == churn_risk)
    if search:
        q_data = q_data.where(
            Customer.full_name.ilike(f"%{search}%")
            | Customer.username.ilike(f"%{search}%")
        )

    q_data = q_data.order_by(Customer.lead_score.desc(), Customer.last_interaction.desc())
    q_data = q_data.offset((page - 1) * limit).limit(limit)
    result = await db.execute(q_data)
    rows = result.all()

    data = []
    for row in rows:
        cust = row[0]
        cust.page_name = row[1]
        if cust.whatsapp_id:
            cust.platform = "whatsapp"
        elif cust.instagram_id:
            cust.platform = "instagram"
        else:
            cust.platform = row[2] or "facebook"
        data.append(cust)

    return data, total


async def get_customer(db: AsyncSession, customer_id: str) -> Customer | None:
    """Fetch a single customer by ID, or None if not found."""
    q = select(Customer, Page.name, Page.platform).outerjoin(Page, Customer.page_id == cast(Page.id, String)).where(Customer.id == customer_id)
    result = await db.execute(q)
    row = result.first()
    if not row:
        return None
    cust = row[0]
    cust.page_name = row[1]
    if cust.whatsapp_id:
        cust.platform = "whatsapp"
    elif cust.instagram_id:
        cust.platform = "instagram"
    else:
        cust.platform = row[2] or "facebook"
    return cust


async def update_customer(
    db: AsyncSession, customer_id: str, update_data: dict
) -> Customer | None:
    """Apply partial update to a customer. Returns updated customer or None."""
    customer = await get_customer(db, customer_id)
    if not customer:
        return None
    for field, value in update_data.items():
        setattr(customer, field, value)
    customer.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(customer)
    return await _attach_page_info(db, customer)


async def add_customer_note(
    db: AsyncSession, customer_id: str, content: str, author: str
) -> Customer | None:
    """Append a note to a customer record. Returns updated customer or None."""
    customer = await get_customer(db, customer_id)
    if not customer:
        return None

    notes = list(customer.notes or [])
    notes.append({
        "id": str(uuid.uuid4()),
        "content": content,
        "author": author,
        "createdAt": datetime.now(timezone.utc).isoformat(),
    })
    customer.notes = notes
    customer.updated_at = datetime.now(timezone.utc)
    await db.commit()
    await db.refresh(customer)
    return await _attach_page_info(db, customer)


async def get_customer_conversations(
    db: AsyncSession, customer_id: str, limit: int = 50
) -> list[Conversation] | None:
    """Fetch conversations for a customer. Returns None if customer not found."""
    customer = await get_customer(db, customer_id)
    if not customer:
        return None

    q = (
        select(Conversation)
        .where(Conversation.customer_id == customer_id)
        .order_by(Conversation.created_at.desc())
        .limit(limit)
    )
    result = await db.execute(q)
    return result.scalars().all()


async def delete_customer(db: AsyncSession, customer_id: str) -> bool:
    """Hard delete a customer record by ID. Returns True if deleted, False otherwise."""
    result = await db.execute(select(Customer).where(Customer.id == customer_id))
    customer = result.scalar_one_or_none()
    if not customer:
        return False
    await db.delete(customer)
    await db.commit()
    return True
