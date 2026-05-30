from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.deps import get_db
from app.schemas import CustomerOut, CustomerListResponse, CustomerUpdate, AddNoteRequest, ConversationOut, BulkUpdateCustomersRequest, BulkDeleteCustomersRequest
from app.services.customers import (
    list_customers as svc_list_customers,
    get_customer as svc_get_customer,
    update_customer as svc_update_customer,
    add_customer_note as svc_add_customer_note,
    get_customer_conversations as svc_get_customer_conversations,
)

router = APIRouter(tags=["customers"])


@router.get("/customers", response_model=CustomerListResponse)
async def list_customers(
    page: int = Query(1, ge=1),
    limit: int = Query(20, ge=1, le=100),
    page_id: str | None = None,
    purchase_intent: str | None = None,
    conversion_status: str | None = None,
    churn_risk: str | None = None,
    search: str | None = None,
    db: AsyncSession = Depends(get_db),
):
    data, total = await svc_list_customers(
        db, page=page, limit=limit, page_id=page_id,
        purchase_intent=purchase_intent, conversion_status=conversion_status,
        churn_risk=churn_risk, search=search
    )
    return CustomerListResponse(data=data, total=total, page=page, limit=limit)


# ── Bulk operations MUST come before /{customer_id} to avoid routing conflicts ──

@router.patch("/customers/bulk", response_model=list[CustomerOut])
async def bulk_update_customers(
    body: BulkUpdateCustomersRequest,
    db: AsyncSession = Depends(get_db),
):
    from app.db import Customer
    from sqlalchemy import select
    from app.services.customers import _attach_page_info

    # Fetch all requested customers
    res = await db.execute(select(Customer).where(Customer.id.in_(body.ids)))
    customers = res.scalars().all()

    update_data = body.update.model_dump(exclude_none=True)
    if not update_data:
        return [await _attach_page_info(db, c) for c in customers]

    from datetime import datetime, timezone
    for customer in customers:
        for field, value in update_data.items():
            setattr(customer, field, value)
        customer.updated_at = datetime.now(timezone.utc)

    await db.commit()
    for customer in customers:
        await db.refresh(customer)

    return [await _attach_page_info(db, c) for c in customers]


@router.post("/customers/bulk-delete", status_code=204)
async def bulk_delete_customers(
    body: BulkDeleteCustomersRequest,
    db: AsyncSession = Depends(get_db),
):
    from app.db import Customer
    from sqlalchemy import select

    if body.ids:
        res = await db.execute(select(Customer).where(Customer.id.in_(body.ids)))
        customers = res.scalars().all()
        for customer in customers:
            await db.delete(customer)
        await db.commit()


# ── Single-customer operations ──────────────────────────────────────────────────

@router.get("/customers/{customer_id}", response_model=CustomerOut)
async def get_customer(customer_id: str, db: AsyncSession = Depends(get_db)):
    customer = await svc_get_customer(db, customer_id)
    if not customer:
        raise HTTPException(status_code=404, detail="Customer not found")
    return customer


@router.patch("/customers/{customer_id}", response_model=CustomerOut)
async def update_customer(
    customer_id: str,
    body: CustomerUpdate,
    db: AsyncSession = Depends(get_db),
):
    customer = await svc_update_customer(db, customer_id, body.model_dump(exclude_none=True))
    if not customer:
        raise HTTPException(status_code=404, detail="Customer not found")
    return customer


@router.delete("/customers/{customer_id}", status_code=204)
async def delete_customer(
    customer_id: str,
    db: AsyncSession = Depends(get_db),
):
    from app.services.customers import delete_customer as svc_delete_customer
    success = await svc_delete_customer(db, customer_id)
    if not success:
        raise HTTPException(status_code=404, detail="Customer not found")


@router.post("/customers/{customer_id}/notes", response_model=CustomerOut)
async def add_note(
    customer_id: str,
    body: AddNoteRequest,
    db: AsyncSession = Depends(get_db),
):
    customer = await svc_add_customer_note(db, customer_id, body.content, body.author)
    if not customer:
        raise HTTPException(status_code=404, detail="Customer not found")
    return customer


@router.get("/customers/{customer_id}/conversations", response_model=list[ConversationOut])
async def get_customer_conversations(
    customer_id: str,
    limit: int = Query(50, ge=1, le=200),
    db: AsyncSession = Depends(get_db),
):
    conversations = await svc_get_customer_conversations(db, customer_id, limit)
    if conversations is None:
        raise HTTPException(status_code=404, detail="Customer not found")
    return conversations
