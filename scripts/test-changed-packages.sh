#!/usr/bin/env bash
# Runs go test for each package touched by added or modified .go files in a diff range.

set -euo pipefail

usage() {
  echo "usage: $0 <base-ref> <head-ref>" >&2
  exit 1
}

if [[ $# -ne 2 ]]; then
  usage
fi

base_ref="$1"
head_ref="$2"

changed_go_files=$(
  git diff --name-only --diff-filter=AM "${base_ref}" "${head_ref}" | grep '\.go$' || true
)

if [[ -z "${changed_go_files}" ]]; then
  echo "No added or modified Go files; skipping tests."
  exit 0
fi

package_dirs=$(
  echo "${changed_go_files}" | while read -r file; do
    dir=$(dirname "${file}")
    if [[ "${dir}" == "." ]]; then
      echo "."
    else
      echo "./${dir}"
    fi
  done | sort -u
)

failed=0
while read -r pkg_dir; do
  [[ -z "${pkg_dir}" ]] && continue
  echo "Testing ${pkg_dir}..."
  if ! go test -count=1 -race "${pkg_dir}"; then
    failed=1
  fi
done <<< "${package_dirs}"

exit "${failed}"
