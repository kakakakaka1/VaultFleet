package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const frontendPlaceholderHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>VaultFleet</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f5f7fb; color: #182033; }
    * { box-sizing: border-box; }
    body { margin: 0; min-height: 100vh; background: #f5f7fb; }
    button, input, textarea, select { font: inherit; }
    button { border: 0; background: #2257d8; color: #fff; padding: 8px 12px; border-radius: 6px; cursor: pointer; }
    button.secondary { background: #eef2f7; color: #182033; border: 1px solid #d8e0ed; }
    button.danger { background: #c83232; }
    button:disabled { opacity: .55; cursor: not-allowed; }
    input, textarea, select { width: 100%; border: 1px solid #cfd8e6; border-radius: 6px; padding: 8px 10px; background: #fff; color: #182033; }
    textarea { min-height: 84px; resize: vertical; }
    .shell { min-height: 100vh; display: grid; grid-template-columns: 240px minmax(0, 1fr); }
    .sidebar { background: #101827; color: #f8fafc; padding: 20px 16px; }
    .brand { font-size: 20px; font-weight: 700; margin-bottom: 24px; }
    .nav { display: grid; gap: 8px; }
    .nav button { width: 100%; text-align: left; background: transparent; color: #dbe4f0; border: 1px solid transparent; }
    .nav button.active { background: #22314a; border-color: #3b4d6a; color: #fff; }
    .content { padding: 24px; }
    .panel { background: #fff; border: 1px solid #dfe6f1; border-radius: 8px; padding: 18px; box-shadow: 0 1px 2px rgba(16, 24, 39, .04); margin-bottom: 18px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; }
    .metric { border: 1px solid #dfe6f1; border-radius: 8px; padding: 14px; background: #fbfcff; }
    .metric strong { display: block; font-size: 28px; margin-top: 4px; }
    .form { display: grid; gap: 12px; max-width: 460px; }
    .row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
    table { width: 100%; border-collapse: collapse; }
    th, td { text-align: left; border-bottom: 1px solid #e5ebf4; padding: 10px 8px; vertical-align: top; }
    th { color: #5b667a; font-size: 13px; font-weight: 600; }
    .status { display: inline-flex; align-items: center; gap: 6px; border-radius: 999px; padding: 3px 8px; font-size: 12px; background: #eef2f7; color: #42506a; }
    .status.online { background: #e7f8ef; color: #11683b; }
    .status.offline { background: #fff0f0; color: #a62828; }
    .muted { color: #657187; }
    .error { color: #b42318; white-space: pre-wrap; }
    .success { color: #14743b; }
    .split { display: grid; grid-template-columns: minmax(0, 1fr) minmax(280px, 420px); gap: 18px; }
    .hidden { display: none !important; }
    @media (max-width: 760px) { .shell { grid-template-columns: 1fr; } .sidebar { position: static; } .split { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <div id="app-root" class="shell">
    <aside class="sidebar">
      <div class="brand">VaultFleet</div>
      <nav class="nav">
        <button id="nav-dashboard" class="active" type="button">Dashboard</button>
        <button id="nav-nodes" type="button">Nodes</button>
        <button id="nav-storage" type="button">Storage</button>
        <button id="nav-policies" type="button">Policies</button>
        <button id="nav-notifications" type="button">Notifications</button>
      </nav>
    </aside>
    <main class="content">
      <section id="auth-panel" class="panel">
        <h1>VaultFleet</h1>
        <p id="auth-mode" class="muted">Checking system state...</p>
        <form id="auth-form" class="form">
          <label>Username<input name="username" autocomplete="username" required></label>
          <label>Password<input name="password" type="password" autocomplete="current-password" required></label>
          <button id="auth-submit" type="submit">Submit</button>
          <div id="auth-error" class="error"></div>
        </form>
      </section>
      <section id="main-panel" class="hidden">
        <div class="panel">
          <div class="row">
            <h1 style="margin-right:auto">Dashboard</h1>
            <button id="refresh" class="secondary" type="button">Refresh</button>
          </div>
          <div class="grid">
            <div class="metric">Online Nodes<strong id="online-count">0</strong></div>
            <div class="metric">Offline Nodes<strong id="offline-count">0</strong></div>
            <div class="metric">Policies<strong id="policy-count">0</strong></div>
            <div class="metric">Recent Tasks<strong id="task-count">0</strong></div>
          </div>
        </div>
        <div id="nodes-view" class="panel">
          <div class="row">
            <h2 style="margin-right:auto">Nodes</h2>
            <input id="new-agent-name" placeholder="Node name" style="max-width:260px">
            <button id="create-agent" type="button">Add Node</button>
          </div>
          <table>
            <thead><tr><th>Name</th><th>Status</th><th>Last Seen</th><th>System</th><th>Actions</th></tr></thead>
            <tbody id="agents-body"></tbody>
          </table>
        </div>
        <div id="detail-view" class="panel hidden">
          <div class="row">
            <h2 id="detail-title" style="margin-right:auto">Node Detail</h2>
            <button id="backup-now" type="button">Backup Now</button>
            <button id="browse-files" class="secondary" type="button">Browse Files</button>
          </div>
          <div id="detail-status" class="muted"></div>
          <h3>Backup History</h3>
          <table>
            <thead><tr><th>Type</th><th>Status</th><th>Snapshot</th><th>Duration</th><th>Error</th></tr></thead>
            <tbody id="tasks-body"></tbody>
          </table>
          <h3>Snapshots</h3>
          <table>
            <thead><tr><th>Snapshot</th><th>Time</th><th>Paths</th><th>Size</th></tr></thead>
            <tbody id="snapshots-body"></tbody>
          </table>
          <h3>File Browser</h3>
          <div class="row">
            <input id="browse-path" value="/" style="max-width:360px">
            <button id="browse-run" class="secondary" type="button">Open</button>
          </div>
          <div id="browse-output" class="muted"></div>
        </div>
        <div class="split">
          <div class="panel">
            <h2>Storage</h2>
            <div id="storage-list" class="muted"></div>
          </div>
          <div class="panel">
            <h2>Policies</h2>
            <div id="policy-list" class="muted"></div>
          </div>
        </div>
      </section>
    </main>
  </div>
  <script>
    const state = { initialized: false, agents: [], policies: [], tasks: [], snapshots: [], currentAgentId: "" };
    state.currentAgentId = routeAgentID();
    const $ = (id) => document.getElementById(id);
    async function api(path, options = {}) {
      const res = await fetch(path, { credentials: "same-origin", headers: { "Content-Type": "application/json", ...(options.headers || {}) }, ...options });
      const text = await res.text();
      let body = null;
      try { body = text ? JSON.parse(text) : null; } catch { body = text; }
      if (!res.ok) throw new Error((body && body.error) || text || res.statusText);
      return body;
    }
    function unwrap(value) { return value && value.ok === true && value.data !== undefined ? value.data : value; }
    function routeAgentID() {
      const match = window.location.pathname.match(/^\/nodes\/([^/]+)$/);
      return match ? decodeURIComponent(match[1]) : "";
    }
    function updateNodeRoute(agentID) {
      const path = "/nodes/" + encodeURIComponent(agentID);
      if (window.location.pathname !== path) history.pushState(null, "", path);
    }
    async function checkAuth() {
      const body = await api("/api/auth/check");
      const auth = body.data || {};
      state.initialized = !!auth.initialized;
      if (!state.initialized) {
        $("auth-mode").textContent = "Initialize administrator";
        $("auth-submit").textContent = "Initialize";
        showAuth();
        return;
      }
      $("auth-mode").textContent = "Login";
      $("auth-submit").textContent = "Login";
      if (!auth.authenticated) { showAuth(); return; }
      await loadAll();
      showMain();
    }
    function showMain() {
      $("auth-panel").classList.add("hidden");
      $("main-panel").classList.remove("hidden");
    }
    function showAuth() {
      $("auth-panel").classList.remove("hidden");
      $("main-panel").classList.add("hidden");
    }
    async function submitAuth(event) {
      event.preventDefault();
      $("auth-error").textContent = "";
      const form = new FormData(event.currentTarget);
      const payload = { username: form.get("username"), password: form.get("password") };
      try {
        await api(state.initialized ? "/api/auth/login" : "/api/auth/init", { method: "POST", body: JSON.stringify(payload) });
        state.initialized = true;
        await loadAll();
        showMain();
      } catch (err) {
        $("auth-error").textContent = err.message;
      }
    }
    async function loadAll() {
      const [agents, policies, tasks] = await Promise.all([
        api("/api/agents").then(unwrap),
        api("/api/policies").then(unwrap).catch(() => []),
        api("/api/tasks?limit=20").then(unwrap).catch(() => []),
      ]);
      state.agents = Array.isArray(agents) ? agents : [];
      state.policies = Array.isArray(policies) ? policies : [];
      state.tasks = Array.isArray(tasks) ? tasks : [];
      render();
      if (state.currentAgentId) await loadAgentDetail(state.currentAgentId);
    }
    function render() {
      const online = state.agents.filter(a => a.status === "online").length;
      $("online-count").textContent = online;
      $("offline-count").textContent = state.agents.length - online;
      $("policy-count").textContent = state.policies.length;
      $("task-count").textContent = state.tasks.length;
      $("agents-body").innerHTML = state.agents.map(agent => {
        const status = agent.status || "offline";
        return "<tr><td>" + escapeHTML(agent.name) + "</td><td><span class=\"status " + status + "\">" + status + "</span></td><td>" + escapeHTML(agent.last_seen_at || "-") + "</td><td>" + escapeHTML(agent.system_info || "-") + "</td><td><button class=\"secondary\" data-agent=\"" + agent.id + "\">Details</button></td></tr>";
      }).join("");
      $("storage-list").textContent = "Use /api/storage to configure MinIO or WebDAV storage.";
      $("policy-list").innerHTML = state.policies.length ? state.policies.map(p => "<div>" + escapeHTML(p.agent_id) + " -> " + escapeHTML(p.repo_path || "") + " synced=" + p.synced + "</div>").join("") : "No policies";
      document.querySelectorAll("[data-agent]").forEach(btn => btn.addEventListener("click", () => loadAgentDetail(btn.dataset.agent)));
    }
    async function createAgent() {
      const name = $("new-agent-name").value.trim();
      if (!name) return;
      const created = unwrap(await api("/api/agents", { method: "POST", body: JSON.stringify({ name }) }));
      $("new-agent-name").value = "";
      await loadAll();
      alert("Enrollment token: " + created.enroll_token);
    }
    async function loadAgentDetail(agentID) {
      state.currentAgentId = agentID;
      updateNodeRoute(agentID);
      const agent = state.agents.find(a => a.id === agentID) || unwrap(await api("/api/agents/" + encodeURIComponent(agentID)));
      const [tasks, snapshots] = await Promise.all([
        api("/api/tasks?agent_id=" + encodeURIComponent(agentID) + "&limit=20").then(unwrap).catch(() => []),
        api("/api/agents/" + encodeURIComponent(agentID) + "/snapshots").then(unwrap).catch(() => []),
      ]);
      $("detail-view").classList.remove("hidden");
      $("detail-title").textContent = agent.name || "Node Detail";
      $("detail-status").textContent = "Status: " + (agent.status || "offline");
      $("tasks-body").innerHTML = tasks.map(t => "<tr><td>" + t.type + "</td><td>" + t.status + "</td><td>" + escapeHTML(t.snapshot_id || "") + "</td><td>" + t.duration_ms + "ms</td><td>" + escapeHTML(t.error_log || "") + "</td></tr>").join("");
      $("snapshots-body").innerHTML = snapshots.map(s => "<tr><td>" + escapeHTML(s.snapshot_id || s.id || "") + "</td><td>" + escapeHTML(s.timestamp || "") + "</td><td>" + escapeHTML((s.paths || []).join(", ")) + "</td><td>" + (s.size || 0) + "</td></tr>").join("");
    }
    async function backupNow() {
      if (!state.currentAgentId) return;
      await api("/api/agents/" + encodeURIComponent(state.currentAgentId) + "/backup-now", { method: "POST", body: "{}" });
      await loadAgentDetail(state.currentAgentId);
    }
    async function browseFiles() {
      if (!state.currentAgentId) return;
      const path = $("browse-path").value || "/";
      const result = unwrap(await api("/api/agents/" + encodeURIComponent(state.currentAgentId) + "/browse", { method: "POST", body: JSON.stringify({ path, depth: 2 }) }));
      renderBrowse(result);
    }
    function renderBrowse(result) {
      const entries = Array.isArray(result.entries) ? result.entries : [];
      if (!entries.length) {
        $("browse-output").textContent = "No entries";
        return;
      }
      $("browse-output").innerHTML = "<table><thead><tr><th>Path</th><th>Type</th><th>Size</th></tr></thead><tbody>" + entries.map(entry => {
        const path = entry.path || "";
        const label = entry.type === "dir" ? "<button class=\"secondary\" data-browse-path=\"" + escapeHTMLAttr(path) + "\">" + escapeHTML(path) + "</button>" : escapeHTML(path);
        return "<tr><td>" + label + "</td><td>" + escapeHTML(entry.type || "") + "</td><td>" + (entry.size || 0) + "</td></tr>";
      }).join("") + "</tbody></table>";
      document.querySelectorAll("[data-browse-path]").forEach(btn => btn.addEventListener("click", () => {
        $("browse-path").value = btn.dataset.browsePath;
        browseFiles();
      }));
    }
    function escapeHTML(value) {
      return String(value ?? "").replace(/[&<>"']/g, ch => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;" }[ch]));
    }
    function escapeHTMLAttr(value) {
      return escapeHTML(value).replace(/` + "`" + `/g, "&#96;");
    }
    $("auth-form").addEventListener("submit", submitAuth);
    $("refresh").addEventListener("click", loadAll);
    $("create-agent").addEventListener("click", createAgent);
    $("backup-now").addEventListener("click", backupNow);
    $("browse-files").addEventListener("click", browseFiles);
    $("browse-run").addEventListener("click", browseFiles);
    setInterval(() => { if (!$("main-panel").classList.contains("hidden")) loadAll().catch(() => showAuth()); }, 10000);
    checkAuth().catch(err => { $("auth-error").textContent = err.message; showAuth(); });
  </script>
</body>
</html>`

func RegisterFrontendRoutes(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		if isBackendRoute(c.Request.URL.Path) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "not found"})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(frontendPlaceholderHTML))
	})
}

func isBackendRoute(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/") ||
		path == "/ws" || strings.HasPrefix(path, "/ws/")
}
