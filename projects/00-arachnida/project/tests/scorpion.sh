#!/usr/bin/env bash

DIR=$(cd "$(dirname "$0")" && pwd)
. "$DIR/lib.sh"

BIN=$1
require_binary "$BIN"
require exiftool find date

if command -v magick >/dev/null 2>&1; then
	IM=magick
elif command -v convert >/dev/null 2>&1; then
	IM=convert
else
	printf '%soutils manquants : imagemagick%s\n' "$RED" "$RESET"
	exit 1
fi

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

field() {
	printf '%s\n' "$OUT" | grep -E "^  $1 " | head -n 1 | sed -E "s/^  $1 +//"
}

report_count() {
	printf '%s\n' "$OUT" | grep -c '^==> '
}

date_state() {
	case "$(field "$1")" in
		[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]\ *) echo date ;;
		*) field "$1" ;;
	esac
}

prefix() {
	local value
	value=$(field "$1")
	printf '%s' "${value:0:$2}"
}

build_images() {
	$IM -size 16x8 xc:red "$WORK/plain.jpg"
	$IM -size 16x8 xc:red -interlace JPEG "$WORK/progressive.jpg"
	$IM -size 16x8 xc:red -set comment "jpeg comment" "$WORK/comment.jpg"
	$IM -size 4x4 xc:red -set comment "$(printf '\033[31mred')" "$WORK/escape.jpg"
	cp "$WORK/plain.jpg" "$WORK/exif.jpg"
	exiftool -q -overwrite_original -Make=TestMake -Model=TestModel -Artist=Tester \
		-DateTimeOriginal="2020:01:02 03:04:05" -UserComment="hello exif" \
		-GPSLatitude=43.5 -GPSLatitudeRef=N -GPSLongitude=11.25 -GPSLongitudeRef=E "$WORK/exif.jpg"
	$IM -size 10x6 xc:blue -set comment "png comment" "$WORK/text.png"
	$IM -size 8x4 xc:green -set comment "gif comment" "$WORK/comment.gif"
	$IM -delay 10 -size 8x4 xc:red xc:blue "$WORK/anim.gif"
	$IM -size 7x5 xc:red "BMP3:$WORK/v3.bmp"
	$IM -size 7x5 xc:red "$WORK/default.bmp"
	cp "$WORK/plain.jpg" "$WORK/disguised.png"
	cp "$WORK/plain.jpg" "$WORK/perm.jpg"
	chmod 600 "$WORK/perm.jpg"
	head -c 100 "$WORK/exif.jpg" > "$WORK/truncated.jpg"
	printf 'not an image at all' > "$WORK/garbage.jpg"
	: > "$WORK/empty.png"
	printf '\xff\xd8\xff\xe1\x00\x22Exif\x00\x00II*\x00\x08\x00\x00\x00\x01\x00\x69\x87\x04\x00\x01\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\xff\xd9' > "$WORK/loop.jpg"
}

test_arguments() {
	title "Arguments"
	run "$BIN" -h
	check "scorpion -h" "exit=0, usage=yes" "exit=$CODE, usage=$(contains 'usage:')"
	run "$BIN"
	check "scorpion (aucun argument)" "exit=2, erreur=at least one file expected" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" -x "$WORK/plain.jpg"
	check "scorpion -x FICHIER (option inconnue)" "exit=2, erreur=flag provided but not defined: -x" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" "$WORK/missing.jpg"
	check "fichier inexistant" "exit=1, erreur=stat $WORK/missing.jpg: no such file or directory" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" "$WORK"
	check "dossier en argument" "exit=1, erreur=$WORK: not a regular file" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" "$WORK/garbage.jpg"
	check "contenu non image avec extension .jpg" "exit=1, erreur=$WORK/garbage.jpg: unsupported format" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" "$WORK/empty.png"
	check "fichier vide" "exit=1, erreur=$WORK/empty.png: unsupported format" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" "$WORK/plain.jpg" "$WORK/text.png" "$WORK/comment.gif"
	check "trois fichiers valides" "exit=0, rapports=3" "exit=$CODE, rapports=$(report_count)"
	run "$BIN" "$WORK/garbage.jpg" "$WORK/plain.jpg"
	check "fichier invalide puis fichier valide" "exit=1, rapports=1" "exit=$CODE, rapports=$(report_count)"
}

