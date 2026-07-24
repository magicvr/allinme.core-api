#!/usr/bin/env bash
# Goal Governance Skills installer (Claude Code + Grok Build + GitHub Copilot)
# Run from the target project root. No network access required.
#
# Typical flow:
#   1. Copy this whole skills package into the project root
#      (may rename, e.g. my-governance-skills)
#   2. cd to project root
#   3. bash ./skills/install.sh --claude --skills-dir ./skills
#      or: bash ./my-governance-skills/install.sh --all --skills-dir ./my-governance-skills

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_ROOT="$SCRIPT_DIR"

TARGET_DIR="${PWD}"
SKILLS_DIR_ARG="./skills"
INSTALL_CLAUDE=0
INSTALL_GROK=0
INSTALL_COPILOT=0
INSTALL_EXTRAS=0
INSTALL_PRIMITIVE_WRAPPERS=0
INIT_WORKSPACE=0
WORKSPACE_SLUG=""
ROOT_SLUG=""
ROOT_TITLE=""
WORKSPACE_NNN="001"
INIT_WORKSPACE_DONE=0

usage() {
  cat <<'EOF'
Goal Governance Skills installer

Prerequisites:
  Copy the entire skills package into the target project root first
  (you may rename it, e.g. my-governance-skills). Then run this script
  from the project root.

Usage (run from target project root):
  ./install.sh --claude [--skills-dir DIR]
  ./install.sh --grok [--skills-dir DIR]
  ./install.sh --copilot [--skills-dir DIR] [--with-primitives]
  ./install.sh --all [--skills-dir DIR] [--with-primitives]
  ./install.sh --init-workspace --workspace-slug SLUG --root-slug SLUG [host flags…]
  ./install.sh --help

Options:
  --claude              Install Claude Code: ./AGENTS.md + project skills
                        ./.claude/skills/govern/  →  /govern
                        ./.claude/skills/audit/   →  /audit  (cross-audit)
  --grok                Install Grok Build project skills
                        ./.grok/skills/govern/  →  /govern
                        ./.grok/skills/audit/   →  /audit
  --copilot             Install GitHub Copilot rules → ./.github/copilot-instructions.md
                        and default slashes → govern.prompt.md + audit.prompt.md
  --with-primitives     Also install advanced Copilot slash wrappers (new-goal, …).
                        Opt-in only — avoids form-menu UX.
  --all                 Install Claude + Grok + Copilot + ensure prompts/, templates/ and
                        contracts/ under --skills-dir; primary entry remains /govern
  --init-workspace      GOAL-019: create docs/workspace-NNN-SLUG/ with workspace.md +
                        goal-tree.md (does NOT create GOAL-* five-pack; use /govern for Root)
  --workspace-slug S    Required with --init-workspace (lowercase hyphen slug)
  --root-slug S         Required with --init-workspace → GOAL-001-<S>
  --root-title T        Optional display title for planned Root (default: pending)
  --workspace-nnn NNN   Optional three-digit workspace number (default: 001)
  --skills-dir DIR      Skills package / destination directory (default: ./skills)
                        Relative paths are resolved from the current working directory.
  --help, -h            Show this help

Behavior:
  - Rule files always install into the current working directory (project root)
  - Claude skills → ./.claude/skills/govern/ + audit/
  - Grok skills → ./.grok/skills/govern/ + audit/
  - Default Copilot slash surface: /govern (primary) + /audit (cross-audit)
  - Advanced form-filling slashes are NOT installed unless --with-primitives
  - Core methodology (GOAL-019 D-004): ALWAYS installs package core/docs → ./docs/
    (architecture + templates + slim docs/README). Missing core = incomplete install.
  - --init-workspace alone is allowed (still installs core); slugs must be explicit (D-005)
  - Core orchestrator: prompts/00-govern-orchestrator.md
  - Cross-audit core: prompts/05-independent-audit.md
  - prompts/, templates/ and contracts/ are placed under --skills-dir (with --all)
  - Source files are read from the package next to this script
  - Prompts before overwriting existing files
  - Offline only; no network calls

Examples:
  cd /path/to/your-project
  bash ./skills/install.sh --claude --skills-dir ./skills
  bash ./skills/install.sh --all --skills-dir ./skills
  bash ./skills/install.sh --all --init-workspace \
    --workspace-slug my-product --root-slug product-vision \
    --root-title "My product vision" --skills-dir ./skills
EOF
}

die() {
  echo "Error: $*" >&2
  exit 1
}

