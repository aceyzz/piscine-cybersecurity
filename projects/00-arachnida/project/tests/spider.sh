#!/usr/bin/env bash

DIR=$(cd "$(dirname "$0")" && pwd)
. "$DIR/lib.sh"

BIN=$1
require_binary "$BIN"
require python3 curl find cmp

WORK=$(mktemp -d)
SITE=$WORK/site
HOST=127.0.0.1
PORT=$((20000 + RANDOM % 20000))
BASE=http://$HOST:$PORT
SERVER_PID=

cleanup() {
	if [ -n "$SERVER_PID" ]; then
		kill "$SERVER_PID" 2>/dev/null
	fi
	rm -rf "$WORK"
}
trap cleanup EXIT

page() {
	local name=$1
	shift
	printf '<html><body>\n%s\n</body></html>\n' "$*" > "$SITE/$name"
}

saved_count() {
	printf '%s\n' "$OUT" | grep -E 'image\(s\) saved' | sed -E 's/ image.*//'
}

build_site() {
	mkdir -p "$SITE/sub"
	for img in a.jpg b.png direct.gif UPPER.JPG x.webp c.gif sub/a.jpg d.bmp e.jpg f.png g.gif h.jpg ext.jpg; do
		printf 'fake image %s' "$img" > "$SITE/$img"
	done
	page index.html \
		'<img src="a.jpg">' '<img src="a.jpg">' '<img src="b.png">' '<img src="UPPER.JPG">' \
		'<img src="x.webp">' '<img src="data:image/png;base64,AAAA">' '<img src="missing.jpg">' \
		'<a href="direct.gif">gif</a>' '<a href="index.html#top">self</a>' \
		"<a href=\"http://localhost:$PORT/external.html\">ext</a>" '<a href="p1.html">p1</a>'
	page external.html '<img src="ext.jpg">'
	page p1.html '<img src="c.gif">' '<img src="sub/a.jpg">' '<a href="p2.html">p2</a>'
	page p2.html '<img src="d.bmp">' '<a href="p3.html">p3</a>'
	page p3.html '<img src="e.jpg">' '<a href="p4.html">p4</a>'
	page p4.html '<img src="f.png">' '<a href="p5.html">p5</a>'
	page p5.html '<img src="g.gif">' '<a href="p6.html">p6</a>'
	page p6.html '<img src="h.jpg">' '<a href="index.html">index</a>'
}

start_server() {
	python3 -m http.server "$PORT" --bind "$HOST" --directory "$SITE" >/dev/null 2>&1 &
	SERVER_PID=$!
	for _ in $(seq 1 50); do
		curl -s -o /dev/null "$BASE/" && return 0
		sleep 0.1
	done
	printf '%sserveur de test injoignable sur le port %s%s\n' "$RED" "$PORT" "$RESET"
	exit 1
}

test_arguments() {
	title "Arguments"
	run "$BIN" -h
	check "spider -h" "exit=0, usage=yes" "exit=$CODE, usage=$(contains 'usage:')"
	run "$BIN"
	check "spider (aucun argument)" "exit=2, erreur=exactly one URL expected" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" "$BASE/" "$BASE/p1.html"
	check "spider URL URL (deux URL)" "exit=2, erreur=exactly one URL expected" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" -l 2 "$BASE/"
	check "spider -l 2 URL (sans -r)" "exit=2, erreur=-l requires -r" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" -r -l -1 "$BASE/"
	check "spider -r -l -1 URL" "exit=2, erreur=depth must be >= 0" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" -r -l abc "$BASE/"
	check "spider -r -l abc URL" "exit=2, erreur=invalid value \"abc\" for flag -l: parse error" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" -x "$BASE/"
	check "spider -x URL (option inconnue)" "exit=2, erreur=flag provided but not defined: -x" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" example.com
	check "spider example.com (sans schéma)" "exit=2, erreur=invalid URL: example.com" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" ftp://example.com/
	check "spider ftp://example.com/ (schéma non HTTP)" "exit=2, erreur=invalid URL: ftp://example.com/" "exit=$CODE, erreur=$(error_message)"
	run "$BIN" http://
	check "spider http:// (hôte vide)" "exit=2, erreur=invalid URL: http://" "exit=$CODE, erreur=$(error_message)"
	touch "$WORK/file"
	run "$BIN" -p "$WORK/file" "$BASE/"
	check "spider -p FICHIER URL (chemin qui n'est pas un dossier)" "exit=1" "exit=$CODE"
}

