<script>
  import { onMount } from 'svelte'
  import {
    getStoredAdminKey,
    setStoredAdminKey,
    listTenants,
    createTenant,
    listUsers,
    createUser,
    setUserPassword
  } from '../lib/adminApi.js'

  let page = 'overview'
  let showAdvanced = false
  let tenants = []
  let users = []
  let selectedTenantId = ''
  let sessionStatus = { msg: '', ok: true }
  let tenantStatus = { msg: '', ok: true }
  let userStatus = { msg: '', ok: true }
  let adminKey = getStoredAdminKey()
  let tenantSlug = ''
  let tenantName = ''
  let tenantSearch = ''
  let userTenantId = ''
  let userEmail = ''
  let userName = ''
  let userPassword = 'password123'
  let userSearch = ''
  let showKey = false

  const pageMeta = {
    overview: { title: 'Overview', subtitle: 'Track platform health and jump into key actions.' },
    session: { title: 'Session', subtitle: 'Store an optional admin key for API requests.' },
    tenants: { title: 'Tenants', subtitle: 'Create workspaces and set your active tenant context.' },
    users: { title: 'Users', subtitle: 'Create users and manage accounts in the selected tenant.' }
  }

  function setSessionStatus(msg, ok = true) {
    sessionStatus = { msg, ok }
  }
  function setTenantStatus(msg, ok = true) {
    tenantStatus = { msg, ok }
  }
  function setUserStatus(msg, ok = true) {
    userStatus = { msg, ok }
  }

  function updateAdminKey(key) {
    adminKey = key
    setStoredAdminKey(key)
  }

  async function loadTenants() {
    try {
      tenants = await listTenants()
      setTenantStatus(`Loaded ${tenants.length} tenants.`, true)
    } catch (e) {
      setTenantStatus(e.message || 'Failed to load tenants.', false)
    }
  }

  async function doCreateTenant() {
    const s = tenantSlug.trim()
    const n = tenantName.trim()
    if (!s || !n) {
      setTenantStatus('Slug and name are required.', false)
      return
    }

    try {
      const tenant = await createTenant(s, n)
      tenantSlug = ''
      tenantName = ''
      selectedTenantId = tenant.id || ''
      userTenantId = selectedTenantId
      setTenantStatus(`Created tenant ${s}.`, true)
      await loadTenants()
    } catch (e) {
      setTenantStatus(e.message || 'Failed to create tenant.', false)
    }
  }

  async function loadUsers() {
    const tid = userTenantId.trim() || selectedTenantId
    if (!tid) {
      setUserStatus('Tenant ID is required.', false)
      return
    }

    try {
      users = await listUsers(tid)
      selectedTenantId = tid
      userTenantId = tid
      setUserStatus(`Loaded ${users.length} users.`, true)
    } catch (e) {
      setUserStatus(e.message || 'Failed to load users.', false)
    }
  }

  async function doCreateUser() {
    const tid = (userTenantId.trim() || selectedTenantId).trim()
    const email = userEmail.trim()
    const displayName = userName.trim()
    const password = userPassword
    if (!tid || !email || !password) {
      setUserStatus('Tenant ID, email, and password are required.', false)
      return
    }

    try {
      const user = await createUser(tid, email, displayName)
      await setUserPassword(tid, user.id, password)
      userEmail = ''
      userName = ''
      setUserStatus(`Created user ${email}.`, true)
      await loadUsers()
    } catch (e) {
      setUserStatus(e.message || 'Failed to create user.', false)
    }
  }

  function saveKey() {
    updateAdminKey(adminKey)
    setSessionStatus(adminKey ? 'Admin key saved locally.' : 'Local key cleared. UI session stays active.', true)
  }

  function clearKey() {
    adminKey = ''
    updateAdminKey('')
    saveKey()
  }

  async function refreshAll() {
    await loadTenants()
    if (selectedTenantId) {
      userTenantId = selectedTenantId
      await loadUsers()
    }
  }

  function selectTenant(tenantId) {
    selectedTenantId = tenantId
    userTenantId = tenantId
    setTenantStatus('Tenant context updated.', true)
  }

  $: filteredTenants = tenantSearch.trim()
    ? tenants.filter(
        (t) =>
          String(t.slug || '').toLowerCase().includes(tenantSearch.trim().toLowerCase()) ||
          String(t.name || '').toLowerCase().includes(tenantSearch.trim().toLowerCase())
      )
    : tenants

  $: filteredUsers = userSearch.trim()
    ? users.filter(
        (u) =>
          String(u.email || '').toLowerCase().includes(userSearch.trim().toLowerCase()) ||
          String(u.display_name || '').toLowerCase().includes(userSearch.trim().toLowerCase())
      )
    : users

  $: selectedTenant = tenants.find((t) => t.id === selectedTenantId)
  $: hasTenantContext = Boolean(selectedTenantId)

  onMount(loadTenants)