same_path() {
  local a b
  a="$(cd "$(dirname "$1")" 2>/dev/null && pwd)/$(basename "$1")"
  b="$(cd "$(dirname "$2")" 2>/dev/null && pwd)/$(basename "$2")"
  [[ "$a" == "$b" ]]
}

confirm_overwrite() {
  local path="$1"
  if [[ ! -e "$path" ]]; then
    return 0
  fi
  printf "File already exists: %s\nOverwrite? [y/N] " "$path"
  read -r answer
  case "$answer" in
    y|Y|yes|YES) return 0 ;;
    *) echo "Skipped: $path"; return 1 ;;
  esac
}

copy_file() {
  local src="$1"
  local dest="$2"
  [[ -f "$src" ]] || die "Source file not found: $src"
  mkdir -p "$(dirname "$dest")"
  if confirm_overwrite "$dest"; then
    cp "$src" "$dest"
    echo "Installed: $dest"
  fi
}

copy_dir_merge() {
  local src="$1"
  local dest="$2"
  local label="$3"
  [[ -d "$src" ]] || die "Source directory not found: $src"

  if [[ -d "$dest" ]] && same_path "$src" "$dest"; then
    echo "Already present: $dest/  (from $label)"
    return 0
  fi

  if [[ -e "$dest" ]]; then
    printf "Directory already exists: %s\nOverwrite contents from %s? [y/N] " "$dest" "$label"
    read -r answer
    case "$answer" in
      y|Y|yes|YES) ;;
      *) echo "Skipped: $dest"; return 0 ;;
    esac
  fi

  mkdir -p "$dest"
  cp -R "$src"/. "$dest"/
  echo "Installed: $dest/  (from $label)"
}

print_next_steps() {
  local step2
  if [[ "$INIT_WORKSPACE_DONE" -eq 1 ]]; then
    step2="2. Workspace skeleton ready: docs/workspace-${WORKSPACE_NNN}-${WORKSPACE_SLUG}/
     Run /govern to create Root GOAL-001-${ROOT_SLUG} (five-pack)."
  else
    step2="2. Create workspace skeleton (pick one):
     - /govern S0 (AI asks for slugs), or
     - re-run install with --init-workspace --workspace-slug S --root-slug S"
  fi
  cat <<EOF

Done.

Next steps:
  1. Review installed rule file(s) and docs/architecture (core methodology; required).
  $step2
  3. DEFAULT user path: /govern (orchestrator) + /audit (cross-audit)
     - Methodology: ./docs/architecture/principles.md
     - Orchestrator: $SKILLS_DIR/prompts/00-govern-orchestrator.md
     - Cross-audit: $SKILLS_DIR/prompts/05-independent-audit.md
     - Contract: $SKILLS_DIR/contracts/skills-consumer-contract.json
     - Claude: /govern + /audit under ./.claude/skills/
     - Grok:   /govern + /audit under ./.grok/skills/
     - Copilot: govern.prompt.md + audit.prompt.md
  4. Advanced Copilot form-filling slashes only if you used --with-primitives.

Project root:  $TARGET_DIR
Skills dir:    $SKILLS_DIR
Package root:  $PACKAGE_ROOT
EOF
}

