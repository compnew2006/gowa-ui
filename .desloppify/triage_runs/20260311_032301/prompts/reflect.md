You are a triage analysis subagent with full codebase access.
Your job is to complete the **REFLECT** stage of triage planning.

Repo root: /Users/noiemany/Downloads/whatomate_GOWA/whatomate

## Standards

You are expected to produce **exceptional** work. The output of this triage becomes the
actual plan that an executor follows — if you are lazy, vague, or sloppy, real work gets
wasted. Concretely:

- **Read the actual source code.** Every opinion you form must come from reading the file,
  not from reading the issue title. Issues frequently exaggerate, miscount, or describe
  code that has already been fixed. Trust nothing until you verify it.
- **Have specific opinions.** "This seems like it could be an issue" is worthless. "This is
  a false positive because line 47 already uses the pattern the issue suggests" is useful.
- **Do the hard thinking.** If two issues seem related, figure out WHY. If something should
  be skipped, explain the specific reason for THIS issue, not a generic category.
- **Don't take shortcuts.** Reading 5 files and extrapolating to 30 is lazy. Read all 30.
  If you have too many, use subagents to parallelize — don't skip.
- The prompt below already contains the authoritative stage contract and prior reports.
  Do NOT search old triage run artifacts for alternate instructions unless you hit a
  concrete mismatch you need to explain.

## Output Contract

- **Do NOT run any `desloppify` commands.**
- **Do NOT debug, repair, reinstall, or inspect the `desloppify` CLI/environment.**
- **Do NOT mutate `plan.json` directly or indirectly.**
- Use shell/read-only repo inspection as needed, but your only deliverable is a plain-text
  stage report for the orchestrator to record and confirm.
- If the prompt mentions CLI commands, treat them as background context for the orchestrator,
  not instructions for you to execute.


## Prior Stage Reports


### OBSERVE Report
Verified key issue set and contradictions across dimensions. Cited issues reviewed: [critical] untested critical modules is genuine risk; [flat_han] handlers flat-directory overload is valid; [cache-mo] cache module fragmentation is valid; [inconsis] authorization helper adoption inconsistency is valid; [type_001] unchecked JSONB assertions are present in settings/parsing paths; [mixed-re] response envelope patterns are inconsistent across handlers; [dual-pro] dual-provider pattern appears intentional (not immediate debt); [module_l] package-level constants are acceptable and not a blocker; [no_init_] explicit initialization is healthy; [handlers] handlers package bloat shows real coupling risk.


## Issue Data

