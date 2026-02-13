package http

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

const adminUICookieName = "locky_admin_ui_session"

const adminUILoginHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Locky Admin Login</title>
  <style>
    :root {
      --bg: #f4f8fb;
      --text: #0f1f2f;
      --muted: #4b6073;
      --line: #c8d8e6;
      --panel: #ffffff;
      --brand: #0e7490;
      --brand-strong: #0b5f76;
      --danger: #b42318;
    }

    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      padding: 24px;
      font-family: "Avenir Next", "SF Pro Text", "Helvetica Neue", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at 10% 20%, rgba(14,116,144,0.16), transparent 34%),
        radial-gradient(circle at 90% 18%, rgba(242,159,5,0.18), transparent 38%),
        linear-gradient(180deg, #f8fbfd 0%, var(--bg) 100%);
    }

    .shell {
      width: min(100%, 420px);
      background: #fcfeff;
      border: 1px solid #d4e0ea;
      border-radius: 18px;
      padding: 28px;
      box-shadow: 0 20px 40px rgba(15, 23, 42, 0.10);
      animation: rise .38s ease-out;
    }

    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 14px;
      padding: 6px 12px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 600;
      letter-spacing: 0.06em;
      text-transform: uppercase;
      color: #0b5f76;
      background: #d7eef5;
      border: 1px solid #a8d3df;
    }

    .dot {
      width: 8px;
      height: 8px;
      border-radius: 999px;
      background: #06b6d4;
      box-shadow: 0 0 0 4px rgba(6, 182, 212, 0.20);
    }

    h1 {
      margin: 0;
      font-size: 28px;
      line-height: 1.1;
      letter-spacing: -0.02em;
    }

    p {
      margin: 10px 0 18px;
      color: var(--muted);
      line-height: 1.5;
    }

    label {
      display: block;
      font-size: 13px;
      margin: 14px 0 6px;
      color: #17354e;
      font-weight: 600;
    }

    input {
      width: 100%;
      border: 1px solid var(--line);
      background: #fbfdff;
      border-radius: 10px;
      padding: 11px 12px;
      font: inherit;
      color: inherit;
      transition: border-color .15s ease, box-shadow .15s ease;
    }

    input:focus {
      outline: none;
      border-color: #0c8bb0;
      box-shadow: 0 0 0 4px rgba(14,116,144,0.18);
    }

    button {
      margin-top: 18px;
      width: 100%;
      border: 0;
      border-radius: 10px;
      padding: 12px 14px;
      font-weight: 700;
      font: inherit;
      color: #fff;
      background: linear-gradient(135deg, var(--brand) 0%, #1498bc 100%);
      cursor: pointer;
      transition: transform .12s ease, box-shadow .12s ease, background .12s ease;
      box-shadow: 0 12px 20px rgba(14, 116, 144, 0.24);
    }

    button:hover {
      background: linear-gradient(135deg, var(--brand-strong) 0%, #1180a0 100%);
      transform: translateY(-1px);
    }

    .err {
      margin: 0 0 12px;
      padding: 10px 12px;
      border-radius: 10px;
      color: var(--danger);
      background: #fff2f0;
      border: 1px solid #f8c9c5;
      font-size: 13px;
    }

    @keyframes rise {
      from { opacity: 0; transform: translateY(14px); }
      to { opacity: 1; transform: translateY(0); }
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="eyebrow"><span class="dot"></span>Locky Admin</div>
    <h1>Welcome back</h1>
    <p>Sign in to manage tenants and users from a single workspace.</p>
    %s
    <form method="POST" action="/admin/ui/login">
      <label for="username">Username</label>
      <input id="username" name="username" autocomplete="username" required>
      <label for="password">Password</label>
      <input id="password" name="password" type="password" autocomplete="current-password" required>
      <button type="submit">Sign In</button>
    </form>
  </div>
</body>
</html>`

const adminUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Locky Admin</title>
  <style>
    :root {
      --bg: #f2f7f9;
      --text: #13263a;
      --muted: #4f6578;
      --line: #d1deea;
      --panel: #ffffff;
      --brand: #0f766e;
      --brand-strong: #0b5f58;
      --accent: #f59e0b;
      --ok: #13795b;
      --err: #b42318;
      --sidebar: #0f2434;
      --sidebar-line: #284156;
    }

    * { box-sizing: border-box; }

    body {
      margin: 0;
      min-height: 100vh;
      font-family: "Avenir Next", "SF Pro Text", "Helvetica Neue", sans-serif;
      color: var(--text);
      background:
        radial-gradient(circle at 8% 12%, rgba(15,118,110,0.14), transparent 35%),
        radial-gradient(circle at 92% 8%, rgba(245,158,11,0.20), transparent 36%),
        linear-gradient(180deg, #f7fbfd 0%, var(--bg) 100%);
    }

    .app {
      width: min(1260px, 100% - 28px);
      margin: 16px auto;
      display: grid;
      grid-template-columns: 250px minmax(0, 1fr);
      gap: 14px;
    }

    .sidebar {
      border: 1px solid var(--sidebar-line);
      border-radius: 16px;
      background: linear-gradient(180deg, #122c40 0%, var(--sidebar) 100%);
      color: #d8e8f5;
      padding: 14px;
      display: flex;
      flex-direction: column;
      gap: 14px;
      box-shadow: 0 16px 30px rgba(8, 23, 35, 0.28);
      min-height: calc(100vh - 32px);
      position: sticky;
      top: 16px;
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .brand-mark {
      width: 34px;
      height: 34px;
      border-radius: 10px;
      background: linear-gradient(135deg, #0f766e 0%, #14b8a6 100%);
      box-shadow: 0 10px 20px rgba(20, 184, 166, 0.28);
    }

    .brand h1 {
      margin: 0;
      font-size: 18px;
      color: #fff;
      letter-spacing: -0.01em;
    }

    .brand p {
      margin: 0;
      font-size: 12px;
      color: #9fbbd1;
    }

    .chip {
      display: flex;
      align-items: center;
      gap: 8px;
      border-radius: 10px;
      font-size: 12px;
      font-weight: 700;
      letter-spacing: 0.04em;
      text-transform: uppercase;
      padding: 8px 10px;
      border: 1px solid #49647a;
      background: #244055;
      color: #d6e8f4;
    }

    .chip-dot {
      width: 8px;
      height: 8px;
      border-radius: 999px;
      background: #f59e0b;
      box-shadow: 0 0 0 4px rgba(245, 158, 11, 0.22);
    }

    .chip.connected {
      color: #d9f8ef;
      background: #154c47;
      border-color: #2d7f75;
    }

    .chip.connected .chip-dot {
      background: #10b981;
      box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.20);
    }

    .nav {
      display: grid;
      gap: 6px;
    }

    .nav button {
      text-align: left;
      border: 1px solid #39556b;
      background: #1a364c;
      color: #dceaf5;
      font: inherit;
      border-radius: 10px;
      padding: 10px 11px;
      cursor: pointer;
      transition: transform .12s ease, background .12s ease, border-color .12s ease;
    }

    .nav button:hover {
      background: #23445c;
      transform: translateX(1px);
    }

    .nav button.active {
      background: linear-gradient(135deg, #0f766e 0%, #14b8a6 100%);
      border-color: #34d4c4;
      color: #fff;
    }

    .sidebar footer {
      margin-top: auto;
      display: grid;
      gap: 8px;
    }

    .sidebar form,
    .sidebar footer button {
      width: 100%;
    }

    .content {
      border: 1px solid #d9e4ef;
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.78);
      box-shadow: 0 14px 28px rgba(15, 23, 42, 0.09);
      overflow: hidden;
      min-height: calc(100vh - 32px);
      display: grid;
      grid-template-rows: auto minmax(0, 1fr);
    }

    .topbar {
      border-bottom: 1px solid #dce8f1;
      padding: 16px 18px 14px;
      background: rgba(255, 255, 255, 0.8);
      backdrop-filter: blur(8px);
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 10px;
    }

    .title {
      margin: 0;
      font-size: 24px;
      letter-spacing: -0.02em;
    }

    .subtitle {
      margin: 2px 0 0;
      color: var(--muted);
      font-size: 13px;
    }

    .workspace {
      padding: 14px;
      overflow: auto;
    }

    .grid {
      display: grid;
      grid-template-columns: repeat(12, minmax(0, 1fr));
      gap: 12px;
    }

    .span-12 { grid-column: span 12; }
    .span-8 { grid-column: span 8; }
    .span-6 { grid-column: span 6; }
    .span-4 { grid-column: span 4; }

    .panel {
      border: 1px solid #d9e5ef;
      border-radius: 16px;
      background: linear-gradient(180deg, #ffffff 0%, #f8fbfd 100%);
      box-shadow: 0 8px 24px rgba(15, 23, 42, 0.05);
      padding: 16px;
    }

    .panel h2 {
      margin: 0;
      font-size: 17px;
      letter-spacing: -0.01em;
    }

    .panel p {
      margin: 8px 0 12px;
      color: var(--muted);
      font-size: 13px;
    }

    .row {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-bottom: 10px;
    }

    label {
      display: block;
      font-size: 12px;
      font-weight: 700;
      letter-spacing: 0.04em;
      text-transform: uppercase;
      color: #31506b;
      margin: 10px 0 6px;
    }

    input, button {
      font: inherit;
      font-size: 14px;
      border-radius: 10px;
    }

    input {
      border: 1px solid var(--line);
      background: #fbfdff;
      color: inherit;
      padding: 10px 12px;
      width: 100%;
      min-width: 0;
      transition: border-color .15s ease, box-shadow .15s ease;
    }

    input:focus {
      outline: none;
      border-color: #0f766e;
      box-shadow: 0 0 0 4px rgba(15, 118, 110, 0.16);
    }

    .input-inline {
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 8px;
      align-items: center;
    }

    button {
      border: 0;
      cursor: pointer;
      color: #fff;
      background: linear-gradient(135deg, var(--brand) 0%, #14b8a6 100%);
      padding: 10px 13px;
      font-weight: 600;
      transition: transform .12s ease, box-shadow .12s ease;
    }

    button:hover {
      transform: translateY(-1px);
      box-shadow: 0 8px 18px rgba(15, 118, 110, 0.25);
    }

    button.secondary {
      background: linear-gradient(135deg, #36566f 0%, #2d455c 100%);
    }

    button.warn {
      background: linear-gradient(135deg, #9a3412 0%, #c2410c 100%);
    }

    button.ghost {
      color: #27425a;
      background: #ecf4fa;
      border: 1px solid #c7d9e8;
    }

    button.small {
      padding: 7px 11px;
      font-size: 12px;
      border-radius: 8px;
    }

    button:disabled {
      opacity: 0.65;
      cursor: not-allowed;
      transform: none;
      box-shadow: none;
    }

    .status {
      margin-top: 8px;
      font-size: 13px;
      min-height: 18px;
    }

    .ok { color: var(--ok); }
    .err { color: var(--err); }

    .table-shell {
      border: 1px solid #dbe7f1;
      border-radius: 12px;
      overflow: hidden;
      background: #fff;
    }

    table {
      border-collapse: collapse;
      width: 100%;
      min-width: 620px;
    }

    th, td {
      padding: 10px 10px;
      border-bottom: 1px solid #ecf2f7;
      text-align: left;
      font-size: 13px;
      vertical-align: middle;
    }

    th {
      color: #4f6579;
      font-weight: 700;
      letter-spacing: 0.03em;
      text-transform: uppercase;
      font-size: 11px;
      background: #f7fbff;
      position: sticky;
      top: 0;
      z-index: 1;
    }

    td code {
      background: #edf4fb;
      border: 1px solid #d6e3ee;
      border-radius: 8px;
      padding: 2px 6px;
      font-size: 12px;
      color: #20425c;
    }

    .status-pill {
      display: inline-block;
      border-radius: 999px;
      padding: 3px 9px;
      font-size: 11px;
      font-weight: 700;
      letter-spacing: 0.03em;
      text-transform: uppercase;
      background: #eaf4fb;
      border: 1px solid #c8ddeb;
      color: #204862;
    }

    .status-pill.active {
      background: #e4f8ef;
      border-color: #b5e6cb;
      color: #0f6f4f;
    }

    .status-pill.pending {
      background: #fff6e8;
      border-color: #f2dcaf;
      color: #8a5a00;
    }

    .table-actions {
      display: flex;
      gap: 6px;
      justify-content: flex-end;
    }

    .table-wrap {
      overflow: auto;
      max-height: 290px;
    }

    .tiny {
      color: var(--muted);
      font-size: 12px;
      margin-top: 8px;
      line-height: 1.4;
    }

    .meta {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 10px;
      margin-bottom: 10px;
      flex-wrap: wrap;
    }

    .meta strong {
      font-size: 14px;
    }

    .selected {
      color: #0b5f58;
      background: #dff4ef;
      border: 1px solid #a5dbc8;
      border-radius: 999px;
      padding: 5px 10px;
      font-size: 12px;
      font-weight: 700;
    }

    .stats {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 10px;
    }

    .stat {
      border: 1px solid #d7e4ef;
      border-radius: 12px;
      padding: 12px;
      background: #fbfdff;
    }

    .stat span {
      display: block;
      font-size: 12px;
      color: var(--muted);
    }

    .stat strong {
      display: block;
      margin-top: 6px;
      font-size: 24px;
      letter-spacing: -0.02em;
    }

    @media (max-width: 1080px) {
      .app {
        grid-template-columns: 1fr;
      }

      .sidebar {
        min-height: auto;
        position: static;
      }

      .content {
        min-height: auto;
      }

      .nav {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }

      .span-8,
      .span-6,
      .span-4 {
        grid-column: span 12;
      }
    }

    @media (max-width: 640px) {
      .app {
        width: min(1260px, 100% - 16px);
        margin: 8px auto;
      }

      .topbar {
        padding: 14px;
      }

      .workspace {
        padding: 10px;
      }

      .panel {
        padding: 14px;
      }

      .stats {
        grid-template-columns: 1fr;
      }

      .row { margin-bottom: 8px; }
      table { min-width: 520px; }
    }
  </style>
</head>
<body>
  <div class="app">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark" aria-hidden="true"></div>
        <div>
          <h1>Locky Admin</h1>
          <p>Control workspace</p>
        </div>
      </div>

      <div id="apiState" class="chip connected"><span class="chip-dot"></span>UI Session</div>

      <nav id="nav" class="nav">
        <button class="nav-item active" data-page="overview" type="button">Overview</button>
        <button class="nav-item" data-page="session" type="button">Session</button>
        <button class="nav-item" data-page="tenants" type="button">Tenants</button>
        <button class="nav-item" data-page="users" type="button">Users</button>
      </nav>

      <footer>
        <button id="refreshAll" type="button">Refresh Data</button>
        <button id="toggleAdvanced" class="ghost" type="button">Show Advanced</button>
        <form method="POST" action="/admin/ui/logout" style="margin: 0;">
          <button class="warn" type="submit">Sign Out</button>
        </form>
      </footer>
    </aside>

    <section class="content">
      <header class="topbar">
        <div>
          <h2 id="pageTitle" class="title">Overview</h2>
          <p id="pageSubtitle" class="subtitle">Quick status across tenants and users.</p>
        </div>
      </header>
      <div id="pageHost" class="workspace"></div>
    </section>
  </div>

  <script>
    let adminKey = localStorage.getItem("locky_admin_key") || "";
    let tenants = [];
    let users = [];
    let selectedTenantId = "";
    let currentPage = "overview";
    let showAdvanced = false;

    const statusState = {
      sessionStatus: { msg: "", ok: true },
      tenantStatus: { msg: "", ok: true },
      userStatus: { msg: "", ok: true }
    };

    const pageMeta = {
      overview: { title: "Overview", subtitle: "Quick status across tenants and users." },
      session: { title: "Session", subtitle: "Optional local API key and connection state." },
      tenants: { title: "Tenants", subtitle: "Create and browse tenant workspaces." },
      users: { title: "Users", subtitle: "Manage users under a selected tenant." }
    };

    function panelComponent(title, subtitle, body) {
      return "<section class='panel'><h2>" + title + "</h2><p>" + subtitle + "</p>" + body + "</section>";
    }

    function overviewPage() {
      return "<div class='grid'>" +
        "<div class='span-12'>" + panelComponent("Workspace Snapshot", "A quick pulse on the admin system.",
          "<div class='stats'>" +
            "<article class='stat'><span>Tenants</span><strong id='statTenantCount'>" + tenants.length + "</strong></article>" +
            "<article class='stat'><span>Loaded Users</span><strong id='statUserCount'>" + users.length + "</strong></article>" +
            "<article class='stat'><span>Tenant Context</span><strong id='statTenantContext'>" + (selectedTenantId ? "1" : "0") + "</strong></article>" +
          "</div>" +
          "<div class='row' style='margin-top:12px;margin-bottom:0;'><button id='overviewLoadTenants' type='button'>Load Tenants</button><button id='overviewLoadUsers' class='secondary' type='button'>Load Users</button><button id='overviewGoSession' class='ghost' type='button'>Edit Session</button></div>") + "</div>" +
        "<div class='span-6'>" + panelComponent("Recent Tenants", "Latest loaded tenants.",
          "<div class='table-shell'><div class='table-wrap'><table><thead><tr><th>Slug</th><th>Name</th><th>Status</th></tr></thead><tbody id='overviewTenantRows'></tbody></table></div></div>") + "</div>" +
        "<div class='span-6'>" + panelComponent("Recent Users", "Users from selected tenant.",
          "<div class='table-shell'><div class='table-wrap'><table><thead><tr><th>Email</th><th>Status</th><th>Name</th></tr></thead><tbody id='overviewUserRows'></tbody></table></div></div>") + "</div>" +
      "</div>";
    }

    function sessionPage() {
      return "<div class='grid'>" +
        "<div class='span-8'>" + panelComponent("Admin Session", "Optional: save an API key in local browser storage.",
          "<label for='adminKey'>X-Admin-Key</label>" +
          "<div class='input-inline'><input id='adminKey' type='password' autocomplete='off' placeholder='Paste admin key'><button id='toggleKey' class='ghost' type='button'>Show</button></div>" +
          "<div class='row' style='margin-top: 10px; margin-bottom: 0;'><button id='saveKey' type='button'>Save Key</button><button id='clearKey' class='ghost' type='button'>Clear</button></div>" +
          "<div id='sessionStatus' class='status'></div>") + "</div>" +
        "<div class='span-4'>" + panelComponent("Quick Actions", "Useful shortcuts while setting up.",
          "<div class='row' style='margin-bottom:0;'><button id='sessionLoadTenants' type='button'>Load Tenants</button><button id='sessionToTenants' class='secondary' type='button'>Go To Tenants</button></div>") + "</div>" +
      "</div>";
    }

    function tenantsPage() {
      return "<div class='grid'>" +
        "<div class='span-4'>" + panelComponent("Create Tenant", "Creates tenant and focuses users on it.",
          "<label for='tenantSlug'>Tenant Slug</label><input id='tenantSlug' placeholder='acme-corp'>" +
          "<label for='tenantName'>Tenant Name</label><input id='tenantName' placeholder='Acme Corporation'>" +
          "<div class='row' style='margin-top:10px;margin-bottom:0;'><button id='createTenant' type='button'>Create Tenant</button></div>" +
          "<div id='tenantStatus' class='status'></div>") + "</div>" +
        "<div class='span-8'>" + panelComponent("Tenant Directory", "Browse and select a tenant context.",
          "<div class='meta'><strong>All Tenants</strong><div class='row' style='margin:0;'><input id='tenantSearch' style='width:220px;' placeholder='Search slug or name'><button id='refreshTenants' class='small ghost' type='button'>Refresh</button></div></div>" +
          "<div class='table-shell'><div class='table-wrap'><table><thead><tr><th>ID</th><th>Slug</th><th>Name</th><th>Status</th><th style='text-align:right;'>Action</th></tr></thead><tbody id='tenantRows'></tbody></table></div></div>" +
          "<div class='tiny'>Tip: Select a tenant, then open the Users page.</div>") + "</div>" +
      "</div>";
    }

    function usersPage() {
      return "<div class='grid'>" +
        "<div class='span-4'>" + panelComponent("Create User", "Create user and set initial password.",
          "<label for='tenantId'>Tenant ID</label><input id='tenantId' placeholder='Select tenant or paste ID'>" +
          "<label for='userEmail'>Email</label><input id='userEmail' placeholder='owner@acme.com'>" +
          "<label for='userName'>Display Name</label><input id='userName' placeholder='Jane Owner'>" +
          "<label for='userPassword'>Initial Password</label><input id='userPassword' type='password' placeholder='password123' value='password123'>" +
          "<div class='row' style='margin-top:10px;margin-bottom:0;'><button id='createUser' type='button'>Create User</button><button id='loadUsers' class='secondary' type='button'>Load Users</button></div>" +
          "<div id='userStatus' class='status'></div>") + "</div>" +
        "<div class='span-8'>" + panelComponent("Users", "Users for the selected tenant.",
          "<div class='meta'><strong>User Directory</strong><div id='selectedTenant' class='selected' style='display:none;'></div></div>" +
          "<div class='row' style='margin-top:0;margin-bottom:10px;'><input id='userSearch' style='width:240px;' placeholder='Search email or name'></div>" +
          "<div class='table-shell'><div class='table-wrap'><table><thead><tr><th>ID</th><th>Email</th><th>Status</th><th>Name</th></tr></thead><tbody id='userRows'></tbody></table></div></div>") + "</div>" +
      "</div>";
    }

    function renderPage() {
      const host = document.getElementById("pageHost");
      const renderer = {
        overview: overviewPage,
        session: sessionPage,
        tenants: tenantsPage,
        users: usersPage
      }[currentPage] || overviewPage;
      host.innerHTML = renderer();

      const meta = pageMeta[currentPage] || pageMeta.overview;
      document.getElementById("pageTitle").textContent = meta.title;
      document.getElementById("pageSubtitle").textContent = meta.subtitle;

      applyStatusState();
      syncPageState();
      bindPageActions();
      renderTenants();
      renderUsers();
      renderOverviewTables();
      renderSelectedTenant();
    }

    function setActiveNav(page) {
      if (!showAdvanced && page === "session") {
        page = "overview";
      }
      currentPage = page;
      document.querySelectorAll(".nav-item").forEach(function (button) {
        button.classList.toggle("active", button.getAttribute("data-page") === page);
      });
      renderPage();
    }

    function setAdvancedMode(enabled) {
      showAdvanced = enabled;
      const sessionNav = document.querySelector(".nav-item[data-page='session']");
      if (sessionNav) {
        sessionNav.style.display = enabled ? "block" : "none";
      }
      const toggle = document.getElementById("toggleAdvanced");
      if (toggle) {
        toggle.textContent = enabled ? "Hide Advanced" : "Show Advanced";
      }
      if (!enabled && currentPage === "session") {
        setActiveNav("overview");
      }
    }

    function esc(value) {
      return String(value || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
    }

    function setStatus(id, message, ok) {
      statusState[id] = { msg: message, ok: ok };
      const el = document.getElementById(id);
      if (!el) return;
      el.className = "status " + (ok ? "ok" : "err");
      el.textContent = message;
    }

    function applyStatusState() {
      Object.keys(statusState).forEach(function (id) {
        const data = statusState[id];
        const el = document.getElementById(id);
        if (!el) return;
        el.className = "status " + (data.ok ? "ok" : "err");
        el.textContent = data.msg;
      });
    }

    function updateConnectionChip(connected) {
      const chip = document.getElementById("apiState");
      chip.className = "chip connected";
      chip.innerHTML = "<span class='chip-dot'></span>" + (connected ? "UI Session + Key" : "UI Session");
    }

    function setSelectedTenant(id, rerender) {
      selectedTenantId = id || "";
      if (rerender) {
        renderPage();
        return;
      }
      renderSelectedTenant();
      syncPageState();
      syncOverviewStats();
    }

    function renderSelectedTenant() {
      const tag = document.getElementById("selectedTenant");
      if (!tag) return;
      if (!selectedTenantId) {
        tag.style.display = "none";
        tag.textContent = "";
        return;
      }
      const tenant = tenants.find(function (t) { return t.id === selectedTenantId; });
      const label = tenant ? (tenant.slug + " (" + tenant.id + ")") : selectedTenantId;
      tag.textContent = "Tenant: " + label;
      tag.style.display = "inline-block";
    }

    function syncPageState() {
      const keyInput = document.getElementById("adminKey");
      if (keyInput) {
        keyInput.value = adminKey;
      }
      const tenantInput = document.getElementById("tenantId");
      if (tenantInput && selectedTenantId) {
        tenantInput.value = selectedTenantId;
      }
      syncOverviewStats();
      updateConnectionChip(Boolean(adminKey));
    }

    function syncOverviewStats() {
      const tenantCount = document.getElementById("statTenantCount");
      const userCount = document.getElementById("statUserCount");
      const tenantContext = document.getElementById("statTenantContext");
      if (tenantCount) tenantCount.textContent = String(tenants.length);
      if (userCount) userCount.textContent = String(users.length);
      if (tenantContext) tenantContext.textContent = selectedTenantId ? "1" : "0";
    }

    function renderTenants() {
      const tbody = document.getElementById("tenantRows");
      if (!tbody) return;
      const search = document.getElementById("tenantSearch");
      const query = search ? search.value.trim().toLowerCase() : "";
      const rows = tenants.filter(function (t) {
        if (!query) return true;
        return String(t.slug || "").toLowerCase().includes(query) || String(t.name || "").toLowerCase().includes(query);
      }).map(function (t) {
        const statusClass = String(t.status || "").toLowerCase() === "active" ? "active" : "pending";
        return "<tr>" +
          "<td><code>" + esc(t.id) + "</code></td>" +
          "<td>" + esc(t.slug) + "</td>" +
          "<td>" + esc(t.name) + "</td>" +
          "<td><span class='status-pill " + statusClass + "'>" + esc(t.status || "unknown") + "</span></td>" +
          "<td><div class='table-actions'><button class='small ghost select-tenant' data-tenant-id='" + esc(t.id) + "' type='button'>Select</button></div></td>" +
          "</tr>";
      }).join("");
      tbody.innerHTML = rows || "<tr><td colspan='5'>No matching tenants.</td></tr>";
    }

    function renderUsers() {
      const tbody = document.getElementById("userRows");
      if (!tbody) return;
      const search = document.getElementById("userSearch");
      const query = search ? search.value.trim().toLowerCase() : "";
      const rows = users.filter(function (u) {
        if (!query) return true;
        return String(u.email || "").toLowerCase().includes(query) || String(u.display_name || "").toLowerCase().includes(query);
      }).map(function (u) {
        const statusClass = String(u.status || "").toLowerCase() === "active" ? "active" : "pending";
        return "<tr>" +
          "<td><code>" + esc(u.id) + "</code></td>" +
          "<td>" + esc(u.email) + "</td>" +
          "<td><span class='status-pill " + statusClass + "'>" + esc(u.status || "unknown") + "</span></td>" +
          "<td>" + esc(u.display_name || "") + "</td>" +
          "</tr>";
      }).join("");
      tbody.innerHTML = rows || "<tr><td colspan='4'>No matching users.</td></tr>";
    }

    function renderOverviewTables() {
      const tenantRows = (tenants || []).slice(0, 6).map(function (t) {
        const statusClass = String(t.status || "").toLowerCase() === "active" ? "active" : "pending";
        return "<tr><td>" + esc(t.slug) + "</td><td>" + esc(t.name) + "</td><td><span class='status-pill " + statusClass + "'>" + esc(t.status || "unknown") + "</span></td></tr>";
      }).join("");
      const userRows = (users || []).slice(0, 6).map(function (u) {
        const statusClass = String(u.status || "").toLowerCase() === "active" ? "active" : "pending";
        return "<tr><td>" + esc(u.email) + "</td><td><span class='status-pill " + statusClass + "'>" + esc(u.status || "unknown") + "</span></td><td>" + esc(u.display_name || "") + "</td></tr>";
      }).join("");
      const tenantHost = document.getElementById("overviewTenantRows");
      const userHost = document.getElementById("overviewUserRows");
      if (tenantHost) {
        tenantHost.innerHTML = tenantRows || "<tr><td colspan='3'>No tenants loaded.</td></tr>";
      }
      if (userHost) {
        userHost.innerHTML = userRows || "<tr><td colspan='3'>No users loaded.</td></tr>";
      }
    }

    function saveKey() {
      const keyInput = document.getElementById("adminKey");
      if (!keyInput) return;
      adminKey = keyInput.value.trim();
      localStorage.setItem("locky_admin_key", adminKey);
      updateConnectionChip(Boolean(adminKey));
      setStatus("sessionStatus", adminKey ? "Admin key saved locally (optional)." : "Local key cleared. UI session still works.", true);
    }

    function clearKey() {
      const keyInput = document.getElementById("adminKey");
      if (!keyInput) return;
      keyInput.value = "";
      saveKey();
    }

    async function api(path, method, body) {
      const headers = {
        "Content-Type": "application/json"
      };
      if (adminKey) {
        headers["X-Admin-Key"] = adminKey;
      }

      const response = await fetch(path, {
        method: method || "GET",
        credentials: "same-origin",
        headers: headers,
        body: body ? JSON.stringify(body) : undefined
      });
      const text = await response.text();
      let data = {};
      try { data = text ? JSON.parse(text) : {}; } catch (_err) {}
      if (!response.ok) throw new Error(data.message || ("HTTP " + response.status));
      return data;
    }

    async function loadTenants() {
      try {
        const data = await api("/admin/tenants");
        tenants = data.tenants || [];
        renderTenants();
        renderOverviewTables();
        syncOverviewStats();
        if (selectedTenantId && !tenants.some(function (t) { return t.id === selectedTenantId; })) {
          setSelectedTenant("", false);
        }
        setStatus("tenantStatus", "Loaded " + tenants.length + " tenants.", true);
      } catch (err) {
        setStatus("tenantStatus", err.message || "Failed to load tenants.", false);
      }
    }

    async function createTenant() {
      const slug = document.getElementById("tenantSlug").value.trim();
      const name = document.getElementById("tenantName").value.trim();
      if (!slug || !name) {
        setStatus("tenantStatus", "Slug and name are required.", false);
        return;
      }
      try {
        const tenant = await api("/admin/tenants", "POST", { slug: slug, name: name });
        const slugInput = document.getElementById("tenantSlug");
        const nameInput = document.getElementById("tenantName");
        if (slugInput) slugInput.value = "";
        if (nameInput) nameInput.value = "";
        setSelectedTenant(tenant.id || "", false);
        setStatus("tenantStatus", "Created tenant " + slug + ".", true);
        await loadTenants();
      } catch (err) {
        setStatus("tenantStatus", err.message || "Failed to create tenant.", false);
      }
    }

    async function loadUsers() {
      const tenantInput = document.getElementById("tenantId");
      const tenantId = tenantInput ? tenantInput.value.trim() : selectedTenantId;
      if (!tenantId) {
        setStatus("userStatus", "Tenant ID is required.", false);
        return;
      }
      try {
        const data = await api("/admin/tenants/" + tenantId + "/users");
        users = data.users || [];
        setSelectedTenant(tenantId, false);
        renderUsers();
        renderOverviewTables();
        syncOverviewStats();
        setStatus("userStatus", "Loaded " + users.length + " users.", true);
      } catch (err) {
        setStatus("userStatus", err.message || "Failed to load users.", false);
      }
    }

    async function createUser() {
      const tenantInput = document.getElementById("tenantId");
      const emailInput = document.getElementById("userEmail");
      const nameInput = document.getElementById("userName");
      const passwordInput = document.getElementById("userPassword");
      const tenantId = tenantInput ? tenantInput.value.trim() : "";
      const email = emailInput ? emailInput.value.trim() : "";
      const displayName = nameInput ? nameInput.value.trim() : "";
      const password = passwordInput ? passwordInput.value : "";

      if (!tenantId || !email || !password) {
        setStatus("userStatus", "Tenant ID, email, and password are required.", false);
        return;
      }

      try {
        const user = await api("/admin/tenants/" + tenantId + "/users", "POST", {
          email: email,
          display_name: displayName
        });
        await api("/admin/tenants/" + tenantId + "/users/" + user.id + "/password", "PUT", {
          password: password
        });
        if (emailInput) emailInput.value = "";
        if (nameInput) nameInput.value = "";
        setStatus("userStatus", "Created user " + email + ".", true);
        await loadUsers();
      } catch (err) {
        setStatus("userStatus", err.message || "Failed to create user.", false);
      }
    }

    async function refreshAll() {
      await loadTenants();
      if (selectedTenantId) {
        const tenantInput = document.getElementById("tenantId");
        if (tenantInput && !tenantInput.value.trim()) {
          tenantInput.value = selectedTenantId;
        }
        await loadUsers();
      }
    }

    function bindPageActions() {
      const saveKeyButton = document.getElementById("saveKey");
      if (saveKeyButton) saveKeyButton.addEventListener("click", saveKey);
      const clearKeyButton = document.getElementById("clearKey");
      if (clearKeyButton) clearKeyButton.addEventListener("click", clearKey);
      const toggleKeyButton = document.getElementById("toggleKey");
      if (toggleKeyButton) {
        toggleKeyButton.addEventListener("click", function () {
          const keyInput = document.getElementById("adminKey");
          if (!keyInput) return;
          const isMasked = keyInput.type === "password";
          keyInput.type = isMasked ? "text" : "password";
          this.textContent = isMasked ? "Hide" : "Show";
        });
      }

      const createTenantButton = document.getElementById("createTenant");
      if (createTenantButton) createTenantButton.addEventListener("click", createTenant);
      const refreshTenantsButton = document.getElementById("refreshTenants");
      if (refreshTenantsButton) refreshTenantsButton.addEventListener("click", loadTenants);
      const tenantSearch = document.getElementById("tenantSearch");
      if (tenantSearch) tenantSearch.addEventListener("input", renderTenants);

      const createUserButton = document.getElementById("createUser");
      if (createUserButton) createUserButton.addEventListener("click", createUser);
      const loadUsersButton = document.getElementById("loadUsers");
      if (loadUsersButton) loadUsersButton.addEventListener("click", loadUsers);
      const userSearch = document.getElementById("userSearch");
      if (userSearch) userSearch.addEventListener("input", renderUsers);

      const overviewLoadTenants = document.getElementById("overviewLoadTenants");
      if (overviewLoadTenants) overviewLoadTenants.addEventListener("click", loadTenants);
      const overviewLoadUsers = document.getElementById("overviewLoadUsers");
      if (overviewLoadUsers) overviewLoadUsers.addEventListener("click", loadUsers);
      const overviewGoSession = document.getElementById("overviewGoSession");
      if (overviewGoSession) {
        overviewGoSession.addEventListener("click", function () {
          setActiveNav("session");
        });
      }

      const sessionLoadTenants = document.getElementById("sessionLoadTenants");
      if (sessionLoadTenants) sessionLoadTenants.addEventListener("click", loadTenants);
      const sessionToTenants = document.getElementById("sessionToTenants");
      if (sessionToTenants) {
        sessionToTenants.addEventListener("click", function () {
          setActiveNav("tenants");
        });
      }
    }

    updateConnectionChip(Boolean(adminKey));
    setStatus("sessionStatus", adminKey ? "Optional API key detected in local storage." : "UI session is active. API key is optional.", true);
    document.getElementById("refreshAll").addEventListener("click", refreshAll);
    document.getElementById("toggleAdvanced").addEventListener("click", function () {
      setAdvancedMode(!showAdvanced);
      if (showAdvanced) {
        setActiveNav("session");
      }
    });

    document.getElementById("nav").addEventListener("click", function (event) {
      const button = event.target.closest(".nav-item");
      if (!button) return;
      const page = button.getAttribute("data-page") || "overview";
      setActiveNav(page);
    });

    document.getElementById("pageHost").addEventListener("click", function (event) {
      const button = event.target.closest(".select-tenant");
      if (!button) return;
      const tenantId = button.getAttribute("data-tenant-id") || "";
      setSelectedTenant(tenantId, false);
      setStatus("userStatus", "Selected tenant. Open Users page to load members.", true);
    });

    renderPage();
    setAdvancedMode(false);

    loadTenants();
  </script>
</body>
</html>`

func (s *Server) handleAdminUI(w http.ResponseWriter, r *http.Request) {
	if !s.isAdminUIAuthenticated(r) {
		s.renderAdminUILogin(w, "")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     adminUICookieName,
		Value:    s.adminUISessionToken(),
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   8 * 60 * 60,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(adminUIHTML))
}

func (s *Server) handleAdminUILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderAdminUILogin(w, "Invalid login form")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username != s.config.AdminUIUsername || password != s.config.AdminUIPassword {
		s.renderAdminUILogin(w, "Invalid username or password")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     adminUICookieName,
		Value:    s.adminUISessionToken(),
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   8 * 60 * 60,
	})
	http.Redirect(w, r, "/admin/ui", http.StatusFound)
}

func (s *Server) handleAdminUILogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminUICookieName,
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     adminUICookieName,
		Value:    "",
		Path:     "/admin/ui",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/admin/ui", http.StatusFound)
}

func (s *Server) isAdminUIAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(adminUICookieName)
	if err != nil {
		return false
	}
	return cookie.Value == s.adminUISessionToken()
}

func (s *Server) adminUISessionToken() string {
	sum := sha256.Sum256([]byte(s.config.AdminUIUsername + ":" + s.config.AdminUIPassword + ":" + s.config.AdminAPIKey))
	return hex.EncodeToString(sum[:])
}

func (s *Server) renderAdminUILogin(w http.ResponseWriter, errMsg string) {
	errHTML := ""
	if errMsg != "" {
		errHTML = `<div class="err">` + errMsg + `</div>`
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(strings.Replace(adminUILoginHTML, "%s", errHTML, 1)))
}
