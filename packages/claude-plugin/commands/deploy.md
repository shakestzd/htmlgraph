# /htmlgraph:deploy

Deploy new version to PyPI using deploy-all.sh script

## Usage

```
/htmlgraph:deploy <version>
```

## Parameters

- `version` (required): Version number (e.g., 0.12.1, 0.13.0, 1.0.0)
- `skip_confirm` (optional) (default: True): Skip confirmation prompts (recommended for AI agents)


## Examples

```bash
/htmlgraph:deploy 0.12.1
```
Deploy patch release (bug fixes)

```bash
/htmlgraph:deploy 0.13.0
```
Deploy minor release (new features)

```bash
/htmlgraph:deploy 1.0.0
```
Deploy major release (breaking changes)



## Instructions for Claude

**CRITICAL: This command deploys to production PyPI.**

**PRE-DEPLOYMENT CHECKLIST:**

1. ✅ **Verify version consistency:**
   ```bash
   ./scripts/verify-versions.sh
   ```
   If versions are inconsistent, fix them before deploying.

2. ✅ **Verify tests pass:**
   ```bash
   uv run pytest
   ```
   ALL tests must pass before deployment.

3. ✅ **Verify all work is committed:**
   Check `git status` - working directory should be clean

4. ✅ **Choose correct version number:**
   - Patch (X.Y.Z+1): Bug fixes only
   - Minor (X.Y+1.0): New features (backward compatible)
   - Major (X+1.0.0): Breaking changes

**DEPLOYMENT STEPS:**

1. **Execute the deployment script:**
   ```bash
   ./scripts/deploy-all.sh {version} --no-confirm
   ```

2. **The script will:**
   - Update version numbers in all files (Step 0)
   - Commit version changes automatically
   - Push to git (Step 1)
   - Build package (Step 2)
   - Publish to PyPI (Step 3)
   - Install locally (Step 4)
   - Update Claude plugin (Step 5)
   - Update Gemini extension (Step 6)
   - Update Codex skill if present (Step 7)

3. **Monitor output for errors:**
   - Watch for "❌" or "⚠️" symbols
   - Check PyPI publish succeeded
   - Verify local install completed

4. **Verify deployment:**
   ```bash
   # Check PyPI (may take 5-10 seconds for CDN)
   curl -s https://pypi.org/pypi/htmlgraph/json | python -c "import sys, json; print(json.load(sys.stdin)['info']['version'])"

   # Check local install
   htmlgraph version
   ```

5. **Report completion:**
   Use the output template to summarize the deployment

**IF DEPLOYMENT FAILS:**
- Check error messages in script output
- Verify PyPI credentials in .env file
- Check network connectivity
- Contact user if manual intervention needed

**NEVER:**
- ❌ Deploy without running tests
- ❌ Deploy with uncommitted changes
- ❌ Deploy without understanding the changes
- ❌ Skip the --no-confirm flag (causes interactive prompts)

**ALWAYS:**
- ✅ Run full test suite first
- ✅ Verify version number is correct
- ✅ Check deployment succeeded before reporting
- ✅ Include PyPI link in success message
```

### Output Format:

🚀 **Deployment Complete: v{version}**

✅ Git: Pushed to origin/main
✅ PyPI: https://pypi.org/project/htmlgraph/{version}/
✅ Local: Installed version {version}
✅ Plugins: Updated

**Verify:**
- PyPI: https://pypi.org/project/htmlgraph/{version}/
- Local: `htmlgraph version`
