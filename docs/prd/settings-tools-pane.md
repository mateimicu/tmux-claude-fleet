# PRD: Settings/Tools Pane with Cache Pre-fill Action

## Goal

Add a "Tools" sub-menu to the existing session list view (`claude-matrix list`) that exposes maintenance actions. The first action is **Pre-fill cache**, which creates git mirror caches for all configured repositories across every source (GitHub API, local file, YAML workspaces) so that future `create` operations clone instantly from local mirrors.

## Background

When a user runs `claude-matrix create`, the tool clones the selected repository. If a mirror cache exists in `~/.tmux-claude-matrix/.cache/mirrors/`, the clone uses `git clone --reference` for near-instant setup. Otherwise, a full `git clone --mirror` is performed first, adding latency to the session creation flow.

Currently, mirrors are only created on-demand (first clone of each repo). Users with many configured repositories must create a session for each repo before the cache is fully warmed. A bulk pre-fill action eliminates this cold-start penalty.

## Requirements

1. **Tools sub-menu via ctrl-s** — In the session list FZF view, pressing `ctrl-s` opens a secondary FZF menu listing available tool actions. The session list header legend must be updated to include the `ctrl-s: tools` hint.

2. **Pre-fill cache action** — The tools menu presents "Pre-fill mirror cache" as a selectable action. When selected, it discovers all repositories from all enabled sources (GitHub API, local repos file, YAML workspaces) and creates/updates mirror caches for each.

3. **Workspace repo inclusion** — Repos listed inside workspace definitions (`workspaces.yaml`) are individually mirrored, not just the workspace entry itself.

4. **Sequential execution with per-repo progress** — Mirrors are created one at a time. Each repo displays its status as it processes:
   - `[1/N] Cloning mirror: org/repo...`
   - `[1/N] Updating mirror: org/repo...` (if mirror already exists)
   - `[1/N] ✓ org/repo` (on success)
   - `[1/N] ✗ org/repo: <error>` (on failure; continue to next repo)

5. **Summary on completion** — After all repos are processed, print a summary: total repos, newly cached, updated, failed. Then return to the session list view.

6. **Return to session list** — After the pre-fill action completes (or is cancelled with ctrl-c), the user is returned to the session list FZF view to continue their workflow.

7. **New SessionAction value** — Add a `SessionActionTools` constant to the `SessionAction` type in `internal/fzf/` to represent the ctrl-s key press, following the existing pattern for `SessionActionDelete` and `SessionActionToggleFilter`.

8. **Reuse existing infrastructure** — Use `repos.Discoverer.ListAll()` for repo discovery and `git.Manager.CloneWithCache()` / `createMirror()` / `updateMirror()` for mirror operations. Use the existing `config.Load()` for cache directory resolution.

9. **Error resilience** — A failure to mirror one repo must not abort the entire pre-fill. Errors are collected and reported in the summary.

## Acceptance Criteria

- [ ] Pressing `ctrl-s` in the session list opens a tools FZF menu
- [ ] Session list header legend includes `ctrl-s: tools` hint
- [ ] "Pre-fill mirror cache" appears as an action in the tools menu
- [ ] Selecting pre-fill discovers repos from all enabled sources (GitHub, local, workspaces)
- [ ] Individual repos inside workspace definitions are each mirrored
- [ ] Each repo shows `[X/N]` progress as mirrors are created/updated
- [ ] Repos with existing mirrors are updated (git fetch --prune), not re-cloned
- [ ] A single repo failure does not abort the batch; error is logged and processing continues
- [ ] Completion summary shows counts: total, new, updated, failed
- [ ] After pre-fill completes, user is returned to the session list
- [ ] Pressing ctrl-c during pre-fill cancels remaining work and returns to the session list
- [ ] New `SessionActionTools` constant added with guard test for duplicate values
- [ ] `ctrl-s` key is included in the `--expect` list for `runFZFWithExpect` in the session list

## Out of Scope

- Parallel/concurrent mirror creation (sequential chosen for simplicity and debuggability)
- Selective pre-fill (e.g., picking specific repos or sources to cache)
- Cache eviction or cleanup of stale mirrors
- Dedicated tmux keybinding for the tools pane (accessed via list view only)
- Additional tool actions beyond pre-fill (future PRDs will add actions to this menu)
- CLI subcommand for pre-fill (e.g., `claude-matrix prefill`); this is UI-only for now
