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
  --skills-dir DIR      Skills package / destination directory (default: ./skills)
                        Relative paths are resolved from the current working directory.
  --help, -h            Show this help

Behavior:
  - Rule files always install into the current working directory (project root)
  - Claude skills → ./.claude/skills/govern/ + audit/
  - Grok skills → ./.grok/skills/govern/ + audit/
  - Default Copilot slash surface: /govern (primary) + /audit (cross-audit)
  - Advanced form-filling slashes are NOT installed unless --with-primitives
  - Core orchestrator: prompts/00-govern-orchestrator.md
  - Cross-audit core: prompts/05-independent-audit.md
  - prompts/, templates/ and contracts/ are placed under --skills-dir (with --all)
  - Source files are read from the package next to this script
  - Prompts before overwriting existing files
  - Offline only; no network calls

Examples:
  cd /path/to/your-project
  bash ./skills/install.sh --claude --skills-dir ./skills
  bash ./skills/install.sh --grok --skills-dir ./skills
  bash ./skills/install.sh --copilot --skills-dir ./skills
  bash ./skills/install.sh --all --skills-dir ./skills
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
  cat <<EOF

Done.

Next steps:
  1. Review installed rule file(s) and adjust paths for your project.
  2. Ensure a docs/workspace-<NNN>-<slug>/ workspace root and its goal-tree.md exist.
  3. DEFAULT user path: /govern (orchestrator) + /audit (cross-audit)
     - Core: $SKILLS_DIR/prompts/00-govern-orchestrator.md
     - Cross: $SKILLS_DIR/prompts/05-independent-audit.md
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

if [[ "$INSTALL_CLAUDE" -eq 0 && "$INSTALL_GROK" -eq 0 && "$INSTALL_COPILOT" -eq 0 ]]; then
  usage
  exit 1
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

[[ -d "$PROMPTS_SRC" ]] || die "Missing package directory: $PROMPTS_SRC"
[[ -d "$TEMPLATES_SRC" ]] || die "Missing package directory: $TEMPLATES_SRC"
[[ -d "$CONTRACTS_SRC" ]] || die "Missing package directory: $CONTRACTS_SRC"
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

print_next_steps
