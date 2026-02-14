import os
import secrets
import uuid
from dataclasses import dataclass
from typing import Dict

import pytest
import requests


@dataclass(frozen=True)
class IntegrationConfig:
    base_url: str
    admin_api_key: str
    tenant_host: str
    seeded_email: str
    seeded_password: str
    seeded_client_id: str
    seeded_redirect_uri: str
    seeded_scope: str


def _trim_url(url: str) -> str:
    return url.rstrip("/")


@pytest.fixture(scope="session")
def config() -> IntegrationConfig:
    base_url = _trim_url(os.getenv("LOCKY_BASE_URL", "http://localhost:8080"))
    admin_api_key = os.getenv("LOCKY_ADMIN_API_KEY", "test-admin-key-123")
    tenant_host = os.getenv("LOCKY_TENANT_HOST", "localhost")

    return IntegrationConfig(
        base_url=base_url,
        admin_api_key=admin_api_key,
        tenant_host=tenant_host,
        seeded_email=os.getenv("LOCKY_SEEDED_EMAIL", "test@example.com"),
        seeded_password=os.getenv("LOCKY_SEEDED_PASSWORD", "password123"),
        seeded_client_id=os.getenv("LOCKY_SEEDED_CLIENT_ID", "test-client-id"),
        seeded_redirect_uri=os.getenv("LOCKY_SEEDED_REDIRECT_URI", "http://localhost:3000/callback"),
        seeded_scope=os.getenv("LOCKY_SEEDED_SCOPE", "openid profile email"),
    )


@pytest.fixture(scope="session")
def session(config: IntegrationConfig) -> requests.Session:
    s = requests.Session()
    s.headers.update({"Accept": "application/json", "Host": config.tenant_host})
    yield s
    s.close()


@pytest.fixture(scope="session")
def admin_headers(config: IntegrationConfig) -> Dict[str, str]:
    return {"X-Admin-Key": config.admin_api_key, "Content-Type": "application/json"}


@pytest.fixture(scope="session")
def require_service_up(session: requests.Session, config: IntegrationConfig) -> None:
    r = session.get(f"{config.base_url}/healthz", timeout=5)
    assert r.status_code == 200, f"Service not healthy: {r.status_code} {r.text}"


@pytest.fixture
def unique_slug() -> str:
    return f"py-it-{uuid.uuid4().hex[:8]}"


@pytest.fixture
def unique_email() -> str:
    return f"py-it-{uuid.uuid4().hex[:8]}@example.com"


@pytest.fixture
def oauth_state() -> str:
    return secrets.token_urlsafe(16)


@pytest.fixture
def oauth_nonce() -> str:
    return secrets.token_urlsafe(16)
