import { Router, type IRouter } from "express";
import { eq, ilike, or, desc } from "drizzle-orm";
import { db, customersTable } from "@workspace/db";
import {
  ListCustomersQueryParams,
  GetCustomerParams,
  UpdateCustomerParams,
  UpdateCustomerBody,
  AddCustomerNoteParams,
  AddCustomerNoteBody,
  ListCustomersResponse,
  GetCustomerResponse,
  UpdateCustomerResponse,
} from "@workspace/api-zod";

const router: IRouter = Router();

function formatCustomer(c: typeof customersTable.$inferSelect) {
  return {
    ...c,
    notes: Array.isArray(c.notes) ? c.notes : [],
    tags: c.tags ?? [],
    escalationHistory: c.escalationHistory ?? [],
  };
}

router.get("/customers", async (req, res): Promise<void> => {
  const parsed = ListCustomersQueryParams.safeParse(req.query);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const { search, purchaseIntent, limit, offset } = parsed.data;

  let query = db.select().from(customersTable).$dynamic();

  if (search) {
    query = query.where(
      or(
        ilike(customersTable.fullName, `%${search}%`),
        ilike(customersTable.username, `%${search}%`)
      )
    );
  }
  if (purchaseIntent) {
    query = query.where(eq(customersTable.purchaseIntent, purchaseIntent as "Low" | "Medium" | "High"));
  }

  const customers = await query
    .orderBy(desc(customersTable.lastInteraction))
    .limit(limit ?? 50)
    .offset(offset ?? 0);

  const total = await db.$count(customersTable);

  res.json(
    ListCustomersResponse.parse({
      data: customers.map(formatCustomer),
      total,
      limit: limit ?? 50,
      offset: offset ?? 0,
    })
  );
});

router.get("/customers/:customerId", async (req, res): Promise<void> => {
  const params = GetCustomerParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [customer] = await db
    .select()
    .from(customersTable)
    .where(eq(customersTable.id, params.data.customerId));
  if (!customer) {
    res.status(404).json({ error: "Customer not found" });
    return;
  }
  res.json(GetCustomerResponse.parse(formatCustomer(customer)));
});

router.patch("/customers/:customerId", async (req, res): Promise<void> => {
  const params = UpdateCustomerParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const body = UpdateCustomerBody.safeParse(req.body);
  if (!body.success) {
    res.status(400).json({ error: body.error.message });
    return;
  }
  const [customer] = await db
    .update(customersTable)
    .set(body.data)
    .where(eq(customersTable.id, params.data.customerId))
    .returning();
  if (!customer) {
    res.status(404).json({ error: "Customer not found" });
    return;
  }
  res.json(UpdateCustomerResponse.parse(formatCustomer(customer)));
});

router.post("/customers/:customerId/notes", async (req, res): Promise<void> => {
  const params = AddCustomerNoteParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const body = AddCustomerNoteBody.safeParse(req.body);
  if (!body.success) {
    res.status(400).json({ error: body.error.message });
    return;
  }
  const [existing] = await db
    .select()
    .from(customersTable)
    .where(eq(customersTable.id, params.data.customerId));
  if (!existing) {
    res.status(404).json({ error: "Customer not found" });
    return;
  }
  const existingNotes = Array.isArray(existing.notes) ? (existing.notes as Array<{ id: string; content: string; createdAt: string; author: string }>) : [];
  const newNote = {
    id: crypto.randomUUID(),
    content: body.data.content,
    createdAt: new Date().toISOString(),
    author: "Admin",
  };
  const [customer] = await db
    .update(customersTable)
    .set({ notes: [...existingNotes, newNote] })
    .where(eq(customersTable.id, params.data.customerId))
    .returning();
  res.status(201).json(GetCustomerResponse.parse(formatCustomer(customer!)));
});

export default router;
