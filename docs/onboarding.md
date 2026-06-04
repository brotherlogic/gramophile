# Onboarding Guide

To get started with Gramophile, you need to connect your Discogs account. This guide details the OAuth login process and first-time setup.

## OAuth Login Process

Gramophile uses a secure, three-legged OAuth 1.0a flow to integrate with Discogs:

1. **Get URL**:
   The client makes a request to the `GetURL` endpoint of the Gramophile API. The server requests a temporary request token, secret, and verifier URL from Discogs using the application's credentials.
2. **Authorize Application**:
   The server persists this attempt in the database and returns the authorization URL and temporary token. The client redirects you to the Discogs login page, where you authorize Gramophile to read and write to your collection and wantlist.
3. **Handle Callback**:
   After you approve access, Discogs redirects your browser back to the Gramophile callback endpoint (e.g., `ServeHTTP` in `callback.go`). The server retrieves the corresponding request token secret from the database, exchanges the temporary token & verifier for a permanent **Discogs User Token** and **User Secret**, and saves them.
4. **Retrieve Auth Token**:
   The client polls the `GetLogin` endpoint with the request token. Once the server confirms the token has been authorized, it generates a unique session **Auth Token** for Gramophile and returns it to the client.

## Authenticating Requests

Subsequent gRPC or HTTP requests must include the resolved session Auth Token in the metadata headers.
- **Header Key**: `auth-token`
- **Verification**: The API server extracts the token using the context metadata, looks up the user record, and grants standard access.

## First-Time User Setup

Upon successful login:
1. The user state is initially set to `USER_STATE_REFRESHING`.
2. The server enqueues a low-priority queue task: **New User Refresh**.
3. This task fetches your Discogs collection page-by-page.
4. Once the sync finishes successfully, your state changes to `USER_STATE_LIVE`, allowing you full access to features like wants management and physical organization.
