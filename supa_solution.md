## Revised Bounty Submission:

### Reward: $500 USD

### Type: Frontend QA / bug-fix bounty

### Target:
MergeOS login and logout modal/session flow across the public frontend.
- Production: https://mergeos.shop/
- Local: http://127.0.0.1:5173/

### Goal:
Test the login modal, social login entry points, logged-in account state, and logout flow across common platforms and responsive viewport sizes. Fix any UI, navigation, or session-state bugs found during testing.

### Scope:

*   Open and close the login modal from all visible entry points using Cypress test suite.
    *   \# Test Login Modal:
        *   `cypress/command/loginModal()` (open)
        *   `cypress/command/loginModalClose()` (close)
    *   `cypress/command/loginButton` (Google login button UI, GitHub login button UI)
*   Test Google login button UI and loading states.
    *   `cypress/command/loginButton` with different states (success, error, loading).
*   Verify the logout flow from the signed-in/account state.
    *   `cypress/command/logout()` followed by redirect to homepage and clearing session data (`localStorage.clear()`)
*   Test logout behavior on different viewport sizes using Cypress- responsive-states plugin
*   Test Cancel/Back behavior for social login entries.

### Approach:

1.  **Cypress Test Suite Setup**:
    *   Create a new test suite in `cypress/integration/login-modal.test.js`.
    *   Set up environment variables (`cypress-env.json`) with production URL.
2.  **Login Modal Tests**: 
    *   Open login modal using `loginModal()` command.
    *   Verify modal content and layout across different viewport sizes.
3.  **Social Login Entry Point Tests**:
    *   Test Google login button UI, GitHub login button UI, loading states, error states, and cancel/back behavior.
4.  **Logout Flow Test**: 
    *   Open logout modal.
    *   Verify logout content and layout.
    *   Perform successful logout by clicking logout button.

### Error Handling:

*   Implement error handling for invalid username/password combinations or failed login attempts using try-catch blocks in test suite.

### Additional Tests:

*   Test responsive viewport sizes (e.g., 320px, 768px, 1024px) with Cypress- responsive-states plugin.
*   Verify correct session state after successful logout (`localStorage.clear()`).

### Deliverables:

1.  **Rapport de bugs**: Detailed bug report including test results and solutions implemented.
2.  **Corrections UI/session validation**:
    *   Updated login modal CSS to handle responsive viewport sizes.
    *   Added responsive states for social login button UI.

3.  `cypress/command/loginModal()` (open) in test suite.

4.  `cypress/command/logout()` in test suite followed by clearing session data (`localStorage.clear()`)