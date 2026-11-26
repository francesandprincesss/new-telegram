const tokenKey = "grpc_chat_access_token";
let eventController = null;

const endpointConfig = {
  register: { method: "POST", path: "/api/v1/auth/register", body: f => ({ username: f.username, password: f.password }), savesToken: true },
  login: { method: "POST", path: "/api/v1/auth/login", body: f => ({ username: f.username, password: f.password }), savesToken: true },
  getMe: { method: "GET", path: "/api/v1/users/me", auth: true },
  getUser: { method: "GET", path: f => `/api/v1/users/${encodeURIComponent(f.user_id)}`, auth: true },
  findUser: { method: "GET", path: f => `/api/v1/users/find?username=${encodeURIComponent(f.username)}`, auth: true },
  setStatus: { method: "PUT", path: "/api/v1/presence/status", body: f => ({ status: f.status }), auth: true }
};

function getToken() { return sessionStorage.getItem(tokenKey) || ""; }
function setToken(token, username) {
  sessionStorage.setItem(tokenKey, token);
  if (username) sessionStorage.setItem("grpc_chat_username", username);
  updateAuthState();
}
function updateAuthState() {
  const token = getToken();
  const label = document.querySelector("#auth-label");
  const logout = document.querySelector("#logout");
  label.textContent = token ? `JWT · ${sessionStorage.getItem("grpc_chat_username") || "авторизован"}` : "Без авторизации";
  logout.hidden = !token;
}
function formValues(form) { return Object.fromEntries(new FormData(form).entries()); }
function toast(message) {
  const node = document.querySelector("#toast");
  node.textContent = message;
  node.classList.add("show");
  window.setTimeout(() => node.classList.remove("show"), 2200);
}
function pretty(value) { return JSON.stringify(value, null, 2); }

async function executeEndpoint(article, form) {
  const config = endpointConfig[article.dataset.endpoint];
  const values = formValues(form);
  const path = typeof config.path === "function" ? config.path(values) : config.path;
  const headers = { Accept: "application/json" };
  if (config.body) headers["Content-Type"] = "application/json";
  if (config.auth) {
    if (!getToken()) { toast("Сначала выполните Register или Login"); return; }
    headers.Authorization = `Bearer ${getToken()}`;
  }
  const button = form.querySelector("button[type=submit]");
  const responseBox = article.querySelector(".response");
  const meta = responseBox.querySelector(".response-meta");
  const output = responseBox.querySelector("pre");
  button.disabled = true;
  meta.className = "response-meta";
  meta.textContent = `${config.method} ${path} · выполняется…`;
  const started = performance.now();
  try {
    const response = await fetch(path, { method: config.method, headers, body: config.body ? JSON.stringify(config.body(values)) : undefined });
    const data = await response.json().catch(() => ({ error: "Ответ не является JSON" }));
    const elapsed = Math.round(performance.now() - started);
    meta.textContent = `${response.status} ${response.statusText} · ${elapsed} ms`;
    meta.classList.add(response.ok ? "success" : "failure");
    output.textContent = pretty(data);
    if (response.ok && config.savesToken && data.access_token) {
      setToken(data.access_token, data.user?.username || values.username);
      const userIDInput = document.querySelector('[data-endpoint="getUser"] input[name="user_id"]');
      if (userIDInput && data.user?.id) userIDInput.value = data.user.id;
      toast("JWT сохранён для этой вкладки");
    }
  } catch (error) {
    meta.textContent = "Сетевая ошибка";
    meta.classList.add("failure");
    output.textContent = pretty({ error: error.message });
  } finally { button.disabled = false; }
}

async function connectEvents() {
  const button = document.querySelector("#connect-events");
  const status = document.querySelector(".console-status");
  const statusText = document.querySelector("#event-status");
  if (eventController) {
    eventController.abort(); button.disabled = true;
    return;
  }
  if (!getToken()) { toast("Сначала выполните Register или Login"); return; }
  const controller = new AbortController();
  eventController = controller;
  button.textContent = "Отключиться"; statusText.textContent = "Подключение…";
  try {
    const response = await fetch("/api/v1/events", { headers: { Authorization: `Bearer ${getToken()}`, Accept: "text/event-stream" }, signal: controller.signal });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    status.classList.add("live"); statusText.textContent = "Подключено · presence активен";
    document.querySelector("#event-log .empty")?.remove();
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const blocks = buffer.split("\n\n"); buffer = blocks.pop();
      blocks.forEach(renderEventBlock);
    }
  } catch (error) {
    if (error.name !== "AbortError") toast(`Event stream: ${error.message}`);
  } finally {
    if (eventController === controller) {
      eventController = null; button.disabled = false; button.textContent = "Подключиться"; status.classList.remove("live"); statusText.textContent = "Отключено";
    }
  }
}

function renderEventBlock(block) {
  if (!block || block.startsWith(":")) return;
  let type = "event", data = null;
  for (const line of block.split("\n")) {
    if (line.startsWith("event:")) type = line.slice(6).trim();
    if (line.startsWith("data:")) { try { data = JSON.parse(line.slice(5).trim()); } catch { data = line.slice(5).trim(); } }
  }
  if (!data) return;
  const item = document.createElement("div"); item.className = "event-item";
  const time = document.createElement("time"); time.textContent = new Date().toLocaleTimeString();
  const content = document.createElement("div");
  const label = document.createElement("div"); label.className = "event-type"; label.textContent = type;
  const pre = document.createElement("pre"); pre.textContent = pretty(data);
  content.append(label, pre); item.append(time, content);
  const log = document.querySelector("#event-log"); log.append(item); log.scrollTop = log.scrollHeight;
}

document.querySelectorAll("article.endpoint").forEach(article => article.querySelector("form")?.addEventListener("submit", event => { event.preventDefault(); executeEndpoint(article, event.currentTarget); }));
document.querySelector("#connect-events").addEventListener("click", connectEvents);
document.querySelector("#clear-events").addEventListener("click", () => { document.querySelector("#event-log").innerHTML = '<div class="empty">Лог очищен.</div>'; });
document.querySelector("#logout").addEventListener("click", () => { if (eventController) eventController.abort(); sessionStorage.removeItem(tokenKey); sessionStorage.removeItem("grpc_chat_username"); updateAuthState(); toast("JWT удалён"); });
document.querySelector("#http-address").textContent = window.location.host;
updateAuthState();

const sections = [...document.querySelectorAll("main [id]")];
const links = [...document.querySelectorAll(".sidebar a[href^='#']")];
const observer = new IntersectionObserver(entries => { const visible = entries.filter(e => e.isIntersecting).sort((a,b) => b.intersectionRatio-a.intersectionRatio)[0]; if (!visible) return; links.forEach(link => link.classList.toggle("active", link.getAttribute("href") === `#${visible.target.id}`)); }, { rootMargin: "-90px 0px -65%", threshold: [0,.2,.5] });
sections.forEach(section => observer.observe(section));
