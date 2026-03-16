You are a focused subagent reviewer for a single holistic investigation batch.

Repository root: /Users/noiemany/Downloads/whatomate_GOWA/whatomate
Blind packet: /Users/noiemany/Downloads/whatomate_GOWA/whatomate/.desloppify/review_packet_blind.json
Batch index: 19
Batch name: api_surface_coherence
Batch rationale: no direct batch mapping for api_surface_coherence; using representative files

DIMENSION TO EVALUATE:

## api_surface_coherence
Inconsistent API shapes, mixed sync/async, overloaded interfaces
Look for:
- Inconsistent API shapes: similar functions with different parameter ordering or naming
- Mixed sync/async in the same module's public API
- Overloaded interfaces: one function doing too many things based on argument types
- Missing error contracts: no documentation or types indicating what can fail
- Public functions with >5 parameters (API boundary may be wrong)
Skip:
- Internal/private APIs where flexibility is acceptable
- Framework-imposed patterns (React hooks must follow rules of hooks)

YOUR TASK: Read the code for this batch's dimension. Judge how well the codebase serves a developer from that perspective. The dimension rubric above defines what good looks like. Cite specific observations that explain your judgment.

Mechanical scan evidence — navigation aid, not scoring evidence:
The blind packet contains `holistic_context.scan_evidence` with aggregated signals from all mechanical detectors — including complexity hotspots, error hotspots, signal density index, boundary violations, and systemic patterns. Use these as starting points for where to look beyond the seed files.

Seed files (start here):
- internal/database/postgres.go
- internal/database/redis.go
- internal/middleware/middleware.go
- internal/middleware/ratelimit.go
- internal/models/chat_status.go
- pkg/whatsmeow/instance_settings.go
- internal/handlers/accounts.go
- internal/handlers/activity_logs.go
- internal/handlers/activity_middleware.go
- pkg/whatsmeow/adapter.go
- pkg/whatsmeow/adapter_actions.go
- pkg/whatsmeow/adapter_client.go
- internal/models/activity_log.go
- internal/models/bulk.go
- internal/models/canned_responses.go
- internal/handlers/chat_close_ratings.go
- internal/handlers/message_template_placeholders.go
- internal/handlers/statuses.go
- internal/handlers/agent_analytics.go
- internal/handlers/analytics_instance_filter.go
- internal/handlers/canned_responses.go
- internal/handlers/instances.go
- internal/handlers/contacts_management.go
- internal/handlers/campaign_policy.go
- internal/handlers/helpers.go
- internal/handlers/campaigns.go
- internal/handlers/webhook.go
- internal/handlers/auth_handlers.go
- internal/handlers/activity_service.go
- internal/handlers/media.go
- internal/handlers/app.go
- internal/handlers/import_export.go
- internal/handlers/cookies.go
- internal/handlers/websocket.go
- internal/middleware/csrf.go
- internal/handlers/canned_response_media.go
- internal/handlers/chatbot_processor.go
- internal/handlers/agent_transfers.go
- internal/handlers/sla_processor.go
- internal/handlers/messages.go
- internal/handlers/chatbot.go
- internal/handlers/cache.go
- pkg/whatsapp/client.go
- pkg/whatsmeow/manager.go
- internal/handlers/custom_actions.go
- internal/queue/redis.go
- internal/worker/worker.go
- pkg/whatsmeow/incoming_media.go
- pkg/whatsmeow/events_call.go
- pkg/whatsapp/adapter.go
- pkg/whatsmeow/message_persist.go
- pkg/provider/interface.go
- pkg/migration/migrate.go
- pkg/whatsmeow/events_message.go
- pkg/whatsmeow/adapter_send.go
- pkg/whatsmeow/events.go
- pkg/whatsapp/flow.go
- pkg/whatsmeow/events_identity.go
- internal/handlers/contacts.go
- internal/handlers/widgets.go
- internal/handlers/flows.go
- internal/handlers/send_restriction_policy.go
- internal/handlers/contacts_messaging.go
- internal/handlers/organization.go
- internal/handlers/users.go
- pkg/whatsapp/profile_extras.go
- internal/models/models.go
- internal/handlers/instance_auto_campaign_worker.go
- internal/handlers/stubs.go
- internal/handlers/teams.go
- internal/handlers/template_engine.go
- internal/handlers/templates.go
- pkg/whatsmeow/chat_close_ratings.go
- internal/handlers/canned_response_send.go
- internal/handlers/catalog.go
- internal/handlers/chat_assignment_reset_worker.go
- internal/handlers/group_message_helpers.go
- internal/handlers/instance_pairing.go
- internal/handlers/roles.go
- internal/worker/campaign_delay.go

Task requirements:
1. Read the blind packet's `system_prompt` — it contains scoring rules and calibration.
2. Start from the seed files, then freely explore the repository to build your understanding.
3. Keep issues and scoring scoped to this batch's dimension.
4. Respect scope controls: do not include files/directories marked by `exclude`, `suppress`, or non-production zone overrides.
5. Return 0-10 issues for this batch (empty array allowed).
6. Complete `dimension_judgment` for your dimension — all three fields (strengths, issue_character, score_rationale) are required. Write the judgment BEFORE setting the score.
7. Do not edit repository files.
8. Return ONLY valid JSON, no markdown fences.

Scope enums:
- impact_scope: "local" | "module" | "subsystem" | "codebase"
- fix_scope: "single_edit" | "multi_file_refactor" | "architectural_change"

Output schema:
{
  "batch": "api_surface_coherence",
  "batch_index": 19,
  "assessments": {"<dimension>": <0-100 with one decimal place>},
  "dimension_notes": {
    "<dimension>": {
      "evidence": ["specific code observations"],
      "impact_scope": "local|module|subsystem|codebase",
      "fix_scope": "single_edit|multi_file_refactor|architectural_change",
      "confidence": "high|medium|low",
      "issues_preventing_higher_score": "required when score >85.0",
      "sub_axes": {"abstraction_leverage": 0-100, "indirection_cost": 0-100, "interface_honesty": 0-100, "delegation_density": 0-100, "definition_directness": 0-100, "type_discipline": 0-100}  // required for abstraction_fitness when evidence supports it; all one decimal place
    }
  },
  "dimension_judgment": {
    "<dimension>": {
      "strengths": ["0-5 specific things the codebase does well from this dimension's perspective"],
      "issue_character": "one sentence characterizing the nature/pattern of issues from this dimension's perspective",
      "score_rationale": "2-3 sentences explaining the score from this dimension's perspective, referencing global anchors"
    }
  },
  "issues": [{
    "dimension": "<dimension>",
    "identifier": "short_id",
    "summary": "one-line defect summary",
    "related_files": ["relative/path.py"],
    "evidence": ["specific code observation"],
    "suggestion": "concrete fix recommendation",
    "confidence": "high|medium|low",
    "impact_scope": "local|module|subsystem|codebase",
    "fix_scope": "single_edit|multi_file_refactor|architectural_change",
    "root_cause_cluster": "optional_cluster_name_when_supported_by_history"
  }],
  "retrospective": {
    "root_causes": ["optional: concise root-cause hypotheses"],
    "likely_symptoms": ["optional: identifiers that look symptom-level"],
    "possible_false_positives": ["optional: prior concept keys likely mis-scoped"]
  }
}