## New issues since last triage (63)
- review::.::holistic::abstraction_fitness::connection_manager_complexity: ConnectionManager accumulates responsibilities without clear boundaries
- review::.::holistic::abstraction_fitness::duplicate_setting_parsing: Chat close rating settings parsing duplicated across organization and instance
- review::.::holistic::abstraction_fitness::jsonb_type_safety: JSONB type used throughout without type safety or validation
- review::.::holistic::abstraction_fitness::pass_through_adapters: MessageProvider adapters are thin wrappers with minimal value-add
- review::.::holistic::ai_generated_debt::boilerplate_error_handling: Identical error handling patterns copied across handlers
- review::.::holistic::ai_generated_debt::commented_out_code_patterns: Struct definitions with extensive field comments but no behavioral documentation
- review::.::holistic::ai_generated_debt::duplicate_struct_definitions: Request/Response struct pairs duplicated with minor variations
- review::.::holistic::ai_generated_debt::generic_function_names: Helper functions use generic names instead of domain terminology
- review::.::holistic::ai_generated_debt::unnecessary_nil_checks: Defensive nil checks on non-nil values throughout codebase
- review::.::holistic::api_surface_coherence::inconsistent-param-ordering: Message sending functions have inconsistent parameter ordering across implementations
- review::.::holistic::api_surface_coherence::mixed-response-patterns: Inconsistent response envelope patterns across handlers
- review::.::holistic::authorization_consistency::inconsistent-auth-checks: Only 35 of 145 handlers use getOrgAndUserID, only 16 use requirePermission
- review::.::holistic::authorization_consistency::missing-resource-level-auth: Contact assignment and chat access policies use manual permission checks
- review::.::holistic::authorization_consistency::super-admin-org-switch: Super admin org switching logic embedded in getOrgID
- review::.::holistic::contract_coherence::contract_001_function_does_more_than_name_indicates: Functions perform side effects not indicated by name
- review::.::holistic::contract_coherence::contract_002_return_type_mismatch_with_docstring: Functions don't clearly document what they return on error
- review::.::holistic::contract_coherence::contract_003_inconsistent_error_handling: Functions in same module have different error handling contracts
- review::.::holistic::contract_coherence::contract_004_optional_parameters_inconsistent: Functions handle nil pointer parameters inconsistently
- review::.::holistic::convention_outlier::inconsistent_file_organization: handlers package mixes domains without subdirectories, making navigation difficult
- review::.::holistic::convention_outlier::inconsistent_request_response_patterns: Request/Response struct definitions scattered across handler files instead of centralized types package
- review::.::holistic::convention_outlier::mixed_naming_patterns: Inconsistent naming between package directories and file purposes
- review::.::holistic::cross_module_architecture::circular_dependency_internal_models: handlers directly import and manipulate internal/models without service layer, creating tight coupling
- review::.::holistic::cross_module_architecture::handlers_package_bloat: handlers package contains 145 files (50% of entire codebase), creating a hub module with excessive blast radius
- review::.::holistic::cross_module_architecture::pkg_whatsapp_duplication: Two WhatsApp packages (pkg/whatsapp and pkg/whatsmeow) with overlapping responsibilities and unclear boundaries
- review::.::holistic::dependency_health::appropriate_external_deps: External dependencies are minimal and appropriate for the domain
- review::.::holistic::dependency_health::whatsmeow_pinning: whatsmeow dependency pinned to specific commit - good for stability but may miss updates
- review::.::holistic::design_coherence::cache-module-fragmentation: cache.go has 31 disconnected function clusters - mixed responsibilities
- review::.::holistic::design_coherence::contacts-messaging-responsibility-bleed: contacts_messaging.go has 7 disconnected function clusters (13 functions total)
- review::.::holistic::design_coherence::monster-functions: Multiple files exceed 1000 LOC with high complexity scores
- review::.::holistic::design_coherence::parameter-bag-anti-pattern: Multiple files use config bag passing instead of structured parameters
- review::.::holistic::error_consistency::error_001_mixed_error_strategies: Inconsistent error handling across modules
- review::.::holistic::error_consistency::error_002_error_context_lost: Errors caught and re-thrown without wrapping original context
- review::.::holistic::error_consistency::error_003_silent_error_swallowing: Errors logged but not propagated or recovered from
- review::.::holistic::error_consistency::error_004_inconsistent_error_types: Mix of string errors, sentinel errors, and custom error types
- review::.::holistic::error_consistency::error_005_missing_io_error_handling: I/O operations without proper error handling
- review::.::holistic::high_level_elegance::handlers_package_bloat: handlers package contains 147 files with unclear domain boundaries
- review::.::holistic::high_level_elegance::monolithic_handler_files: Multiple handler files exceed 1000 LOC indicating poor decomposition
- review::.::holistic::incomplete_migration::deprecated-flow-status: Flow status uses deprecated marker but flows are still active
- review::.::holistic::incomplete_migration::dual-provider-pattern: Provider abstraction exists but both Meta and Whatsmeow are actively used
- review::.::holistic::initialization_coupling::module_level_constants: Package-level constants and compiled regexes are safe and appropriate
- review::.::holistic::initialization_coupling::no_init_functions: No init() functions found - all initialization is explicit and controlled
- review::.::holistic::logic_clarity::logic_001_dead_code_after_return: Dead code paths after unconditional returns
- review::.::holistic::logic_clarity::logic_002_redundant_null_checks: Redundant nil checks on values that cannot be nil
- review::.::holistic::logic_clarity::logic_003_complex_nested_conditionals: Deeply nested conditional logic that's hard to follow
- review::.::holistic::logic_clarity::logic_004_duplicate_code_blocks: Identical or very similar code in multiple branches
- review::.::holistic::low_level_elegance::message_send_options_pattern: Well-designed options pattern for message sending configuration
- review::.::holistic::low_level_elegance::restating_comments: Comments that merely restate the code without adding insight
- review::.::holistic::mid_level_elegance::app_god_object: App struct accumulates too many responsibilities and dependencies
- review::.::holistic::mid_level_elegance::authorization_pattern_inconsistency: Permission checking scattered and inconsistent across handlers
- review::.::holistic::mid_level_elegance::provider_abstraction_leak: MessageProvider interface creates unnecessary translation layers
- review::.::holistic::naming_quality::naming_001_generic_helper_functions: Generic helper function names that don't communicate intent
- review::.::holistic::naming_quality::naming_002_inconsistent_naming_patterns: Inconsistent naming between resolve/load/get prefixes
- review::.::holistic::naming_quality::naming_003_abbreviations_without_convention: Inconsistent abbreviations (org, ctx, req, resp)
- review::.::holistic::package_organization::ambiguous_package_boundaries: pkg/whatsmeow contains 67 files with domain logic, blurring library vs application boundary
- review::.::holistic::package_organization::flat_handlers_structure: handlers package is flat with 145 files, violating the 'flat directory overload' anti-pattern
- review::.::holistic::package_organization::missing_internal_abstractions: No internal service or repository layer - handlers directly access models and external services
- review::.::holistic::test_strategy::critical-untested-modules: High-complexity modules have zero test coverage
- review::.::holistic::test_strategy::missing-integration-tests: No evidence of cross-module integration tests
- review::.::holistic::test_strategy::test-production-coupling: Test files show evidence of implementation coupling
- review::.::holistic::type_safety::type_001_jsonb_type_assertions_without_checks: Type assertions on JSONB values without checking ok return
- review::.::holistic::type_safety::type_002_any_type_overuse: Excessive use of 'any' type loses type safety
- review::.::holistic::type_safety::type_003_string_uuid_conversions: Inconsistent UUID handling between string and uuid.UUID types
- review::.::holistic::type_safety::type_004_untyped_error_returns: Functions return error without specifying expected error types

## All open review issues (63)
- [medium] review::.::holistic::abstraction_fitness::connection_manager_complexity
  File: .
  Dimension: abstraction_fitness
  Summary: ConnectionManager accumulates responsibilities without clear boundaries
  Suggestion: Split into focused components: ClientPool (connection management), MetricsCollector (metrics), AvatarSyncer (avatars), CallManager (calls), QRCodeCache (caching). Use composition to assemble features.