validate_slug() {
  local label="$1"
  local value="$2"
  [[ -n "$value" ]] || die "$label is required"
  [[ "$value" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || die "$label must be lowercase hyphen slug (got: $value)"
}

validate_nnn() {
  local value="$1"
  [[ "$value" =~ ^[0-9]{3}$ ]] || die "--workspace-nnn must be three digits (got: $value)"
}

init_workspace_skeleton() {
  validate_slug "--workspace-slug" "$WORKSPACE_SLUG"
  validate_slug "--root-slug" "$ROOT_SLUG"
  validate_nnn "$WORKSPACE_NNN"
  local title="${ROOT_TITLE:-Root Goal (pending definition)}"
  local today
  today="$(date +%Y-%m-%d 2>/dev/null || echo '2026-07-24')"
  local ws_id="workspace-${WORKSPACE_NNN}-${WORKSPACE_SLUG}"
  local root_id="GOAL-001-${ROOT_SLUG}"
  local scope="docs/${ws_id}/"
  local ws_dir="$TARGET_DIR/$scope"
  local ws_file="${ws_dir}workspace.md"
  local tree_file="${ws_dir}goal-tree.md"

  if [[ -e "$ws_dir" ]]; then
    die "Workspace path already exists (refuse overwrite): $ws_dir"
  fi
  # Prefer installed templates; fall back to package core
  local tmpl="$TARGET_DIR/docs/templates/workspace-context.md"
  if [[ ! -f "$tmpl" ]]; then
    tmpl="$PACKAGE_ROOT/core/docs/templates/workspace-context.md"
  fi
  [[ -f "$tmpl" ]] || die "Missing workspace template: $tmpl (install core first)"

  mkdir -p "$ws_dir"
  # Write concrete workspace.md (template is example-filled; generate authoritative frontmatter + body)
  cat >"$ws_file" <<EOF
---
id: ${ws_id}
title: ${ws_id}
status: active
root_goal: ${root_id}
canonical_scope: ${scope}
shared_materials_catalog: none
created: ${today}
updated: ${today}
version: 0.1.0
---

# 工作区上下文 · ${ws_id}

本工作区由 \`install --init-workspace\` 脚手架创建（GOAL-019）。目标状态只存在于本目录的 \`goal-tree.md\` 与 \`GOAL-*\` 五件套。

## 绑定

| 字段 | 当前值 | 说明 |
|------|--------|------|
| 工作区 ID | \`${ws_id}\` | 与共享资料引用的 \`workspace_id\` 一致。 |
| Root Goal | \`${root_id}\` | **尚未创建五件套**；用 \`/govern\` 创建后 \`parent: null\`。 |
| canonical 范围 | \`${scope}\` | 本工作区唯一目标状态范围。 |
| 共享资料目录 | \`none\` | 需要资料时再改为固定路径并补引用表。 |

## 固定共享资料引用

| reference_id | workspace_id | material_id | source | version | sha256 | purpose | local_record | status |
|--------------|--------------|-------------|--------|---------|--------|---------|--------------|--------|

## 备注

- Root 计划标题（可改）：${title}
- 脚手架**不**创建 \`GOAL-*\` 文件夹；下一步：\`/govern\` 设立 Root。
EOF

  cat >"$tree_file" <<EOF
---
title: Goal Tree · 目标树与进展总览
status: active
created: ${today}
updated: ${today}
parent: null
version: 0.1.0
---

# Goal Tree

> 工作区 \`${ws_id}\` 已 scaffold。Root \`${root_id}\` 尚未创建五件套 — 运行 \`/govern\` 完成设立。

## 树状结构

\`\`\`text
(empty — pending Root ${root_id})
\`\`\`

## 状态总览

| ID | 标题 | Parent | Status | Progress | 路径 |
|----|------|--------|--------|----------|------|
| ${root_id} | ${title} | — | draft | 0% | _(not created yet)_ |
EOF

  echo "Scaffolded workspace: ${scope}"
  echo "  workspace.md + goal-tree.md (Root ${root_id} pending /govern)"
  INIT_WORKSPACE_DONE=1
}

install_core_docs() {
  local core_docs="$PACKAGE_ROOT/core/docs"
  [[ -d "$core_docs/architecture" ]] || die "Missing package core mirror: $core_docs/architecture (GOAL-019 D-004)"
  [[ -d "$core_docs/templates" ]] || die "Missing package core mirror: $core_docs/templates (GOAL-019 D-004)"
  [[ -f "$core_docs/README.md" ]] || die "Missing package core mirror: $core_docs/README.md (GOAL-019 D-004)"
  [[ -f "$core_docs/architecture/principles.md" ]] || die "Missing principles.md in core mirror"
  [[ -f "$core_docs/architecture/workspace-protocol.md" ]] || die "Missing workspace-protocol.md in core mirror"
  if [[ -f "$core_docs/architecture/tech-stack.md" ]]; then
    die "core mirror must not ship tech-stack.md (D-004)"
  fi
  echo "Installing core methodology → ./docs/ (architecture + templates + README)"
  copy_file "$core_docs/README.md" "$TARGET_DIR/docs/README.md"
  copy_dir_merge "$core_docs/architecture" "$TARGET_DIR/docs/architecture" "core architecture"
  copy_dir_merge "$core_docs/templates" "$TARGET_DIR/docs/templates" "core templates"
}

# --- parse args ---
if [[ $# -eq 0 ]]; then
  usage
  exit 1
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    --claude)
      INSTALL_CLAUDE=1
      shift
      ;;
    --grok)
      INSTALL_GROK=1
      shift
      ;;
    --copilot)
      INSTALL_COPILOT=1
      shift
      ;;
    --all)
      INSTALL_CLAUDE=1
      INSTALL_GROK=1
      INSTALL_COPILOT=1
      INSTALL_EXTRAS=1
      shift
      ;;
    --with-primitives)
      INSTALL_PRIMITIVE_WRAPPERS=1
      shift
      ;;
    --init-workspace)
      INIT_WORKSPACE=1
      shift
      ;;
    --workspace-slug)
      [[ $# -ge 2 ]] || die "--workspace-slug requires a value"
      WORKSPACE_SLUG="$2"
      shift 2
      ;;
    --workspace-slug=*)
      WORKSPACE_SLUG="${1#--workspace-slug=}"
      shift
      ;;
    --root-slug)
      [[ $# -ge 2 ]] || die "--root-slug requires a value"
      ROOT_SLUG="$2"
      shift 2
      ;;
    --root-slug=*)
      ROOT_SLUG="${1#--root-slug=}"
      shift
      ;;
    --root-title)
      [[ $# -ge 2 ]] || die "--root-title requires a value"
      ROOT_TITLE="$2"
      shift 2
      ;;
    --root-title=*)
      ROOT_TITLE="${1#--root-title=}"
      shift
      ;;
    --workspace-nnn)
      [[ $# -ge 2 ]] || die "--workspace-nnn requires a value"
      WORKSPACE_NNN="$2"
      shift 2
      ;;
    --workspace-nnn=*)
      WORKSPACE_NNN="${1#--workspace-nnn=}"
      shift
      ;;
    --skills-dir)
      [[ $# -ge 2 ]] || die "--skills-dir requires a path argument"
      SKILLS_DIR_ARG="$2"
      shift 2
      ;;
    --skills-dir=*)
      SKILLS_DIR_ARG="${1#--skills-dir=}"
      [[ -n "$SKILLS_DIR_ARG" ]] || die "--skills-dir requires a path argument"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      die "Unknown option: $1 (use --help)"
      ;;
  esac
done

if [[ "$INSTALL_CLAUDE" -eq 0 && "$INSTALL_GROK" -eq 0 && "$INSTALL_COPILOT" -eq 0 && "$INIT_WORKSPACE" -eq 0 ]]; then
  usage
  exit 1
fi

if [[ "$INIT_WORKSPACE" -eq 1 ]]; then
  [[ -n "$WORKSPACE_SLUG" ]] || die "--init-workspace requires --workspace-slug (D-005: no silent default)"
  [[ -n "$ROOT_SLUG" ]] || die "--init-workspace requires --root-slug (D-005: no silent default)"
fi

# Resolve skills-dir (relative to CWD / project root)
if [[ "$SKILLS_DIR_ARG" = /* ]]; then
  SKILLS_DIR="$SKILLS_DIR_ARG"
else
  SKILLS_DIR="$TARGET_DIR/$SKILLS_DIR_ARG"
fi
if [[ -d "$SKILLS_DIR" ]]; then
  SKILLS_DIR="$(cd "$SKILLS_DIR" && pwd)"
fi

CLAUDE_AGENTS_SRC="$PACKAGE_ROOT/install/claude/AGENTS.md"
CLAUDE_GOVERN_SRC="$PACKAGE_ROOT/install/claude/skills/govern/SKILL.md"
CLAUDE_AUDIT_SRC="$PACKAGE_ROOT/install/claude/skills/audit/SKILL.md"
GROK_GOVERN_SRC="$PACKAGE_ROOT/install/grok/skills/govern/SKILL.md"
GROK_AUDIT_SRC="$PACKAGE_ROOT/install/grok/skills/audit/SKILL.md"
COPILOT_SRC="$PACKAGE_ROOT/install/copilot/copilot-instructions.md"
COPILOT_WRAPPERS_SRC="$PACKAGE_ROOT/install/copilot/prompts"
PROMPTS_SRC="$PACKAGE_ROOT/prompts"
TEMPLATES_SRC="$PACKAGE_ROOT/templates"
CONTRACTS_SRC="$PACKAGE_ROOT/contracts"
CORE_DOCS_SRC="$PACKAGE_ROOT/core/docs"

[[ -d "$PROMPTS_SRC" ]] || die "Missing package directory: $PROMPTS_SRC"
[[ -d "$TEMPLATES_SRC" ]] || die "Missing package directory: $TEMPLATES_SRC"
[[ -d "$CONTRACTS_SRC" ]] || die "Missing package directory: $CONTRACTS_SRC"
[[ -d "$CORE_DOCS_SRC" ]] || die "Missing package directory: $CORE_DOCS_SRC (GOAL-019 core mirror)"
[[ -d "$TARGET_DIR" ]] || die "Current working directory is not a directory: $TARGET_DIR"

if [[ "$INSTALL_CLAUDE" -eq 1 ]]; then
  [[ -f "$CLAUDE_AGENTS_SRC" ]] || die "Missing package file: $CLAUDE_AGENTS_SRC"
  [[ -f "$CLAUDE_GOVERN_SRC" ]] || die "Missing package file: $CLAUDE_GOVERN_SRC"
  [[ -f "$CLAUDE_AUDIT_SRC" ]] || die "Missing package file: $CLAUDE_AUDIT_SRC"
fi
if [[ "$INSTALL_GROK" -eq 1 ]]; then
  [[ -f "$GROK_GOVERN_SRC" ]] || die "Missing package file: $GROK_GOVERN_SRC"
  [[ -f "$GROK_AUDIT_SRC" ]] || die "Missing package file: $GROK_AUDIT_SRC"
fi
if [[ "$INSTALL_COPILOT" -eq 1 ]]; then
  [[ -f "$COPILOT_SRC" ]] || die "Missing package file: $COPILOT_SRC"
  [[ -d "$COPILOT_WRAPPERS_SRC" ]] || die "Missing package directory: $COPILOT_WRAPPERS_SRC"
fi

echo "Project root:  $TARGET_DIR"
echo "Skills dir:    $SKILLS_DIR"
echo "Package root:  $PACKAGE_ROOT"
echo

if [[ "$INSTALL_CLAUDE" -eq 1 ]]; then
  copy_file "$CLAUDE_AGENTS_SRC" "$TARGET_DIR/AGENTS.md"
  copy_file "$CLAUDE_GOVERN_SRC" "$TARGET_DIR/.claude/skills/govern/SKILL.md"
  copy_file "$CLAUDE_AUDIT_SRC" "$TARGET_DIR/.claude/skills/audit/SKILL.md"
  echo "Claude skills: /govern  (./.claude/skills/govern/)  +  /audit  (./.claude/skills/audit/)"
fi

if [[ "$INSTALL_GROK" -eq 1 ]]; then
  copy_file "$GROK_GOVERN_SRC" "$TARGET_DIR/.grok/skills/govern/SKILL.md"
  copy_file "$GROK_AUDIT_SRC" "$TARGET_DIR/.grok/skills/audit/SKILL.md"
  echo "Grok skills: /govern  (./.grok/skills/govern/)  +  /audit  (./.grok/skills/audit/)"
  # Optional: also ensure AGENTS if missing (Grok reads AGENTS.md as project rules)
  if [[ ! -f "$TARGET_DIR/AGENTS.md" && -f "$CLAUDE_AGENTS_SRC" ]]; then
    echo "Note: no AGENTS.md yet; consider --claude or copy install/claude/AGENTS.md for project rules."
  fi
fi

if [[ "$INSTALL_COPILOT" -eq 1 ]]; then
  mkdir -p "$TARGET_DIR/.github"
  copy_file "$COPILOT_SRC" "$TARGET_DIR/.github/copilot-instructions.md"

  mkdir -p "$TARGET_DIR/.github/prompts"
  # Default product surface: primary orchestrator + cross-audit (not form-fill primitives)
  WRAPPER_NAMES=(govern audit)
  if [[ "$INSTALL_PRIMITIVE_WRAPPERS" -eq 1 ]]; then
    WRAPPER_NAMES+=(new-goal log-decision update-execution write-audit)
    echo "Including advanced primitive slash wrappers (--with-primitives)"
  else
    echo "Copilot slash surface: /govern + /audit (pass --with-primitives for form-fill ops)"
  fi
  for name in "${WRAPPER_NAMES[@]}"; do
    copy_file \
      "$COPILOT_WRAPPERS_SRC/${name}.md" \
      "$TARGET_DIR/.github/prompts/${name}.prompt.md"
  done
fi

if [[ "$INSTALL_EXTRAS" -eq 1 ]]; then
  mkdir -p "$SKILLS_DIR"
  copy_dir_merge "$PROMPTS_SRC" "$SKILLS_DIR/prompts" "prompts"
  copy_dir_merge "$TEMPLATES_SRC" "$SKILLS_DIR/templates" "templates"
  copy_dir_merge "$CONTRACTS_SRC" "$SKILLS_DIR/contracts" "contracts"
fi

# GOAL-019 D-003/D-004: core methodology is co-required with any host or workspace init
install_core_docs

if [[ "$INIT_WORKSPACE" -eq 1 ]]; then
  init_workspace_skeleton
fi

print_next_steps
