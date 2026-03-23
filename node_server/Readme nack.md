# Branche feature/nack

## Résumé

Mécanisme de **NACK + Retry** pour garantir la livraison des messages quand des relais tombent.

---

## Problème

Quand un relai tombe pendant le transit, le message est perdu. Le sender ne le sait pas et attend un timeout pour rien.

## Solution

Le relai qui détecte l'échec envoie un NACK qui remonte hop par hop jusqu'au sender. Le sender reconstruit une nouvelle route et renvoie le message automatiquement (max 3 tentatives).

---

## Ce qui a changé

### OnionLayer.go
Simplifié de 7 à 6 champs. Les champs sont recyclés entre RELAY et FINAL :
- `Next` = NextHop (RELAY) ou ReturnAddr (FINAL)
- `Data` = Payload chiffré (RELAY) ou ReturnOnion (FINAL)

### Node.go
- Interception des NACK plaintext (`NACK:<nackID>`)
- `PendingRelays` map pour propager les NACK hop par hop
- NackIDs aléatoires entre chaque hop (un observateur voit des IDs différents)
- `SIM_LATENCY` : variable d'env pour simuler la latence réseau (optionnel, 0 par défaut)

### main.go
- `SendWithRetry()` : reconstruit une route et renvoie sur NACK
- Commande `BENCH:<nbr_messages>:<nbr_relays>:<maxRetries>:<ip>:<port>` pour les tests
- NackArray généré à la construction de l'onion


---

## NackIDs

Chaque couche RELAY contient `"nackID_tosend:nackID_toreceive"` dans MsgID :

```
Couche R1 : MsgID = "nack-111:nack-222"
Couche R2 : MsgID = "nack-222:nack-333"

R2 fail → envoie NACK:nack-222 → R1
R1 lookup nack-222 → trouve nack-111 → envoie NACK:nack-111 → Sender
```

Chaque hop voit un ID différent → pas de corrélation par un observateur.

---

## Limites connues

1. NACK en plaintext (pas chiffré) — un observateur voit qu'un NACK passe mais ne peut pas le relier au message (IDs différents)
2. Même msgID entre sender et destination — nécessaire pour le ACK
3. Pas de liste noire des nœuds morts — le retry peut rechoisir un nœud mort

---

## Résultats benchmark

24 configurations testées avec différents paramètres : taille du réseau (5-20 relais), pannes simultanées (1-4 kills), fréquence du churn, latence simulée (0-200ms), et charge réseau (1-5 senders).

| Scénario | Latence | Sans retry | Avec retry | Gain |
|----------|---------|-----------|-----------|------|
| Petit réseau (5r) | 0ms | 76.3% | 91.2% | +14.9% |
| Réseau moyen (10r) | 0ms | 87.0% | 96.5% | +9.5% |
| Grand réseau (15r) | 0ms | 88.2% | 98.8% | +10.6% |
| Très grand réseau (20r) | 0ms | 87.0% | 99.8% | +12.8% |
| 2 pannes simultanées | 0ms | 83.7% | 97.5% | +13.8% |
| 3 pannes simultanées | 0ms | 68.3% | 93.7% | +25.4% |
| 4 pannes simultanées | 0ms | 74.3% | 88.7% | +14.4% |
| Churn faible (10s) | 0ms | 93.0% | 99.7% | +6.7% |
| Réseau stable | 0ms | 99.2% | 100% | +0.8% |
| Churn extrême (1s) | 0ms | 80.7% | 95.3% | +14.6% |
| 1 sender, 80ms | 80ms | 70.7% | 89.0% | +18.3% |
| 3 senders, 80ms | 80ms | 80.7% | 94.8% | +14.1% |
| 5 senders, 80ms | 80ms | 88.3% | 95.2% | +6.9% |
| Latence 50ms | 50ms | 81.5% | 97.3% | +15.8% |
| Latence 100ms | 100ms | 76.9% | 96.7% | +19.8% |
| Latence 200ms | 200ms | 83.6% | 92.7% | +9.1% |
| Réseau stable, pannes groupées | 50ms | 94.6% | 98.6% | +4.0% |
| Réseau moyen, latence variable | 100ms | 88.8% | 98.4% | +9.6% |
| Réseau mobile, déconnexions fréquentes | 150ms | 92.5% | 98.5% | +6.0% |
| Réseau instable, haute latence | 200ms | 82.7% | 98.4% | +15.7% |
| Conditions extrêmes | 100ms | 82.2% | 93.4% | +11.2% |

