# HtmlGraph Scripts

Development and deployment scripts for the HtmlGraph project.

## Deployment

```bash
# Full release (non-interactive)
./scripts/deploy-all.sh 0.41.0 --no-confirm

# Preview (dry-run)
./scripts/deploy-all.sh 0.41.0 --dry-run

# Build-only (quality gates)
./scripts/deploy-all.sh --build-only

# Docs-only (commit + push)
./scripts/deploy-all.sh --docs-only
```

See `deploy-all.sh --help` for all options.

## Worktree Helpers

```bash
scripts/worktree-setup.sh <branch>     # Create worktree
scripts/worktree-merge.sh <branch>     # Merge worktree to main
scripts/worktree-cleanup.sh <branch>   # Remove worktree
scripts/worktree-status.sh             # Show all worktrees
```

## Other

```bash
scripts/generate-docs.sh               # Generate documentation
scripts/git-commit-push.sh             # Stage, commit, push
```
