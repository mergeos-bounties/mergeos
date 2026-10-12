#!/bin/bash

# Configuration
REPO_OWNER="mergeos-bounties"
REPO_NAME="mergeos"
REPO_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}.git"
LOCAL_REPO_DIR="${REPO_NAME}"

# --- Functions ---

log_info() {
    echo -e "\n\033[0;34m[INFO]\033[0m $1"
}

log_success() {
    echo -e "\n\033[0;32m[SUCCESS]\033[0m $1"
}

log_error() {
    echo -e "\n\033[0;31m[ERROR]\033[0m $1"
    exit 1
}

# --- Main Script ---

if [ -z "$1" ]; then
    log_error "Usage: $0 <PR_NUMBER>"
fi

PR_NUMBER=$1
PR_BRANCH="pull/${PR_NUMBER}/head" # Standard GitHub PR head ref

log_info "Starting verification for PR #${PR_NUMBER}"

# 1. Clone or update repository
if [ ! -d "${LOCAL_REPO_DIR}" ]; then
    log_info "Cloning ${REPO_URL} into ${LOCAL_REPO_DIR}..."
    git clone "${REPO_URL}" "${LOCAL_REPO_DIR}" || log_error "Failed to clone repository."
else
    log_info "Repository already exists. Fetching latest changes..."
    (cd "${LOCAL_REPO_DIR}" && git fetch origin) || log_error "Failed to fetch origin."
fi

cd "${LOCAL_REPO_DIR}" || log_error "Failed to change directory to ${LOCAL_REPO_DIR}."

# 2. Fetch and checkout PR branch
log_info "Fetching PR #${PR_NUMBER} and checking out its head commit..."
git fetch origin "${PR_BRANCH}:${PR_BRANCH}" || log_error "Failed to fetch PR branch ${PR_BRANCH}."
git checkout "${PR_BRANCH}" || log_error "Failed to checkout PR branch ${PR_BRANCH}."

HEAD_COMMIT_SHA=$(git rev-parse HEAD)
log_info "Checked out PR #${PR_NUMBER} (Head Commit: ${HEAD_COMMIT_SHA})"

# 3. Install dependencies
log_info "Installing Node.js dependencies..."
npm install || log_error "Failed to install Node.js dependencies."

# 4. Run tests
log_info "Running 'npm test'..."
NPM_TEST_OUTPUT=$(npm test 2>&1)
NPM_TEST_STATUS=$?

if [ ${NPM_TEST_STATUS} -eq 0 ]; then
    log_success "'npm test' passed."
    NPM_TEST_RESULT="PASS"
else
    log_error "'npm test' failed. Output:\n${NPM_TEST_OUTPUT}"
    NPM_TEST_RESULT="FAIL"
fi

# 5. Run build
log_info "Running 'npm run build:local'..."
NPM_BUILD_OUTPUT=$(npm run build:local 2>&1)
NPM_BUILD_STATUS=$?

if [ ${NPM_BUILD_STATUS} -eq 0 ]; then
    log_success "'npm run build:local' passed."
    NPM_BUILD_RESULT="PASS"
else
    log_error "'npm run build:local' failed. Output:\n${NPM_BUILD_OUTPUT}"
    NPM_BUILD_RESULT="FAIL"
fi

log_info "Automated checks complete. Please proceed with manual testing and evidence review."

echo "--- Automated Check Results ---"
echo "PR Number: ${PR_NUMBER}"
echo "Head Commit SHA: ${HEAD_COMMIT_SHA}"
echo "npm test result: ${NPM_TEST_RESULT}"
echo "npm run build:local result: ${NPM_BUILD_RESULT}"
echo "-------------------------------"

# Optional: Keep the branch checked out for manual testing, or revert to main
# To revert: git checkout main
# To clean up PR branch: git branch -D "${PR_BRANCH}"

