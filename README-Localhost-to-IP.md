# Transition localhost → Réseau réel (IP) — Compatible TLS

## Résumé

Cette branche transforme le projet DOR pour fonctionner sur un **réseau réel entre plusieurs machines**, tout en conservant le **chiffrement TLS** pour les communications nœud↔serveur.

Testé et validé entre une machine Windows (192.168.1.71) et une VM Linux VirtualBox (192.168.1.90).

---

## Ce qui a changé et pourquoi

### 1. Adresse du serveur configurable (`SERVER_ADDR`)

**Problème** : l'adresse `localhost:8080` était écrite en dur partout. Si le serveur tourne sur une autre machine, les nœuds ne peuvent pas le trouver.

**Solution** : les nœuds lisent la variable d'environnement `SERVER_ADDR` au démarrage. Si elle n'est pas définie, la valeur par défaut reste `localhost:8080`.

**Fichiers modifiés** : `node_server/node/main.go`, `node_server/model/Node.go` (ajout du champ `ServerAddr` dans la struct `Node`)

### 2. Identification des nœuds par ip:port

**Problème** : les nœuds étaient identifiés par leur port uniquement (ex: `4567`). Pour communiquer, on faisait `localhost:4567`. Impossible de joindre un nœud sur une autre machine.

**Solution** : partout dans le code, le simple port `int` a été remplacé par une adresse complète `string` au format `ip:port` (ex: `192.168.1.90:4567`).

**Impact** :
- `SendTo(targetPort int)` → `SendTo(targetAddr string)`
- `publicKeys map[int]` → `publicKeys map[string]`
- Format RELAY : `RELAY:<port>:<payload>` → `RELAY:<ip>:<port>:<payload>`
- Format liste serveur : `name|port|key` → `name|ip|port|key`
- Commande RELAY : séparateur `:` → séparateur `,` (car `ip:port` contient déjà un `:`)
- GET_KEY : recherche par `ip + port` au lieu de `port` seul

### 3. Écoute sur toutes les interfaces (0.0.0.0)

**Problème** : par défaut, un programme peut n'écouter que sur localhost, refusant les connexions d'autres machines.

**Solution** : le serveur et les nœuds écoutent explicitement sur `0.0.0.0` avec le protocole `tcp4` (IPv4 uniquement pour éviter les problèmes avec les adresses IPv6 comme `::1`).

**Fichiers modifiés** : `node_server/List_Serveur/serveur.go` (`tls.Listen("tcp4", "0.0.0.0:8080", config)`), `node_server/node/main.go` (`net.Listen("tcp4", addr)`)

### 4. Détection automatique de l'IP du nœud (INIT_ACK)

**Problème** : un nœud ne connaît pas sa propre adresse IP vue de l'extérieur. Il en a besoin pour s'exclure des routes aléatoires dans la commande SEND.

**Solution** : quand un nœud s'enregistre, le serveur voit son IP via `conn.RemoteAddr()` et la lui renvoie dans `INIT_ACK:<nom>:<ip>`. Le nœud la stocke dans le champ `NodeIP`.