test_download() {
	title "Téléchargement sans -r"
	local out=$WORK/flat
	run "$BIN" -p "$out" "$BASE/"
	check "spider -p DIR URL" "exit=0, total=4, fichiers=4" "exit=$CODE, total=$(saved_count), fichiers=$(count_files "$out")"
	check "images .jpg .png .JPG et lien <a> vers .gif" "a.jpg=yes, b.png=yes, UPPER.JPG=yes, direct.gif=yes" \
		"a.jpg=$(exists "$out/a.jpg"), b.png=$(exists "$out/b.png"), UPPER.JPG=$(exists "$out/UPPER.JPG"), direct.gif=$(exists "$out/direct.gif")"
	check "extension .webp et URI data: ignorées" "x.webp=no" "x.webp=$(exists "$out/x.webp")"
	check "image présente deux fois dans la page" "a_1.jpg=no" "a_1.jpg=$(exists "$out/a_1.jpg")"
	check "image 404 : avertissement, exécution poursuivie" "exit=0, avertissement=yes" "exit=$CODE, avertissement=$(contains 'missing.jpg: 404')"
	check "page liée non suivie sans -r" "c.gif=no" "c.gif=$(exists "$out/c.gif")"
	check "contenu identique à la source" "a.jpg=identique" "a.jpg=$(same_file "$SITE/a.jpg" "$out/a.jpg")"

	run "$BIN" -p "$out" "$BASE/"
	check "relance dans le même dossier : renommage _N" "fichiers=8, a_1.jpg=yes" "fichiers=$(count_files "$out"), a_1.jpg=$(exists "$out/a_1.jpg")"

	mkdir -p "$WORK/default"
	run_in "$WORK/default" "$BIN" "$BASE/"
	check "dossier par défaut ./data/" "exit=0, data=4" "exit=$CODE, data=$(count_files "$WORK/default/data")"

	run "$BIN" -p "$WORK/a/b/c" "$BASE/"
	check "-p crée les dossiers manquants" "exit=0, fichiers=4" "exit=$CODE, fichiers=$(count_files "$WORK/a/b/c")"

	run "$BIN" -p "$WORK/down" "http://$HOST:1/"
	check "serveur injoignable" "exit=0, total=0, avertissement=yes" "exit=$CODE, total=$(saved_count), avertissement=$(contains 'spider: ')"
}

test_recursion() {
	title "Récursion"
	local out
	for spec in "0 4" "1 6" "2 7" "3 8" "6 11"; do
		set -- $spec
		out=$WORK/r$1
		run "$BIN" -r -l "$1" -p "$out" "$BASE/"
		check "spider -r -l $1" "exit=0, fichiers=$2" "exit=$CODE, fichiers=$(count_files "$out")"
	done

	out=$WORK/rdefault
	run "$BIN" -r -p "$out" "$BASE/"
	check "spider -r (profondeur par défaut 5)" "exit=0, fichiers=10, g.gif=yes, h.jpg=no" \
		"exit=$CODE, fichiers=$(count_files "$out"), g.gif=$(exists "$out/g.gif"), h.jpg=$(exists "$out/h.jpg")"
	check "homonyme dans un sous-dossier : renommage _N" "a.jpg=yes, a_1.jpg=yes" "a.jpg=$(exists "$out/a.jpg"), a_1.jpg=$(exists "$out/a_1.jpg")"
	check "lien vers un autre hôte non suivi" "ext.jpg=no" "ext.jpg=$(exists "$WORK/r6/ext.jpg")"
	check "boucle p6 vers index : fin du programme" "exit=0" "exit=$CODE"
}

build_site
start_server
test_arguments
test_download
test_recursion
summary "spider"
