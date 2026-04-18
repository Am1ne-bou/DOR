# P1 : Système de Clustering Dynamique par Scores

L'objectif est de regrouper les nœuds du réseau en **clusters** de sorte que chaque étape du circuit oignon soit fiable de manière statistique, cela malgrè la présence éventuelle d'appareils volatiles.

## 1. Le modèle de scoring

Pour cela, chaque nœud calcule et transmet deux métriques fondamentales qui sont en fait deux scores. Ces scores sont normalisés entre **0 et 100** afin de permettre une évolution future (au début les scores seront basés sur les profils docker puis éventuellement basés sur des mesures réelles).

### 1.1 Score disponibilité (\(S_a\))

Il représente grossièrement la probabilité que l'équipement reste en ligne, nous fixeront de manière arbitraire (cela nous semble pertinent) les scores suivant :

- Serveur : `90 - 100`
- Laptop : `50 - 70`
- Smartphone : `20 - 40`

### 1.2 Score réseau (\(S_n\))

Qualité de la liaison (quantifié via le débit, la latence, la perte de paquet), nous fixeront de manière arbitraire (cela nous semble pertinent) les scores suivant :

- Filaire / Fibre : `90 - 100`
- WiFi 6 / 5G : `70 - 80`
- 2G / EDGE : `10 - 20`

### 1.3 Score global du nœud (\(S_{node}\))

Le calcul se fait de la manière suivante :
\[
S_{node} = (w_1 \cdot S_a) + (w_2 \cdot S_n)
\]
où \(w_1\) et \(w_2\) sont des coefficients de pondération ajustables selon les besoins du réseau et a des fins de tests.

## 2. Protocole de communication (évolutivité)

Pour assurer la transition entre simulation et réalité, le message d'initialisation devient dorénavent une sorte de "vecteur de données".

### 2.1 Enregistrement (nœud → serveur)

`INIT:<ID>:<PORT>:<PUB_KEY>:<S_a>:<S_n>`

Le serveur d'autorité stocke ces valeurs en base de données. Attention néanmoins, le rôle du serveur s'arrête là et il ne va en aucun cas les interpréter (sinon on a une centralisation qui commence a être contre le principe du sujet).

### 2.2 Récupération (sender → serveur)

`GET_LIST` renvoie désormais : `LIST:<ID>|<IP>|<PORT>|<KEY>|<S_a>|<S_n>,...`

## 3. Algorithme de clustering (côté sender)

C'est ici que l'intelligence est centralisée (au niveau des noeuds). Ainsi, le **sender** devient l'architecte de la route en suivant une logique de remplissage décrite ci-dessous.

### 3.1 Paramètres de contrôle

Cela se manifeste par trois paramètres avec lesquels on jouera pour voir comment le réseau se comporte :

- **MinClusters** : minimum `3` pour des communications anonymes.
- **MaxNodesPerCluster (\(K\))** : limite pour éviter les oignons trop lourds
- **TargetClusterScore** : score cumulé minimal pour valider un cluster

### 3.2 Logique de distribution (hétérogène)

1. **Le tri**  
   On trie tous les nœuds disponibles par \(S_{node}\) décroissant.

2. **L'ancrage**  
   On sélectionne les \(N\) meilleurs nœuds (généralement il s'agit des "serveurs") pour servir d'ancres. 
   → Une ancre par cluster, nous en reparlerons plus bas.

3. **Le remplissage par le bas**  
   Pour chaque nœud restant, on affecte ce nœud au cluster ayant actuellement le **score cumulé le plus faible**.

4. **Conditions d'arrêt**  
   Le remplissage d'un cluster s'arrête si l'une des conditions suivantes est atteinte :
   - Condition 1 : le score total du cluster atteint `TargetClusterScore`
   - Condition 2 : le nombre de nœuds atteint `MaxNodesPerCluster`

## 4. Matérialisation technique

### 4.1 Côté SQLManager

- Ajout des colonnes :
  - `availability_score`
  - `network_score`
- Mode **WAL** activé pour gérer les écritures concurrentes lors des tests massifs

### 4.2 Côté `model/Broadcast.go`

- Utilisation inchangée du format :
  `K1;K2;...:Payload`
- La taille de la liste des clés (\(K\)) est désormais bornée par `MaxNodesPerCluster`

### 4.3 Côté `node/super_send.go`

- La fonction `PickLayer` évolue en : `BuildSmartClusters`
- Elle implémente l'algorithme de distribution décrit plus bas

## 5. Résilience et livraison

Prenons l'exemple d'un cluster composé d'un serveur (**l'ancre**) et de 2 smartphones (**volatiles**)

### 5.1 Fonctionnement

- Le message est chiffré pour les **3 clés**
- Le nœud précédent tente d'envoyer à **l'un des trois**

### 5.2 Cas de défaillance

- Si les smartphones sont déconnectés :
  - l'ancre reçoit le message
  - elle assure la continuité vers le cluster suivant

→ Le score du cluster reflète cette "assurance de survie".

-----

# P2 : Stratégie d'ancrage dynamique

L'objectif est de distribuer la **stabilité relative** équitablement entre les clusters.

## 1. Définition de l'ancre

On définit comme **ancre** les \(N\) nœuds possédant les meilleurs scores globaux \(S_{node}\), quel que soit leur type :

- Serveur
- Laptop
- Smartphone

## 2. Distribution

- Une ancre est placée **prioritairement dans chaque cluster**

## 3. Gestion du cas pessimiste "No-High-Score"

Si le score de l'ancre est inférieur à un seuil de sécurité (par ex : \(S_{node} < 50\)), le système bascule en mode **Stacking compensatoire**.

Le rôle de l'ancre devient alors **purement structurel** (premier nœud du cluster), mais l'algorithme ne compte plus sur elle pour assurer seule la survie.

**Conséquence :**  
l'étape de remplissage devra ajouter davantage de nœuds volatiles pour atteindre le `TargetClusterScore`.

-----

# P3 : Algorithme de remplissage et compensation

L'algorithme doit être capable de s'adapter à la **pauvreté du réseau**.

## 1. Distribution des noeuds

On distribue les noeuds restants vers le cluster ayant le **score le plus faible**.

## 2. Boucle de compensation

Si un cluster n'atteint pas le `TargetClusterScore` avec son ancre, il continue de piocher des nœuds jusqu'à :

- atteindre le score cible
- ou atteindre `MaxNodesPerCluster`

## 3. Arbitrage final

Si, malgré le remplissage maximal, un cluster reste sous un seuil critique, le sender fonctionnera en **mode dégradé**.
Cela signifie qu'il va envoyer quand même le message (priorité à la connectivité), mais avertira l'utilisateur par exeple avec un message du genre "Attention, route construite avec un score de fiabilité de seulement $30\%$"