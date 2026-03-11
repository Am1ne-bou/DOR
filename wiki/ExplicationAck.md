# feature/ack_return_route — Système ACK / Route retour

## Résumé

Cette branche ajoute un système d'accusé de réception (ACK) qui permet à l'expéditeur de savoir si son message a bien été reçu par le destinataire. Le destinataire renvoie un ACK chiffré via une **route retour** (inverse de la route forward).

## Fichiers modifiés / ajoutés

### `OnionLayer.go` (NOUVEAU)
Structure `OnionLayer` qui remplace le parsing manuel des messages. Chaque couche contient :
- `Type` : `RELAY`, `FINAL`, ou `ACK`
- `MsgID` : identifiant unique du message (ex: `msg-482910`)
- `NextHop` : adresse du prochain nœud (pour RELAY)
- `ReturnAddr` / `ReturnData` : adresse et données chiffrées pour la route retour (pour FINAL)
- `Message` : le message en clair (pour FINAL)
- `Payload` : la couche chiffrée suivante (pour RELAY)

Sérialisation via `|` comme séparateur (`OnionlayerToString` / `StringToOnionLayer`).

### `Node.go`
- Ajout de `PendingACKs map[string]chan bool` + `sync.Mutex` pour gérer l'attente des ACKs
- `handlerroutine` utilise maintenant `OnionLayer` au lieu du parsing `MSG:`/`RELAY:` brut
- Gestion de 3 types de couches :
    - **RELAY** → forward vers `NextHop`
    - **FINAL** → affiche le message + envoie le ACK via `ReturnAddr` si présent
    - **ACK** → notifie le canal `PendingACKs[msgID]`

### `main.go`
- `Encapsulator_func` prend maintenant une `returnRoute []string` (nil = pas de ACK)
- Retourne `(onion, msgID, error)` au lieu de `(onion, error)`
- Construction de 2 oignons :
    1. **Return onion** : couches chiffrées pour la route retour, couche interne = `ACK`
    2. **Forward onion** : couche interne = `FINAL` avec `ReturnAddr` + `ReturnData` embarqués
- Nouvelle fonction `encryptOnionLayers` (factorise le chiffrement couche par couche)
- Nouvelle fonction `encryptForNode` (AES + RSA pour un nœud)
- Commande `SEND` : construit automatiquement la route retour (inverse des relais), attend le ACK avec timeout de 10s

## Comment ça marche

```
Sender → R1 → R2 → Dest (FINAL: affiche msg, envoie ACK)
                       ↓
                      R2 → R1 → Sender (ACK: confirme réception)
```

1. `SEND:2:ip:port:hello` construit une route forward `[R1, R2, dest]` et une route retour `[R2, R1, sender]`
2. Le message est encapsulé avec les 2 oignons (forward + return)
3. Le destinataire reçoit le `FINAL`, lit le message, et envoie le ACK chiffré via la route retour
4. L'expéditeur reçoit le `ACK` et affiche la confirmation (ou timeout après 10s)

## Commandes impactées

| Commande | ACK ? | Remarque |
|----------|-------|----------|
| `MSG`    | Non   | Envoi direct, pas de route retour |
| `RELAY`  | Non   | Route manuelle, pas de route retour |
| `SEND`   | **Oui** | Route auto + route retour + attente ACK |

