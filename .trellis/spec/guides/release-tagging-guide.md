# Release Tagging Thinking Guide

Use this guide when a task mentions tagging, releases, publishing a version, or GitHub Release creation.

## Decide Project Identity First

- Main project / owned project / source project / non-fork: use normal semver tags such as `v0.0.17`.
- Fork / derived project: use fork coordinates only when the user or repository context clearly says the project is a fork.
- If project identity is ambiguous, ask before choosing a tag format.

## Approval Gate

Before creating or publishing a tag/release, decide and show the release plan first, then stop.

List these fields:
- Project identity: main project / fork project / uncertain
- Target commit: `<short-sha> <subject>`
- Proposed tag
- Release title
- Release notes summary
- Commands that would run

Do not run `git tag`, `git push <tag>`, or `gh release create` until the user explicitly approves with words such as `可以`, `确认`, `就这样`, or `按这个发`.

## Main Project Tag Flow

1. Check local state and latest tags:
   ```bash
   git status --short --branch
   git tag --sort=-v:refname
   ```
2. Pick the next semver tag from the latest `vMAJOR.MINOR.PATCH` tag.
   - Patch default: `v0.0.16` -> `v0.0.17` for ordinary fixes or small updates.
   - Ask before minor/major bumps unless the user has already specified the version.
3. Check the tag does not already exist locally or remotely:
   ```bash
   git tag --list 'v0.0.17'
   git ls-remote --tags origin 'refs/tags/v0.0.17'
   ```
4. Create and push the tag only after the approval gate has passed and the target commit is confirmed:
   ```bash
   git show --no-patch --format='%h %s' HEAD
   git tag v0.0.17
   git push origin v0.0.17
   ```
5. Create the GitHub Release with real Markdown notes:
   ```bash
   gh release create v0.0.17 --title "v0.0.17" --notes "$RELEASE_NOTES"
   ```

## Do Not

- Do not use `v<upstream>-fork.<timestamp>` for owned main projects.
- Do not infer fork status only from a skill name or old instructions.
- Do not push the branch when the user only asked to push a tag or create a release.
- Do not expose credentials embedded in remote URLs when reporting results.

## Verify After Release

```bash
git ls-remote --tags origin 'refs/tags/<tag>'
gh release view <tag> --json tagName,name,url
git status --short --branch
```

Report whether the branch commit was pushed separately from the tag.