- [high] review::.::holistic::abstraction_fitness::duplicate_setting_parsing
  File: .
  Dimension: abstraction_fitness
  Summary: Chat close rating settings parsing duplicated across organization and instance
  Suggestion: Extract settings parsing into a helper function that takes a settings map and applies overrides to a base struct. Use functional options pattern for settings composition.
- [medium] review::.::holistic::abstraction_fitness::jsonb_type_safety
  File: .
  Dimension: abstraction_fitness
  Summary: JSONB type used throughout without type safety or validation
  Suggestion: Define typed structs for settings with JSON marshaling. Use code generation or schema definition to ensure type safety. Consider protobuf or JSON Schema for validation.
- [high] review::.::holistic::abstraction_fitness::pass_through_adapters
  File: .
  Dimension: abstraction_fitness
  Summary: MessageProvider adapters are thin wrappers with minimal value-add
  Suggestion: Remove MessageProvider abstraction. Consolidate into a single client type or use configuration-based branching. The interface adds indirection without enabling true polymorphism.
- [high] review::.::holistic::ai_generated_debt::boilerplate_error_handling
  File: .
  Dimension: ai_generated_debt
  Summary: Identical error handling patterns copied across handlers
  Suggestion: Create middleware for common authorization patterns. Use method decorators for resource-level authorization. Extract handler setup into template/generic functions.
- [low] review::.::holistic::ai_generated_debt::commented_out_code_patterns
  File: .
  Dimension: ai_generated_debt
  Summary: Struct definitions with extensive field comments but no behavioral documentation
  Suggestion: Add package-level documentation explaining workflows and field interactions. Group related fields into sub-structs. Add validation documentation with examples.
- [medium] review::.::holistic::ai_generated_debt::duplicate_struct_definitions
  File: .
  Dimension: ai_generated_debt
  Summary: Request/Response struct pairs duplicated with minor variations
  Suggestion: Use code generation or struct embedding to reduce duplication. Consider using same struct with JSON tags for both request/response where validation allows.
- [low] review::.::holistic::ai_generated_debt::generic_function_names
  File: .
  Dimension: ai_generated_debt
  Summary: Helper functions use generic names instead of domain terminology
  Suggestion: Move generic utilities to appropriate packages (timeutil, dbutil, httputil). Rename functions to reflect domain context (parseMessageListPagination, parseAnalyticsDateRange).
- [medium] review::.::holistic::ai_generated_debt::unnecessary_nil_checks
  File: .
  Dimension: ai_generated_debt
  Summary: Defensive nil checks on non-nil values throughout codebase
  Suggestion: Remove nil receiver checks. If called with nil, let it panic - that's a bug to fix. Add nil checks only for interface/pointer parameters that can legitimately be nil.
- [high] review::.::holistic::api_surface_coherence::inconsistent-param-ordering
  File: .
  Dimension: api_surface_coherence
  Summary: Message sending functions have inconsistent parameter ordering across implementations
  Suggestion: Standardize message sending API surface - either consolidate on provider.MessageProvider interface or create unified request/response types. The current mix of struct-based and positional parameters creates confusion.
- [medium] review::.::holistic::api_surface_coherence::mixed-response-patterns
  File: .
  Dimension: api_surface_coherence
  Summary: Inconsistent response envelope patterns across handlers
  Suggestion: Create consistent response helper functions or standardize on one pattern. The variation makes the API surface harder to learn and use consistently.
- [high] review::.::holistic::authorization_consistency::inconsistent-auth-checks
  File: .
  Dimension: authorization_consistency
  Summary: Only 35 of 145 handlers use getOrgAndUserID, only 16 use requirePermission
  Suggestion: Audit all 516 handler functions and categorize: 1) Public routes (health, webhooks), 2) Authenticated routes missing RBAC, 3) Routes with proper auth. Create middleware or decorator pattern to enforce consistent authorization.
- [medium] review::.::holistic::authorization_consistency::missing-resource-level-auth
  File: .
  Dimension: authorization_consistency
  Summary: Contact assignment and chat access policies use manual permission checks
  Suggestion: Consolidate chat access and send restriction policies into a single authorization layer. Create a ChatAccessAuthorizer interface that consolidates all permission checks.
- [medium] review::.::holistic::authorization_consistency::super-admin-org-switch
  File: .
  Dimension: authorization_consistency
  Summary: Super admin org switching logic embedded in getOrgID
  Suggestion: Extract org switching into authorizeOrgSwitch(orgID, userID, overrideOrgID) function. Add audit logging for org switches by super admins. The current inline logic is hard to test and audit.
- [high] review::.::holistic::contract_coherence::contract_001_function_does_more_than_name_indicates
  File: .
  Dimension: contract_coherence
  Summary: Functions perform side effects not indicated by name
  Suggestion: Rename functions to reflect all side effects (e.g., getAndDecryptChatbotSettings, sendAndStoreCloseRatingPrompt). Split functions if they do multiple unrelated things
- [high] review::.::holistic::contract_coherence::contract_002_return_type_mismatch_with_docstring
  File: .
  Dimension: contract_coherence
  Summary: Functions don't clearly document what they return on error
  Suggestion: Separate validation from response sending. Use sentinel error pattern consistently. Document behavior when input is invalid (clamp vs error vs default)
- [high] review::.::holistic::contract_coherence::contract_003_inconsistent_error_handling
  File: .
  Dimension: contract_coherence
  Summary: Functions in same module have different error handling contracts
  Suggestion: Establish standard error handling contract: either always return error OR always send response and return nil. Don't mix patterns in same module
