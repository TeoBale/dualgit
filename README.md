# dualgit (dgit)

DualGit is a Git orchestrator that keeps two histories for the same project:
- `public` history (`main` -> public remote)
- `private` superset history (`private-main` -> private remote)

## MVP commands

- `dualgit init`
- `dualgit context public|private`
- `dualgit status`
- `dualgit push [--all-contexts] [--dry-run]`
- `dualgit publish`
- `dualgit hooks install|check`
- `dualgit shell-init zsh|bash`

## Setup

1. Build:

```bash
go build -o dualgit ./cmd/dualgit
```

2. Install shell integration:

```bash
eval "$(./dualgit shell-init zsh)"
# or bash
# eval "$(./dualgit shell-init bash)"
```

3. Initialize in a repo:

```bash
./dualgit init
```

## Security model

- `.gitignore`: excluded from both public and private histories.
- `.publicignore`: paths allowed only in private context.
- In `public` context:
  - pre-commit hook blocks commits touching `.publicignore` paths.
  - pre-push hook blocks raw `git push` (must use `dualgit push`).
- `dualgit push` validates outgoing public commits against `.publicignore`.

## Notes

- Public commit replication to private is implemented through shell wrapper flow:
  - `git commit` -> `dualgit _post_commit_sync`.
- If you bypass `shell-init`, commit replication is not automatic; protections still apply at hook/push level.
