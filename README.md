# go-tui-template

Personal base template for Go terminal UIs built on the [Charm](https://charm.sh) stack.

## Stack

- [Bubble Tea v2](https://charm.land/bubbletea/v2) — TUI framework
- [Lip Gloss v2](https://charm.land/lipgloss/v2) — styling and layout
- [Bubbles](https://github.com/charmbracelet/bubbles) — components (table, spinner, list, etc.)
- [GoReleaser v2](https://goreleaser.com) — release pipeline

## What's included

- Header / sidebar / main / footer layout
- Screen router with sidebar navigation
- Command palette (`ctrl+k`)
- Global keybindings and help overlay (`?`)
- Theme system with 15+ built-in themes (`ctrl+t` to cycle)
- Debug log screen
- GoReleaser config + GitHub Actions release workflow (push tag or manual bump)

## Starting a new project

```bash
gh repo create my-new-app --template dev/go-tui-template --clone
cd my-new-app

# Rename the module and binary
OLD=github.com/dev/go-tui-template
NEW=github.com/yourname/my-new-app
find . -type f -name "*.go" -exec sed -i "s|$OLD|$NEW|g" {} +
sed -i "s|$OLD|$NEW|g" go.mod
sed -i "s|go-tui-template|my-new-app|g" .goreleaser.yaml Dockerfile
mv cmd/go-tui-template cmd/my-new-app

# Update AppName in internal/app/view.go
# Update README and AGENTS.md

go mod tidy
go run ./cmd/my-new-app
```

## Development

```bash
go run ./cmd/go-tui-template   # run
go test ./...                   # test
go build ./cmd/go-tui-template  # build check
```

## Release

Releases are handled by GoReleaser via GitHub Actions.

```bash
# Auto-trigger on tag push
git tag v0.1.0 && git push origin v0.1.0

# Or use the manual publish workflow (patch/minor/major bump)
gh workflow run publish.yml -f bump=patch
```

## Notes

- `Width(n)` / `Height(n)` in Lip Gloss v2 set the **total outer size** including borders. Do not pre-subtract `GetFrameSize()` before passing to these methods.
- Some Bubbles components still return Bubble Tea v1 types — check for type mismatches when wiring new ones into the model.
