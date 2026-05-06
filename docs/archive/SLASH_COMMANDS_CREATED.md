# Slash Commands Created

**Date**: 2025-12-25
**Commands**: `/wipnote:git-commit` and `/wipnote:deploy`

---

## What Was Created

Two new slash commands for the deployment scripts:

### 1. `/wipnote:git-commit`

**Script**: `./scripts/git-commit-push.sh`

**Usage**:
```
/wipnote:git-commit "feat: add new feature"
/wipnote:git-commit "fix(parser): handle edge case"
```

**Features**:
- Commits and pushes in one command
- Conventional commit format
- Auto-includes Wipnote footer
- Supports multi-line messages

### 2. `/wipnote:deploy`

**Script**: `./scripts/deploy-all.sh`

**Usage**:
```
/wipnote:deploy 0.12.1    # Patch release
/wipnote:deploy 0.13.0    # Minor release
/wipnote:deploy 1.0.0     # Major release
```

**Features**:
- Full deployment pipeline
- Version bump automation
- PyPI publishing
- Plugin updates

---

## Files Generated

### YAML Definitions (Source of Truth)
- `packages/common/command_definitions/git-commit.yaml`
- `packages/common/command_definitions/deploy.yaml`

### Claude Code Commands
- `packages/claude-plugin/commands/git-commit.md`
- `packages/claude-plugin/commands/deploy.md`

### Codex Skill Sections
- `packages/codex-skill/command_git-commit.md`
- `packages/codex-skill/command_deploy.md`

### Gemini Extension Sections
- `packages/gemini-extension/command_git-commit.md`
- `packages/gemini-extension/command_deploy.md`

**Total**: 8 files (2 YAML + 6 platform-specific)

---

## How It Works

1. **Single Source of Truth**: YAML files in `packages/common/command_definitions/`
2. **Generator**: `packages/common/generators/generate_commands.py`
3. **Multi-Platform**: Generates commands for Claude Code, Codex, and Gemini
4. **Regenerate**: Run `python generate_commands.py --platform all` to regenerate

---

## Usage Examples

### For Users (Claude Code)

```bash
# Commit and push changes
/wipnote:git-commit "feat: add OAuth support"

# Deploy new version
/wipnote:deploy 0.12.2
```

### For Developers (Updating Commands)

```bash
# Edit YAML definition
vim packages/common/command_definitions/git-commit.yaml

# Regenerate all commands
cd packages/common/generators
python generate_commands.py --platform all

# Commit changes
git add packages/common/command_definitions/*.yaml
git add packages/{claude-plugin,codex-skill,gemini-extension}/command*.md
git commit -m "feat: add /git-commit and /deploy slash commands"
```

---

## Benefits

✅ **DRY Principle**: Single YAML file → Multiple platform outputs
✅ **Consistency**: Same behavior across all platforms
✅ **Easy Updates**: Edit YAML once, regenerate all
✅ **Version Control**: YAML files are source of truth
✅ **Documentation**: Auto-generated from definitions

---

## Next Steps

1. **Test the commands**:
   ```bash
   # In Claude Code, try:
   /wipnote:git-commit "test: verify slash command"
   ```

2. **Deploy updated plugin**:
   ```bash
   ./scripts/deploy-all.sh 0.12.2 --no-confirm
   ```

3. **Document in README**:
   - Add `/git-commit` and `/deploy` to command list
   - Add examples to deployment section

---

## Command Generator Details

**Script**: `packages/common/generators/generate_commands.py`

**Platforms Supported**:
- Claude Code (`.md` files in `claude-plugin/commands/`)
- Codex (`.md` sections in `codex-skill/`)
- Gemini (`.md` sections in `gemini-extension/`)

**Generation Command**:
```bash
cd packages/common/generators
python generate_commands.py --platform all
```

**Options**:
- `--platform claude` - Generate only Claude Code commands
- `--platform codex` - Generate only Codex sections
- `--platform gemini` - Generate only Gemini sections
- `--platform all` - Generate for all platforms (default)
