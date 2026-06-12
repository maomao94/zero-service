# uix production framework foundation Design

## Boundary

`cli/uix/` provides a generic Bubble Tea shell and reusable components. It may depend on Bubble Tea, Bubbles, Lip Gloss, and existing chart dependencies. It must not import `cli/dtui` or Docker packages.

## Key Design Points

- Keep global prompt prefixes shell-owned: `/` command/module palette, `@` reference placeholders, `#` file picker, `!` disabled warning.
- Make help/status a shell-level concern with safe truncation/wrapping for long module binding text.
- Let modules drive dynamic help through existing `StatusMsg` or a minimal shell refresh path rather than adding a large framework abstraction.
- Keep modal as the only full-screen overlay; keep dropdown/file picker inline above status/prompt.
- Keep native terminal text selection by avoiding mouse capture.

## Contracts To Preserve

- `NewShell`, `NewApp`, `RegisterModule`, `RegisterCommand`, `SetRunner`, `AppendMessage`, `EnterModule`, `Run`.
- `Module` interface and module-root ESC semantics.
- `Runner` interface and default `MockRunner` behavior.
- Shell messages: `ShowModalMsg`, `ConfirmMsg`, `FileSelectedMsg`, `AppendMessageMsg`, `StatusMsg`.

## Testing Strategy

Use unit tests around shell methods and component views instead of snapshotting full terminal screens. Focus on behavior: routing, visible content, safe width, and commands returned.
