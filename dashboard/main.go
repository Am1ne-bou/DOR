package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Event struct {
	From string    `json:"from"`
	To   string    `json:"to"`
	Type string    `json:"type"`
	Msg  string    `json:"msg,omitempty"`
	ExID string    `json:"exid,omitempty"`
	Ts   time.Time `json:"ts"`
}

var (
	clients   = make(map[chan Event]bool)
	mutex     sync.Mutex
	history   []Event
	histMutex sync.RWMutex
)

func broadcast(ev Event) {
	mutex.Lock()
	defer mutex.Unlock()
	for ch := range clients {
		select {
		case ch <- ev:
		default:
		}
	}
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="UTF-8">
<title>DOR — God Mode</title>
<script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
<style>
* { box-sizing: border-box; margin: 0; padding: 0 }
body { background: #0d1117; color: #c9d1d9; font-family: system-ui, sans-serif; height: 100vh; display: flex; flex-direction: column; overflow: hidden; }

.header { display: flex; align-items: center; justify-content: space-between; padding: 10px 20px; background: #161b22; border-bottom: 1px solid #30363d; flex-shrink: 0; flex-wrap: wrap; gap: 8px; }
.header h1 { font-size: 1.1rem; color: #58a6ff; }
.header p  { font-size: .75rem; color: #8b949e; }

.legend { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; background: #0d1117; border: 1px solid #30363d; border-radius: 8px; padding: 8px 14px; }
.leg { display: flex; align-items: center; gap: 6px; font-size: .78rem; color: #c9d1d9; font-weight: 500; }
.dot { width: 12px; height: 12px; border-radius: 50%; flex-shrink: 0; }

.stats-bar { display: flex; gap: 16px; padding: 6px 20px; background: #0d1117; border-bottom: 1px solid #30363d; font-size: .75rem; flex-shrink: 0; flex-wrap: wrap; }
.stat { display: flex; gap: 6px; align-items: center; }
.stat-val { font-weight: 700; color: #58a6ff; }
.stat-val.green  { color: #3fb950; }
.stat-val.red    { color: #f85149; }
.stat-val.purple { color: #bc8cff; }
.stat-val.yellow { color: #e3b341; }

.toolbar { display: flex; gap: 10px; padding: 6px 20px; background: #161b22; border-bottom: 1px solid #30363d; font-size: .75rem; flex-shrink: 0; align-items: center; flex-wrap: wrap; }
.toolbar label { color: #8b949e; display: flex; align-items: center; gap: 4px; }
.toolbar input, .toolbar select { background: #0d1117; border: 1px solid #30363d; color: #c9d1d9; padding: 3px 7px; border-radius: 5px; font-size: .75rem; }
.toolbar button { background: #21262d; border: 1px solid #30363d; color: #c9d1d9; padding: 4px 12px; border-radius: 5px; cursor: pointer; font-size: .75rem; }
.toolbar button:hover { background: #30363d; }
.toolbar .sep { color: #30363d; }

.body { display: flex; flex: 1; min-height: 0; }
#mynetwork { flex: 1; }

.right-panel { width: 340px; min-width: 340px; background: #161b22; border-left: 1px solid #30363d; display: flex; flex-direction: column; overflow: hidden; }

.tabs { display: flex; border-bottom: 1px solid #30363d; flex-shrink: 0; }
.tab { flex: 1; padding: 7px; text-align: center; font-size: .72rem; font-weight: 600; color: #8b949e; cursor: pointer; text-transform: uppercase; border-bottom: 2px solid transparent; }
.tab.active { color: #58a6ff; border-bottom-color: #58a6ff; }

.tab-content { flex: 1; overflow-y: auto; display: none; }
.tab-content.active { display: block; }

.log-list { padding: 4px 0; font-family: monospace; font-size: .68rem; }
.log-entry { padding: 4px 10px; border-bottom: 1px solid #21262d; display: flex; gap: 6px; align-items: baseline; cursor: pointer; }
.log-entry:hover { background: #21262d; }
.log-entry .badge { flex-shrink: 0; padding: 1px 5px; border-radius: 4px; font-size: .65rem; font-weight: 700; color: #fff; }
.log-entry .route { color: #8b949e; word-break: break-all; flex: 1; font-size: .67rem; }
.log-entry .exid-badge { flex-shrink: 0; padding: 1px 5px; border-radius: 4px; font-size: .6rem; background: #21262d; color: #8b949e; border: 1px solid #30363d; }
.log-entry .hop-num { flex-shrink: 0; font-size: .6rem; color: #58a6ff; font-weight: 700; min-width: 28px; text-align: right; }
.log-entry .ts { color: #484f58; font-size: .6rem; flex-shrink: 0; }
.log-entry.simulated { opacity: 0.55; font-style: italic; }

.exchange-list { padding: 8px; }
.ex-card { background: #0d1117; border: 1px solid #30363d; border-radius: 8px; margin-bottom: 8px; padding: 10px 12px; cursor: pointer; transition: border-color .15s; }
.ex-card:hover { border-color: #58a6ff44; }
.ex-card.active { border-color: #58a6ff; }
.ex-card .ex-id { font-size: .78rem; font-weight: 700; color: #58a6ff; display: flex; justify-content: space-between; align-items: center; }
.ex-card .ex-meta { font-size: .67rem; color: #8b949e; margin-top: 3px; }
.ex-card .ex-route { margin-top: 8px; }
.ex-card .ex-row { display: flex; align-items: center; gap: 4px; margin-bottom: 3px; font-size: .65rem; }
.ex-card .ex-row-label { color: #484f58; font-weight: 700; width: 40px; flex-shrink: 0; }
.ex-hop { padding: 2px 7px; border-radius: 4px; font-size: .62rem; font-weight: 600; display: inline-flex; align-items: center; gap: 3px; }
.ex-arrow { color: #484f58; font-size: .7rem; }
.ex-result { font-size: .72rem; font-weight: 700; margin-top: 6px; display: flex; align-items: center; gap: 6px; }
.ex-result.ack     { color: #3fb950; }
.ex-result.nack    { color: #f85149; }
.ex-result.pending { color: #e3b341; }

.node-info { padding: 12px; font-size: .72rem; }
.node-info h3 { color: #58a6ff; margin-bottom: 8px; font-size: .85rem; word-break: break-all; }
.node-stat { display: flex; justify-content: space-between; padding: 5px 0; border-bottom: 1px solid #21262d; }
.node-stat .k { color: #8b949e; }
.node-stat .v { color: #c9d1d9; font-weight: 600; }
.node-ex-list { margin-top: 10px; }
.node-ex-list h4 { color: #8b949e; font-size: .7rem; margin-bottom: 5px; }
.node-ex-item { font-size: .67rem; padding: 3px 0; border-bottom: 1px solid #21262d; display: flex; justify-content: space-between; cursor: pointer; color: #58a6ff; }
.node-ex-item:hover { color: #c9d1d9; }

#no-selection { padding: 20px; color: #484f58; font-size: .75rem; text-align: center; margin-top: 20px; }

/* Scrollbar */
::-webkit-scrollbar { width: 5px; }
::-webkit-scrollbar-track { background: #0d1117; }
::-webkit-scrollbar-thumb { background: #30363d; border-radius: 3px; }
</style>
</head>
<body>

<div class="header">
  <div>
    <h1>🕸 DOR Network — God Mode</h1>
    <p>Visualisation temps réel · <span id="conn-status" style="color:#3fb950">Connecté</span></p>
  </div>
  <div class="legend">
    <div class="leg"><div class="dot" style="background:#238636"></div><span>SEND/RELAY</span></div>
    <div class="leg"><div class="dot" style="background:#a371f7"></div><span>SSEND</span></div>
    <div class="leg"><div class="dot" style="background:#e3b341"></div><span>FINAL</span></div>
    <div class="leg"><div class="dot" style="background:#1f6feb"></div><span>ACK</span></div>
    <div class="leg"><div class="dot" style="background:#da3633"></div><span>NACK</span></div>
    <div class="leg"><div class="dot" style="background:#1f6feb;opacity:.4"></div><span>ACK reconstruit</span></div>
  </div>
</div>

<div class="stats-bar">
  <div class="stat">📨 Msgs : <span class="stat-val" id="s-total">0</span></div>
  <div class="stat">✅ ACK : <span class="stat-val green" id="s-ack">0</span></div>
  <div class="stat">❌ NACK : <span class="stat-val red" id="s-nack">0</span></div>
  <div class="stat">🔀 Échanges : <span class="stat-val purple" id="s-ex">0</span></div>
  <div class="stat">📦 Livraison : <span class="stat-val green" id="s-rate">—</span></div>
  <div class="stat">⏱ Latence moy. : <span class="stat-val yellow" id="s-lat">—</span></div>
  <div class="stat">🖥 Nœuds : <span class="stat-val" id="s-nodes">0</span></div>
</div>

<div class="toolbar">
  <label>ExID : <input id="filter-ex" placeholder="filtrer…" style="width:90px" oninput="applyFilter()"></label>
  <label>Type :
    <select id="filter-type" onchange="applyFilter()">
      <option value="">Tous</option>
      <option>SEND</option><option>SSEND</option><option>RELAY</option>
      <option>ACK</option><option>NACK</option><option>FINAL</option>
    </select>
  </label>
  <span class="sep">|</span>
  <button onclick="togglePhysics()" id="btn-physics">⚡ Physics ON</button>
  <button onclick="network.fit()">🎯 Fit</button>
  <button onclick="clearAll()">🗑 Effacer</button>
  <span class="sep">|</span>
  <label><input type="checkbox" id="chk-persist" checked> Persistant</label>
  <label><input type="checkbox" id="chk-reconstruct" checked> Reconstituer ACK</label>
</div>

<div class="body">
  <div id="mynetwork"></div>
  <div class="right-panel">
    <div class="tabs">
      <div class="tab active" onclick="switchTab('log')">📋 Log</div>
      <div class="tab" onclick="switchTab('exchanges')">🔄 Échanges</div>
      <div class="tab" onclick="switchTab('nodeinfo')">🖥 Nœud</div>
    </div>
    <div class="tab-content active" id="tab-log">
      <div class="log-list" id="log-list"></div>
    </div>
    <div class="tab-content" id="tab-exchanges">
      <div class="exchange-list" id="exchange-list"></div>
    </div>
    <div class="tab-content" id="tab-nodeinfo">
      <div id="no-selection">Cliquez sur un nœud<br>pour voir ses statistiques</div>
      <div class="node-info" id="node-info" style="display:none"></div>
    </div>
  </div>
</div>

<script>
// ── Couleurs ──────────────────────────────────────────────────
const COLOR = {
  RELAY:'#238636', SEND:'#238636', SSEND:'#a371f7',
  ACK:'#1f6feb',   NACK:'#da3633', FINAL:'#e3b341',
  ACK_SIM:'#1f6feb55',
};
const BADGE = {
  RELAY:'background:#238636', SEND:'background:#238636', SSEND:'background:#a371f7',
  ACK:'background:#1f6feb',   NACK:'background:#da3633',
  FINAL:'background:#e3b341;color:#0d1117',
};
const NODE_DEFAULT = { border:'#30363d', background:'#21262d' };

// ── State ─────────────────────────────────────────────────────
const allEvents = [];
const exchanges = {};   // exid → Exchange
const nodeStats = {};   // addr → stats
let physicsOn   = true;
let selectedEx   = null;
let selectedNode = null;
let statTotal=0, statAck=0, statNack=0;
const latencies = [];

// Exchange shape :
// { id, rawEvents[], fwdRoute[], retRoute[], fwdCount, retCount,
//   startTs, endTs, result, msg }

// ── vis-network ───────────────────────────────────────────────
const nodesDS = new vis.DataSet();
const edgesDS = new vis.DataSet();

const network = new vis.Network(
  document.getElementById('mynetwork'),
  { nodes: nodesDS, edges: edgesDS },
  {
    nodes:{
      shape:'dot', size:24,
      font:{ color:'#c9d1d9', size:13, face:'system-ui' },
      borderWidth:2, color: NODE_DEFAULT,
    },
    edges:{
      width:2,
      color:{ color:'#30363d', highlight:'#58a6ff' },
      arrows:{ to:{ enabled:true, scaleFactor:.7 } },
      smooth:{ type:'curvedCW', roundness:.18 },
      font:{ color:'#c9d1d9', size:12, strokeWidth:2, strokeColor:'#0d1117', align:'horizontal' },
    },
    physics:{
      barnesHut:{ gravitationalConstant:-5000, centralGravity:.2, springLength:180 },
      stabilization:{ iterations:100 },
    },
  }
);

network.on('click', p => {
  if (p.nodes.length) { selectedNode = p.nodes[0]; switchTab('nodeinfo'); renderNodeInfo(selectedNode); }
});

// ── Helpers ───────────────────────────────────────────────────
function ensureNode(id) {
  if (!nodesDS.get(id)) {
    nodesDS.add({ id, label: id.replace(':','\n') });
    nodeStats[id] = { sent:0, recv:0, relay:0, ack:0, nack:0, final:0 };
  }
}

function edgeKey(f,t) { return f+'→'+t; }

function ensureEdge(f,t) {
  const id = edgeKey(f,t);
  if (!edgesDS.get(id)) edgesDS.add({ id, from:f, to:t, color:{ color:'#30363d' }, width:2, label:'' });
  return id;
}

function flashEdge(from, to, col, label, simulated) {
  ensureNode(from); ensureNode(to);
  const eid = ensureEdge(from, to);
  const c = simulated ? COLOR.ACK_SIM : col;
  edgesDS.update({ id:eid, color:{ color:c }, width: simulated ? 3 : 5, label, dashes: simulated });
  nodesDS.update({ id:from, color:{ border:col, background:col+'22' } });
  nodesDS.update({ id:to,   color:{ border:col, background:col+'22' } });
  if (!document.getElementById('chk-persist').checked) {
    setTimeout(() => {
      if (edgesDS.get(eid))  edgesDS.update({ id:eid, color:{ color:'#30363d' }, width:2, label:'', dashes:false });
      if (nodesDS.get(from)) nodesDS.update({ id:from, color:NODE_DEFAULT });
      if (nodesDS.get(to))   nodesDS.update({ id:to,   color:NODE_DEFAULT });
    }, 4000);
  }
}

// ── Exchange ──────────────────────────────────────────────────
function getEx(exid) {
  if (!exchanges[exid]) {
    exchanges[exid] = {
      id:exid, rawEvents:[], fwdRoute:[], retRoute:[],
      fwdCount:0, retCount:0,
      startTs:null, endTs:null, result:'pending', msg:''
    };
  }
  return exchanges[exid];
}

// Reconstruit la route retour depuis la route forward
function buildReturnRoute(ex) {
  if (!document.getElementById('chk-reconstruct').checked) return [];
  // fwdRoute = [src, R1, R2, ..., dest]
  // retRoute = [dest, ..., R2, R1, src] (inverse sans src ni dest doublé)
  const r = ex.fwdRoute;
  if (r.length < 2) return [];
  return [...r].reverse();
}

// ── Traitement event ──────────────────────────────────────────
function processEvent(ev) {
  allEvents.push(ev);
  statTotal++;

  const exid = ev.exid || '__noex__';
  const ex   = getEx(exid);
  if (!ex.startTs) ex.startTs = ev.ts ? new Date(ev.ts) : new Date();
  if (ev.msg) ex.msg = ev.msg;

  ex.rawEvents.push(ev);

  // Construire fwdRoute au fil des events
  if (ev.type === 'SEND' || ev.type === 'SSEND') {
    ex.fwdRoute = [ev.from];
  }
  if (ev.type === 'RELAY' && ex.fwdRoute.length > 0) {
    if (!ex.fwdRoute.includes(ev.to)) ex.fwdRoute.push(ev.to);
  }
  if (ev.type === 'FINAL') {
    if (ex.fwdRoute.length === 0) ex.fwdRoute = [ev.from];
    if (!ex.fwdRoute.includes(ev.to || ev.from)) ex.fwdRoute.push(ev.to || ev.from);
  }

  const col = COLOR[ev.type] || COLOR.RELAY;

  if (ev.type === 'ACK') {
    ex.result = 'ack';
    ex.endTs  = ev.ts ? new Date(ev.ts) : new Date();
    statAck++;
    if (ex.startTs && ex.endTs) latencies.push(ex.endTs - ex.startTs);

    // Reconstituer visuellement les hops retour
    const retRoute = buildReturnRoute(ex);
    ex.retRoute = retRoute;

    if (retRoute.length >= 2) {
      for (let i = 0; i < retRoute.length - 1; i++) {
        ex.retCount++;
        const label = '#R' + ex.retCount;
        flashEdge(retRoute[i], retRoute[i+1], COLOR.ACK, label, i > 0);
        addLogEntry({ from:retRoute[i], to:retRoute[i+1], type:'ACK', exid, ts:ev.ts },
          ex.retCount, true, i > 0);
      }
    } else {
      // pas assez de route connue, on affiche juste le saut direct
      ex.retCount++;
      flashEdge(ev.from, ev.to, col, '#R1', false);
      addLogEntry(ev, ex.retCount, true, false);
    }

    updateStats();
    renderExchangeCard(exid);
    if (selectedNode) renderNodeInfo(selectedNode);
    return;
  }

  if (ev.type === 'NACK') {
    ex.result = 'nack';
    statNack++;
    ex.retCount++;
    flashEdge(ev.from, ev.to, col, '#R' + ex.retCount, false);
    addLogEntry(ev, ex.retCount, true, false);
    updateStats();
    renderExchangeCard(exid);
    return;
  }

  // Events forward
  ex.fwdCount++;
  const label = '#F' + ex.fwdCount;

  if (ev.type === 'FINAL') {
    ex.result = ex.result === 'pending' ? 'pending' : ex.result;
  }

  // node stats
  ensureNode(ev.from); ensureNode(ev.to);
  const ns = nodeStats[ev.from];
  if (ev.type === 'SEND' || ev.type === 'SSEND') { ns.sent++; nodeStats[ev.to].recv++; }
  if (ev.type === 'RELAY') ns.relay++;
  if (ev.type === 'FINAL') { ns.final++; nodeStats[ev.to] && nodeStats[ev.to].recv++; }

  flashEdge(ev.from, ev.to, col, label, false);
  addLogEntry(ev, ex.fwdCount, false, false);
  updateStats();
  renderExchangeCard(exid);
  if (selectedNode === ev.from || selectedNode === ev.to) renderNodeInfo(selectedNode);
}

// ── Stats bar ─────────────────────────────────────────────────
function updateStats() {
  document.getElementById('s-total').textContent = statTotal;
  document.getElementById('s-ack').textContent   = statAck;
  document.getElementById('s-nack').textContent  = statNack;
  document.getElementById('s-ex').textContent    = Object.keys(exchanges).filter(k=>k!=='__noex__').length;
  document.getElementById('s-nodes').textContent = nodesDS.length;
  const tot = statAck + statNack;
  document.getElementById('s-rate').textContent  = tot > 0 ? Math.round(statAck/tot*100)+'%' : '—';
  if (latencies.length > 0) {
    const avg = Math.round(latencies.reduce((a,b)=>a+b,0)/latencies.length);
    document.getElementById('s-lat').textContent = avg+' ms';
  }
}

// ── Log ───────────────────────────────────────────────────────
function addLogEntry(ev, hopNum, isReturn, isSimulated) {
  const exF = document.getElementById('filter-ex').value.trim();
  const tyF = document.getElementById('filter-type').value;
  if (exF && !(ev.exid||'').includes(exF)) return;
  if (tyF && ev.type !== tyF) return;

  const el = document.createElement('div');
  el.className = 'log-entry' + (isSimulated ? ' simulated' : '');
  el.dataset.exid = ev.exid || '';
  el.onclick = () => { if (ev.exid) { highlightExchange(ev.exid); switchTab('exchanges'); } };

  const dir   = isReturn ? 'R' : 'F';
  const ts    = ev.ts ? new Date(ev.ts).toLocaleTimeString() : '';
  const badge = BADGE[ev.type] || 'background:#30363d';
  const sim   = isSimulated ? ' <span style="color:#484f58;font-size:.58rem">(sim)</span>' : '';

  el.innerHTML =
    '<span class="badge" style="'+badge+'">'+ev.type+'</span>'+
    '<span class="route">'+ev.from+' → '+ev.to+
      (ev.msg?'  <span style="color:#484f58">"'+ev.msg.slice(0,30)+'"</span>':'')+
      sim+'</span>'+
    (ev.exid?'<span class="exid-badge">'+ev.exid+'</span>':'')+
    '<span class="hop-num">#'+dir+hopNum+'</span>'+
    '<span class="ts">'+ts+'</span>';

  const list = document.getElementById('log-list');
  list.prepend(el);
  while (list.children.length > 400) list.removeChild(list.lastChild);
}

// ── Exchange card ─────────────────────────────────────────────
function renderExchangeCard(exid) {
  if (!exid || exid === '__noex__') return;
  const ex = exchanges[exid];
  if (!ex) return;

  const list = document.getElementById('exchange-list');
  let card = document.getElementById('ex-'+exid);
  if (!card) {
    card = document.createElement('div');
    card.className = 'ex-card';
    card.id = 'ex-'+exid;
    card.onclick = () => highlightExchange(exid);
    list.prepend(card);
  }
  card.classList.toggle('active', selectedEx === exid);

  const dur = (ex.startTs && ex.endTs) ? Math.round(ex.endTs - ex.startTs)+'ms' : '⏳';
  const resultClass = ex.result==='ack'?'ack': ex.result==='nack'?'nack':'pending';
  const resultLabel = ex.result==='ack'?'✅ Livré': ex.result==='nack'?'❌ Échoué':'⏳ En cours';

  // Ligne forward
  const fwdHops = ex.fwdRoute.map((n,i) => {
    const c = i===0 ? (COLOR.SEND) : i===ex.fwdRoute.length-1 ? COLOR.FINAL : COLOR.RELAY;
    return hop(n, c) + (i<ex.fwdRoute.length-1 ? '<span class="ex-arrow">→</span>' : '');
  }).join('');

  // Ligne retour
  const retRoute = ex.retRoute.length > 0 ? ex.retRoute :
    (ex.result==='ack' && ex.fwdRoute.length>=2 ? [...ex.fwdRoute].reverse() : []);
  const retHops = retRoute.map((n,i) => {
    return hop(n, COLOR.ACK, i>0 && i<retRoute.length-1) +
      (i<retRoute.length-1 ? '<span class="ex-arrow">→</span>' : '');
  }).join('');

  card.innerHTML =
    '<div class="ex-id">'+exid+
      '<span style="font-size:.65rem;font-weight:400;color:#8b949e">'+dur+
        (ex.startTs?' · '+ex.startTs.toLocaleTimeString():'')+
      '</span>'+
    '</div>'+
    (ex.msg?'<div class="ex-meta">💬 "'+ex.msg.slice(0,50)+'"</div>':'')+
    '<div class="ex-route">'+
      '<div class="ex-row"><span class="ex-row-label">→ FWD</span>'+fwdHops+'</div>'+
      (retHops?'<div class="ex-row"><span class="ex-row-label">← ACK</span>'+retHops+'</div>':'')+
    '</div>'+
    '<div class="ex-result '+resultClass+'">'+resultLabel+'</div>';
}

function hop(addr, col, dashed) {
  const short = addr.split(':')[1] || addr;
  const st = 'background:'+col+'22;color:'+col+';border:1px solid '+col+'55'+(dashed?';border-style:dashed':'');
  return '<span class="ex-hop" style="'+st+'" title="'+addr+'">'+short+'</span>';
}

// ── Highlight exchange ────────────────────────────────────────
function highlightExchange(exid) {
  selectedEx = exid;
  edgesDS.get().forEach(e => edgesDS.update({ id:e.id, color:{color:'#30363d'}, width:2, label:'', dashes:false }));
  nodesDS.get().forEach(n => nodesDS.update({ id:n.id, color:NODE_DEFAULT }));

  const ex = exchanges[exid];
  if (!ex) return;

  // Forward hops
  let fi = 0;
  ex.rawEvents.filter(e => !isRet(e.type)).forEach(ev => {
    fi++;
    const col = COLOR[ev.type]||COLOR.RELAY;
    ensureNode(ev.from); ensureNode(ev.to);
    const eid = ensureEdge(ev.from, ev.to);
    edgesDS.update({ id:eid, color:{color:col}, width:6, label:'#F'+fi, dashes:false });
    nodesDS.update({ id:ev.from, color:{border:col, background:col+'33'} });
    nodesDS.update({ id:ev.to,   color:{border:col, background:col+'33'} });
  });

  // Return hops (real or reconstructed)
  const retRoute = ex.retRoute.length > 0 ? ex.retRoute :
    (ex.fwdRoute.length>=2 ? [...ex.fwdRoute].reverse() : []);
  for (let i=0; i<retRoute.length-1; i++) {
    const sim = i>0 && ex.retRoute.length===0;
    const eid = ensureEdge(retRoute[i], retRoute[i+1]);
    edgesDS.update({ id:eid, color:{color:COLOR.ACK}, width: sim?3:6, label:'#R'+(i+1), dashes:sim });
    nodesDS.update({ id:retRoute[i],   color:{border:COLOR.ACK, background:COLOR.ACK+'33'} });
    nodesDS.update({ id:retRoute[i+1], color:{border:COLOR.ACK, background:COLOR.ACK+'33'} });
  }

  document.querySelectorAll('.ex-card').forEach(c =>
    c.classList.toggle('active', c.id==='ex-'+exid));
  network.fit({ animation:{ duration:600, easingFunction:'easeInOutQuad' } });
}

function isRet(type) { return type==='ACK'||type==='NACK'; }

// ── Node info ─────────────────────────────────────────────────
function renderNodeInfo(addr) {
  if (!addr || !nodeStats[addr]) return;
  const s = nodeStats[addr];
  const info  = document.getElementById('node-info');
  const nosel = document.getElementById('no-selection');
  info.style.display = 'block'; nosel.style.display = 'none';

  const involved = Object.values(exchanges).filter(ex =>
    ex.rawEvents.some(e => e.from===addr||e.to===addr)
  );

  info.innerHTML =
    '<h3>'+addr+'</h3>'+
    row('Messages envoyés', s.sent)+
    row('Messages reçus',   s.recv)+
    row('Relais effectués', s.relay)+
    row('ACK émis',         s.ack)+
    row('NACK émis',        s.nack)+
    row('FINAL reçus',      s.final)+
    row('Échanges',         involved.length)+
    (involved.length>0?
      '<div class="node-ex-list"><h4>Échanges impliqués</h4>'+
      involved.slice(0,10).map(ex=>
        '<div class="node-ex-item" onclick="highlightExchange(\''+ex.id+'\');switchTab(\'exchanges\')">'+
          ex.id+
          '<span style="color:'+(ex.result==='ack'?'#3fb950':ex.result==='nack'?'#f85149':'#e3b341')+'">'+
            (ex.result==='ack'?'✅':ex.result==='nack'?'❌':'⏳')+
          '</span>'+
        '</div>'
      ).join('')+
      '</div>'
    :'');
}

function row(k,v) {
  return '<div class="node-stat"><span class="k">'+k+'</span><span class="v">'+v+'</span></div>';
}

// ── Filtres ───────────────────────────────────────────────────
function applyFilter() {
  document.getElementById('log-list').innerHTML = '';
  const exF = document.getElementById('filter-ex').value.trim();
  const tyF = document.getElementById('filter-type').value;
  allEvents.slice().reverse().forEach(ev => {
    if (exF && !(ev.exid||'').includes(exF)) return;
    if (tyF && ev.type !== tyF) return;
    const ex = exchanges[ev.exid||'__noex__'];
    const ret = isRet(ev.type);
    let n = 1;
    if (ex) {
      let fi=0, ri=0;
      for (const e of ex.rawEvents) {
        if (isRet(e.type)) ri++; else fi++;
        if (e === ev) { n = ret ? ri : fi; break; }
      }
    }
    addLogEntry(ev, n, ret, false);
  });
}

// ── UI ────────────────────────────────────────────────────────
function switchTab(name) {
  ['log','exchanges','nodeinfo'].forEach((n,i) => {
    document.querySelectorAll('.tab')[i].classList.toggle('active', n===name);
    document.getElementById('tab-'+n).classList.toggle('active', n===name);
  });
  document.querySelectorAll('.tab-content').forEach(c => c.style.display='none');
  document.getElementById('tab-'+name).style.display='block';
}

// Fix: init tab display
document.querySelectorAll('.tab-content').forEach(c => c.style.display='none');
document.getElementById('tab-log').style.display='block';

function togglePhysics() {
  physicsOn = !physicsOn;
  network.setOptions({ physics:{ enabled:physicsOn } });
  document.getElementById('btn-physics').textContent = '⚡ Physics '+(physicsOn?'ON':'OFF');
}

function clearAll() {
  allEvents.length = 0;
  Object.keys(exchanges).forEach(k=>delete exchanges[k]);
  Object.keys(nodeStats).forEach(k=>delete nodeStats[k]);
  statTotal=0; statAck=0; statNack=0; latencies.length=0;
  nodesDS.clear(); edgesDS.clear();
  document.getElementById('log-list').innerHTML='';
  document.getElementById('exchange-list').innerHTML='';
  selectedEx=null; selectedNode=null;
  updateStats();
}

// ── SSE ───────────────────────────────────────────────────────
const es = new EventSource('/stream');
es.onmessage = e => processEvent(JSON.parse(e.data));
es.onopen  = () => { const s=document.getElementById('conn-status'); s.textContent='Connecté';     s.style.color='#3fb950'; };
es.onerror = () => { const s=document.getElementById('conn-status'); s.textContent='⚠ Déconnecté'; s.style.color='#f85149'; };

fetch('/history').then(r=>r.json()).then(evs=>evs.forEach(processEvent)).catch(()=>{});
</script>
</body>
</html>`

// TODO: add live network graph -- visualize routing topology as nodes relay messages (D3.js or similar)
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(dashboardHTML))
	})

	http.HandleFunc("/telemetry", func(w http.ResponseWriter, r *http.Request) {
		var ev Event
		if err := json.NewDecoder(r.Body).Decode(&ev); err == nil {
			ev.Ts = time.Now()
			histMutex.Lock()
			history = append(history, ev)
			histMutex.Unlock()
			broadcast(ev)
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		histMutex.RLock()
		defer histMutex.RUnlock()
		json.NewEncoder(w).Encode(history)
	})

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := make(chan Event, 64)
		mutex.Lock()
		clients[ch] = true
		mutex.Unlock()
		defer func() {
			mutex.Lock()
			delete(clients, ch)
			mutex.Unlock()
			close(ch)
		}()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-ch:
				b, _ := json.Marshal(ev)
				fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
			}
		}
	})

	fmt.Println("🌐 Dashboard God Mode → http://localhost:8888")
	http.ListenAndServe(":8888", nil)
}
