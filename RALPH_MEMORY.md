# Ralph Memory

## 2026-05-30 Issue: Whatsmeow adapter concrete type assertion in async background worker and test suite compilation

- **The Trap:** Type asserting `w.MessageProvider` to concrete `*whatsmeow.WhatsmeowAdapter` inside the background worker prevented constructing elegant unit tests without having to instantiate complex database connections, connection pools, and redis clients for whatsmeow adapter's dependencies. Furthermore, `MockQueue` and `MockJobHandler` in the shared `testutil` package were missing the newly added `EnqueueWhatsAppFilter` and `HandleWhatsAppFilterJob` queue method signatures, causing broad test suite compilation failures.
- **The Reality:** The background worker only needs to query the `whatsmeow.Client` from the provider. Type-asserting to a local, minimal interface `interface { GetClient(ctx context.Context, instanceID string) (*whatsmeow.Client, error) }` decouples the worker cleanly, allowing test harnesses to supply mock providers. `MockQueue` and `MockJobHandler` need to maintain perfect synchronization with the master `Queue` and `JobHandler` interface signatures.
- **The Fix:** Changed the concrete type assertion in `checkWhatsmeowContacts` to a local `whatsmeowClientGetter` interface assertion. Implemented `EnqueueWhatsAppFilter` on `MockQueue` and `HandleWhatsAppFilterJob` on `MockJobHandler` in `test/testutil/mocks.go`, completely resolving compile errors. Created dedicated handler and worker test suites running and compiling with 100% success.
- **The Law:** Always use local interface assertions instead of concrete provider type assertions for multi-provider adapters in async tasks to ensure clean mock-ability, and keep the test suite mocks fully in sync with standard interface signature expansions.

## 2026-05-30 Issue: Vue 3 Generic computed ref type inference compiler limits inside custom pagination composable

- **The Trap:** Attempting to store selection objects with a generic type `ref<Map<string | number, T>>` caused severe `UnwrapRefSimple<T>` compiler errors under Vue 3's strict type checker, since the generic type parameter cannot be parsed automatically for nested UnwrapRef structures inside maps.
- **The Reality:** Vue's type-wrapper systems have strict limitations when generic parameters are nested inside Map or Set templates inside reactive references. We can declare the Map value type as `any` locally to allow flawless reactive setter and getter calls, while exposing the mapped values list via a strongly typed computed list casted as `T[]` to preserve downstream type safety.
- **The Fix:** Changed the Map generic type in `useSelectableTable.ts` from `Map<string | number, T>` to `Map<string | number, any>`, and safely typecast the computed array return using `as T[]`.
- **The Law:** Never map raw generic parameters directly inside reactive Map/Set references in Vue 3 composables; use clean internal `any` maps and cast exposed computed arrays to retain type safety while bypassing UnwrapRef compilation limits.

## 2026-05-30 Issue: Playwright E2E Nested API Interception Failures and Strict Selector Collisions

- **The Trap:** Playwright's standard glob path matcher `*` matches any sequence of characters *except* a slash `/`. A pattern like `**/api/whatsapp-filter/batches*` did not intercept nested results requests like `/api/whatsapp-filter/batches/batch-12345678/results`. In addition, matching text locators such as `page.locator("text=Contact Name 1")` or `page.getByRole("checkbox", { name: "Select row 1" })` resolve to multiple elements (e.g. `Contact Name 10`, `Select row 11`) and trigger strict-mode violations in Playwright.
- **The Reality:** Playwright needs a robust RegExp route interceptor: `page.route(/\/api\/whatsapp-filter\/batches/, ...)` which cleanly intercepts all sub-routes without escaping boundaries. We must enforce exact matches on all dynamic indices by leveraging `{ exact: true }` parameters inside Playwright locator queries.
- **The Fix:** Refactored mock interceptors in `whatsapp-filter.spec.ts` to use explicit RegExps and passed exact matcher arguments into Playwright role selection locators.
- **The Law:** Always use regular expression patterns instead of standard globs when routing nested sub-routes in Playwright, and apply `{ exact: true }` on cell and checkbox selectors containing indexed values to eliminate strict-mode element collisions.

## 2026-05-30 Issue: Strict Typechecking Mismatch on UI Components and Mismatched Pagination Props in Existing Views

- **The Trap:** Existing Vue components such as `GroupSearch.vue` failed typecheck validation because class attributes inside custom `<Card>` wrapper components were bound using objects (e.g. `:class="{ 'ring-2 ring-primary': active }"`) which are rejected by strict typescript definitions expecting simple `string` types. Furthermore, standard pagination subcomponents were wired with incorrect prop names (e.g. `:page` instead of `:current-page`), leading to typescript compilation blockages.
- **The Reality:** External/custom wrapped shadcn-vue components often enforce a strict `string` signature on class attributes. Changing bindings to conditional strings ensures flawless type check alignment. Shared subcomponents like `PaginationControls` must have their prop bindings mapped exactly to the declared types (`currentPage`, `totalPages`, `totalItems`, `pageSize`).
- **The Fix:** Refactored object class bindings on `<Card>` inside `GroupSearch.vue` to conditional string expressions, and updated the `PaginationControls` props and emits to exactly match their typed signatures.
- **The Law:** Always use conditional string expressions instead of class objects when binding classes on custom UI wrappers, and double-check shared subcomponent definitions to ensure prop names align precisely with their typescript definitions.


