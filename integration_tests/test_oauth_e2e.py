import base64
import hashlib
import secrets
from urllib.parse import parse_qs, urlparse

import pytest


def _pkce_pair():
    verifier = secrets.token_urlsafe(64)
    digest = hashlib.sha256(verifier.encode("utf-8")).digest()
    challenge = base64.urlsafe_b64encode(digest).rstrip(b"=").decode("ascii")
    return verifier, challenge


def _ensure_seeded_environment(session, config):
    tenants_resp = session.get(
        f"{config.base_url}/admin/tenants",
        headers={"X-Admin-Key": config.admin_api_key},
        timeout=10,
    )
    if tenants_resp.status_code != 200:
        pytest.skip(f"Could not list tenants: {tenants_resp.status_code} {tenants_resp.text}")

    tenants = tenants_resp.json().get("tenants", [])
    if not any(t.get("slug") == "test" for t in tenants):
        pytest.skip(
            "Seeded tenant not found. Run with docker compose seed or set env vars for pre-seeded test tenant/client."
        )


def test_oidc_discovery(require_service_up, session, config):
    _ensure_seeded_environment(session, config)
    r = session.get(f"{config.base_url}/.well-known/openid-configuration", timeout=10)
    assert r.status_code == 200, r.text
    payload = r.json()
    assert payload["authorization_endpoint"].endswith("/oauth2/authorize")
    assert "authorization_code" in payload["grant_types_supported"]
    assert payload["issuer"].startswith("https://")


def test_authorization_code_pkce_flow(require_service_up, session, config, oauth_state, oauth_nonce):
    _ensure_seeded_environment(session, config)

    verifier, challenge = _pkce_pair()
    authorize_params = {
        "response_type": "code",
        "client_id": config.seeded_client_id,
        "redirect_uri": config.seeded_redirect_uri,
        "scope": config.seeded_scope,
        "state": oauth_state,
        "nonce": oauth_nonce,
        "code_challenge": challenge,
        "code_challenge_method": "S256",
    }

    login_page = session.get(
        f"{config.base_url}/oauth2/authorize",
        params=authorize_params,
        timeout=10,
    )
    if login_page.status_code == 500 and "issue access token" in login_page.text.lower():
        pytest.skip("OAuth signing key missing for seeded tenant. Seed signing keys before running OAuth E2E tests.")
    assert login_page.status_code == 200, login_page.text

    login_submit = session.post(
        f"{config.base_url}/oauth2/authorize",
        data={
            **authorize_params,
            "email": config.seeded_email,
            "password": config.seeded_password,
        },
        allow_redirects=False,
        timeout=10,
    )
    if login_submit.status_code == 400 and "client not found" in login_submit.text.lower():
        pytest.skip("Seeded OAuth client not found. Ensure seed data includes a client for LOCKY_SEEDED_CLIENT_ID.")
    if login_submit.status_code >= 500 and "issue access token" in login_submit.text.lower():
        pytest.skip("OAuth signing key missing for seeded tenant. Seed signing keys before running OAuth E2E tests.")

    assert login_submit.status_code == 302, login_submit.text
    redirect_location = login_submit.headers.get("Location", "")
    parsed = urlparse(redirect_location)
    query = parse_qs(parsed.query)

    assert "code" in query, redirect_location
    assert query["state"][0] == oauth_state
    auth_code = query["code"][0]

    token_resp = session.post(
        f"{config.base_url}/oauth2/token",
        data={
            "grant_type": "authorization_code",
            "code": auth_code,
            "redirect_uri": config.seeded_redirect_uri,
            "code_verifier": verifier,
            "client_id": config.seeded_client_id,
        },
        timeout=10,
    )
    if token_resp.status_code >= 500 and "issue access token" in token_resp.text.lower():
        pytest.skip("OAuth signing key missing for seeded tenant. Seed signing keys before running OAuth E2E tests.")
    assert token_resp.status_code == 200, token_resp.text

    tokens = token_resp.json()
    access_token = tokens.get("access_token")
    refresh_token = tokens.get("refresh_token")
    assert access_token
    assert refresh_token
    assert tokens["token_type"] == "Bearer"

    introspect_before = session.post(
        f"{config.base_url}/oauth2/introspect",
        data={"token": refresh_token},
        timeout=10,
    )
    assert introspect_before.status_code == 200, introspect_before.text
    assert introspect_before.json().get("active") is True

    revoke = session.post(
        f"{config.base_url}/oauth2/revoke",
        data={"token": refresh_token, "token_type_hint": "refresh_token"},
        timeout=10,
    )
    assert revoke.status_code == 200, revoke.text

    introspect_after = session.post(
        f"{config.base_url}/oauth2/introspect",
        data={"token": refresh_token},
        timeout=10,
    )
    assert introspect_after.status_code == 200, introspect_after.text
    assert introspect_after.json().get("active") is False
