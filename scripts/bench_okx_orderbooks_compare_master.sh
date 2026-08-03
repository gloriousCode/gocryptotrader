#!/usr/bin/env bash
# Runs the OKX orderbook websocket benchmarks against master and the current
# checkout with the same benchmark file, then prints a markdown comparison.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

MASTER_REF="${MASTER_REF:-master}"
COUNT="${COUNT:-5}"
BENCHTIME="${BENCHTIME:-3s}"
BENCH="${BENCH:-Benchmark(WsProcessOrderBooksInstrumentTypeEmpty|WsProcessOrderBooksInstrumentTypeEmptyUpdate|WsOrderBookUnmarshalInstrumentTypeEmpty|CalculateOrderbookChecksum|GenerateOrderbookChecksum)$}"
OUT_DIR="${OUT_DIR:-/tmp/gct-okx-orderbook-bench-$(date +%Y%m%d%H%M%S)}"
MASTER_DIR="$OUT_DIR/master-worktree"
MASTER_OUT="$OUT_DIR/master.txt"
CURRENT_OUT="$OUT_DIR/current.txt"

mkdir -p "$OUT_DIR"

cleanup() {
	git worktree remove --force "$MASTER_DIR" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "Output directory: $OUT_DIR"
echo "Benchmark regex: $BENCH"
echo "Count: $COUNT"
echo "Benchtime: $BENCHTIME"

git worktree add --detach "$MASTER_DIR" "$MASTER_REF" >/dev/null
cp exchanges/okx/okx_websocket_benchmark_test.go "$MASTER_DIR/exchanges/okx/okx_websocket_benchmark_test.go"

echo "Running master benchmarks..."
(
	cd "$MASTER_DIR"
	go test ./exchanges/okx -run '^$' -bench "$BENCH" -benchmem -benchtime="$BENCHTIME" -count="$COUNT"
) | tee "$MASTER_OUT"

echo "Running current benchmarks..."
go test ./exchanges/okx -run '^$' -bench "$BENCH" -benchmem -benchtime="$BENCHTIME" -count="$COUNT" | tee "$CURRENT_OUT"

echo
echo "Markdown comparison:"
awk -v masterFile="$MASTER_OUT" -v currentFile="$CURRENT_OUT" '
function capture(file, bucket, line, name) {
	while ((getline line < file) > 0) {
		if (line !~ /^Benchmark/) {
			continue
		}
		split(line, fields)
		name = fields[1]
		sub(/-[0-9]+$/, "", name)
		sum[bucket, name, "ns"] += fields[3]
		sum[bucket, name, "bytes"] += fields[5]
		sum[bucket, name, "allocs"] += fields[7]
		count[bucket, name]++
		if (bucket == "current" && !(name in seen)) {
			order[++orderCount] = name
			seen[name] = 1
		}
	}
	close(file)
}
function pct(current, master) {
	if (master == 0) {
		return "n/a"
	}
	return sprintf("%+.1f%%", ((current / master) - 1) * 100)
}
BEGIN {
	capture(masterFile, "master")
	capture(currentFile, "current")
	print "| Benchmark | master ns/op | current ns/op | ns change | master B/op | current B/op | B change | master allocs/op | current allocs/op | alloc change |"
	print "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|"
	for (i = 1; i <= orderCount; i++) {
		name = order[i]
		if (count["master", name] == 0 || count["current", name] == 0) {
			continue
		}
		mns = sum["master", name, "ns"] / count["master", name]
		cns = sum["current", name, "ns"] / count["current", name]
		mb = sum["master", name, "bytes"] / count["master", name]
		cb = sum["current", name, "bytes"] / count["current", name]
		ma = sum["master", name, "allocs"] / count["master", name]
		ca = sum["current", name, "allocs"] / count["current", name]
		printf "| `%s` | %.1f | %.1f | %s | %.0f | %.0f | %s | %.1f | %.1f | %s |\n", name, mns, cns, pct(cns, mns), mb, cb, pct(cb, mb), ma, ca, pct(ca, ma)
	}
}
'