- [medium] review::.::holistic::contract_coherence::contract_004_optional_parameters_inconsistent
  File: .
  Dimension: contract_coherence
  Summary: Functions handle nil pointer parameters inconsistently
  Suggestion: Document nil parameter handling in function comments. Use consistent pattern (nil = unfiltered, nil = error, or nil = use default). Consider using functional options pattern for multiple optional params
- [high] review::.::holistic::convention_outlier::inconsistent_file_organization
  File: .
  Dimension: convention_outlier
  Summary: handlers package mixes domains without subdirectories, making navigation difficult
  Suggestion: Organize handlers by domain: handlers/accounts/, handlers/campaigns/, handlers/chatbot/, handlers/contacts/. This improves discoverability without adding abstraction layers.
- [medium] review::.::holistic::convention_outlier::inconsistent_request_response_patterns
  File: .
  Dimension: convention_outlier
  Summary: Request/Response struct definitions scattered across handler files instead of centralized types package
  Suggestion: Create internal/types or internal/dto package for shared request/response structures. Keep handler-specific types in handlers, but extract common patterns.
- [medium] review::.::holistic::convention_outlier::mixed_naming_patterns
  File: .
  Dimension: convention_outlier
  Summary: Inconsistent naming between package directories and file purposes
  Suggestion: Establish naming convention: entity files for CRUD (contacts.go), feature files for cross-cutting concerns (chat_policy.go), worker files for background jobs (campaign_worker.go). Document this in CONTRIBUTE.md.
- [high] review::.::holistic::cross_module_architecture::circular_dependency_internal_models
  File: .
  Dimension: cross_module_architecture
  Summary: handlers directly import and manipulate internal/models without service layer, creating tight coupling
  Suggestion: Introduce a repository/service layer between handlers and models. Create domain-specific packages (e.g., internal/rating, internal/campaign) that encapsulate business logic and data access.
- [high] review::.::holistic::cross_module_architecture::handlers_package_bloat
  File: .
  Dimension: cross_module_architecture
  Summary: handlers package contains 145 files (50% of entire codebase), creating a hub module with excessive blast radius
  Suggestion: Split handlers into focused packages: handlers/ (HTTP only), services/ (business logic), policies/ (authorization), workers/ (background jobs). Extract domain models from handlers to reduce coupling.
- [medium] review::.::holistic::cross_module_architecture::pkg_whatsapp_duplication
  File: .
  Dimension: cross_module_architecture
  Summary: Two WhatsApp packages (pkg/whatsapp and pkg/whatsmeow) with overlapping responsibilities and unclear boundaries
  Suggestion: Consider consolidating WhatsApp provider logic under a single package with clear internal structure. Move domain-specific whatsmeow logic to internal/whatsmeow if it's not a reusable library.
- [high] review::.::holistic::dependency_health::appropriate_external_deps
  File: .
  Dimension: dependency_health
  Summary: External dependencies are minimal and appropriate for the domain
  Suggestion: Current dependency management is healthy. Consider periodic audits to remove unused dependencies.
- [low] review::.::holistic::dependency_health::whatsmeow_pinning
  File: .
  Dimension: dependency_health
  Summary: whatsmeow dependency pinned to specific commit - good for stability but may miss updates
  Suggestion: Document the update process for whatsmeow dependency. Consider automated dependency scanning to alert on new versions.
- [high] review::.::holistic::design_coherence::cache-module-fragmentation
  File: .
  Dimension: design_coherence
  Summary: cache.go has 31 disconnected function clusters - mixed responsibilities
  Suggestion: Extract a CacheManager interface with concrete implementation. Separate concerns: CacheManager (get/set/invalidate), CryptoHelper (decrypt/encrypt), and DomainCacheLayer (chatbot, permissions, etc.).
- [high] review::.::holistic::design_coherence::contacts-messaging-responsibility-bleed
  File: .
  Dimension: design_coherence
  Summary: contacts_messaging.go has 7 disconnected function clusters (13 functions total)
  Suggestion: Split into focused files: message_send.go, message_reactions.go, message_media.go, typing_indicators.go. Each should have a single clear responsibility.
- [high] review::.::holistic::design_coherence::monster-functions
  File: .
  Dimension: design_coherence
  Summary: Multiple files exceed 1000 LOC with high complexity scores
  Suggestion: Break down these mega-files into focused modules. For example, chatbot_processor.go could be split into: rule_engine.go, flow_executor.go, ai_context.go, session_manager.go. Aim for files under 500 LOC.
- [medium] review::.::holistic::design_coherence::parameter-bag-anti-pattern
  File: .
  Dimension: design_coherence
  Summary: Multiple files use config bag passing instead of structured parameters
  Suggestion: Refactor OutgoingMessageRequest into focused request types: TextMessageRequest, MediaMessageRequest, TemplateMessageRequest, FlowMessageRequest. Use interface or builder pattern instead of god-object.
- [high] review::.::holistic::error_consistency::error_001_mixed_error_strategies
  File: .
  Dimension: error_consistency
  Summary: Inconsistent error handling across modules
  Suggestion: Choose one error strategy per layer (handlers send responses, services return errors). Use consistent error wrapping. Document error handling strategy for each layer
