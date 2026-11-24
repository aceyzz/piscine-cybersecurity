<img title="42_piscine-cybersecurity-arachnida" alt="42_piscine-cybersecurity-arachnida" src="assets/banner.png" width="100%">

<br>

# Arachnida

Ce projet a pour but de développer deux programmes, `spider` et `scorpion`, qui permettent de télécharger des images à partir d'une page web puis d'examiner et gérer les metadata EXIF de ces images, respectivement.

- [spider](#programme-spider)    
- [scorpion](#programme-scorpion)  
- [Utilisation](#utilisation)    
- [Statut](#statut)  

<br>

## Programme `spider`

Le programme `spider` télécharge les images d'une page web, puis celles des pages liées jusqu'à une profondeur maximale.

```bash
./spider [-r] [-l N] [-p PATH] URL
```
`-r` : téléchargement récursif des images des pages liées  
`-l N` : profondeur maximale, nécessite `-r` (défaut 5)  
`-p PATH` : dossier de sortie (défaut `./data/`)  
`-h` : aide  

Extensions compatibles : `.jpg`, `.jpeg`, `.png`, `.gif`, `.bmp`.

<details>
<summary>Fonctionnement</summary>

### Arguments (`main.go`)
`parseArgs` lit `-r`, `-l` et `-p`, puis l'URL  
Règles validées : une seule URL, `-l` seulement avec `-r`, profondeur positive, URL en `http` ou `https`  
Sans `-r`, la profondeur vaut 0 : seule la page donnée est traitée  
`-h` affiche l'aide (code 0). Une erreur affiche l'usage (code 2)  
Le dossier de sortie est créé s'il n'existe pas  

### Parcours (`crawler.go`)
Une file de pages démarre avec l'URL de départ, au niveau 0  
`run` traite les pages une par une, dans l'ordre de la file  
`visit` récupère une page, télécharge ses images, puis ajoute ses liens à la file tant que la profondeur maximale n'est pas atteinte  
`nextTargets` ne garde que les liens du même hôte, et ignore les liens déjà vus  
`seenPages` empêche de traiter deux fois une page et évite les boucles  

### Extraction (`crawler.go`)
`fetch` fait un GET avec un User-Agent et un timeout de 15 s. Tout statut autre que 200 est une erreur  
`scrape` ignore les réponses qui ne sont pas en `text/html` et limite la lecture à 10 Mo  
`extractLinks` parcourt le HTML balise par balise et relève `img src` et `a href`  
`resolveURL` convertit les liens relatifs en URL absolues et retire l'ancre `#`  
Un lien qui finit par une extension d'image va dans la liste des images. Un autre lien `<a>` va dans la liste des pages  

### Téléchargement (`crawler.go`)
`seenImages` empêche de télécharger deux fois une même URL  
`download` vérifie que la réponse a un type `image/`  
`uniquePath` ajoute un suffixe `_N` si le nom de fichier existe déjà  
`writeFile` crée le fichier sans écraser l'existant, et le supprime si l'écriture échoue  
Chaque fichier enregistré est affiché, puis le total  

### Méthode de scraping

Parcours en largeur (BFS), limité en profondeur : toutes les pages d'un niveau sont traitées avant le niveau suivant  
Le HTML est découpé balise par balise au fil de la lecture avec le tokenizer de `golang.org/x/net/html`, sans arbre DOM ni sélecteurs  
Seul ce découpage est délégué à une bibliothèque. Parcours, filtrage, déduplication et enregistrement sont codés à la main  
Les requêtes passent l'une après l'autre, sans parallélisme  

</details>

<br>

## Programme `scorpion`

Le programme `scorpion` lit les fichiers image et affiche leurs métadonnées EXIF, ainsi que les attributs de base du fichier.

```bash
./scorpion FILE1 [FILE2 ...]
```
`-h` : aide  

Formats compatibles : JPEG, PNG, GIF, BMP (détection par signature, indépendante de l'extension).

<details>
<summary>Fonctionnement</summary>

### Arguments (`main.go`)
`parseArgs` lit les options, puis la liste des fichiers  
Au moins un fichier est requis  
`-h` affiche l'aide (code 0). Une erreur d'argument affiche l'usage (code 2)  
`showAll` traite les fichiers un par un. Une erreur sur un fichier n'arrête pas les suivants (code 1 en fin d'exécution)  

### Rapport (`report.go`)
`buildReport` refuse ce qui n'est pas un fichier régulier, lit le fichier et détecte son format  
`detectFormat` compare les premiers octets aux signatures JPEG, PNG, GIF87a, GIF89a et BMP  
`fileSection` donne les attributs de base : nom, format, taille, permissions, dates de création et de modification  
`birthTime` lit la date de création avec `statx` sous Linux (`birthtime_linux.go`) et `Birthtimespec` sous macOS (`birthtime_darwin.go`)  
`printReport` affiche une section par bloc de métadonnées. Les valeurs longues sont tronquées à l'affichage  

### JPEG (`jpeg.go`)
`jpegSegments` parcourt les segments jusqu'au début des données image (SOS) ou à la fin du fichier (EOI)  
SOF : encodage, dimensions, bits par composante, nombre de composantes  
APP0 : version JFIF et densité  
APP1 : EXIF, transmis au parseur TIFF, et taille du bloc XMP  
COM : commentaire  

### PNG (`png.go`)
`pngChunks` parcourt les chunks jusqu'à `IEND`  
IHDR : dimensions, profondeur, type de couleur, entrelacement  
pHYs : densité de pixels. tIME : date de dernière modification  
tEXt, zTXt, iTXt : textes clé/valeur, avec décompression zlib limitée à 1 Mo  
eXIf : EXIF, transmis au parseur TIFF  

### GIF et BMP (`gif.go`, `bmp.go`)
GIF : version, dimensions, palette globale, nombre de frames, commentaires  
BMP : taille déclarée, offset des pixels, type d'en-tête DIB, dimensions, ordre des lignes, bits par pixel, compression, résolution  

### EXIF (`exif.go`, `tags.go`)
`newTIFFReader` lit l'ordre des octets (`II` ou `MM`) et l'offset du premier IFD  
`walk` lit un IFD et suit les pointeurs vers SubIFD, GPS et Interoperability, puis lit IFD1 (miniature)  
`readEntry` décode chaque entrée : tag, type, nombre de valeurs, valeur stockée sur place ou à un offset  
`formatValue` met en forme les valeurs : texte, entiers, rationnels, flottants, octets bruts, UserComment  
`tags.go` associe chaque identifiant de tag à son nom. Un tag inconnu s'affiche en hexadécimal  

### Robustesse
Chaque lecture vérifie les bornes avant d'accéder aux données  
`seen` empêche de relire un IFD déjà visité, ce qui protège des fichiers piégés en boucle  
`sanitize` remplace les caractères de contrôle, contre l'injection de séquences dans le terminal  
Un fichier tronqué ou malformé produit un rapport partiel, sans arrêt du programme  

### Méthode de parsing

Le fichier est lu entièrement en mémoire, puis découpé selon la structure binaire de son format  
Aucune bibliothèque de métadonnées : JPEG, PNG, GIF, BMP et TIFF/EXIF sont décodés à la main avec `encoding/binary`  
Seules la décompression zlib (`compress/zlib`) et la lecture de la date de création (`golang.org/x/sys/unix`) reposent sur une bibliothèque  

</details>

<br>

## Utilisation

### Prérequis
Compilation : `go` (version 1.26 ou plus, voir `go.mod`) et `make`  
Tests : `bash`, `python3`, `curl`, `imagemagick`, `exiftool`  

Installation des outils de test sous Ubuntu/WSL :
```bash
sudo apt install -y python3 curl imagemagick libimage-exiftool-perl
```

Installation des outils de test sous macOS (Homebrew) :
```bash
brew install python curl imagemagick exiftool
```

### Compilation
Depuis le dossier `project/` :
```bash
make
```
Compile les deux programmes et copie `spider` et `scorpion` dans `project/`  
Au premier lancement, `go mod init` et `go mod tidy` sont exécutés si `go.mod` ou `go.sum` manquent (réseau requis)  

Compilation d'un seul programme :
```bash
make -C 00-spider
make -C 01-scorpion
```

### Règles Makefile
`make` : compile les deux programmes  
`make clean` : supprime les binaires des sous-dossiers  
`make fclean` : supprime tous les binaires et le dossier `data`  
`make re` : `fclean` puis `make`  
`make test` : compile puis lance les deux batteries de tests  

### Lancement
```bash
./spider -r -l 2 -p ./data https://scrapepark.org
./scorpion data/*.jpg data/*.png
```

### Tests
Depuis le dossier `project/` :
```bash
make re
make test
```
Une seule batterie :
```bash
make -C 00-spider test
make -C 01-scorpion test
```

`tests/spider.sh` génère un mini site local, servi par `python3 -m http.server` sur un port aléatoire. Les tests couvrent les arguments, le téléchargement, les extensions, la déduplication, le renommage, la profondeur, les liens externes et les boucles  
`tests/scorpion.sh` génère des images avec ImageMagick et exiftool. Les tests couvrent les arguments, les attributs de fichier, les formats JPEG, PNG, GIF et BMP, l'EXIF comparé à exiftool, et les fichiers malformés  
Chaque test affiche la commande, le résultat attendu et le résultat obtenu, puis OK ou KO  
Code de sortie : 0 si tous les tests passent, 1 sinon  
Les fichiers temporaires sont supprimés en fin d'exécution

<br>

## Statut

> En attente de correction  
<!-- <img src="assets/100.png" alt="Grade" width="150"> -->

