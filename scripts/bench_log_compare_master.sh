#!/usr/bin/env bash

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
base_ref="${BASE_REF:-master}"
count="${COUNT:-5}"
bench_re='^Benchmark(InfolnDiscard|InfofDiscard|InfofDiscardWithCustomHook)$'
out_dir="${OUT_DIR:-/tmp/gct-log-bench-$(date +%Y%m%d_%H%M%S)}"
keep_worktree="${KEEP_WORKTREE:-0}"

if ! git -C "$repo_root" rev-parse --verify "$base_ref^{commit}" >/dev/null 2>&1; then
	echo "Base ref '$base_ref' was not found locally. Fetch it first (for example: git fetch origin $base_ref:$base_ref)." >&2
	exit 1
fi

mkdir -p "$out_dir"

current_sha="$(git -C "$repo_root" rev-parse --short HEAD)"
current_branch="$(git -C "$repo_root" rev-parse --abbrev-ref HEAD)"
base_sha="$(git -C "$repo_root" rev-parse --short "$base_ref")"
base_slug="$(echo "$base_ref" | tr '/:' '__')"
worktree_dir="$out_dir/worktree-$base_slug"
current_out="$out_dir/current-$current_sha.txt"
base_out="$out_dir/${base_slug}-$base_sha.txt"
compare_out="$out_dir/compare.txt"
benchmark_support_file="$repo_root/log/logger_benchmark_test.go"

render_plain_diff() {
	awk '
	function trim(v) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", v); return v }
	function pct(base, current) {
		if (base == 0) { return "n/a" }
		return sprintf("%.2f%%", ((current-base)/base)*100)
	}
	/^Benchmark/ {
		name = $1
		sub(/-[0-9]+$/, "", name)
		if (FNR == NR) {
			base_ns[name] = $(NF-5)
			base_b[name] = $(NF-3)
			base_a[name] = $(NF-1)
		} else {
			cur_ns[name] = $(NF-5)
			cur_b[name] = $(NF-3)
			cur_a[name] = $(NF-1)
			names[name] = 1
		}
	}
	END {
		printf "%-38s  %-26s  %-26s  %-26s\n", "Benchmark", "ns/op (base -> current)", "B/op (base -> current)", "allocs/op (base -> current)"
		for (n in names) {
			printf "%-38s  %s -> %s (%s)  %s -> %s (%s)  %s -> %s (%s)\n",
				n,
				base_ns[n], cur_ns[n], pct(base_ns[n]+0, cur_ns[n]+0),
				base_b[n], cur_b[n], pct(base_b[n]+0, cur_b[n]+0),
				base_a[n], cur_a[n], pct(base_a[n]+0, cur_a[n]+0)
		}
	}
	' "$base_out" "$current_out"
}

cleanup() {
	if [[ "$keep_worktree" == "1" ]]; then
		return
	fi
	if [[ -d "$worktree_dir" ]]; then
		git -C "$repo_root" worktree remove --force "$worktree_dir" >/dev/null 2>&1 || true
	fi
}
trap cleanup EXIT

git -C "$repo_root" worktree add --detach "$worktree_dir" "$base_ref" >/dev/null

if [[ -f "$benchmark_support_file" ]]; then
	cp "$benchmark_support_file" "$worktree_dir/log/logger_benchmark_test.go"
fi

echo "=================================================================="
echo "Running CURRENT benchmark set"
echo "Branch: $current_branch"
echo "Ref:    HEAD"
echo "Commit: $current_sha"
echo "=================================================================="
(
	cd "$repo_root"
	GOCACHE=/tmp/go-build-cache \
		go test ./log -run '^$' -bench "$bench_re" -benchmem -count="$count"
) | tee "$current_out"

echo
echo "=================================================================="
echo "Running BASE benchmark set"
echo "Branch/Ref: $base_ref"
echo "Commit:     $base_sha"
echo "=================================================================="
(
	cd "$worktree_dir"
	GOCACHE=/tmp/go-build-cache \
		go test ./log -run '^$' -bench "$bench_re" -benchmem -count="$count"
) | tee "$base_out"

echo
echo "=================================================================="
echo "Quick Summary (plain go test output)"
echo "CURRENT ($current_branch @ $current_sha):"
grep '^Benchmark' "$current_out" || true
echo
echo "BASE ($base_ref @ $base_sha):"
grep '^Benchmark' "$base_out" || true
echo "=================================================================="
echo

if command -v benchstat >/dev/null 2>&1; then
	echo "Generating comparison with benchstat (BASE -> CURRENT)"
	benchstat "$base_out" "$current_out" | tee "$compare_out"
else
	{
		echo "benchstat is not installed; raw outputs were written instead."
		echo "Install with: go install golang.org/x/perf/cmd/benchstat@latest"
		echo
		echo "CURRENT: $current_branch @ $current_sha"
		echo "BASE:    $base_ref @ $base_sha"
		echo
		echo "Delta Summary (BASE -> CURRENT):"
		render_plain_diff
		echo
		echo "Current output: $current_out"
		echo "Base output:    $base_out"
	} | tee "$compare_out"
fi

echo
echo "Benchmark artefacts written to: $out_dir"