- [high] review::.::holistic::error_consistency::error_002_error_context_lost
  File: .
  Dimension: error_consistency
  Summary: Errors caught and re-thrown without wrapping original context
  Suggestion: Always wrap errors with context using fmt.Errorf or errors.Wrap. Include what operation failed, not just that it failed. Use structured logging with original error
- [medium] review::.::holistic::error_consistency::error_003_silent_error_swallowing
  File: .
  Dimension: error_consistency
  Summary: Errors logged but not propagated or recovered from
  Suggestion: Either propagate the error or explicitly document why it's safe to ignore. For cache, consider corrupted data eviction. For DB updates, consider retry or user notification
- [medium] review::.::holistic::error_consistency::error_004_inconsistent_error_types
  File: .
  Dimension: error_consistency
  Summary: Mix of string errors, sentinel errors, and custom error types
  Suggestion: Define custom error types for domain errors. Use sentinel errors for control flow (validation failures). Use string errors only for user-facing messages. Document error type hierarchy
- [medium] review::.::holistic::error_consistency::error_005_missing_io_error_handling
  File: .
  Dimension: error_consistency
  Summary: I/O operations without proper error handling
  Suggestion: Check for specific I/O error types (os.ErrPermission, os.ErrExist). Handle disk space errors explicitly. Add context about what file/path was being accessed
- [high] review::.::holistic::high_level_elegance::handlers_package_bloat
  File: .
  Dimension: high_level_elegance
  Summary: handlers package contains 147 files with unclear domain boundaries
  Suggestion: Split handlers into domain subpackages: handlers/contacts, handlers/campaigns, handlers/chatbot, handlers/analytics, handlers/instances, handlers/organizations. Each subpackage should have clear ownership boundaries.
- [high] review::.::holistic::high_level_elegance::monolithic_handler_files
  File: .
  Dimension: high_level_elegance
  Summary: Multiple handler files exceed 1000 LOC indicating poor decomposition
  Suggestion: Break down monolithic files into focused single-responsibility modules. Extract validation, persistence, and business logic into separate packages. Apply Command/Query pattern to reduce handler complexity.
- [medium] review::.::holistic::incomplete_migration::deprecated-flow-status
  File: .
  Dimension: incomplete_migration
  Summary: Flow status uses deprecated marker but flows are still active
  Suggestion: Either complete the migration by removing deprecated flows from queries/API, or remove the deprecated status if it's not being enforced. The current state is partial.
- [low] review::.::holistic::incomplete_migration::dual-provider-pattern
  File: .
  Dimension: incomplete_migration
  Summary: Provider abstraction exists but both Meta and Whatsmeow are actively used
  Suggestion: This is actually good design - the abstraction allows multi-provider support. However, document which provider should be used for new features or if both should be maintained indefinitely.
- [medium] review::.::holistic::initialization_coupling::module_level_constants
  File: .
  Dimension: initialization_coupling
  Summary: Package-level constants and compiled regexes are safe and appropriate
  Suggestion: Current patterns are acceptable. Document the immutability contract for package-level variables to prevent future misuse.
- [high] review::.::holistic::initialization_coupling::no_init_functions
  File: .
  Dimension: initialization_coupling
  Summary: No init() functions found - all initialization is explicit and controlled
  Suggestion: Continue current pattern. The explicit initialization approach is excellent for testability and avoiding boot-order dependencies.
- [high] review::.::holistic::logic_clarity::logic_001_dead_code_after_return
  File: .
  Dimension: logic_clarity
  Summary: Dead code paths after unconditional returns
  Suggestion: Either return the error to the caller or remove the error path entirely. If fallback is intentional, add explanatory comment. Consider extracting date parsing logic to avoid duplication
- [high] review::.::holistic::logic_clarity::logic_002_redundant_null_checks
  File: .
  Dimension: logic_clarity
  Summary: Redundant nil checks on values that cannot be nil
  Suggestion: Remove nil checks on method receivers. If defensive programming is needed, use assertions or document preconditions. Consider making these free functions if nil checks are truly needed
- [medium] review::.::holistic::logic_clarity::logic_003_complex_nested_conditionals
  File: .
  Dimension: logic_clarity
  Summary: Deeply nested conditional logic that's hard to follow
  Suggestion: Extract nested logic into separate functions with clear names. Use early returns to reduce nesting. Consider strategy pattern for type handling
- [medium] review::.::holistic::logic_clarity::logic_004_duplicate_code_blocks
  File: .
  Dimension: logic_clarity
  Summary: Identical or very similar code in multiple branches
  Suggestion: Extract common logic to helper functions. Create shared error handling helpers. Use table-driven approaches for repetitive validation
- [low] review::.::holistic::low_level_elegance::message_send_options_pattern
  File: .
  Dimension: low_level_elegance
  Summary: Well-designed options pattern for message sending configuration
  Suggestion: Consider applying this options pattern to other complex operations (contact queries, campaign operations). The current design is solid.
- [high] review::.::holistic::low_level_elegance::restating_comments
  File: .
  Dimension: low_level_elegance
  Summary: Comments that merely restate the code without adding insight
  Suggestion: Remove restating comments. Keep comments that explain WHY not WHAT. For public APIs, document edge cases, preconditions, and invariants instead of repeating names.
- [high] review::.::holistic::mid_level_elegance::app_god_object
  File: .
  Dimension: mid_level_elegance
  Summary: App struct accumulates too many responsibilities and dependencies
  Suggestion: Extract domain services from App into separate packages (orgservice, authservice, contactservice). Handlers should be thin wrappers that delegate to services. App should only hold infrastructure dependencies.
