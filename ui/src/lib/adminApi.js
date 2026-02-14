const ADMIN_KEY_STORAGE = 'locky_admin_key'

export function getStoredAdminKey() {
  return typeof localStorage !== 'undefined' ? localStorage.getItem(ADMIN_KEY_STORAGE) || '' : ''
}

export function setStoredAdminKey(key) {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(ADMIN_KEY_STORAGE, key)
  }
}

export function clearStoredAdminKey() {
  if (typeof localStorage !== 'undefined') {
    localStorage.removeItem(ADMIN_KEY_STORAGE)
  }
}

export async function adminApi(path, method = 'GET', body = null) {
  const adminKey = getStoredAdminKey()
  const headers = { 'Content-Type': 'application/json' }
  if (adminKey) headers['X-Admin-Key'] = adminKey

  const res = await fetch(path, {
    method,
    credentials: 'same-origin',
    headers,
    body: body ? JSON.stringify(body) : undefined
  })
  const text = await res.text()
  let data = {}
  try {
    if (text) data = JSON.parse(text)
  } catch (_) {}
  if (!res.ok) throw new Error(data.message || `HTTP ${res.status}`)
  return data
}

export async function listTenants() {
  const data = await adminApi('/admin/tenants')
  return data.tenants || []
}

export async function createTenant(slug, name) {
  return adminApi('/admin/tenants', 'POST', { slug, name })
}

export async function listUsers(tenantId) {
  const data = await adminApi(`/admin/tenants/${tenantId}/users`)
  return data.users || []
}

export async function createUser(tenantId, email, displayName) {
  return adminApi(`/admin/tenants/${tenantId}/users`, 'POST', { email, display_name: displayName })
}

export async function setUserPassword(tenantId, userId, password) {
  return adminApi(`/admin/tenants/${tenantId}/users/${userId}/password`, 'PUT', { password })
}
