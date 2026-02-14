def test_healthz(require_service_up):
    assert require_service_up is None


def test_admin_endpoint_requires_key(session, config):
    r = session.get(f"{config.base_url}/admin/tenants", timeout=10)
    assert r.status_code == 401
    payload = r.json()
    assert payload["error"] == "unauthorized"


def test_admin_tenant_user_password_flow(
    require_service_up,
    session,
    config,
    admin_headers,
    unique_slug,
    unique_email,
):
    create_tenant = session.post(
        f"{config.base_url}/admin/tenants",
        headers=admin_headers,
        json={"slug": unique_slug, "name": "Python Integration Tenant"},
        timeout=10,
    )
    assert create_tenant.status_code == 201, create_tenant.text
    tenant = create_tenant.json()
    assert tenant["slug"] == unique_slug
    assert tenant["status"] == "active"
    tenant_id = tenant["id"]

    list_tenants = session.get(
        f"{config.base_url}/admin/tenants",
        headers={"X-Admin-Key": admin_headers["X-Admin-Key"]},
        timeout=10,
    )
    assert list_tenants.status_code == 200, list_tenants.text
    listed_tenants = list_tenants.json()["tenants"]
    assert any(t["id"] == tenant_id for t in listed_tenants)

    create_user = session.post(
        f"{config.base_url}/admin/tenants/{tenant_id}/users",
        headers=admin_headers,
        json={"email": unique_email, "display_name": "Py Integration User"},
        timeout=10,
    )
    assert create_user.status_code == 201, create_user.text
    user = create_user.json()
    assert user["email"] == unique_email
    user_id = user["id"]

    list_users = session.get(
        f"{config.base_url}/admin/tenants/{tenant_id}/users",
        headers={"X-Admin-Key": admin_headers["X-Admin-Key"]},
        timeout=10,
    )
    assert list_users.status_code == 200, list_users.text
    listed_users = list_users.json()["users"]
    assert any(u["id"] == user_id for u in listed_users)

    set_password = session.put(
        f"{config.base_url}/admin/tenants/{tenant_id}/users/{user_id}/password",
        headers=admin_headers,
        json={"password": "Password123!"},
        timeout=10,
    )
    assert set_password.status_code == 204, set_password.text