- [medium] review::.::holistic::mid_level_elegance::authorization_pattern_inconsistency
  File: .
  Dimension: mid_level_elegance
  Summary: Permission checking scattered and inconsistent across handlers
  Suggestion: Implement consistent authorization middleware or helper. Choose one pattern (error return vs bool) and apply uniformly. Consider RBAC middleware that sets context before handlers.
- [medium] review::.::holistic::mid_level_elegance::provider_abstraction_leak
  File: .
  Dimension: mid_level_elegance
  Summary: MessageProvider interface creates unnecessary translation layers
  Suggestion: Consider removing MessageProvider abstraction or merge implementations into a single unified client. The two-provider pattern (Meta vs Whatsmeow) creates more complexity than it solves.
- [high] review::.::holistic::naming_quality::naming_001_generic_helper_functions
  File: .
  Dimension: naming_quality
  Summary: Generic helper function names that don't communicate intent
  Suggestion: Rename helpers to indicate their domain context (e.g., parseUUIDFromPath, parsePaginationParams, parseDateQueryParam). Group related helpers by domain (contact helpers, message helpers, etc.)
- [high] review::.::holistic::naming_quality::naming_002_inconsistent_naming_patterns
  File: .
  Dimension: naming_quality
  Summary: Inconsistent naming between resolve/load/get prefixes
  Suggestion: Establish naming convention: 'get' for simple retrieval, 'load' for database queries, 'resolve' for computed/derived values, 'find' for searches. Add 'Cached' suffix consistently to all cached accessors
- [medium] review::.::holistic::naming_quality::naming_003_abbreviations_without_convention
  File: .
  Dimension: naming_quality
  Summary: Inconsistent abbreviations (org, ctx, req, resp)
  Suggestion: Establish standard abbreviations: org (organization), ctx (context), req (request), resp (response). Use consistently or document exceptions. Prefer full names in public APIs
- [medium] review::.::holistic::package_organization::ambiguous_package_boundaries
  File: .
  Dimension: package_organization
  Summary: pkg/whatsmeow contains 67 files with domain logic, blurring library vs application boundary
  Suggestion: Move whatsmeow application-specific logic to internal/whatsmeow/. Keep only generic whatsmeow wrappers in pkg/ if they're reusable across projects.
- [high] review::.::holistic::package_organization::flat_handlers_structure
  File: .
  Dimension: package_organization
  Summary: handlers package is flat with 145 files, violating the 'flat directory overload' anti-pattern
  Suggestion: Reorganize handlers into domain-based subdirectories: handlers/accounts/, handlers/campaigns/, handlers/chatbot/, handlers/contacts/, handlers/analytics/. Keep HTTP handlers in these dirs, move business logic to services/.
- [medium] review::.::holistic::package_organization::missing_internal_abstractions
  File: .
  Dimension: package_organization
  Summary: No internal service or repository layer - handlers directly access models and external services
  Suggestion: Introduce internal/services/ package with domain services (e.g., services/campaign, services/rating, services/chatbot). Handlers become thin HTTP adapters that delegate to services.
- [high] review::.::holistic::test_strategy::critical-untested-modules
  File: .
  Dimension: test_strategy
  Summary: High-complexity modules have zero test coverage
  Suggestion: Prioritize test coverage for these 5 modules immediately. They are critical paths (caching, messaging, data import) with zero coverage. Start with integration tests before unit tests.
- [high] review::.::holistic::test_strategy::missing-integration-tests
  File: .
  Dimension: test_strategy
  Summary: No evidence of cross-module integration tests
  Suggestion: Add integration test suite that covers critical workflows: 1) User auth → Permission check → Resource access, 2) Campaign launch → Message queue → Delivery, 3) Webhook receipt → Message processing → WebSocket broadcast.
- [medium] review::.::holistic::test_strategy::test-production-coupling
  File: .
  Dimension: test_strategy
  Summary: Test files show evidence of implementation coupling
  Suggestion: Extract test utilities into test/testutil package (already exists). Break large test files into focused test suites. Add interface boundaries to reduce coupling to App internals.
- [high] review::.::holistic::type_safety::type_001_jsonb_type_assertions_without_checks
  File: .
  Dimension: type_safety
  Summary: Type assertions on JSONB values without checking ok return
  Suggestion: Always check both return values from type assertions. Create helper functions for common JSONB access patterns. Consider using a typesafe wrapper for JSONB
- [high] review::.::holistic::type_safety::type_002_any_type_overuse
  File: .
  Dimension: type_safety
  Summary: Excessive use of 'any' type loses type safety
  Suggestion: Define specific types for metadata (e.g., type-safe structs with JSON tags). Use union types or interfaces for Content instead of any. Document the expected structure when any is necessary
- [medium] review::.::holistic::type_safety::type_003_string_uuid_conversions
  File: .
  Dimension: type_safety
  Summary: Inconsistent UUID handling between string and uuid.UUID types
  Suggestion: Standardize on uuid.UUID in internal code, convert to string only at API boundaries. Use consistent pointer vs value semantics (prefer value for UUID)
- [medium] review::.::holistic::type_safety::type_004_untyped_error_returns
  File: .
  Dimension: type_safety
  Summary: Functions return error without specifying expected error types
  Suggestion: Define custom error types for expected failure modes. Use error wrapping with fmt.Errorf. Document in comments what errors can be returned

