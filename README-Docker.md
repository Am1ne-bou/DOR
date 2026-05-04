# DOR - Guide de lancement Docker

## Prérequis

- [Docker Desktop](https://www.docker.com/products/docker-desktop) installé et lancé
- [Go 1.25.6+](https://go.dev/dl/)

## Lancement rapide

### Windows
```powershell
cd Projet-DOR
.\start-dor.ps1
```

### Linux
```bash
cd Projet-DOR
chmod +x start-dor.sh
./start-dor.sh
```

### macOS
```bash
cd Projet-DOR
chmod +x start-dor-mac.sh
./start-dor-mac.sh
```

4 terminaux s'ouvrent automatiquement : 1 directory server + 3 nœuds.

## Lancement manuel (si le script ne marche pas)

### 1. Builder + lancer

```bash
docker compose up -d
```

### 2. Vérifier que tout tourne

```bash
docker compose ps
docker compose logs directory
```

Les 3 nœuds doivent apparaître comme "registered" dans les logs du directory.

### 3. Ouvrir les terminaux (un par nœud)

```bash
docker attach node1
docker attach node2
docker attach node3
docker attach directory
```

> **Important** : pour détacher un terminal sans tuer le conteneur → `Ctrl+P` puis `Ctrl+Q`

### 4. Récupérer les IP des nœuds

```bash
docker inspect node1 --format "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}"
docker inspect node2 --format "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}"
docker inspect node3 --format "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}"
```

## Commandes dans un nœud

| Commande | Exemple |
|----------|---------|
| Message direct | `MSG:<ip>:<port>:<message>` |
| Relay multi-hop | `RELAY:<ip>:<port>,<ip>:<port>,<message>` |
| Envoi auto + ACK | `SEND:<nb_relais>:<ip>:<port>:<message>` |
| Liste des nœuds | `LIST` |

Exemple avec node1 (172.18.0.5) qui envoie via node2 (172.18.0.4) vers node3 (172.18.0.3) :

```
RELAY:172.18.0.4:9002,172.18.0.3:9003,Hello en oignon
```

## Profils réseau des nœuds

Définis dans `docker-compose.yml` :

| Nœud | Profil | Débit | Latence |
|------|--------|-------|---------|
| node1 | laptop_WIFI5 | 200 Mbps | 10ms |
| node2 | smartphone_4G | 20 Mbps | 50ms |
| node3 | laptop_WIFI6 | 400 Mbps | 8ms |

Modifiable dans `docker-compose.yml` → variable `NETWORK_PROFILE`.

Profils disponibles : `smartphone_2G`, `smartphone_EDGE`, `smartphone_3G`, `smartphone_4G`, `smartphone_5G`, `laptop_WIFI3`, `laptop_WIFI4`, `laptop_WIFI5`, `laptop_WIFI6`, `laptop_WIFI7`, `server`.

## Tout arrêter

```bash
docker compose down
```
