#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ "$(git rev-parse --show-toplevel)" != "$(pwd)" ]; then
    printf >&2 "Working directory %s is not a git repository root directory. Check where you run this script." "$(pwd)"
    exit 1
fi

if [ -z ${UPSTREAM_URL+x} ]; then
    printf >&2 "Set UPSTREAM_URL to the upstream repo URL."
    exit 1
fi

if [ -z ${UPSTREAM_BRANCH+x} ]; then
    printf >&2 "Set UPSTREAM_BRANCH to the upstream branch to mirror."
    exit 1
fi

# __ACTIONS_BRANCH is the branch where the downstream repo stores GitHub Actions workflows.
__ACTIONS_BRANCH="actions"

# __DOWNSTREAM_BRANCH is the downstream branch, derived from the upstream branch.
__DOWNSTREAM_BRANCH="downstream-$UPSTREAM_BRANCH"

# __WORKFLOWS_DIR is the directory that contains GitHub Actions workflows.
__WORKFLOWS_DIR=".github/workflows"

# __RELEASE_BRANCH_WORKFLOWS_DIR is the directory that contains GitHub Actions workflows that are committed to
# downstream release branches.
__RELEASE_BRANCH_WORKFLOWS_DIR=".github/release-branch-workflows"

# Check if the downstream branch already exists. If it does, then we are done.
if ! git ls-remote --heads --quiet --exit-code origin "refs/heads/$__DOWNSTREAM_BRANCH"; then
    printf >&2 "Downstream branch %s already exists in the origin remote." "$__DOWNSTREAM_BRANCH"
    exit 0
fi

# Add upstream remote, and fetch branches and tags.
git remote add upstream "$UPSTREAM_URL" --fetch --tags

# Create downstream branch from HEAD of upstream branch
git branch "$__DOWNSTREAM_BRANCH" --track "upstream/$UPSTREAM_BRANCH"

# Remove upstream GitHub Actions workflows
git checkout "$__DOWNSTREAM_BRANCH"
rm --recursive "$__WORKFLOWS_DIR"
git commit --all --message "Remove upstream GitHub Actions workflows"

# Add downstream GitHub Actions workflows
git checkout "$__ACTIONS_BRANCH" -- "$__RELEASE_BRANCH_WORKFLOWS_DIR"
mv "$__RELEASE_BRANCH_WORKFLOWS_DIR" "$__WORKFLOWS_DIR"
git add "$__WORKFLOWS_DIR"
git commit --all --message "Add downstream GitHub Actions workflows"

# Push downstream branch
git push origin "$__DOWNSTREAM_BRANCH"