Le retry améliore le taux de livraison dans **chaque** configuration. Sans retry, le taux descend jusqu'à 68%. Avec retry, il ne descend jamais sous 88%.

Les résultats détaillés et graphiques sont dans `node_server/tests/results_*/`.
### Fichiers

```
node_server/tests/
├── benchmark.sh       # Lance serveur + receiver + relais + chaos (kill aléatoire)
├── full_bench.sh      # Compare avec/sans retry pour UNE config
├── run_all.sh         # Lance toutes les configs définies
├── run_analysis.sh    # Extrait les RESULT des logs et lance l'analyse
├── analyze_bench.py   # Parse les logs, génère rapport texte + 5 graphiques PNG
└── logs/              # Logs temporaires des nœuds
```

### Comment lancer un test manuel

Terminal 1 — infra + chaos :
```bash
cd node_server/tests
./benchmark.sh 5 3 5
# Args: <nb_relais> <kill_interval> <dead_time>
# Le script affiche l'adresse du receiver
```

Terminal 2 — sender :
```bash
cd node_server/node
go run main.go mon-node
# Tape: BENCH:<nbr_messages>:<nbr_relays>:<maxRetries>:<ip>:<port>
```

### Comment lancer un benchmark automatisé

Une config :
```bash
cd node_server/tests
./full_bench.sh 10 3 5
# Fait 2 runs : avec retry (maxRetries=3) et sans (maxRetries=1)
# Génère rapport + graphiques dans un dossier
```

Avec multi-sender et latence simulée :
```bash
SIM_LATENCY=80 NBR_SENDERS=3 MAX_KILLS=2 ./full_bench.sh 20 3 8
```

Toutes les configs :
```bash
./run_all.sh
# Modifie le tableau CONFIGS dans run_all.sh pour changer les scénarios
```

### Format des configs dans run_all.sh

```
"relais kill dead max_kills senders messages latency"
   │      │    │      │       │       │        │
   │      │    │      │       │       │        └─ latence simulée par hop (ms, 0 = off)
   │      │    │      │       │       └────────── messages par sender
   │      │    │      │       └────────────────── senders en parallèle
   │      │    │      └────────────────────────── relais tués par cycle
   │      │    └───────────────────────────────── secondes qu'un relai reste mort
   │      └────────────────────────────────────── intervalle entre les kills (s)
   └───────────────────────────────────────────── nombre de relais dans le réseau
```

### Sortie d'un test

Chaque config génère un dossier avec :
```
10r_3k_5d_2mk_3s_80ms/
├── report.txt                  # Tableau comparatif texte
├── sender_with_retry_results.log
├── sender_without_retry_results.log
├── 01_delivery_rate.png        # Taux de livraison (barres)
├── 02_results_breakdown.png    # Répartition ACK/Timeout/Abandon
├── 03_latency_boxplot.png      # Distribution des latences
├── 04_latency_comparison.png   # Latence moyenne vs P95
└── 05_avg_retries.png          # Nombre moyen de retries
```

---

## Avant de merge

- [ ] On merge les scripts de test dans le repo ou on les garde séparés ?
- [ ] Le `SIM_LATENCY` dans Node.go reste ? (inactif par défaut, 0 coût si pas utilisé)
- [ ] Le `BENCH` command dans main.go reste ? (utile pour les démos)