</script>

<a href="#admin-main" class="skip-link">Skip To Main Content</a>

<div class="admin-app">
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-mark" aria-hidden="true"></div>
      <div class="brand-copy">
        <p class="eyebrow">Locky Console</p>
        <h1>Admin Workspace</h1>
      </div>
    </div>

    <div class="system-chip" class:connected={!!adminKey}>
      <span class="chip-dot" aria-hidden="true"></span>
      <span>{adminKey ? 'Session + Key' : 'Session Only'}</span>
    </div>

    <nav class="nav" aria-label="Admin pages">
      <button class="nav-item" class:active={page === 'overview'} on:click={() => (page = 'overview')} type="button">
        Overview
      </button>
      {#if showAdvanced}
        <button class="nav-item" class:active={page === 'session'} on:click={() => (page = 'session')} type="button">
          Session
        </button>
      {/if}
      <button class="nav-item" class:active={page === 'tenants'} on:click={() => (page = 'tenants')} type="button">
        Tenants
      </button>
      <button class="nav-item" class:active={page === 'users'} on:click={() => (page = 'users')} type="button">
        Users
      </button>
    </nav>

    <div class="sidebar-actions">
      <button type="button" on:click={refreshAll}>Refresh Data</button>
      <button class="btn-secondary" type="button" on:click={() => (showAdvanced = !showAdvanced)}>
        {showAdvanced ? 'Hide Advanced Tools' : 'Show Advanced Tools'}
      </button>
      <form method="POST" action="/admin/ui/logout">
        <button class="btn-danger" type="submit">Sign Out</button>
      </form>
    </div>
  </aside>

  <main id="admin-main" class="content">
    <header class="topbar">
      <div class="topbar-title">
        <p class="eyebrow">Control Plane</p>
        <h2>{pageMeta[page]?.title ?? 'Overview'}</h2>
        <p class="subtitle">{pageMeta[page]?.subtitle ?? ''}</p>
      </div>
      <div class="quick-metrics">
        <article>
          <span>Tenants</span>
          <strong>{tenants.length}</strong>
        </article>
        <article>
          <span>Users</span>
          <strong>{users.length}</strong>
        </article>
        <article>
          <span>Context</span>
          <strong>{hasTenantContext ? 'Ready' : 'Unset'}</strong>
        </article>
      </div>
    </header>

    <section class="workspace">
      {#if page === 'overview'}
        <div class="grid">
          <section class="panel span-8">
            <h3>Workspace Snapshot</h3>
            <p>Review live counts, then move directly into tenant or user flows.</p>
            <div class="stats">
              <article class="stat-card">
                <span>Total Tenants</span>
                <strong>{tenants.length}</strong>
              </article>
              <article class="stat-card">
                <span>Loaded Users</span>
                <strong>{users.length}</strong>
              </article>
              <article class="stat-card">
                <span>Tenant Context</span>
                <strong>{selectedTenant ? selectedTenant.slug : 'None'}</strong>
              </article>
            </div>
            <div class="row">
              <button type="button" on:click={loadTenants}>Load Tenants</button>
              <button class="btn-secondary" type="button" on:click={loadUsers}>Load Users</button>
              <button class="btn-ghost" type="button" on:click={() => (page = 'session')}>Edit Session</button>
            </div>
          </section>

          <section class="panel span-4">
            <h3>Context Status</h3>
            <p>Current selection used for user queries and creation.</p>
            {#if selectedTenant}
              <div class="context-pill">
                <span class="label">Tenant</span>
                <strong>{selectedTenant.slug}</strong>
                <code>{selectedTenant.id}</code>
              </div>
            {:else}
              <p class="empty">No tenant selected. Choose one from the Tenants page.</p>
            {/if}
          </section>

          <section class="panel span-6">
            <h3>Recent Tenants</h3>
            <p>Latest entries from your tenant directory.</p>
            <div class="table-shell">
              <div class="table-wrap">
                <table>
                  <thead>
                    <tr><th>Slug</th><th>Name</th><th>Status</th></tr>
                  </thead>
                  <tbody>
                    {#each tenants.slice(0, 6) as t}
                      <tr>
                        <td>{t.slug}</td>
                        <td>{t.name}</td>
                        <td><span class="status-pill" class:active={(t.status || '').toLowerCase() === 'active'}>{t.status || 'unknown'}</span></td>
                      </tr>
                    {:else}
                      <tr><td colspan="3">No tenants loaded.</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </section>

          <section class="panel span-6">
            <h3>Recent Users</h3>
            <p>Users from the most recent query.</p>
            <div class="table-shell">
              <div class="table-wrap">
                <table>
                  <thead>
                    <tr><th>Email</th><th>Status</th><th>Name</th></tr>
                  </thead>
                  <tbody>
                    {#each users.slice(0, 6) as u}
                      <tr>
                        <td>{u.email}</td>
                        <td><span class="status-pill" class:active={(u.status || '').toLowerCase() === 'active'}>{u.status || 'unknown'}</span></td>
                        <td>{u.display_name || ''}</td>
                      </tr>
                    {:else}
                      <tr><td colspan="3">No users loaded.</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        </div>
      {:else if page === 'session'}
        <div class="grid">
          <section class="panel span-8">
            <h3>Admin Session</h3>
            <p>Store an optional `X-Admin-Key` in local storage for API calls.</p>
            <label for="adminKey">X-Admin-Key</label>
            <div class="input-inline">
              {#if showKey}
                <input
                  id="adminKey"
                  name="adminKey"
                  type="text"
                  bind:value={adminKey}
                  autocomplete="off"
                  placeholder="Paste admin key…"
                />
              {:else}
                <input
                  id="adminKey"
                  name="adminKey"
                  type="password"
                  bind:value={adminKey}
                  autocomplete="off"
                  placeholder="Paste admin key…"
                />
              {/if}
              <button class="btn-ghost" type="button" on:click={() => (showKey = !showKey)}>
                {showKey ? 'Hide Key' : 'Show Key'}
              </button>
            </div>
            <div class="row">
              <button type="button" on:click={saveKey}>Save Key</button>
              <button class="btn-ghost" type="button" on:click={clearKey}>Clear Key</button>
            </div>
            <p class="status" class:ok={sessionStatus.ok} class:err={!sessionStatus.ok} aria-live="polite">{sessionStatus.msg}</p>
          </section>

          <section class="panel span-4">
            <h3>Quick Actions</h3>
            <p>Move quickly between setup and provisioning tasks.</p>
            <div class="stack">
              <button type="button" on:click={loadTenants}>Load Tenants</button>
              <button class="btn-secondary" type="button" on:click={() => (page = 'tenants')}>Open Tenants</button>
              <button class="btn-ghost" type="button" on:click={() => (page = 'users')}>Open Users</button>
            </div>
          </section>
        </div>
      {:else if page === 'tenants'}
        <div class="grid">
          <section class="panel span-4">
            <h3>Create Tenant</h3>
            <p>Create a tenant and instantly set it as your active context.</p>
            <label for="tenantSlug">Tenant Slug</label>
            <input
              id="tenantSlug"
              name="tenantSlug"
              bind:value={tenantSlug}
              autocomplete="off"
              placeholder="acme-corp…"
            />
            <label for="tenantName">Tenant Name</label>
            <input
              id="tenantName"
              name="tenantName"
              bind:value={tenantName}
              autocomplete="off"
              placeholder="Acme Corporation…"
            />
            <div class="row">
              <button type="button" on:click={doCreateTenant}>Create Tenant</button>
            </div>
            <p class="status" class:ok={tenantStatus.ok} class:err={!tenantStatus.ok} aria-live="polite">{tenantStatus.msg}</p>
          </section>

          <section class="panel span-8">
            <div class="meta">
              <div>
                <h3>Tenant Directory</h3>
                <p>Search, inspect, and set the tenant context.</p>
              </div>
              <div class="meta-controls">
                <label class="inline-label" for="tenantSearch">Search</label>
                <input
                  id="tenantSearch"
                  name="tenantSearch"
                  bind:value={tenantSearch}
                  autocomplete="off"
                  placeholder="Search slug or name…"
                />
                <button class="btn-ghost" type="button" on:click={loadTenants}>Refresh</button>
              </div>
            </div>
            <div class="table-shell">
              <div class="table-wrap">
                <table>
                  <thead>
                    <tr><th>ID</th><th>Slug</th><th>Name</th><th>Status</th><th class="align-right">Action</th></tr>
                  </thead>
                  <tbody>
                    {#each filteredTenants as t}
                      <tr>
                        <td><code>{t.id}</code></td>
                        <td>{t.slug}</td>
                        <td>{t.name}</td>
                        <td><span class="status-pill" class:active={(t.status || '').toLowerCase() === 'active'}>{t.status || 'unknown'}</span></td>
                        <td class="align-right">
                          <button class="btn-ghost btn-small" type="button" on:click={() => selectTenant(t.id)}>
                            Select
                          </button>
                        </td>
                      </tr>
                    {:else}
                      <tr><td colspan="5">No matching tenants.</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        </div>
      {:else if page === 'users'}
        <div class="grid">
          <section class="panel span-4">
            <h3>Create User</h3>
            <p>Create a user and set an initial password for first access.</p>
            <label for="tenantId">Tenant ID</label>
            <input
              id="tenantId"
              name="tenantId"
              bind:value={userTenantId}
              autocomplete="off"
              placeholder="Paste tenant ID…"
            />
            <label for="userEmail">Email</label>
            <input
              id="userEmail"
              name="userEmail"
              type="email"
              inputmode="email"
              spellcheck="false"
              bind:value={userEmail}
              autocomplete="off"
              placeholder="owner@acme.com…"
            />
            <label for="userName">Display Name</label>
            <input
              id="userName"
              name="userName"
              bind:value={userName}
              autocomplete="off"
              placeholder="Jane Owner…"
            />
            <label for="userPassword">Initial Password</label>
            <input
              id="userPassword"
              name="userPassword"
              type="password"
              bind:value={userPassword}
              autocomplete="new-password"
              placeholder="password123…"
            />
            <div class="row">
              <button type="button" on:click={doCreateUser}>Create User</button>
              <button class="btn-secondary" type="button" on:click={loadUsers}>Load Users</button>
            </div>
            <p class="status" class:ok={userStatus.ok} class:err={!userStatus.ok} aria-live="polite">{userStatus.msg}</p>
          </section>

          <section class="panel span-8">
            <div class="meta">
              <div>
                <h3>User Directory</h3>
                <p>Explore users for the active tenant context.</p>
              </div>
              <div class="meta-stack">
                {#if selectedTenant}
                  <span class="selected">
                    Tenant: {selectedTenant.slug} <code>{selectedTenant.id}</code>
                  </span>
                {/if}
                <label class="inline-label" for="userSearch">Search</label>
                <input
                  id="userSearch"
                  name="userSearch"
                  bind:value={userSearch}
                  autocomplete="off"
                  placeholder="Search email or display name…"
                />
              </div>
            </div>
            <div class="table-shell">
              <div class="table-wrap">
                <table>
                  <thead>
                    <tr><th>ID</th><th>Email</th><th>Status</th><th>Name</th></tr>
                  </thead>
                  <tbody>
                    {#each filteredUsers as u}
                      <tr>
                        <td><code>{u.id}</code></td>
                        <td>{u.email}</td>
                        <td><span class="status-pill" class:active={(u.status || '').toLowerCase() === 'active'}>{u.status || 'unknown'}</span></td>
                        <td>{u.display_name || ''}</td>
                      </tr>
                    {:else}
                      <tr><td colspan="4">No matching users.</td></tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        </div>
      {/if}
    </section>
  </main>
</div>

<style>
  :global(body) {
    margin: 0;
    color-scheme: light;
    background: #f4f7fb;
  }

  .skip-link {
    position: absolute;
    top: -60px;
    left: 12px;
    padding: 10px 14px;
    border-radius: 10px;
    border: 1px solid #c7d3e4;
    background: #ffffff;
    color: #11253d;
    text-decoration: none;
    z-index: 15;
  }

  .skip-link:focus-visible {
    top: 12px;
    outline: 3px solid rgba(96, 165, 250, 0.45);
    outline-offset: 3px;
  }

  .admin-app {
    width: min(1320px, calc(100% - 24px));
    margin: 12px auto;
    display: grid;
    gap: 14px;
    grid-template-columns: 280px minmax(0, 1fr);
    font-family: 'Inter', 'Avenir Next', 'SF Pro Text', 'Helvetica Neue', sans-serif;
    color: #0f2640;
    min-height: calc(100vh - 24px);
    background:
      radial-gradient(circle at 14% 0%, rgba(45, 212, 191, 0.2), transparent 36%),
      radial-gradient(circle at 88% 4%, rgba(96, 165, 250, 0.2), transparent 34%),
      linear-gradient(180deg, #f7faff 0%, #f3f7fd 100%);
    border: 1px solid #d9e3f3;
    border-radius: 24px;
    padding: 12px;
    box-sizing: border-box;
  }

  .sidebar {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 18px;
    border-radius: 18px;
    border: 1px solid #233b5a;
    color: #dbe8f8;
    background:
      radial-gradient(circle at 12% 10%, rgba(45, 212, 191, 0.22), transparent 40%),
      linear-gradient(180deg, #10233d 0%, #112742 55%, #102238 100%);
    box-shadow: 0 20px 38px rgba(10, 22, 39, 0.28);
    position: sticky;
    top: 12px;
    min-height: calc(100vh - 48px);
    box-sizing: border-box;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .brand-mark {
    width: 42px;
    height: 42px;
    border-radius: 12px;
    background: linear-gradient(145deg, #2dd4bf 0%, #38bdf8 100%);
    box-shadow: 0 10px 24px rgba(45, 212, 191, 0.35);
  }

  .brand-copy {
    min-width: 0;
  }

  .brand-copy h1 {
    margin: 2px 0 0;
    font-size: 20px;
    line-height: 1.1;
    letter-spacing: -0.02em;
    text-wrap: balance;
  }

  .eyebrow {
    margin: 0;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-size: 11px;
    font-weight: 700;
    color: #8fb5db;
  }

  .system-chip {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    border-radius: 999px;
    width: fit-content;
    padding: 8px 12px;
    border: 1px solid #35557c;
    background: #1a3858;
    color: #d6e9ff;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.03em;
  }

  .system-chip.connected {
    color: #dcfff4;
    background: #145149;
    border-color: #2d9c89;
  }

  .chip-dot {
    width: 8px;
    height: 8px;
    border-radius: 999px;
    background: #fbbf24;
    box-shadow: 0 0 0 4px rgba(251, 191, 36, 0.22);
  }

  .system-chip.connected .chip-dot {
    background: #34d399;
    box-shadow: 0 0 0 4px rgba(52, 211, 153, 0.2);
  }

  .nav {
    display: grid;
    gap: 8px;
  }

  .nav-item {
    border: 1px solid #35557c;
    border-radius: 12px;
    background: #1a3656;
    color: #e5f0ff;
    font: inherit;
    text-align: left;
    padding: 11px 12px;
    cursor: pointer;
    transition: background-color 140ms ease, border-color 140ms ease, transform 140ms ease;
    touch-action: manipulation;
  }

  .nav-item:hover {
    background: #214163;
    border-color: #4f79a3;
  }

  .nav-item.active {
    background: linear-gradient(135deg, #34d399 0%, #38bdf8 100%);
    border-color: #7dd3fc;
    color: #062337;
    font-weight: 700;
  }

  .sidebar-actions {
    margin-top: auto;
    display: grid;
    gap: 8px;
  }

  .sidebar-actions form {
    margin: 0;
  }

  .content {
    border-radius: 18px;
    border: 1px solid #d5e2f3;
    background: rgba(255, 255, 255, 0.84);
    backdrop-filter: blur(8px);
    display: grid;
    grid-template-rows: auto minmax(0, 1fr);
    min-width: 0;
    overflow: hidden;
  }

  .topbar {
    border-bottom: 1px solid #dde8f8;
    padding: 20px;
    display: flex;
    justify-content: space-between;
    gap: 14px;
    align-items: flex-start;
    flex-wrap: wrap;
    background: linear-gradient(180deg, rgba(255, 255, 255, 0.95), rgba(250, 253, 255, 0.88));
  }

  .topbar-title h2 {
    margin: 2px 0 0;
    font-size: clamp(24px, 2vw, 30px);
    letter-spacing: -0.02em;
    line-height: 1.12;
    text-wrap: balance;
  }

  .subtitle {
    margin: 6px 0 0;
    color: #425e7e;
    font-size: 14px;
    max-width: 65ch;
  }

  .quick-metrics {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
    min-width: 360px;
  }

  .quick-metrics article {
    border: 1px solid #dce8f8;
    border-radius: 12px;
    background: #f9fbff;
    padding: 10px;
    min-width: 0;
  }

  .quick-metrics span {
    display: block;
    font-size: 12px;
    color: #4b6684;
  }

  .quick-metrics strong {
    display: block;
    margin-top: 5px;
    font-size: 19px;
    font-variant-numeric: tabular-nums;
  }

  .workspace {
    padding: 16px;
    overflow: auto;
    min-width: 0;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: 12px;
    min-width: 0;
  }

  .span-4 { grid-column: span 4; }
  .span-6 { grid-column: span 6; }
  .span-8 { grid-column: span 8; }

  .panel {
    border: 1px solid #dbe6f6;
    border-radius: 16px;
    background: linear-gradient(180deg, #ffffff 0%, #f9fbff 100%);
    box-shadow: 0 8px 24px rgba(15, 23, 42, 0.06);
    padding: 16px;
    min-width: 0;
  }

  .panel h3 {
    margin: 0;
    font-size: 18px;
    letter-spacing: -0.02em;
    text-wrap: balance;
  }

  .panel p {
    margin: 8px 0 12px;
    color: #4f6884;
    line-height: 1.45;
  }

  .stats {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
    margin-bottom: 12px;
  }

  .stat-card {
    border: 1px solid #dce8f8;
    border-radius: 12px;
    background: #f8fbff;
    padding: 12px;
    min-width: 0;
  }

  .stat-card span {
    display: block;
    font-size: 12px;
    color: #4f6987;
  }

  .stat-card strong {
    display: block;
    margin-top: 6px;
    font-size: 22px;
    line-height: 1.15;
    text-wrap: balance;
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 10px;
  }

  .stack {
    display: grid;
    gap: 8px;
  }

  label {
    display: block;
    margin: 10px 0 6px;
    font-size: 12px;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    font-weight: 700;
    color: #25435f;
  }

  .inline-label {
    margin: 0;
  }

  input,
  button {
    font: inherit;
  }

  input {
    width: 100%;
    min-width: 0;
    box-sizing: border-box;
    border: 1px solid #cbdaee;
    border-radius: 11px;
    background: #fcfdff;
    color: inherit;
    padding: 10px 12px;
  }

  input:focus-visible {
    border-color: #38bdf8;
    outline: 3px solid rgba(56, 189, 248, 0.22);
    outline-offset: 2px;
  }

  .input-inline {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 8px;
    align-items: center;
  }

  button {
    border: 1px solid transparent;
    border-radius: 11px;
    padding: 10px 14px;
    color: #ffffff;
    background: linear-gradient(135deg, #0f766e 0%, #14b8a6 100%);
    font-weight: 700;
    cursor: pointer;
    transition: transform 130ms ease, box-shadow 130ms ease, filter 130ms ease;
    touch-action: manipulation;
    -webkit-tap-highlight-color: rgba(20, 184, 166, 0.2);
  }

  button:hover {
    transform: translateY(-1px);
    box-shadow: 0 8px 20px rgba(15, 118, 110, 0.28);
    filter: saturate(1.08);
  }

  button:focus-visible {
    outline: 3px solid rgba(20, 184, 166, 0.28);
    outline-offset: 2px;
  }

  .btn-secondary {
    background: linear-gradient(135deg, #2f567e 0%, #3d6b9a 100%);
  }

  .btn-danger {
    background: linear-gradient(135deg, #b42318 0%, #dc2626 100%);
  }

  .btn-ghost {
    background: #eef4ff;
    border-color: #c7d6ea;
    color: #1e3f60;
  }

  .btn-small {
    padding: 7px 10px;
    font-size: 12px;
  }

  .status {
    margin-top: 10px;
    min-height: 1.2em;
    font-size: 13px;
  }

  .status.ok { color: #0f766e; }
  .status.err { color: #b42318; }

  .context-pill {
    display: grid;
    gap: 6px;
    border: 1px solid #cbe0f3;
    border-radius: 13px;
    padding: 12px;
    background: #f5faff;
  }

  .context-pill .label {
    display: inline-block;
    width: fit-content;
    background: #dff1ff;
    border-radius: 999px;
    padding: 4px 10px;
    font-size: 12px;
    color: #1d4f7a;
  }

  .context-pill code,
  td code,
  .selected code {
    border: 1px solid #c8d9ed;
    border-radius: 8px;
    background: #edf5ff;
    color: #143f64;
    font-size: 12px;
    padding: 2px 6px;
    width: fit-content;
  }

  .empty {
    margin: 0;
    padding: 12px;
    border-radius: 12px;
    border: 1px dashed #bed0e8;
    background: #f8fbff;
    color: #4b6788;
  }

  .meta {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 12px;
  }

  .meta p {
    margin-bottom: 0;
  }

  .meta-controls,
  .meta-stack {
    display: grid;
    gap: 7px;
    width: min(320px, 100%);
    min-width: 0;
  }

  .selected {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    width: fit-content;
    border: 1px solid #a8d3bf;
    border-radius: 999px;
    background: #e3f6ee;
    color: #145945;
    padding: 5px 10px;
    font-size: 12px;
    font-weight: 700;
  }

  .table-shell {
    border: 1px solid #dbe7f5;
    border-radius: 13px;
    overflow: hidden;
    background: #ffffff;
  }

  .table-wrap {
    max-height: 340px;
    overflow: auto;
    min-width: 0;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    min-width: 620px;
  }

  th,
  td {
    border-bottom: 1px solid #edf3fb;
    text-align: left;
    padding: 10px;
    vertical-align: middle;
    min-width: 0;
  }

  th {
    background: #f5f9ff;
    color: #4a6584;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    font-size: 11px;
    position: sticky;
    top: 0;
    z-index: 1;
  }

  td {
    color: #122c46;
    font-size: 13px;
    word-break: break-word;
  }

  .align-right {
    text-align: right;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    border-radius: 999px;
    border: 1px solid #c8dcee;
    background: #eaf3ff;
    color: #1f486a;
    padding: 3px 9px;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    font-weight: 700;
  }

  .status-pill.active {
    border-color: #a8dfc8;
    background: #e2f8ee;
    color: #0f6b4d;
  }

  @media (max-width: 1140px) {
    .admin-app {
      grid-template-columns: 1fr;
      width: min(1320px, calc(100% - 14px));
      border-radius: 18px;
    }

    .sidebar {
      position: static;
      min-height: 0;
      border-radius: 14px;
    }

    .quick-metrics {
      min-width: 0;
      width: 100%;
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 980px) {
    .span-4,
    .span-6,
    .span-8 {
      grid-column: span 12;
    }

    .stats {
      grid-template-columns: 1fr;
    }

    .quick-metrics {
      grid-template-columns: 1fr;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    *,
    *::before,
    *::after {
      animation: none !important;
      transition: none !important;
      scroll-behavior: auto !important;
    }
  }
</style>
