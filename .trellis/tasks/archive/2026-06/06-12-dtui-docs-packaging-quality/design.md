# dtui docs packaging quality Design

## Boundary

This child owns documentation, packaging/build script polish, validation command documentation, and final quality review. It should not introduce new runtime features except small build/doc-support changes required for accuracy.

## Documentation Shape

README should be organized around user tasks:

- What `dtui` is.
- Quick start.
- Startup without Docker vs Docker-required operations.
- Command palette and global prompt modes.
- Module references and keys.
- Config file and deploy safety.
- Build/package outputs.
- Validation commands.

## Packaging Shape

Keep `build.sh` simple and local. It should build deterministic binary names under `cli/dtui/bin/` and avoid hidden dependencies beyond Go toolchain.

## Quality Shape

Docs and packaging are verified after all feature children finish. The final parent quality gate should execute all required commands or document any environment-specific reason they cannot run.
