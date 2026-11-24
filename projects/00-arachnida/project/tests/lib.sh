#!/usr/bin/env bash

TOTAL=0
PASSED=0

if [ -t 1 ]; then
	GREEN=$'\033[32m'
	RED=$'\033[31m'
	BOLD=$'\033[1m'
	RESET=$'\033[0m'
else
	GREEN=
	RED=
	BOLD=
	RESET=
fi

title() {
	printf '\n%s== %s ==%s\n' "$BOLD" "$1" "$RESET"
}

require() {
	local missing=""
	for cmd in "$@"; do
		command -v "$cmd" >/dev/null 2>&1 || missing="$missing $cmd"
	done
	if [ -n "$missing" ]; then
		printf '%soutils manquants :%s%s\n' "$RED" "$missing" "$RESET"
		exit 1
	fi
}

require_binary() {
	if [ ! -x "$1" ]; then
		printf '%sbinaire introuvable : %s%s\n' "$RED" "$1" "$RESET"
		exit 1
	fi
}

run() {
	OUT=$("$@" 2>&1)
	CODE=$?
}

run_in() {
	local dir=$1
	shift
	OUT=$(cd "$dir" && "$@" 2>&1)
	CODE=$?
}

check() {
	TOTAL=$((TOTAL + 1))
	printf '%s[%02d] %s%s\n' "$BOLD" "$TOTAL" "$1" "$RESET"
	printf '     attendu : %s\n' "$2"
	if [ "$3" = "$2" ]; then
		PASSED=$((PASSED + 1))
		printf '     obtenu  : %s %sOK%s\n' "$3" "$GREEN" "$RESET"
	else
		printf '     obtenu  : %s %sKO%s\n' "$3" "$RED" "$RESET"
	fi
}

summary() {
	local color=$GREEN
	[ "$PASSED" -eq "$TOTAL" ] || color=$RED
	printf '\n%s%s : %d/%d tests OK%s\n' "$color" "$1" "$PASSED" "$TOTAL" "$RESET"
	[ "$PASSED" -eq "$TOTAL" ]
}

contains() {
	if printf '%s' "$OUT" | grep -qF -- "$1"; then
		echo yes
	else
		echo no
	fi
}

exists() {
	if [ -e "$1" ]; then
		echo yes
	else
		echo no
	fi
}

error_message() {
	printf '%s\n' "$OUT" | grep -m 1 -E '^[a-z]+: ' | sed -E 's/^[a-z]+: //'
}

count_files() {
	if [ ! -d "$1" ]; then
		echo 0
		return
	fi
	find "$1" -type f | wc -l | tr -d ' '
}

same_file() {
	if cmp -s "$1" "$2"; then
		echo identique
	else
		echo différent
	fi
}

file_size() {
	wc -c < "$1" | tr -d ' '
}
