# Construire l'image docker

```bash
docker build -t device-amd <path_of_Dockerfile>
```

# Lancer le conteneur

## Liste des profils disponibles

| Nom   |
|---------|
| laptop_WIFI3 |
| laptop_WIFI4 |
| laptop_WIFI5 |
| laptop_WIFI6 |
| laptop_WIFI7 |
| serveur |
| smartphone_2G |
| smartphone_EDGE |
| smartphone_3G |
| smartphone_4G |
| smartphone_5G |
| smartphone_2G |

> [!WARNING]
> Pour le profil "serveur", aucun changement de config.

## Avec terminal
```bash
docker run -it --cap-add=NET_ADMIN -e NETWORK_PROFILE=<PROFILE> device-amd:latest       
```

## Sans terminal
```bash
docker run --cap-add=NET_ADMIN -e NETWORK_PROFILE=<PROFILE> device-amd:latest       
```

> [!NOTE]
> Exemple avec terminal et profil "smartphone_4G"
> ```bash
> docker run -it --cap-add=NET_ADMIN -e NETWORK_PROFILE=smartphone_4G device-amd:latest       
> ```
