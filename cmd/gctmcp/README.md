# GCT MCP server

`gctmcp` serves GoCryptoTrader market queries, permitted RPC operations and saved
runtime diagnostics over MCP standard input/output. Public market queries are
available when MCP is enabled. Optional permissions control diagnostic access,
private account information and execution.

For a plain-English introduction, setup steps and example AI requests, see
[Use GoCryptoTrader with an AI assistant](../../docs/mcp/README.md).

Build the executables from the repository root:

```sh
go build -o gctmcp ./cmd/gctmcp
go build -o gctdiagnosticrun ./cmd/gctdiagnosticrun
```

## Configuration

The MCP policy is `remoteControl.mcp` in the JSON file selected by `--config`.
The default file is `config.json` inside GCT's platform-specific data directory.
MCP is disabled by default. An enabled policy permits public market queries;
all five optional permissions default to false.

MCP permissions govern the AI's tool requests. GCT continues operating with its
configured exchange credentials and services.

```json
{
  "remoteControl": {
    "mcp": {
      "enabled": true,
      "mode": "readonly",
      "permissions": {
        "diagnostics": false,
        "accountData": false,
        "orderManagement": false,
        "withdrawals": false,
        "systemControl": false
      }
    }
  }
}
```

`mode` defaults to `readonly`, which permits authorised reads. `execution`
permits state changes only when their relevant optional permissions are also
true. Execution mode grants no optional permissions by itself.

| Permission | Default | Access when enabled |
| --- | --- | --- |
| `diagnostics` | `false` | Operational inspection, saved runtime observations, effective MCP policy and pair-refresh receipts. |
| `accountData` | `false` | Private balances, portfolio, order history, positions, collateral and balance-refresh receipts. Account refreshes also require execution mode. |
| `orderManagement` | `false` | Order submission, modification and cancellation, plus margin and leverage changes, in execution mode. It also protects controls that can cancel orders indirectly. |
| `withdrawals` | `false` | Fiat and cryptocurrency withdrawals in execution mode. |
| `systemControl` | `false` | Runtime configuration, exchange, pair, subsystem and websocket enablement in execution mode, plus pair-refresh receipt inspection. Controls that can cancel orders also require `orderManagement`. |

The policy is loaded when the MCP process starts. Restart the MCP connection to
load policy changes. Loading the policy does not save or migrate GCT
configuration. For an encrypted GCT configuration, use a separate unencrypted
JSON policy file containing only `remoteControl.mcp`; protocol input is reserved
for MCP messages.

Only permitted tools are advertised, and the same policy is enforced before
execution. Authentication secrets, API keys, connection URLs and raw RPC error
text are withheld from results. Exchange connection destinations and proxies
are configured by the operator through GCT configuration or CLI commands.

## Connect

Start GCT with its RPC server enabled, then configure the MCP client to launch:

```sh
./gctmcp --config /path/to/policy.json --rpchost localhost:9052 --cert /path/to/gct/tls/cert.pem
```

Supply the RPC password through `GCT_RPC_PASSWORD`. `GCT_RPC_USERNAME` defaults
to `admin` and can also be selected with `--rpcuser`. Keep credentials in the
client's private environment or secret settings. The certificate defaults to
`tls/cert.pem` inside GCT's platform-specific data directory. Certificate and key
files are ignored by Git; tests generate their certificates in memory.

Startup errors and help use standard error. Standard output carries MCP
messages. Run `gctmcp --help` for the launch options.

## Runtime recording

GCT runtime recording is optional and disabled by default. Enable it in the root
configuration block:

```json
{
  "diagnostics": {
    "enabled": true
  }
}
```

