# dtui Docker resource modules Design

## Boundary

This child owns `plugins/containers`, `plugins/images`, and only the `internal/docker` methods needed to support those modules. It must not change `uix` contracts unless the framework child has already planned the change.

## Container Module

Use root table mode plus subviews:

- Root: container table and action/status surface.
- Detail mode: inspect output from `InspectContainer`.
- Log mode: existing `LogViewer` with follow/scroll.
- Stats mode: stream or sample `StatsEntry` data and display recent history.

`IsRoot()` should return false for subviews so `esc` closes the subview before leaving the module.

## Image Module

Use root table mode plus forms/subviews:

- Root: image table and action/status surface.
- History mode: image layer history.
- Tag form: source image plus target tag input.
- Save/export form: output path input, defaulting from `Image.DefaultSaveFile()`.

## Safety

Container remove, image remove, and image prune require second confirmation. Confirmation text must include the selected resource name/ref/short ID and explain the impact. Read-only list/detail/logs/stats/history views should not require confirmation.

## Testing

Prefer pure module tests using injected fake Docker clients if minimal interfaces are introduced. Avoid requiring a live daemon for state-machine tests.