## Dimension scores (context)
- AI generated debt: 68.0% (strict: 68.0%, 5 issues)
- API coherence: 72.0% (strict: 72.0%, 2 issues)
- Abstraction fit: 54.0% (strict: 54.0%, 4 issues)
- Auth consistency: 68.0% (strict: 68.0%, 3 issues)
- Code quality: 84.4% (strict: 84.4%, 6 issues)
- Contracts: 70.0% (strict: 70.0%, 4 issues)
- Convention drift: 68.0% (strict: 68.0%, 3 issues)
- Cross-module arch: 72.0% (strict: 72.0%, 3 issues)
- Dep health: 85.0% (strict: 85.0%, 2 issues)
- Design coherence: 55.0% (strict: 55.0%, 4 issues)
- Duplication: 97.5% (strict: 97.5%, 49 issues)
- Error consistency: 58.0% (strict: 58.0%, 5 issues)
- File health: 85.8% (strict: 85.8%, 37 issues)
- High elegance: 62.0% (strict: 62.0%, 2 issues)
- Init coupling: 95.0% (strict: 95.0%, 2 issues)
- Logic clarity: 68.0% (strict: 68.0%, 4 issues)
- Low elegance: 71.0% (strict: 71.0%, 2 issues)
- Mid elegance: 58.0% (strict: 58.0%, 3 issues)
- Naming quality: 72.0% (strict: 72.0%, 3 issues)
- Security: 100.0% (strict: 98.9%, 0 issues)
- Stale migration: 85.0% (strict: 85.0%, 2 issues)
- Structure nav: 62.0% (strict: 62.0%, 3 issues)
- Test health: 60.0% (strict: 51.7%, 85 issues)
- Test strategy: 45.0% (strict: 45.0%, 3 issues)
- Type safety: 65.0% (strict: 65.0%, 4 issues)


## Required Issue Hashes
Total open review issues: 63
Every one of these hashes must appear exactly once in your cluster/skip blueprint.
Do not repeat hashes outside that blueprint.
ambiguous_package_boundaries, app_god_object, appropriate_external_deps, authorization_pattern_inconsistency, boilerplate_error_handling, cache-module-fragmentation, circular_dependency_internal_models, commented_out_code_patterns, connection_manager_complexity, contacts-messaging-responsibility-bleed, contract_001_function_does_more_than_name_indicates, contract_002_return_type_mismatch_with_docstring, contract_003_inconsistent_error_handling, contract_004_optional_parameters_inconsistent, critical-untested-modules, deprecated-flow-status, dual-provider-pattern, duplicate_setting_parsing, duplicate_struct_definitions, error_001_mixed_error_strategies, error_002_error_context_lost, error_003_silent_error_swallowing, error_004_inconsistent_error_types, error_005_missing_io_error_handling, flat_handlers_structure, generic_function_names, handlers_package_bloat, handlers_package_bloat, inconsistent-auth-checks, inconsistent-param-ordering, inconsistent_file_organization, inconsistent_request_response_patterns, jsonb_type_safety, logic_001_dead_code_after_return, logic_002_redundant_null_checks, logic_003_complex_nested_conditionals, logic_004_duplicate_code_blocks, message_send_options_pattern, missing-integration-tests, missing-resource-level-auth, missing_internal_abstractions, mixed-response-patterns, mixed_naming_patterns, module_level_constants, monolithic_handler_files, monster-functions, naming_001_generic_helper_functions, naming_002_inconsistent_naming_patterns, naming_003_abbreviations_without_convention, no_init_functions, parameter-bag-anti-pattern, pass_through_adapters, pkg_whatsapp_duplication, provider_abstraction_leak, restating_comments, super-admin-org-switch, test-production-coupling, type_001_jsonb_type_assertions_without_checks, type_002_any_type_overuse, type_003_string_uuid_conversions, type_004_untyped_error_returns, unnecessary_nil_checks, whatsmeow_pinning

