# Grimhold

Un jeu de donjon en terminal écrit en Go, avec six régions, des combats aux dés et des monstres récupérés depuis Open5e. Le projet utilise uniquement la bibliothèque standard.

## Lancer le jeu

Il faut Go 1.22 ou plus récent et une connexion Internet pour consulter Open5e ou rencontrer des monstres. Les déplacements, le personnage et les sauvegardes fonctionnent sans connexion.

Depuis le dossier `grimhold` :

```powershell
go run ./cmd/grimhold
```

Pour compiler et lancer sous Windows :

```powershell
go build -o grimhold.exe ./cmd/grimhold
.\grimhold.exe
```

Sous Linux ou macOS :

```sh
go build -o grimhold ./cmd/grimhold
./grimhold
```

Lancer le programme depuis le dossier contenant `world.json`.

## Jouer

`help` affiche toutes les commandes à partir de leur map d'enregistrement.

| Commande | Utilisation |
| --- | --- |
| `map` | Afficher les régions, leur danger et leurs sorties |
| `travel ashen-marsh` | Aller dans une région reliée à la région actuelle |
| `look` | Observer les lieux et le monstre présent |
| `explore` | Rencontrer un monstre choisi parmi ceux de la région |
| `fight` | Jouer un round : votre attaque, puis celle du monstre survivant |
| `flee` | Essayer de fuir le combat |
| `character` | Consulter les PV, l'XP, le niveau, l'arme et l'or |
| `rest` | Récupérer tous les PV, une fois par visite |
| `bestiary` | Afficher la première page des monstres Open5e |
| `next` / `prev` | Suivre la pagination du bestiaire |
| `inspect goblin` | Consulter un monstre avec son identifiant Open5e |
| `slain` | Afficher les monstres vaincus et leur nombre |
| `save` / `load` | Sauvegarder ou charger la partie |
| `quit` | Quitter |

Un `fight` joue un seul round. Répéter la commande pour continuer, ou utiliser `flee` entre deux rounds. L'attaque utilise `1d20 + bonus` contre l'armure. L'épée inflige `1d8 + 3`. Le monstre utilise la première action de sa fiche contenant des dés utilisables, avec son bonus d'attaque et ses dégâts Open5e.

La fuite réussit avec `1d20 + 2 >= 12`. Un échec donne une attaque gratuite au monstre et le combat continue. Après une défaite, le personnage revient au Carrefour avec 1 PV et un repos disponible.

Le danger d'une région détermine le nombre de tirages de monstres : le jeu conserve celui dont l'indice de danger est le plus élevé. Une victoire donne `indice de danger × 100` XP, avec un minimum de 10, et de 1 à 6 pièces d'or. Le prochain niveau demande `niveau × niveau × 50` XP cumulés. Chaque niveau ajoute 8 PV maximum et 1 au bonus d'attaque, puis restaure les PV.

## Sauvegarder

`save` écrit `save.json` dans le dossier courant. La sauvegarde contient le personnage, la région, l'or, les monstres vaincus, le repos et le combat en cours. Un fichier temporaire est écrit avant de remplacer la sauvegarde existante. Une défaite ne déclenche aucune écriture automatique.

`load` restaure la partie. Une sauvegarde absente ou invalide affiche une erreur et conserve la partie actuelle. Pour reprendre au démarrage :

```powershell
go run ./cmd/grimhold --continue
```

Il faut utiliser `save` avant de quitter pour conserver sa progression.

## Exemple de session

```text
grimhold > travel ashen-marsh
Le Marais Cendré (ashen-marsh)
Le brouillard sent le fer. Quelque chose respire sous l'eau.
Danger : 1/5
Sorties : crossroads, drowned-chapel
grimhold > rest
Vous retrouvez tous vos points de vie.
grimhold > rest
Erreur : vous avez déjà utilisé votre repos pour cette visite
grimhold > save
Partie sauvegardée.
grimhold > load
Partie chargée.
Le Marais Cendré (ashen-marsh)
Le brouillard sent le fer. Quelque chose respire sous l'eau.
Danger : 1/5
Sorties : crossroads, drowned-chapel
grimhold > quit
À bientôt.
```

## Organisation

| Dossier | Rôle |
| --- | --- |
| `cmd/grimhold` | Démarrage et option `--continue` |
| `internal/repl` | Lecture et exécution des commandes |
| `internal/game` | Personnage, rencontres, combats et sauvegardes |
| `internal/open5e` | Client HTTP, structures JSON et cache |
| `internal/world` | Chargement et validation de la carte |

Les notions reprises du cours sont les variables, conditions, boucles, fonctions, slices, maps, pointeurs, structures, méthodes, interfaces, erreurs, packages, goroutines, channels et mutex. L'interface `MonsterSource` permet au jeu d'utiliser le client Open5e et aux tests de fournir des monstres sans réseau.

Le cache est une map protégée par un mutex. Une réponse reste valable cinq minutes. Une goroutine vérifie les expirations chaque minute, avec un ticker et un `select`. Un channel arrête cette goroutine à la fermeture du jeu.

Les données viennent de [l'API Open5e](https://api.open5e.com/v1/monsters/). Les identifiants sont ceux de l'API, par exemple `goblin` et `skeleton`. Une erreur réseau est affichée et le joueur peut réessayer sa commande.

## Vérifications

```powershell
go fmt ./...
go vet ./...
go test ./...
```

Bonus réalisé : suite de tests sans accès réseau. Elle vérifie notamment la victoire et la montée de niveau, la défaite, les déplacements et le repos, la sauvegarde complète, le rejet d'un chargement invalide, le décodage HTTP avec `httptest.Server`, la pagination et l'expiration du cache. Les combats de test utilisent des caractéristiques garantissant le résultat malgré les tirages aléatoires.
