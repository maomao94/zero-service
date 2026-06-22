# Optimize GIS Docker image build

## Goal

Optimize the GIS service Docker image build so it follows Go + CGO + GEOS container best practices, reduces repeat build time, and remains compatible with future multi-architecture builds.

## Requirements

- Keep GIS compilation inside the Docker builder stage because the service requires C toolchain and GEOS headers.
- Keep runtime image minimal: include runtime GEOS dependency, timezone data, and certificates, but not compiler or GEOS development packages.
- Avoid hard-coding `GOARCH=amd64` so future ARM builds can be driven by Docker build platform selection.
- Enable Go module and compile cache reuse during Docker builds with BuildKit cache mounts.
- Ensure configured `GOPROXY` is actually used by Go commands.
- Reduce Docker build context noise with a root `.dockerignore`.

## Acceptance Criteria

- [x] `app/gis/Dockerfile` uses a multi-stage builder/runtime layout for CGO + GEOS.
- [x] Builder installs `pkgconf` and `geos-dev`; runtime installs `geos` only for GEOS.
- [x] Dockerfile no longer sets `GOARCH=amd64` directly.
- [x] `go mod download` and `go build` use BuildKit cache mounts for Go module/build caches.
- [x] `GOPROXY` build arg is exported as an environment variable for Go commands.
- [x] Root `.dockerignore` excludes common local/generated artifacts from the Docker context.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
