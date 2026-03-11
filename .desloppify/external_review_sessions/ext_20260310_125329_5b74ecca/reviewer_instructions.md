# External Blind Review Session

Session id: ext_20260310_125329_5b74ecca
Session token: 221fd2cd0cd536eadb4f748d7353005b
Blind packet: /Users/noiemany/Downloads/whatomate_GOWA/whatomate/.desloppify/review_packet_blind.json
Template output: /Users/noiemany/Downloads/whatomate_GOWA/whatomate/.desloppify/external_review_sessions/ext_20260310_125329_5b74ecca/review_result.template.json
Claude launch prompt: /Users/noiemany/Downloads/whatomate_GOWA/whatomate/.desloppify/external_review_sessions/ext_20260310_125329_5b74ecca/claude_launch_prompt.md
Expected reviewer output: /Users/noiemany/Downloads/whatomate_GOWA/whatomate/.desloppify/external_review_sessions/ext_20260310_125329_5b74ecca/review_result.json

Happy path:
1. Open the Claude launch prompt file and paste it into a context-isolated subagent task.
2. Reviewer writes JSON output to the expected reviewer output path.
3. Submit with the printed --external-submit command.

Reviewer output requirements:
1. Return JSON with top-level keys: session, assessments, issues.
2. session.id must be `ext_20260310_125329_5b74ecca`.
3. session.token must be `221fd2cd0cd536eadb4f748d7353005b`.
4. Include issues with required schema fields (dimension/identifier/summary/related_files/evidence/suggestion/confidence).
5. Use the blind packet only (no score targets or prior context).