**Fichiers modifiés** : `node_server/List_Serveur/serveur.go` (réponse INIT_ACK enrichie), `node_server/model/Node.go` (parsing de l'IP + champ `NodeIP`)

### 5. Compatibilité TLS

**Problème** : le passage en TLS (commit précédent) utilisait un certificat vérifiant `ServerName: "localhost"`. En réseau réel avec des IP, la vérification échoue.

**Solution** : `InsecureSkipVerify: true` dans la config TLS du client. C'est un compromis temporaire — en production, il faudrait générer un certificat avec les bonnes IP dans les SAN (Subject Alternative Names).

**Fichier modifié** : `node_server/model/Node.go` (fonction `DialDirectoryServer`)

### 6. Format des messages avec UUID

Chaque message envoyé contient un UUID unique pour identifier les messages. Le format est `MSG:<uuid>:<message>`. Le handler MSG dans `Node.go` parse ce format pour afficher le UUID et le message séparément.

### 7. Scripts de lancement

Les scripts `start_node.sh` et `start_nodes_tmux.sh` acceptent la variable `SERVER_ADDR` et la passent aux nœuds.

---

## Guide d'utilisation

### En local (rien ne change)

```bash
./start_nodes_tmux.sh -n 4
```

### Entre plusieurs machines

#### Étape 1 — Trouver les adresses IP

Sur Windows :
```
ipconfig
```
Chercher l'adresse IPv4 sous Wi-Fi ou Ethernet (ex: `192.168.1.71`).

Sur Linux :
```bash
hostname -I
```

#### Étape 2 — Configuration pare-feu (Windows)

Ouvrir PowerShell **en administrateur** :

```powershell
New-NetFirewallRule -DisplayName "Allow ICMP" -Protocol ICMPv4 -Action Allow
New-NetFirewallRule -DisplayName "Allow DOR Server" -Protocol TCP -LocalPort 8080 -Action Allow
New-NetFirewallRule -DisplayName "Allow DOR Nodes" -Protocol TCP -LocalPort 49000-65535 -Action Allow
```

#### Étape 3 — Vérifier la connectivité

Depuis chaque machine :
```bash
ping 192.168.1.71
```

#### Étape 4 — Lancer le serveur

Sur la machine serveur (ex: `192.168.1.71`) :
```bash
cd node_server/List_Serveur
go run serveur.go
```

#### Étape 5 — Lancer les nœuds

Sur Windows (PowerShell) :
```powershell
cd node_server\node
$env:SERVER_ADDR="192.168.1.71:8080"
go run main.go node-1
```

Sur Linux :
```bash
cd node_server/node
SERVER_ADDR=192.168.1.71:8080 go run main.go node-2
```

Chaque nœud affiche son IP détectée :
```
[node-2] Registered to directory server (IP: 192.168.1.90)
```

#### Étape 6 — Vérifier l'enregistrement

Dans le terminal serveur, taper `LIST` :
```
=== Noeuds connectés ===
  . node-1 - Addr: 192.168.1.71:56814, Key: MIIBIjAN...
  . node-vm - Addr: 192.168.1.90:45253, Key: MIIBIjAN...
Total: 2
```

### Commandes

```
MSG:<ip>:<port>:<message>                     Message direct chiffré
RELAY:<ip>:<port>,<ip>:<port>,...,<message>    Route manuelle (virgule entre les adresses)
SEND:<nbr_relais>:<ip>:<port>:<message>       Route aléatoire automatique
FETCH:<ip>:<port>                             Récupérer la clé publique d'un nœud
LIST:                                         Afficher les nœuds enregistrés
QUIT:                                         Quitter proprement
```

### Exemple de test avec VirtualBox

**Configuration VirtualBox** :
1. Éteindre la VM
2. Configuration → Réseau → Adaptateur 1 → Mode : **Accès par pont** (Bridged Adapter)
3. Sélectionner la carte Wi-Fi
4. Démarrer la VM

**Machine Windows (192.168.1.71)** :
```powershell
# Terminal 1 - Serveur
cd node_server\List_Serveur
go run serveur.go

# Terminal 2 - Node-1
cd node_server\node
$env:SERVER_ADDR="192.168.1.71:8080"
go run main.go node-1

# Terminal 3 - Node-2
cd node_server\node
$env:SERVER_ADDR="192.168.1.71:8080"
go run main.go node-2
```

**VM Linux (192.168.1.90)** :
```bash
cd Projet-DOR/node_server/node
SERVER_ADDR=192.168.1.71:8080 go run main.go node-vm
```

**Tests validés** :
```
# Message Windows → VM
MSG:192.168.1.90:45253:Hello depuis Windows !

# Message VM → Windows
MSG:192.168.1.71:56814:Hello depuis la VM !

# Relai manuel
RELAY:192.168.1.71:52896,192.168.1.90:45253,Message via relai

# Route automatique (1 relai)
SEND:1:192.168.1.71:56814:Message auto-routé cross-machine
```

---

## Prochaines étapes possibles

- **Certificat TLS propre** : générer un certificat avec les IP dans les SAN au lieu de `InsecureSkipVerify`
- **Port fixe configurable** : variable `NODE_PORT` pour éviter d'ouvrir une large plage de ports
- **Démonstrateur** : interface web ou Android
- **Évaluation** : tests de résistance aux attaques et performance