test_file() {
	title "Attributs du fichier"
	run "$BIN" "$WORK/perm.jpg"
	check "Name" "perm.jpg" "$(field Name)"
	check "Size identique à wc -c" "$(file_size "$WORK/perm.jpg") bytes" "$(field Size)"
	check "Permissions après chmod 600" "-rw-------" "$(field Permissions)"
	check "Modified identique à date -r" "$(date -r "$WORK/perm.jpg" '+%Y-%m-%d %H:%M:%S')" "$(field Modified)"
	check "Created renseigné" "date" "$(date_state Created)"
}

test_jpeg() {
	title "JPEG"
	run "$BIN" "$WORK/plain.jpg"
	check "format détecté" "JPEG" "$(field Format)"
	check "dimensions (SOF)" "16x8" "$(field Width)x$(field Height)"
	check "encodage baseline" "Baseline DCT" "$(field Encoding)"
	check "segment JFIF" "yes" "$(contains 'JFIF version')"
	run "$BIN" "$WORK/progressive.jpg"
	check "encodage progressif" "Progressive DCT" "$(field Encoding)"
	run "$BIN" "$WORK/comment.jpg"
	check "commentaire (COM)" "jpeg comment" "$(field Comment)"
	run "$BIN" "$WORK/disguised.png"
	check "JPEG renommé en .png : détection par signature" "JPEG" "$(field Format)"
}

test_exif() {
	title "EXIF"
	run "$BIN" "$WORK/exif.jpg"
	for tag in Make Model Artist DateTimeOriginal; do
		check "$tag comparé à exiftool" "$(exiftool -s3 -"$tag" "$WORK/exif.jpg")" "$(field "$tag")"
	done
	check "UserComment (préfixe ASCII)" "hello exif" "$(field UserComment)"
	check "GPSLatitude" "N 43, 30" "$(field GPSLatitudeRef) $(prefix GPSLatitude 6)"
	check "GPSLongitude" "E 11, 15" "$(field GPSLongitudeRef) $(prefix GPSLongitude 6)"
	check "sections IFD0, SubIFD et GPS" "yes, yes, yes" "$(contains '[EXIF IFD0]'), $(contains '[EXIF SubIFD]'), $(contains '[EXIF GPS]')"
}

test_png() {
	title "PNG"
	run "$BIN" "$WORK/text.png"
	check "format détecté" "PNG" "$(field Format)"
	check "dimensions (IHDR)" "10x6" "$(field Width)x$(field Height)"
	check "chunk texte" "yes" "$(contains 'png comment')"
}

test_gif() {
	title "GIF"
	run "$BIN" "$WORK/comment.gif"
	check "format et version" "GIF 89a" "$(field Format) $(field Version)"
	check "dimensions" "8x4" "$(field Width)x$(field Height)"
	check "commentaire" "gif comment" "$(field Comment)"
	check "image fixe" "1" "$(field Frames)"
	run "$BIN" "$WORK/anim.gif"
	check "GIF animé" "2" "$(field Frames)"
}

test_bmp() {
	title "BMP"
	run "$BIN" "$WORK/v3.bmp"
	check "format détecté" "BMP" "$(field Format)"
	check "en-tête DIB (BMP3)" "BITMAPINFOHEADER" "$(field 'DIB header')"
	check "dimensions" "7x5" "$(field Width)x$(field Height)"
	check "ordre des lignes" "bottom-up" "$(field 'Row order')"
	run "$BIN" "$WORK/default.bmp"
	check "dimensions (en-tête par défaut)" "7x5" "$(field Width)x$(field Height)"
}

test_robustness() {
	title "Robustesse"
	run "$BIN" "$WORK/truncated.jpg"
	check "JPEG tronqué à 100 octets" "exit=0, format=JPEG, panic=no" "exit=$CODE, format=$(field Format), panic=$(contains 'panic')"
	run "$BIN" "$WORK/loop.jpg"
	check "IFD EXIF qui pointe sur lui-même" "exit=0, IFD0=yes" "exit=$CODE, IFD0=$(contains '[EXIF IFD0]')"
	run "$BIN" "$WORK/escape.jpg"
	check "séquence d'échappement dans un commentaire" "exit=0, ESC=no, commentaire=yes" \
		"exit=$CODE, ESC=$(contains $'\033'), commentaire=$(contains '[31mred')"
}

build_images
test_arguments
test_file
test_jpeg
test_exif
test_png
test_gif
test_bmp
test_robustness
summary "scorpion"
