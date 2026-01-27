# Prototype Noeud P2P - Base DOR

Prototype de noeud réseau pour le projet Dynamic Onion Routing.

## Fonctionnalités

- Envoyer/recevoir des messages directs
- Relayer des messages à travers plusieurs noeuds

## Lancer un noeud
```bash
go run main.go  
```

## Commandes
```
MSG:<port>:<message>                  Message direct
RELAY:<port>:<port>:...:<message>     Relai multi-hop
QUIT:000:                             Quitter
```

## Exemple

3 terminaux :
```bash
go run main.go node-1 9010
go run main.go node-2 9011
go run main.go node-3 9012
```

Dans node-1 :
```
MSG:9011:Hello
RELAY:9011:9012:Secret
```