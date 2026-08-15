#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
caller_dir=$(pwd -P)
artifact_path=
signature_path=

absolute_path() {
	case "$1" in
		/*) printf '%s\n' "$1" ;;
		*) printf '%s/%s\n' "$caller_dir" "$1" ;;
	esac
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--artifact)
			[ "$#" -ge 2 ] || { printf '%s\n' 'artifact path is required' >&2; exit 2; }
			artifact_path=$(absolute_path "$2")
			shift 2
			;;
		--artifact=*)
			artifact_path=$(absolute_path "${1#--artifact=}")
			shift
			;;
		--signature)
			[ "$#" -ge 2 ] || { printf '%s\n' 'signature path is required' >&2; exit 2; }
			signature_path=$(absolute_path "$2")
			shift 2
			;;
		--signature=*)
			signature_path=$(absolute_path "${1#--signature=}")
			shift
			;;
		*)
			printf '%s\n' 'unexpected signing argument' >&2
			exit 2
			;;
	esac
done

[ -n "$artifact_path" ] || { printf '%s\n' 'artifact path is required' >&2; exit 2; }
[ -n "$signature_path" ] || { printf '%s\n' 'signature path is required' >&2; exit 2; }

cd "$script_dir/sign-release-asset"
exec go run . --artifact "$artifact_path" --signature "$signature_path"
