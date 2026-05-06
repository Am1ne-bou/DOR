package main

const webPage = `<!DOCTYPE html>
<html lang="fr" data-theme="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>DOR Node UI</title>
<style>
:root {
  --bg:#0d1117; --surface:#161b22; --surface2:#21262d; --border:#30363d;
  --text:#c9d1d9; --text-muted:#8b949e; --accent:#238636; --accent-hov:#2ea043;
  --blue:#1f6feb; --blue-hov:#388bfd; --red:#da3633; --red-dark:#b62324;
  --purple:#8957e5; --purple-hov:#a371f7;
  --radius-sm:6px; --radius-md:10px; --radius-full:9999px; --trans:150ms ease;
}
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
html{-webkit-font-smoothing:antialiased}
body{
  font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
  font-size:14px; background:var(--bg); color:var(--text);
  height:100vh; overflow:hidden; display:flex; flex-direction:column;
}
.app{display:flex; flex:1; min-height:0}
.sidebar{
  width:220px; min-width:220px; background:var(--surface);
  border-right:1px solid var(--border); display:flex; flex-direction:column; overflow:hidden;
}
.sidebar-header{
  padding:10px 12px; border-bottom:1px solid var(--border);
  display:flex; justify-content:space-between; align-items:center;
  font-weight:600; font-size:.8rem; letter-spacing:.04em;
  text-transform:uppercase; color:var(--text-muted); flex-shrink:0;
}
.quit-btn{
  background:var(--red-dark); color:#fff; border:none;
  border-radius:var(--radius-sm); padding:4px 10px;
  font-size:.75rem; font-weight:600; cursor:pointer; transition:background var(--trans);
}
.quit-btn:hover{background:var(--red)}
.nodes-list{flex:1; overflow-y:auto}
.inbox-item{
  display:flex; align-items:center; justify-content:space-between;
  padding:9px 12px; border-bottom:1px solid var(--border);
  cursor:pointer; font-size:.83rem; font-weight:600; transition:background var(--trans);
}
.inbox-item:hover{background:var(--surface2)}
.inbox-item.active{background:var(--blue); color:#fff}
.node-item{
  display:flex; align-items:center; justify-content:space-between;
  padding:9px 12px; border-bottom:1px solid var(--surface2);
  cursor:pointer; font-size:.83rem; transition:background var(--trans);
}
.node-item:hover{background:var(--surface2)}
.node-item.active{background:var(--blue); color:#fff}
.node-item.active .node-addr{color:#aac8ff}
.node-info{display:flex; flex-direction:column; gap:2px; min-width:0}
.node-name{font-weight:600; white-space:nowrap; overflow:hidden; text-overflow:ellipsis}
.node-addr{font-size:.72rem; color:var(--text-muted)}
.unread-badge{
  min-width:20px; height:20px; background:var(--red); color:#fff;
  border-radius:var(--radius-full); font-size:.68rem; font-weight:700;
  display:flex; align-items:center; justify-content:center; padding:0 5px; flex-shrink:0;
}
.node-item.active .unread-badge,.inbox-item.active .unread-badge{background:#fff; color:var(--blue)}
.main{flex:1; display:flex; flex-direction:column; min-width:0; overflow:hidden}
.chat-header{
  padding:8px 14px; background:var(--surface); border-bottom:1px solid var(--border);
  display:flex; justify-content:space-between; align-items:center; flex-shrink:0; z-index:10;
}
.chat-header-node{font-weight:600; font-size:.85rem; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;}
.mode-badge{
  display:inline-flex; align-items:center; gap:5px;
  padding:3px 10px; border-radius:var(--radius-full);
  font-size:.72rem; font-weight:700; white-space:nowrap; flex-shrink:0;
}
.mode-badge.anon{background:rgba(139,148,158,.18); color:var(--text-muted)}
.mode-badge.auth{background:rgba(35,134,54,.22); color:#7ee787}
.mode-badge.ssend{background:rgba(137,87,229,.22); color:#a371f7}
.video-feed-container{
  display:none; padding:10px; background:#000;
  border-bottom:1px solid var(--border); text-align:center;
}
#remote-video{max-height:180px; border-radius:8px; border:2px solid var(--accent)}
.video-label{font-size:.7rem; color:var(--text-muted); margin-bottom:5px}
.chat-body{flex:1; display:flex; flex-direction:column; min-height:0; overflow:hidden}
.messages{flex:1; overflow-y:auto; padding:10px 14px; display:flex; flex-direction:column; gap:6px}
.msg-row{display:flex; width:100%; align-items:flex-end; gap:6px}
.msg-row.me-row{justify-content:flex-end}
.msg-row.peer-row{justify-content:flex-start}
.msg-row.meta-row{justify-content:center}
.peer-wrap{display:flex; flex-direction:column; align-items:flex-start; max-width:68%}
.msg{
  padding:8px 12px; border-radius:16px; font-size:.83rem; line-height:1.45;
  word-wrap:break-word; word-break:break-word; white-space:pre-wrap; min-width:30px;
}
.msg.me{max-width:68%; background:var(--accent); color:#fff; border-bottom-right-radius:4px}
.msg.me.ssend-me{background:var(--purple)}
.msg.peer{background:var(--blue); color:#fff; border-bottom-left-radius:4px; width:100%}
.msg.auth-peer{background:#1a3a2a; border:1px solid var(--accent); color:#c9d1d9; border-bottom-left-radius:4px; width:100%}
.msg.meta{background:transparent; color:var(--text-muted); font-size:.72rem; text-align:center; max-width:90%}
.msg-from{font-size:.68rem; color:var(--text-muted); margin-bottom:3px; margin-left:4px}
.msg-status{font-size:.72rem; color:var(--text-muted); flex-shrink:0; padding-bottom:2px}
.logs-panel{
  height:100px; flex-shrink:0; border-top:1px solid var(--border);
  background:var(--bg); padding:4px 8px; overflow-y:auto;
  font-family:monospace; font-size:.7rem; color:var(--text-muted); line-height:1.4;
}
.chat-input{
  padding:7px 10px; border-top:1px solid var(--border);
  background:var(--surface); display:flex; gap:6px; align-items:center; flex-shrink:0;
}
.send-type-select,.relay-select,.mode-select{
  background:var(--bg); border:1px solid var(--border); border-radius:var(--radius-full);
  color:var(--text); font-size:.75rem; padding:5px 8px; cursor:pointer; flex-shrink:0;
}
.send-type-select{width:90px}
.relay-select{width:82px}
.mode-select{width:118px}
.mode-select.auth{border-color:var(--accent); color:#7ee787}
.send-type-select.ssend{border-color:var(--purple); color:#a371f7}
select:focus{border-color:var(--blue-hov); outline:none}
.ssend-group-wrap{display:none; align-items:center; gap:4px; flex-shrink:0}
.ssend-group-wrap.visible{display:flex}
.ssend-label{font-size:.72rem; color:#a371f7; white-space:nowrap; flex-shrink:0}
.ssend-input{width:52px; background:var(--bg); border:1px solid rgba(137,87,229,.4); border-radius:var(--radius-sm); color:var(--text); font-size:.78rem; padding:4px 6px; outline:none}
.ssend-input:focus{border-color:var(--purple)}
.input-wrapper{
  flex:1; display:flex; align-items:center; background:var(--bg);
  border:1px solid var(--border); border-radius:var(--radius-full); padding:5px 14px; gap:6px;
}
.input-wrapper:focus-within{border-color:var(--blue-hov)}
.input-wrapper input{flex:1; border:none; background:transparent; color:var(--text); font-size:.83rem; outline:none}
.target-hint{font-size:.7rem; color:var(--text-muted); white-space:nowrap; flex-shrink:0}
.send-btn{
  background:var(--accent); color:#fff; border:none; border-radius:var(--radius-full);
  padding:8px 16px; font-size:.83rem; font-weight:600; cursor:pointer; white-space:nowrap;
  transition:background var(--trans); flex-shrink:0;
}
.send-btn:hover{background:var(--accent-hov)}
.send-btn:disabled{background:var(--surface2); color:var(--text-muted); cursor:not-allowed}
.send-btn.ssend-active{background:var(--purple)}
.send-btn.ssend-active:hover{background:var(--purple-hov)}
.cam-btn{
  background:transparent; color:var(--text); border:1px solid var(--border);
  border-radius:var(--radius-full); padding:8px 12px; cursor:pointer; transition:var(--trans);
}
.cam-btn:hover{background:var(--surface2)}
.cam-btn.active{background:var(--red); color:#fff; border-color:var(--red)}
</style>
</head>
<body>
<div class="app">
  <div class="sidebar">
    <div class="sidebar-header">Nœuds <button class="quit-btn" onclick="quitNode()">⏻ Quit</button></div>
    <div class="nodes-list">
      <div id="inbox-item" class="inbox-item active" onclick="selectInbox()">
        <span>📬 Reçus</span><span></span>
      </div>
      <div id="nodes-list"></div>
    </div>
  </div>
  <div class="main">
    <div class="chat-header">
      <div class="chat-header-node" id="current-node-name">📬 Messages reçus</div>
      <div style="display:flex;align-items:center;gap:8px;flex-shrink:0">
        <span id="mode-indicator" class="mode-badge anon">🔒 Anonyme</span>
        <span style="font-size:.72rem;color:var(--text-muted)">Relais : <span id="current-relays">3</span></span>
      </div>
    </div>

    <div id="video-feed-container" class="video-feed-container">
      <div id="video-sender-name" class="video-label">Vidéo en direct</div>
      <img id="remote-video" src="" alt="En attente de flux..." />
    </div>

    <div class="chat-body">
      <div id="messages" class="messages"></div>
      <div id="logs" class="logs-panel"></div>
    </div>
    <div class="chat-input">
      <select id="send-type" class="send-type-select" onchange="onSendTypeChange()">
        <option value="send">📨 SEND</option>
        <option value="ssend">⚡ SSEND</option>
      </select>
      <select id="relay-count" class="relay-select" onchange="updateRelayLabel()">
        <option value="1">1 saut</option>
        <option value="2">2 sauts</option>
        <option value="3" selected>3 sauts</option>
        <option value="4">4 sauts</option>
      </select>
      <div class="ssend-group-wrap" id="ssend-group-wrap">
        <span class="ssend-label">Grp</span>
        <input id="ssend-group" class="ssend-input" type="number" min="1" max="20" value="3"
               title="group_size : nb de nœuds dans le super-groupe">
      </div>
      <select id="mode-select" class="mode-select" onchange="updateModeStyle()">
        <option value="anon">🔒 Anonyme</option>
        <option value="auth">🔐 Authentifié</option>
      </select>
      <div class="input-wrapper">
        <input id="msg-input" type="text" placeholder="Écrire un message…"
               onkeydown="if(event.key==='Enter'){event.preventDefault();sendMessage();}">
        <span class="target-hint" id="target-hint">aucun nœud</span>
      </div>
      <button id="cam-btn" class="cam-btn" onclick="toggleStream()" title="Diffuser la caméra" disabled>📷</button>
      <button class="send-btn" id="send-btn" onclick="sendMessage()" disabled>Envoyer ➤</button>
    </div>
  </div>
</div>

<video id="local-video" autoplay playsinline muted style="display:none;"></video>
<canvas id="video-canvas" style="display:none;"></canvas>

<script>
const INBOX = '__inbox__';
let selectedNode = null, selectedNodeKey = INBOX;
const conversations = {}, unread = {};
let msgCounter = 0;
let isStreaming = false, streamInterval = null;

// Ordre stable : knownNodes map + nodeOrder tableau d'insertion
const knownNodes = {};
const nodeOrder  = [];

const messagesDiv    = document.getElementById('messages');
const logsDiv        = document.getElementById('logs');
const nodesListDiv   = document.getElementById('nodes-list');
const nodeNameSpan   = document.getElementById('current-node-name');
const targetHintSpan = document.getElementById('target-hint');
const relaysSpan     = document.getElementById('current-relays');
const modeIndicator  = document.getElementById('mode-indicator');
const modeSelect     = document.getElementById('mode-select');
const sendBtn        = document.getElementById('send-btn');
const camBtn         = document.getElementById('cam-btn');
const inboxItem      = document.getElementById('inbox-item');
const sendTypeSelect = document.getElementById('send-type');
const ssendGroupWrap = document.getElementById('ssend-group-wrap');

// ── Conversations ────────────────────────────────────────────
function ensureConv(k) {
  if (!conversations[k]) conversations[k] = [];
  if (!(k in unread)) unread[k] = 0;
}

function renderConversation() {
  messagesDiv.innerHTML = '';
  const conv = conversations[selectedNodeKey] || [];
  if (!conv.length) { messagesDiv.appendChild(makeMeta('Aucun message.')); return; }
  conv.forEach(m => {
    const row = document.createElement('div');
    if (m.from === 'me') {
      row.className = 'msg-row me-row';
      const b = document.createElement('div');
      b.className = 'msg me' + (m.sendType === 'ssend' ? ' ssend-me' : '');
      b.textContent = (m.sendType === 'ssend' ? '⚡ ' : '') + m.text;
      row.appendChild(b);
      const st = document.createElement('span'); st.className = 'msg-status';
      st.textContent = m.status === 'pending' ? '⏳' : m.status === 'ok' ? '✔' : '✖';
      row.appendChild(st);
    } else if (m.from === 'peer') {
      row.className = 'msg-row peer-row';
      const wrap = document.createElement('div'); wrap.className = 'peer-wrap';
      if (m.mode === 'auth' && m.senderID) {
        const f = document.createElement('div'); f.className = 'msg-from';
        f.textContent = '🔐 ' + m.senderID; wrap.appendChild(f);
      }
      const b = document.createElement('div');
      b.className = m.mode === 'auth' ? 'msg auth-peer' : 'msg peer';
      b.textContent = m.text; wrap.appendChild(b); row.appendChild(wrap);
    } else {
      return void messagesDiv.appendChild(makeMeta(m.text));
    }
    messagesDiv.appendChild(row);
  });
  messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function makeMeta(text) {
  const row = document.createElement('div'); row.className = 'msg-row meta-row';
  const b = document.createElement('div'); b.className = 'msg meta'; b.textContent = text;
  row.appendChild(b); return row;
}

function addMessage(key, from, text, status, mode, senderID, sendType) {
  ensureConv(key);
  conversations[key].push({
    id: ++msgCounter, from, text,
    status: status || 'ok',
    mode: mode || 'anon',
    senderID: senderID || null,
    sendType: sendType || 'send'
  });
  if (key === selectedNodeKey) { renderConversation(); }
  else if (from === 'peer' || from === 'meta') { unread[key] = (unread[key] || 0) + 1; updateBadge(key); }
}

function updateLastPending(destKey, status) {
  const conv = conversations[destKey]; if (!conv) return;
  for (let i = conv.length - 1; i >= 0; i--) {
    if (conv[i].from === 'me' && conv[i].status === 'pending') { conv[i].status = status; break; }
  }
  if (destKey === selectedNodeKey) renderConversation();
}

function updateBadge(key) {
  const count = unread[key] || 0;
  const el = key === INBOX ? inboxItem : document.querySelector('.node-item[data-key="' + key + '"]');
  if (!el) return;
  let badge = el.querySelector('.unread-badge');
  if (!count) { if (badge) badge.remove(); return; }
  if (!badge) { badge = document.createElement('span'); badge.className = 'unread-badge'; el.appendChild(badge); }
  badge.textContent = count > 99 ? '99+' : count;
}

// ── Sidebar ordre stable ─────────────────────────────────────
function rerenderSidebar() {
  nodesListDiv.innerHTML = '';
  if (!nodeOrder.length) {
    nodesListDiv.innerHTML = '<div style="padding:10px 12px;font-size:.78rem;color:var(--text-muted);text-align:center">(aucun nœud)</div>';
    return;
  }
  for (const key of nodeOrder) {
    const node = knownNodes[key];
    ensureConv(key);
    const item = document.createElement('div'); item.className = 'node-item'; item.dataset.key = key;
    if (key === selectedNodeKey) item.classList.add('active');
    const info = document.createElement('div'); info.className = 'node-info';
    const n = document.createElement('div'); n.className = 'node-name'; n.textContent = node.name;
    const a = document.createElement('div'); a.className = 'node-addr'; a.textContent = key;
    info.appendChild(n); info.appendChild(a); item.appendChild(info);
    item.onclick = () => selectNode(node);
    if ((unread[key] || 0) > 0) {
      const badge = document.createElement('span'); badge.className = 'unread-badge';
      badge.textContent = unread[key] > 99 ? '99+' : unread[key]; item.appendChild(badge);
    }
    nodesListDiv.appendChild(item);
  }
}

// ── Chargement des nœuds ─────────────────────────────────────
let lastNodeListRaw = '';

function loadNodes() {
  fetch('/nodes').then(r => r.json()).then(d => {
    const listStr = d.list || '';
    if (listStr === lastNodeListRaw) return;
    lastNodeListRaw = listStr;

    const freshKeys = new Set();
    if (listStr && listStr !== 'LIST:empty') {
      let raw = listStr.startsWith('LIST:') ? listStr.slice(5) : listStr;
      raw.split(',').forEach(entry => {
        const f = entry.trim().split('|'); if (f.length < 3) return;
        const [name, ip, port] = f; const key = ip + ':' + port;
        freshKeys.add(key);
        if (!knownNodes[key]) {
          knownNodes[key] = { name, ip, port };
          nodeOrder.push(key);
        } else {
          knownNodes[key].name = name;
        }
      });
    }

    // Supprimer les nœuds disparus
    for (let i = nodeOrder.length - 1; i >= 0; i--) {
      const key = nodeOrder[i];
      if (!freshKeys.has(key)) {
        onNodeDisappeared(key);
        nodeOrder.splice(i, 1);
        delete knownNodes[key];
      }
    }

    rerenderSidebar();
  }).catch(err => appendLog('Erreur /nodes : ' + err));
}

// ── Nœud disparu ─────────────────────────────────────────────
function onNodeDisappeared(key) {
  const conv = conversations[key];
  if (conv) {
    conv.forEach(m => { if (m.from === 'me' && m.status === 'pending') m.status = 'fail'; });
    if (key === selectedNodeKey) renderConversation();
  }
  addMessage(key,   'meta', '⚠️ Nœud ' + key + ' hors ligne — messages non délivrés', 'ok', 'anon', null, 'send');
  if (key !== selectedNodeKey)
    addMessage(INBOX, 'meta', '⚠️ Nœud ' + key + ' a disparu du réseau', 'ok', 'anon', null, 'send');
  if (selectedNodeKey === key) {
    sendBtn.disabled = true; camBtn.disabled = true;
    if (isStreaming) {
      clearInterval(streamInterval);
      document.getElementById('local-video').srcObject?.getTracks().forEach(t => t.stop());
      document.getElementById('local-video').srcObject = null;
      isStreaming = false; camBtn.classList.remove('active');
    }
    selectedNode = null;
    selectInbox();
  }
}

// ── SEND / SSEND toggle ──────────────────────────────────────
function onSendTypeChange() {
  const isSSend = sendTypeSelect.value === 'ssend';
  ssendGroupWrap.classList.toggle('visible', isSSend);
  sendTypeSelect.classList.toggle('ssend', isSSend);
  if (isSSend) {
    sendBtn.classList.add('ssend-active');
    sendBtn.textContent = 'SSEND ⚡';
  } else {
    sendBtn.classList.remove('ssend-active');
    sendBtn.textContent = 'Envoyer ➤';
  }
}

// ── Sélection ────────────────────────────────────────────────
function selectInbox() {
  selectedNode = null; selectedNodeKey = INBOX; unread[INBOX] = 0; updateBadge(INBOX);
  document.querySelectorAll('.node-item').forEach(el => el.classList.remove('active'));
  inboxItem.classList.add('active');
  nodeNameSpan.textContent = '📬 Messages reçus'; targetHintSpan.textContent = 'aucun nœud';
  sendBtn.disabled = true; camBtn.disabled = true;
  setModeIndicator('anon'); ensureConv(INBOX); renderConversation();
}

function selectNode(node) {
  const key = node.ip + ':' + node.port;
  selectedNode = node; selectedNodeKey = key; unread[key] = 0; updateBadge(key);
  document.querySelectorAll('.node-item').forEach(el => el.classList.remove('active'));
  inboxItem.classList.remove('active');
  const item = document.querySelector('.node-item[data-key="' + key + '"]'); if (item) item.classList.add('active');
  nodeNameSpan.textContent = node.name + '  (' + key + ')';
  targetHintSpan.textContent = key;
  sendBtn.disabled = false; camBtn.disabled = false;
  updateModeStyle(); ensureConv(key); renderConversation();
}

// ── Envoi ─────────────────────────────────────────────────────
function sendMessage() {
  const input = document.getElementById('msg-input');
  const text = input.value.trim();
  const relays = document.getElementById('relay-count').value;
  const mode = modeSelect.value;
  const sendType = sendTypeSelect.value;
  if (!selectedNode || !text || sendBtn.disabled) return;

  const key = selectedNode.ip + ':' + selectedNode.port;
  if (!knownNodes[key]) {
    addMessage(INBOX, 'meta', '⚠️ Impossible d\'envoyer : nœud ' + key + ' hors ligne', 'ok', 'anon', null, 'send');
    onNodeDisappeared(key);
    return;
  }

  let cmd;
  if (sendType === 'ssend') {
    const groupSize = parseInt(document.getElementById('ssend-group').value) || 3;
    cmd = 'SSEND:' + groupSize + ':' + relays + ':' + selectedNode.ip + ':' + selectedNode.port + ':' + text;
    setModeIndicator('ssend');
  } else {
    cmd = 'SEND:' + relays + ':' + selectedNode.ip + ':' + selectedNode.port + ':' + text;
    setModeIndicator(mode);
  }

  addMessage(key, 'me', text, 'pending', mode, null, sendType);
  fetch('/cmd', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ Cmd: cmd, Mode: mode }) })
    .catch(err => addMessage(key, 'meta', 'Erreur réseau : ' + err, 'fail'));
  input.value = '';
}

// ── Vidéo reçue ───────────────────────────────────────────────
function displayRemoteVideo(base64Data, senderName) {
  const container = document.getElementById('video-feed-container');
  const img = document.getElementById('remote-video');
  const label = document.getElementById('video-sender-name');
  container.style.display = 'block';
  label.textContent = 'Flux en direct de ' + senderName;
  img.src = base64Data;
  clearTimeout(window.videoTimeout);
  window.videoTimeout = setTimeout(() => { container.style.display = 'none'; }, 2500);
}

// ── Logs SSE ──────────────────────────────────────────────────
function appendLog(line) {
  const d = document.createElement('div'); d.textContent = line; logsDiv.appendChild(d); logsDiv.scrollTop = logsDiv.scrollHeight;
  const resultM = line.match(/^RESULT(?:_SUPER)?\|([^|]+)\|([A-Z]+)\|/);
  if (resultM) { updateLastPending(resultM[1], resultM[2] === 'ACK' ? 'ok' : 'fail'); return; }
  const msgM = line.match(/Message recu \(MsgID\s*:[^)]+\)[^"]*"([^"]*)"/);
  if (!msgM) return;
  const raw = msgM[1].trim();
  const authM = raw.match(/^AUTH:([^:]+):(.+)$/s);
  if (authM) {
    setModeIndicator('auth');
    const sender = authM[1], text = authM[2].trim();
    if (text.startsWith('VIDEO:')) { displayRemoteVideo(text.substring(6), '🔐 ' + sender); return; }
    const targetKey = findNodeKey(sender);
    addMessage(targetKey || INBOX, 'peer', text, 'ok', 'auth', sender, 'send');
    return;
  }
  setModeIndicator('anon');
  if (raw.startsWith('VIDEO:')) { displayRemoteVideo(raw.substring(6), '🔒 Nœud Anonyme'); return; }
  addMessage(INBOX, 'peer', raw, 'ok', 'anon', null, 'send');
}

// ── Helpers ───────────────────────────────────────────────────
function findNodeKey(senderID) {
  for (const [key, node] of Object.entries(knownNodes)) {
    if (node.name === senderID || key === senderID) return key;
  }
  return null;
}

function setModeIndicator(mode) {
  if (mode === 'auth') { modeIndicator.textContent = '🔐 Authentifié'; modeIndicator.className = 'mode-badge auth'; }
  else if (mode === 'ssend') { modeIndicator.textContent = '⚡ SSEND'; modeIndicator.className = 'mode-badge ssend'; }
  else { modeIndicator.textContent = '🔒 Anonyme'; modeIndicator.className = 'mode-badge anon'; }
}

function updateModeStyle() {
  const val = modeSelect.value;
  modeSelect.className = val === 'auth' ? 'mode-select auth' : 'mode-select';
  if (selectedNodeKey && selectedNodeKey !== INBOX) setModeIndicator(val);
}

function updateRelayLabel() { relaysSpan.textContent = document.getElementById('relay-count').value; }

// ── Webcam ────────────────────────────────────────────────────
async function toggleStream() {
  if (!selectedNode) return;
  const video = document.getElementById('local-video');
  const canvas = document.getElementById('video-canvas');
  if (isStreaming) {
    clearInterval(streamInterval);
    video.srcObject?.getTracks().forEach(t => t.stop());
    video.srcObject = null; isStreaming = false; camBtn.classList.remove('active');
  } else {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
      video.srcObject = stream; isStreaming = true; camBtn.classList.add('active');
      streamInterval = setInterval(() => {
        if (!selectedNode || !isStreaming) return;
        const key = selectedNode.ip + ':' + selectedNode.port;
        if (!knownNodes[key]) { toggleStream(); return; }
        canvas.width = 320; canvas.height = 240;
        canvas.getContext('2d').drawImage(video, 0, 0, 320, 240);
        const frameBase64 = canvas.toDataURL('image/webp', 0.6);
        const relays = document.getElementById('relay-count').value;
        const mode = modeSelect.value;
        const sendType = sendTypeSelect.value;
        let cmd;
        if (sendType === 'ssend') {
          const groupSize = parseInt(document.getElementById('ssend-group').value) || 3;
          cmd = 'SSEND:' + groupSize + ':' + relays + ':' + selectedNode.ip + ':' + selectedNode.port + ':VIDEO:' + frameBase64;
        } else {
          cmd = 'SEND:' + relays + ':' + selectedNode.ip + ':' + selectedNode.port + ':VIDEO:' + frameBase64;
        }
        fetch('/cmd', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ Cmd: cmd, Mode: mode }) });
      }, 66);
    } catch (err) { alert('Erreur webcam : ' + err); }
  }
}

// ── Quit ──────────────────────────────────────────────────────
function quitNode() {
  if (!confirm('Arrêter ce nœud ?')) return;
  fetch('/cmd', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ Cmd: 'QUIT', Mode: 'anon' }) });
}

// ── Init ──────────────────────────────────────────────────────
const es = new EventSource('/logs');
es.onmessage = e => appendLog(e.data);
es.onerror = () => appendLog('⚠️ SSE déconnecté…');

ensureConv(INBOX); selectInbox(); loadNodes(); setInterval(loadNodes, 3000);
updateRelayLabel(); updateModeStyle(); onSendTypeChange();
</script>
</body>
</html>`
