#!/usr/bin/env bash
# scripts/changelog.sh
# Generate a changelog section from Conventional Commits and prepend to CHANGELOG.md
# Usage:
#   scripts/changelog.sh NEW_TAG [FROM_TAG] [TO_REF]
# Examples:
#   scripts/changelog.sh v0.2.0 v0.1.0
#   scripts/changelog.sh v0.2.0 "" v0.2.0   # when running on the tag itself

set -euo pipefail

NEW_TAG="${1:-}"
FROM_TAG="${2:-}"
TO_REF="${3:-HEAD}"

if [[ -z "${NEW_TAG}" ]]; then
  echo "Usage: $0 NEW_TAG [FROM_TAG] [TO_REF]" >&2
  exit 2
fi

# Resolve FROM_TAG if not provided (fallback: initial commit)
if [[ -z "${FROM_TAG}" ]]; then
  if git describe --tags --abbrev=0 >/dev/null 2>&1; then
    FROM_TAG="$(git describe --tags --abbrev=0)"
  else
    FROM_TAG="$(git rev-list --max-parents=0 HEAD | tail -n1)"
  fi
fi

DATE="$(date +%Y-%m-%d)"
RANGE="${FROM_TAG}..${TO_REF}"

# Collect commits: full sha | short sha | subject | body
# --no-merges keeps the log clean
LOG="$(git log --no-merges --pretty=format:'%H|%h|%s|%b' ${RANGE} || true)"

if [[ -z "${LOG}" ]]; then
  echo "No commits found in range ${RANGE}. Nothing to do."
  exit 0
fi

# Buckets
features=()
fixes=()
perf=()
refactors=()
docs=()
chores=()
tests=()
ci=()
deps=()
breaking=()

strip_type_scope() {
  # Remove "type(scope)!: " or "type: " prefix; keep message
  echo "$1" | sed -E 's/^[a-z]+(\([^)]+\))?!:\s*//; s/^[a-z]+:\s*//'
}

# Iterate commits
while IFS='|' read -r full short subject body; do
  s_lower="$(echo "${subject}" | tr 'A-Z' 'a-z')"
  msg="$(strip_type_scope "${subject}")"

  # Detect breaking change: "type!: ..." OR body contains "BREAKING CHANGE:"
  if [[ "${s_lower}" =~ ^[a-z]+(\([^)]+\))?!: ]] || echo "${body}" | grep -qiE '^breaking change:'; then
    breaking+=("${msg} (${short})")
  fi

  case "${s_lower}" in
    feat:*|feat(\)*:*)      features+=("${msg} (${short})") ;;
    fix:*|fix(\)*:*)        fixes+=("${msg} (${short})") ;;
    perf:*|perf(\)*:*)      perf+=("${msg} (${short})") ;;
    refactor:*|refactor(\)*:*) refactors+=("${msg} (${short})") ;;
    docs:*|docs(\)*:*)      docs+=("${msg} (${short})") ;;
    chore:*|chore(\)*:*)    chores+=("${msg} (${short})") ;;
    test:*|tests:*|test(\)*:*) tests+=("${msg} (${short})") ;;
    ci:*|ci(\)*:*)          ci+=("${msg} (${short})") ;;
    build:*|build(\)*:*)    ci+=("${msg} (${short})") ;;
    deps:*|dependency:*|dependencies:*)
                            deps+=("${msg} (${short})") ;;
    *)
      # Heuristic: untyped but looks like fix/feat
      if [[ "${s_lower}" =~ ^fix ]]; then fixes+=("${msg} (${short})");
      elif [[ "${s_lower}" =~ ^feat ]]; then features+=("${msg} (${short})");
      else
        # default bucket: chores
        chores+=("${msg} (${short})")
      fi
      ;;
  esac
done <<< "${LOG}"

# Build section
BODY_TMP="$(mktemp)"
{
  echo "## ${NEW_TAG} — ${DATE}"
  echo

  if (( ${#breaking[@]} )); then
    echo "### ⚠️ Breaking Changes"
    for i in "${breaking[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#features[@]} )); then
    echo "### Features"
    for i in "${features[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#fixes[@]} )); then
    echo "### Bug Fixes"
    for i in "${fixes[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#perf[@]} )); then
    echo "### Performance"
    for i in "${perf[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#refactors[@]} )); then
    echo "### Refactors"
    for i in "${refactors[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#docs[@]} )); then
    echo "### Documentation"
    for i in "${docs[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#ci[@]} )); then
    echo "### CI / Build"
    for i in "${ci[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#deps[@]} )); then
    echo "### Dependencies"
    for i in "${deps[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#tests[@]} )); then
    echo "### Tests"
    for i in "${tests[@]}"; do echo "- ${i}"; done
    echo
  fi

  if (( ${#chores[@]} )); then
    echo "### Chores"
    for i in "${chores[@]}"; do echo "- ${i}"; done
    echo
  fi
} > "${BODY_TMP}"

# If all sections are empty (unlikely), bail out
if [[ ! -s "${BODY_TMP}" ]]; then
  echo "No categorized commits; not updating CHANGELOG.md"
  rm -f "${BODY_TMP}"
  exit 0
fi

# Prepend to CHANGELOG.md
if [[ -f CHANGELOG.md ]]; then
  TMP_ALL="$(mktemp)"
  cat "${BODY_TMP}" > "${TMP_ALL}"
  echo >> "${TMP_ALL}"
  cat CHANGELOG.md >> "${TMP_ALL}"
  mv "${TMP_ALL}" CHANGELOG.md
else
  echo -e "# Changelog\n\n$(cat "${BODY_TMP}")" > CHANGELOG.md
fi

echo "CHANGELOG.md updated for range ${RANGE}"