The daemon's `--diagnostics=true` or `--diagnostics=false` overrides that choice
for one launch. The recorder saves bounded JSON observations under `diagnostics`
in GCT's data directory. It reads public market caches and runtime state without
requiring a history database. Recording and the MCP `diagnostics` permission are
separate choices: one produces evidence, and the other permits its inspection.
See the [configuration reference](../../config/README.md#enable-runtime-diagnostics-via-config-example).

To inspect saved evidence without an RPC connection, enable the MCP
`diagnostics` permission and launch:

```sh
./gctmcp --config /path/to/policy.json --offline --diagnostics-dir /path/to/gct/diagnostics
```

`--offline` serves permitted saved diagnostics without RPC credentials or a TLS
certificate. `--diagnostics-dir` defaults to `diagnostics` inside the
platform-specific GCT data directory. Paths are selected at process startup.

Saved observations identify their capture time separately from the time of an
MCP read. A recent heartbeat establishes recorder progress; worker and exchange
progress have their own recorded evidence. Stale or missing evidence can reflect
sleep, storage failures, a stalled process or downtime. It does not establish the
cause by itself.

### Recorder ownership and recovery

`owner.lock` permits one GCT recorder to publish to a diagnostic directory. On
Linux, macOS and Windows, a subsequent launch can reclaim it when the operating
system confirms that the recorded process has exited. A running process, a
reused process ID or an uncertain status keeps the lock in place. GCT continues
operating if recording is unavailable.

If recording remains unavailable, check that no GCT process is using that
diagnostic directory. Only after confirming the previous recorder has stopped,
remove `owner.lock` and start GCT with recording enabled. Preserve the journal
files and refresh receipts. Do not delete the diagnostic directory or remove a
lock merely because its recorded observations are old. The optional launcher's
`supervisor.lock` and an unresolved `repair-active.json` require their separate
reconciliation described below.

## Diagnostic commands

Operational inspection commands from `gct_runtime_health` through
`gct_market_snapshots` require `diagnostics`. Refresh execution and receipt
inspection use the target permissions described below. Saved data is historical,
including recent snapshots. Unknown measurements remain unknown.

| Command | Description |
| --- | --- |
| `gct_runtime_health` | Inspect the saved heartbeat, recorded worker progress and bounded goroutine code locations independently of RPC. |
| `gct_crash_report` | Inspect the optional launcher's exit evidence or previous engine observations, preserving uncertainty about incomplete records. |
| `gct_performance` | Inspect saved memory, goroutine, GC and runtime measurements. |
| `gct_events` | Inspect bounded warning/error categories, counts and first/last observation times without raw log messages. |
| `gct_run_report` | Read the recorded run's activity, progress and diagnostic coverage summary. |
| `gct_identity` | Identify the recorded process, build revision, Go version and run start time. |
| `gct_configuration` | Inspect allowlisted effective settings, counts and configuration fingerprints. |
| `gct_permissions` | Inspect the current MCP policy and effective access, with saved daemon startup permissions labelled separately. |
| `gct_market_health` | Inspect native market-cache availability, source timestamps and retained depth counts. |
| `gct_websocket_health` | Inspect saved connection and subscription counts. Connection state and market-data arrival are separate observations. |
| `gct_instrument_metadata` | Inspect native market identities, enabled states and recorded metadata coverage. |
| `gct_storage` | Inspect saved recorder and storage availability observations. File availability alone does not establish database integrity. |
| `gct_market_snapshots` | Read saved public prices and up to twenty retained book levels per side, with source timestamps and native identities. |
| `gct_repair` | Execute one tracked balance or pair refresh through the live RPC server, with execution mode and the target's optional permission. |
| `gct_repair_status` | Inspect an authorised receipt by `id`, or omit `id` to list authorised saved receipts, independently of RPC. Balance receipts require `accountData`; pair receipts require `systemControl` or `diagnostics`. |

Saved diagnostic commands accept optional `exchange`, `asset`, `base` and
`quote` filters for rows containing those identity fields. For example, an asset
filter retains exchange-wide websocket status. `limit` defaults to 50 and allows
1–200; `offset` is a non-negative offset within filtered saved rows. Scalar run
observations are independent of market filters. Each page can observe a newer
snapshot; `truncated` identifies bounded output.

The general cache's contract multiplier and exchange quantity/price-rule coverage
is reported explicitly. Native cached amounts need their corresponding contract
metadata before they can establish underlying notional values or executable
sizing. Saved depth is an observation rather than an executable quote.

## Tracked refreshes

`gct_repair` refreshes a target identified by observed stale data or a retrieval
failure. It requires an online RPC connection, execution mode and the permission
for its target.
Supply `id`, `action`, `exchange` and `reason`; `asset` depends on the action.
IDs contain 8–64 lowercase letters, digits or hyphens. Exchange names use native
lowercase identifiers. Reasons are `stale_data` or `retrieval_failed`.

| Action | Target and required permission | Result |
| --- | --- | --- |
| `refresh_balances` | Supply `asset`; requires `accountData`. Refresh balances for that enabled exchange asset. | A successful retrieval is `completed_unverified`; the receipt excludes balances and does not establish reconciliation. |
| `refresh_pairs` | Select the exchange and omit `asset`; requires `systemControl` and `orderManagement`. Refresh supported pairs, including subscription handling that can reconnect account sessions. | A successful refresh is `completed_unverified`. |

Intent is saved before execution. One repair executes at a time, with a minimum
minute between admissions. A directory admits at most 64 receipts. Reconcile
and archive its history before deliberately rotating the directory.

The same ID and arguments return the saved receipt without executing again,
including after an MCP restart. An ID used with different arguments is rejected.
After an uncertain result, inspect `gct_repair_status` with that ID and reconcile
the actual state. Receipt inspection requires `accountData` for balance refreshes,
or `systemControl` or `diagnostics` for pair refreshes. Ledger listings include
only receipts authorised by those permissions. An unresolved
`repair-active.json` guard requires operator reconciliation after confirming that the previous request has stopped.

## RPC command reference

Use the [GCT CLI command reference](../gctcli/README.md) for RPC commands.
`gctcli --help` lists commands, and `gctcli <command> --help` describes their
flags. MCP exposes permitted protobuf methods as `gct_<MethodName>`, with inputs
specified by each tool's schema and the [RPC contract](../../gctrpc/rpc.proto).
Inputs use a `request` object with protobuf JSON field names.

For example, `gct_GetTicker` accepts:

```json
{
  "request": {
    "exchange": "binance",
    "pair": { "base": "BTC", "quote": "USDT", "delimiter": "-" },
    "assetType": "spot"
  }
}
```

Public requests retain GCT's exchange support, validation and availability
requirements. Private queries require `accountData`; state changes require
execution mode and their relevant optional permission. `GetOrder` also stores
the returned order in GCT's managed orders, so it requires `accountData`,
`orderManagement` and execution mode. `GetOrders` supports read-only order queries
with `accountData`.

### Exchange-backed candles

`gct_GetHistoricCandles` retrieves candles from an enabled exchange, asset and
native pair. It is available when MCP is enabled and the RPC connection is
configured. Use `exRequest` for the exchange's extended retrieval path. Supported
requests set `useDb`, `sync`, `fillMissingWithTrades` and `force` to false.

```json
{
  "request": {
    "exchange": "binance",
    "pair": { "base": "BTC", "quote": "USDT", "delimiter": "-" },
    "assetType": "spot",
    "start": "2026-11-11 00:00:00 UTC",
    "end": "2026-11-12 00:00:00 UTC",
    "timeInterval": "3600000000000",
    "exRequest": true,
    "useDb": false,
    "sync": false,
    "fillMissingWithTrades": false,
    "force": false
  }
}
```

The interval uses nanoseconds; this example requests hourly candles. Candle
retrieval operates in read-only mode and uses the exchange-backed path.

## Optional exit recorder

| Executable command | Description |
| --- | --- |
| `gctmcp` | Serve permitted MCP tools over standard input/output. |
| `gctdiagnosticrun -- [executable] [arguments...]` | Launch one explicitly selected executable and save bounded launch and exit evidence. |

For GCT's default data directory:

```sh
./gctdiagnosticrun -- /absolute/path/gocryptotrader --diagnostics
```

The `--` separator is required. `--diagnostics-dir` defaults to `diagnostics`
inside GCT's platform-specific data directory. Set it to the running GCT data
location's `diagnostics` directory when using a custom location.

The launcher protects that directory with owner-only permissions and records
intent before starting its child. It records PID, observation times, exit code,
interruption state and whether stderr contained a panic marker. The marker is an
observation rather than a complete crash diagnosis. Arguments, environment values
and raw output stay out of `crash.json`; stdout and stderr pass through to the
operator.

A `supervisor.lock` gives one launcher exclusive ownership of the directory's
exit record. If the launcher is killed or power is lost, confirm the previous
supervisor and child have stopped before reconciling the record and removing its
lock. An incomplete record leaves the child outcome unknown.

Interrupting or terminating the launcher sends an interrupt to its own child,
with up to ten seconds for shutdown before that child is killed. Launch-intent
persistence failure prevents the child from starting. Completion persistence
failure is reported to the operator. The launcher runs its child once.

## Output limits

`--timeout` defaults to `10s`, `--max-messages` to `20`, and
`--max-response-bytes` to `1048576`. Allowed ranges are positive durations up to
one minute, 1–1000 stream messages, and 4096–4194304 response bytes. Stream calls
can select a smaller sample with `maxMessages`.

Streams return bounded samples, with partial observations and application
failures reported explicitly. Retrieval time and exchange-data freshness are
separate. A failed RPC is never automatically retried; reconcile an uncertain
operation before attempting it again.
