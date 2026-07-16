# Bug Tracker

## Bug 1: Organisation context not switching on login with different organisation

**Description:**
When a new organisation is created and then a login attempt is made with a different organisation, the system still loads the previously created organisation instead of the one specified during login.

**Steps to Reproduce:**
1. Create a new organisation.
2. Attempt to log in using credentials belonging to a different organisation.
3. Observe that the system loads the first (created) organisation instead of the correct one.

**Expected Behaviour:**
The system should resolve and load the organisation that matches the credentials used during login.

**Actual Behaviour:**
The system incorrectly returns/uses the organisation that was most recently created, ignoring the organisation tied to the login credentials.

---

## Bug 2: Refresh token error redirects user out of patient section after fresh org creation

**Description:**
After a fresh organisation is created and the user logs in successfully, navigating to the patient section triggers a logout or redirect due to a `refresh_token` issue.

**Steps to Reproduce:**
1. Create a fresh organisation.
2. Log in with the newly created organisation's credentials.
3. Navigate to the patient section.
4. Observe that the user is redirected out (logged out or sent to login page).

**Expected Behaviour:**
The user should remain authenticated and be able to access the patient section without being redirected.

**Actual Behaviour:**
The user is kicked out of the patient section, likely because the `refresh_token` is missing, invalid, or not correctly associated with the new organisation session.
