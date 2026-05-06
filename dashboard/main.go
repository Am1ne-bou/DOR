package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Event struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
	Msg  string `json:"msg,omitempty"`
}

var (
	clients = make(map[chan Event]bool)
	mutex   sync.Mutex
)

func broadcast(ev Event) {
	mutex.Lock()
	defer mutex.Unlock()
	for ch := range clients {
		select {
		case ch <- ev:
		default: // ← évite le blocage si le client est lent
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

/* ── HEADER ── */
.header { display: flex; align-items: center; justify-content: space-between; padding: 10px 20px; background: #161b22; border-bottom: 1px solid #30363d; flex-shrink: 0; }
.header h1 { font-size: 1.1rem; color: #58a6ff; }
.header p  { font-size: .75rem; color: #8b949e; }
.legend {
  display: flex; gap: 10px; align-items: center;
  flex-wrap: wrap; background: #0d1117;
  border: 1px solid #30363d; border-radius: 8px;
  padding: 8px 14px;
}
.leg {
  display: flex; align-items: center; gap: 6px;
  font-size: .78rem; color: #c9d1d9; font-weight: 500;
}
.dot {
  width: 13px; height: 13px; border-radius: 50%; flex-shrink: 0;
  box-shadow: 0 0 6px currentColor;
}
.leg-label { color: #8b949e; font-size: .7rem; margin-left: 2px; }

/* ── LAYOUT ── */
.body { display: flex; flex: 1; min-height: 0; }
#mynetwork { flex: 1; }

/* ── LOG PANEL ── */
.log-panel { width: 280px; min-width: 280px; background: #161b22; border-left: 1px solid #30363d; display: flex; flex-direction: column; overflow: hidden; }
.log-title { padding: 8px 12px; font-size: .75rem; font-weight: 600; color: #8b949e; text-transform: uppercase; border-bottom: 1px solid #30363d; flex-shrink: 0; }
.log-list { flex: 1; overflow-y: auto; padding: 4px 0; font-family: monospace; font-size: .68rem; }
.log-entry { padding: 3px 10px; border-bottom: 1px solid #21262d; display: flex; gap: 6px; align-items: baseline; }
.log-entry .badge { flex-shrink: 0; padding: 1px 5px; border-radius: 4px; font-size: .65rem; font-weight: 700; color: #fff; }
.log-entry .route { color: #8b949e; word-break: break-all; }
</style>
</head>
<body>
<div class="header">
  <div>
    <h1>🕸 DOR Network — God Mode</h1>
    <p>Visualisation en temps réel des transmissions</p>
  </div>
	<div class="legend">
	<div class="leg">
		<div class="dot" style="background:#238636;color:#238636"></div>
		<span>RELAY</span><span class="leg-label">/ SEND</span>
	</div>
	<div class="leg">
		<div class="dot" style="background:#1f6feb;color:#1f6feb"></div>
		<span>ACK</span>
	</div>
	<div class="leg">
		<div class="dot" style="background:#da3633;color:#da3633"></div>
		<span>NACK</span>
	</div>
	<div class="leg">
		<div class="dot" style="background:#a371f7;color:#a371f7"></div>
		<span>VIDEO</span>
	</div>
	<div class="leg">
		<div class="dot" style="background:#e3b341;color:#e3b341"></div>
		<span>FINAL</span><span class="leg-label">destination atteinte</span>
	</div>
	</div>
</div>
<div class="body">
  <div id="mynetwork"></div>
  <div class="log-panel">
    <div class="log-title">📋 Événements</div>
    <div class="log-list" id="log-list"></div>
  </div>
</div>

<script>
const COLOR = {
  RELAY:  '#238636',
  SEND:   '#238636',
  ACK:    '#1f6feb',
  NACK:   '#da3633',
  VIDEO:  '#a371f7',
  FINAL:  '#e3b341',
};
const NODE_DEFAULT = { border: '#30363d', background: '#21262d' };

const nodesDS = new vis.DataSet();
const edgesDS = new vis.DataSet();

// --- état d'un "échange" ---
let currentExchangeId = 0;
let currentHopIndex   = 0;          // 1, 2, 3… pour numéroter les flèches

const edgeMeta = new Map();         // eid -> { exchangeId, hopIndex, type }

const network = new vis.Network(
  document.getElementById('mynetwork'),
  { nodes: nodesDS, edges: edgesDS },
  {
    nodes: {
      shape: 'dot', size: 22,
      font: { color: '#c9d1d9', size: 13, face: 'system-ui' },
      borderWidth: 2,
      color: NODE_DEFAULT,
    },
    edges: {
      width: 2,
      color: { color: '#30363d', highlight: '#58a6ff' },
      arrows: { to: { enabled: true, scaleFactor: 0.7 } },
      smooth: { type: 'curvedCW', roundness: 0.15 },
      font: { color: '#8b949e', size: 10, strokeWidth: 0, align: 'horizontal' } // pour afficher le numéro
    },
    physics: {
      barnesHut: { gravitationalConstant: -4000, centralGravity: 0.15, springLength: 160 },
      stabilization: { iterations: 80 },
    },
  }
);

function ensureNode(id) {
  if (!nodesDS.get(id)) {
    nodesDS.add({ id, label: id.replace(':', '\n') });
  }
}

function edgeId(from, to) { return from + '→' + to; }

function ensureEdge(from, to) {
  const id = edgeId(from, to);
  if (!edgesDS.get(id)) {
    edgesDS.add({ id, from, to, color: { color: '#30363d' }, width: 2 });
  }
  return id;
}

// --- gestion des échanges ---
// Appelé pour démarrer un NOUVEL échange et effacer les surbrillances précédentes
function startNewExchange() {
  currentExchangeId += 1;
  currentHopIndex = 0;

  // Réinitialiser toutes les arêtes et tous les nœuds
  const allEdges = edgesDS.get();
  allEdges.forEach(e => {
    edgesDS.update({
      id: e.id,
      color: { color: '#30363d' },
      width: 2,
      label: ''               // enlève le #1, #2…
    });
  });
  const allNodes = nodesDS.get();
  allNodes.forEach(n => {
    nodesDS.update({
      id: n.id,
      color: NODE_DEFAULT
    });
  });
  edgeMeta.clear();
}

// Enregistre un "hop" dans l'échange courant et met la flèche en évidence
function registerHop(from, to, type, msg) {
  ensureNode(from);
  ensureNode(to);
  const eid = ensureEdge(from, to);
  const col = COLOR[type] || COLOR.RELAY;

  // Numéro de saut dans cet échange
  currentHopIndex += 1;
  const hopLabel = '#' + currentHopIndex;

  // Mise en évidence persistante (pas de timeout)
  edgesDS.update({
    id: eid,
    color: { color: col },
    width: 5,
    label: hopLabel
  });

  nodesDS.update({
    id: from,
    color: { border: col, background: col + '33' }
  });
  nodesDS.update({
    id: to,
    color: { border: col, background: col + '33' }
  });

  edgeMeta.set(eid, { exchangeId: currentExchangeId, hopIndex: currentHopIndex, type });
}

// ── Log panel ──
const logList = document.getElementById('log-list');
const BADGE_STYLE = {
  RELAY: 'background:#238636',
  SEND:  'background:#238636',
  ACK:   'background:#1f6feb',
  NACK:  'background:#da3633',
  VIDEO: 'background:#a371f7',
  FINAL: 'background:#e3b341; color:#000',
};

function addLog(ev) {
  const entry = document.createElement('div');
  entry.className = 'log-entry';
  const badge = document.createElement('span');
  badge.className = 'badge';
  badge.style.cssText = BADGE_STYLE[ev.type] || 'background:#30363d';
  badge.textContent = ev.type;
  const route = document.createElement('span');
  route.className = 'route';
  route.textContent = ev.from + ' → ' + ev.to + (ev.msg ? '  "' + ev.msg.slice(0, 60) + '"' : '');
  entry.appendChild(badge);
  entry.appendChild(route);
  logList.prepend(entry);
  while (logList.children.length > 200) logList.removeChild(logList.lastChild);
}

// ── SSE ──
const es = new EventSource('/stream');

es.onmessage = e => {
  const ev = JSON.parse(e.data);

  // On considère qu’un nouveau SEND (ou VIDEO) démarre un nouvel échange
  if (ev.type === 'SEND' || ev.type === 'VIDEO') {
    startNewExchange();
  }

  registerHop(ev.from, ev.to, ev.type, ev.msg);
  addLog(ev);
};

es.onerror = () => console.warn('SSE déconnecté');
</script>
</body>
</html>`

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(dashboardHTML))
	})

	http.HandleFunc("/telemetry", func(w http.ResponseWriter, r *http.Request) {
		var ev Event
		if err := json.NewDecoder(r.Body).Decode(&ev); err == nil {
			broadcast(ev)
		}
		w.WriteHeader(http.StatusOK)
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
