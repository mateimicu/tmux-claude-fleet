# Organization Filtering

## Overview

If you have access to many GitHub repositories (hundreds or thousands), you can filter to show only repositories from specific organizations.

## Configuration

### Method 1: Config File (Recommended)

Create or edit `~/.tmux-claude-fleet/config`:

```bash
# Filter by organizations (comma-separated)
GITHUB_ORGS=myorg,anotherorg,personal-account
```

Example:
```bash
# Only show repos from these organizations
GITHUB_ORGS=AdventOfVim,mateimicu
```

### Method 2: Environment Variable

```bash
export TMUX_CLAUDE_FLEET_GITHUB_ORGS="org1,org2,org3"
```

Add to your shell profile for persistence:

```bash
# In ~/.zshrc or ~/.bashrc
export TMUX_CLAUDE_FLEET_GITHUB_ORGS="myorg,anotherorg"
```

## Examples

### Example 1: Filter by Work Organization

```bash
# ~/.tmux-claude-fleet/config
GITHUB_ORGS=my-company
```

Result: Only shows repos from `my-company` organization

### Example 2: Multiple Organizations

```bash
# ~/.tmux-claude-fleet/config
GITHUB_ORGS=company1,company2,personal-username
```

Result: Shows repos from all three organizations

### Example 3: Personal Repos Only

```bash
# Use your GitHub username
GITHUB_ORGS=your-github-username
```

Result: Only shows your personal repositories

## How It Works

### Without Filter (Default)

```bash
# No GITHUB_ORGS set
# Result: Shows ALL 1845 repositories
```

When you run `claude-fleet create`:
```
✓ GitHub integration enabled (using gh CLI)
🔍 Discovering repositories...
✓ Found 1845 repositories
```

### With Filter

```bash
# GITHUB_ORGS=AdventOfVim,mateimicu
# Result: Shows only repos from these 2 orgs
```

When you run `claude-fleet create`:
```
✓ GitHub integration enabled (using gh CLI)
  Filtering by organizations: AdventOfVim, mateimicu
🔍 Discovering repositories...
✓ Found 42 repositories
```

## Verifying Configuration

Use the diagnostic command:

```bash
claude-fleet diagnose
```

**Without filter:**
```
🐙 GitHub Repository Source:
  Enabled: true
  Authentication: ✓ Using gh CLI
  Testing GitHub API...
  Status: ✓ API working
  Repositories found: 1845
```

**With filter:**
```
🐙 GitHub Repository Source:
  Enabled: true
  Authentication: ✓ Using gh CLI
  Organization filter: AdventOfVim, mateimicu
  Testing GitHub API...
  Status: ✓ API working
  Repositories found: 42
```

## Finding Your Organizations

### List Your Organizations

```bash
gh api user/orgs --jq '.[].login'
```

This shows all organizations you're a member of.

### List Your Repos by Organization

```bash
# See which orgs have the most repos
gh repo list --limit 100 --json nameWithOwner --jq '.[].nameWithOwner' | cut -d'/' -f1 | sort | uniq -c | sort -rn
```

Example output:
```
    523 my-company
    142 side-project-org
     89 personal-username
     12 another-org
```

## Use Cases

### 1. Filter Out Work Repos at Home

```bash
# Only show personal repos
GITHUB_ORGS=your-username
```

### 2. Filter Out Personal Repos at Work

```bash
# Only show work repos
GITHUB_ORGS=work-org1,work-org2
```

### 3. Focus on Specific Projects

```bash
# Only show repos from active projects
GITHUB_ORGS=current-client,internal-tools
```

### 4. Exclude Archived Organizations

```bash
# Only show repos from current organizations
# (manually exclude old/archived org names)
GITHUB_ORGS=current-org
```

## Combining with Local Repos

You can use organization filtering for GitHub repos while also having local repos:

```bash
# ~/.tmux-claude-fleet/config
GITHUB_ORGS=myorg
LOCAL_CONFIG_ENABLED=1

# ~/.tmux-claude-fleet/repos.txt
https://github.com/other-org/special-repo:Special project
https://gitlab.com/company/private-repo:GitLab repo
```

Result:
- GitHub repos: Only from `myorg`
- Local repos: All repos from `repos.txt`

## Disabling Organization Filter

To show all repos again:

**Method 1:** Remove from config file
```bash
# Edit ~/.tmux-claude-fleet/config
# Delete or comment out the GITHUB_ORGS line
# GITHUB_ORGS=...
```

**Method 2:** Set to empty
```bash
GITHUB_ORGS=
```

**Method 3:** Unset environment variable
```bash
unset TMUX_CLAUDE_FLEET_GITHUB_ORGS
```

## Performance

Organization filtering is done **client-side** after fetching repos from GitHub.

**Impact:**
- ✅ **API calls**: Same (fetches all repos once)
- ✅ **Caching**: Works normally (5min cache)
- ✅ **FZF performance**: Much faster with fewer repos!

Example:
- Without filter: FZF searches through 1845 items
- With filter (2 orgs): FZF searches through 42 items

## Troubleshooting

### "No repositories found" after setting filter

**Problem:** The organizations don't match exactly.

**Solution:**
```bash
# Check exact organization names
gh api user/repos --jq '.[].owner.login' | sort -u

# Use exact names (case-sensitive!)
GITHUB_ORGS=AdventOfVim  # Correct
# Not: adventofvim, advent-of-vim, etc.
```

### Filter not working

**Solution:**
```bash
# Clear cache to force refresh
rm -rf ~/.tmux-claude-fleet/.cache/

# Run diagnose to verify config
claude-fleet diagnose

# Check for typos in organization names
```

### Want to see all orgs in repos

```bash
# List all unique organizations in your repos
gh api user/repos --paginate --jq '.[].full_name' | cut -d'/' -f1 | sort -u
```

## Configuration Reference

```bash
# ~/.tmux-claude-fleet/config

# Organization filter (comma-separated, no spaces around commas)
GITHUB_ORGS=org1,org2,org3

# Enable/disable GitHub integration
GITHUB_ENABLED=1

# Cache settings
CACHE_TTL=5m

# Other settings
LOCAL_CONFIG_ENABLED=1
LOCAL_REPOS_FILE=~/.tmux-claude-fleet/repos.txt
```

## Example Configurations

### Work Setup
```bash
# ~/.tmux-claude-fleet/config
GITHUB_ORGS=my-company,client-org
CACHE_TTL=10m
```

### Personal Setup
```bash
# ~/.tmux-claude-fleet/config
GITHUB_ORGS=my-username,side-project-org
LOCAL_CONFIG_ENABLED=1
```

### Minimal Setup
```bash
# ~/.tmux-claude-fleet/config
GITHUB_ORGS=my-main-org
LOCAL_CONFIG_ENABLED=0  # Disable local repos
```

## Quick Reference

```bash
# Set organizations
echo 'GITHUB_ORGS=org1,org2' >> ~/.tmux-claude-fleet/config

# Verify
claude-fleet diagnose

# Test
claude-fleet create
```