## Coverage Ledger Template
Your final report MUST contain a `## Coverage Ledger` section with one line per issue.
Allowed forms:
- `- abcd1234 -> cluster "cluster-name"`
- `- abcd1234 -> skip "specific-reason-tag"`
Do not mention hashes outside the `## Coverage Ledger` section.
- ambiguous_package_boundaries -> TODO
- app_god_object -> TODO
- appropriate_external_deps -> TODO
- authorization_pattern_inconsistency -> TODO
- boilerplate_error_handling -> TODO
- cache-module-fragmentation -> TODO
- circular_dependency_internal_models -> TODO
- commented_out_code_patterns -> TODO
- connection_manager_complexity -> TODO
- contacts-messaging-responsibility-bleed -> TODO
- contract_001_function_does_more_than_name_indicates -> TODO
- contract_002_return_type_mismatch_with_docstring -> TODO
- contract_003_inconsistent_error_handling -> TODO
- contract_004_optional_parameters_inconsistent -> TODO
- critical-untested-modules -> TODO
- deprecated-flow-status -> TODO
- dual-provider-pattern -> TODO
- duplicate_setting_parsing -> TODO
- duplicate_struct_definitions -> TODO
- error_001_mixed_error_strategies -> TODO
- error_002_error_context_lost -> TODO
- error_003_silent_error_swallowing -> TODO
- error_004_inconsistent_error_types -> TODO
- error_005_missing_io_error_handling -> TODO
- flat_handlers_structure -> TODO
- generic_function_names -> TODO
- handlers_package_bloat -> TODO
- handlers_package_bloat -> TODO
- inconsistent-auth-checks -> TODO
- inconsistent-param-ordering -> TODO
- inconsistent_file_organization -> TODO
- inconsistent_request_response_patterns -> TODO
- jsonb_type_safety -> TODO
- logic_001_dead_code_after_return -> TODO
- logic_002_redundant_null_checks -> TODO
- logic_003_complex_nested_conditionals -> TODO
- logic_004_duplicate_code_blocks -> TODO
- message_send_options_pattern -> TODO
- missing-integration-tests -> TODO
- missing-resource-level-auth -> TODO
- missing_internal_abstractions -> TODO
- mixed-response-patterns -> TODO
- mixed_naming_patterns -> TODO
- module_level_constants -> TODO
- monolithic_handler_files -> TODO
- monster-functions -> TODO
- naming_001_generic_helper_functions -> TODO
- naming_002_inconsistent_naming_patterns -> TODO
- naming_003_abbreviations_without_convention -> TODO
- no_init_functions -> TODO
- parameter-bag-anti-pattern -> TODO
- pass_through_adapters -> TODO
- pkg_whatsapp_duplication -> TODO
- provider_abstraction_leak -> TODO
- restating_comments -> TODO
- super-admin-org-switch -> TODO
- test-production-coupling -> TODO
- type_001_jsonb_type_assertions_without_checks -> TODO
- type_002_any_type_overuse -> TODO
- type_003_string_uuid_conversions -> TODO
- type_004_untyped_error_returns -> TODO
- unnecessary_nil_checks -> TODO
- whatsmeow_pinning -> TODO

## REFLECT Stage Instructions

Your task: using the verdicts from observe, design the cluster structure.

**A strategy is NOT a restatement of observe.** Observe says "here's what I found." Reflect
says "here's what we should DO about it, and here's what we should NOT do, and here's WHY."

**The Structured Observe Assessments table (provided below) is your primary input.** It contains
a per-issue verdict (genuine/false-positive/exaggerated/over-engineering) with reasoning. Use
these verdicts as authoritative — do not second-guess observe unless you have specific evidence.
Issues with verdict `false-positive` or `over-engineering` should go into skip lines, not clusters.

### What you must do:

1. **Filter:** which issues are genuine (from the observe assessments table)?
2. **Map:** for each genuine issue, what file/directory does it touch?
3. **Group:** which issues share files or directories? These become clusters.
4. **Skip:** which issues should be skipped? (with per-issue justification — "low priority" is
   not a justification; "the fix would add a 50-line abstraction to save 3 lines of duplication" is)
5. **Order:** which clusters depend on others? What's the execution sequence?
6. **Check recurring patterns** — compare current issues against resolved history. If the same
   dimension keeps producing issues, that's a root cause that needs addressing, not just
   another round of fixes.
7. **Account for every issue exactly once** — every open issue hash must appear in exactly one
   cluster line or one skip line. Do not drop hashes, and do not repeat a hash in multiple
   clusters or in both a cluster and a skip.

### Your report MUST include both a coverage ledger and a concrete cluster blueprint

This blueprint is what the organize stage will execute. Be specific:
```
## Coverage Ledger
- a5996373 -> cluster "travel-structure-contract-unification"
- fb113678 -> skip "false-positive-current-code"

## Cluster Blueprint
Cluster "media-lightbox-hooks" (all in src/domains/media-lightbox/)
Cluster "task-typing" (both touch src/types/database.ts)

## Skip Decisions
Skip "false-positive-current-code" (false positive per observe)
```

### Hard accounting rule

- Start your report with a `## Coverage Ledger` section.
- In that section, mention each issue hash **once and only once** on its own ledger line.
- Do **not** mention issue hashes again in cluster rationale paragraphs, recurring-pattern notes,
  or ordering explanations. After the ledger, refer to clusters by name.
- Before finishing, do a self-check: the ledger must cover all open issue hashes exactly once.

### What a LAZY reflect looks like (will be rejected):
- Restating observe findings in slightly different words
- "We should prioritize high-impact items and defer low-priority ones"
- A bulleted list of dimensions without any strategic thinking
- Ignoring recurring patterns
- No `## Coverage Ledger`
- No cluster blueprint (just vague grouping ideas)
- Missing or duplicated issue hashes

### What a GOOD reflect looks like:
- "50% false positive rate. Of 34 issues, 17 are genuine. 10 of those are batch-scriptable
  convention fixes (zero risk, 30 min) — cluster 'convention-batch'. The remaining 7 split into
  3 clusters by file proximity: 'media-lightbox-hooks' (issues X,Y,Z — all in src/domains/media-lightbox/),
  'timeline-cleanup' (issues A,B,C — touching Timeline components), 'task-typing' (issues D,E).
  Skip: issue W (false positive), issue V (over-engineering).
  design_coherence recurs (2 resolved, 5 open) but only 1 of the 5 actually warrants work."

When done, write a plain-text reflect report with a concrete cluster blueprint.
The orchestrator records and confirms the stage.



## Validation Requirements
- Stage must be recorded with a 100+ char report
- Report must mention recurring dimension names (if any exist)
- Report must include a `## Coverage Ledger` section
- Report must account for every open review issue exactly once (no missing or duplicate hashes)
- Stage must be confirmed with an 80+ char attestation
