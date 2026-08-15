#!/bin/sh

set -eu

usage() {
	printf 'usage: %s DIST_DIR PUBLIC_KEY_PATH\n' "$0" >&2
	exit 2
}

fail() {
	printf 'release asset verification failed: %s\n' "$1" >&2
	exit 1
}

[ "$#" -eq 2 ] || usage

dist_dir=$1
public_key_path=$2
binary=${BINARY:-llmhub}
minisign_bin=${MINISIGN_BIN:-minisign}
checksums_path=$dist_dir/checksums.txt

[ -d "$dist_dir" ] || fail "distribution directory is missing"
[ -f "$checksums_path" ] || fail "checksums.txt is missing"
[ -f "$public_key_path" ] || fail "public key is missing"
command -v "$minisign_bin" >/dev/null 2>&1 || fail "minisign verifier is unavailable"

if command -v sha256sum >/dev/null 2>&1; then
	sha256() {
		sha256sum "$1" | awk '{print $1}'
	}
elif command -v shasum >/dev/null 2>&1; then
	sha256() {
		shasum -a 256 "$1" | awk '{print $1}'
	}
else
	fail "no SHA-256 utility is available"
fi

for os in linux darwin windows freebsd; do
	for arch in amd64 arm64; do
		asset=$binary-$os-$arch
		if [ "$os" = windows ]; then
			asset=$asset.exe
		fi
		case "$os/$arch" in
			windows/amd64) target=${os}_${arch}_v1; filename=$binary.exe ;;
			windows/arm64) target=${os}_${arch}_v8.0; filename=$binary.exe ;;
			*/amd64) target=${os}_${arch}_v1; filename=$binary ;;
			*/arm64) target=${os}_${arch}_v8.0; filename=$binary ;;
			*) fail "unsupported release target $os/$arch" ;;
		esac
		if [ -e "$dist_dir/$asset" ]; then
			asset_path=$dist_dir/$asset
			signature_path=$dist_dir/$asset.minisig
		else
			asset_path=$dist_dir/${binary}_$target/$filename
			signature_path=$asset_path.minisig
		fi

		[ -f "$asset_path" ] || fail "missing binary asset $asset"
		[ ! -L "$asset_path" ] || fail "binary asset is a symlink $asset"
		[ -f "$signature_path" ] || fail "missing signature asset $asset.minisig"
		[ ! -L "$signature_path" ] || fail "signature asset is a symlink $asset.minisig"

		checksum=$(awk -v name="$asset" '
			$2 == name {
				count++
				digest = $1
			}
			END {
				if (count != 1 || length(digest) != 64 || digest ~ /[^0-9A-Fa-f]/) {
					exit 1
				}
				print digest
			}
		' "$checksums_path") || fail "checksums.txt must contain exactly one valid entry for $asset"

		actual=$(sha256 "$asset_path")
		[ "$actual" = "$checksum" ] || fail "checksum mismatch for $asset"
		"$minisign_bin" -Vm "$asset_path" -x "$signature_path" -p "$public_key_path" >/dev/null 2>&1 || fail "signature verification failed for $asset"
	done
done

printf 'verified %s release binary assets\n' "8"
