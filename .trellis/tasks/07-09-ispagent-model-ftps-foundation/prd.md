# ispagent model ftps foundation

## Goal

Provide the ispagent foundation for generating ISP model XML files and uploading them through FTPS. This phase covers the device point model, the patrol device model, and reusable upload primitives only.

## Requirements

- Define field-safe Go models for the ISP device point model XML (`Device_Model`) and patrol device model XML (`PatrolDevice_Model`).
- Cover all required fields from the provided protocol tables and existing sample XML files.
- Generate XML using structured escaping, not manual string concatenation, so JSON-like attribute values remain valid XML attributes.
- Support streaming XML generation to an `io.Writer` so large point model exports do not require building the entire XML in memory.
- Provide FTPS upload primitives with configurable address, credentials, TLS mode, remote directory, timeout, and optional temporary remote filename flow.
- Keep this phase service-internal to `app/ispagent`; do not change gRPC proto contracts or wire the model sync commands yet.
- Do not hard-code production FTPS credentials in source code or default config.

## Acceptance Criteria

- [ ] Device point model generation emits `<Device_Model>` with `<Item .../>` attributes matching protocol field names.
- [ ] Patrol device model generation emits `<PatrolDevice_Model>` with `<Item .../>` attributes matching protocol field names.
- [ ] Generated XML escapes quotes, ampersands, and JSON-like attribute values correctly.
- [ ] FTPS uploader can upload a local file to a configured remote path and optionally use a temporary remote path before rename.
- [ ] Focused unit tests cover XML field names and escaping behavior.
- [ ] Relevant package tests compile and pass.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
