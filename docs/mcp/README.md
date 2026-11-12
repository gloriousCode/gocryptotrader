# Use GoCryptoTrader with an AI assistant

GCT's MCP connection lets an AI assistant use GoCryptoTrader tools to answer
questions about markets, inspect recorded runtime information and carry out
operations you permit. You write ordinary requests in your AI app; the app calls
the appropriate GCT tools and explains their results.

GCT continues operating with its configured exchanges, credentials and services.
The MCP permissions control what the AI can request through this connection.
MCP is disabled by default, and its default mode is read-only.

## Example AI requests

Replace the example exchange, pair and asset with markets enabled in your GCT
configuration. Ask for observation times so you can judge how current a result is.

| Ask the AI | What it needs |
| --- | --- |
| "Read your GCT MCP permissions. Explain what you may read and change." | `diagnostics`; works offline, including without a saved runtime record. |
| "Summarise the last recorded GCT run. Include when it was captured and which information is unavailable." | `diagnostics` and a saved runtime record; works offline. |
| "Which recorded Binance spot feeds look stale? Separate recorder progress from actual market updates." | `diagnostics` and a saved runtime record; works offline. |
| "Summarise recorded warning and error categories, with their counts and observation times." | `diagnostics` and a saved runtime record. Results contain categories rather than raw log messages. |
| "Show the cached BTC-USDT spot ticker on Binance, including its source timestamp." | An online RPC connection; public market queries need no optional permission. |
| "Compare the cached BTC-USDT spot orderbooks on Binance and Kraken. Include timestamps and explain missing data." | An online RPC connection and both markets enabled in GCT. |
| "Fetch hourly BTC-USDT spot candles from Binance for the previous complete UTC day using exchange data. Identify gaps without inventing candles." | An online RPC connection and exchange support for candle retrieval. |
| "Show my cached Binance spot balances by currency. Include any supplied update time; otherwise say their age is unknown." | An online RPC connection and `accountData`; read-only mode is sufficient. |
| "List my active BTC-USDT spot orders on Binance. Only inspect them." | An online RPC connection and `accountData`; read-only mode is sufficient. |
| "Cancel only order `<order-id>` for BTC-USDT spot on Binance. Report whether cancellation is confirmed, and do not repeat an uncertain request." | An online RPC connection, execution mode and `orderManagement`. Supply the actual order ID. |

Examples describe requests you can make; exchange support and your permissions
determine which tools can fulfil them. A request in chat does not grant an MCP
permission.

## Start with read-only inspection

This setup lets you inspect saved diagnostics without supplying an RPC password.
You need an AI app that supports launching a local MCP connection.

### 1. Build the MCP executable

From the GCT repository root, using its required Go toolchain:

```sh
go build -o gctmcp ./cmd/gctmcp
```

On Windows, use `go build -o gctmcp.exe ./cmd/gctmcp`. Note the executable's full
path; the AI app needs that path when it launches the connection.

### 2. Save a local MCP policy

Save this as a separate local file, for example
`/absolute/path/mcp-policy.json`. This file contains only the MCP policy; keep
your normal GCT configuration in its usual location.

```json
{
  "remoteControl": {
    "mcp": {
      "enabled": true,
      "mode": "readonly",
      "permissions": {
        "diagnostics": true,
        "accountData": false,
        "orderManagement": false,
        "withdrawals": false,
        "systemControl": false
      }
    }
  }
}
```

This permits operational inspection while keeping private account queries and
state changes unavailable through MCP. A separate policy file also works when
your normal GCT configuration is encrypted.

### 3. Register the connection in your AI app

For Codex CLI, add this entry to `~/.codex/config.toml`, replacing every example
path with a full path on your computer:

```toml
[mcp_servers.gct]
command = "/absolute/path/gocryptotrader/gctmcp"
args = [
  "--config", "/absolute/path/mcp-policy.json",
  "--offline",
  "--diagnostics-dir", "/absolute/path/gct-data/diagnostics"
]
```

On Windows, select the `gctmcp.exe` executable. Other AI apps use their local or
STDIO MCP settings with the same executable and arguments. The AI app launches
`gctmcp` and exchanges tool messages with it. Type your questions in the AI app;
running `gctmcp` directly in a terminal waits for protocol messages.

Follow the client's connection or reload instructions after saving the entry.
See the [official OpenAI documentation](https://developers.openai.com/codex/mcp)
for client configuration details.

### 4. Check the connection

Ask:

> Read your GCT MCP permissions. Tell me the current mode, which permissions are
> enabled and whether this connection is offline.

The assistant should report read-only mode, `diagnostics` enabled and the other
four optional permissions disabled. It should identify the connection as
offline. This check works even if GCT has not recorded any runtime information.

### 5. Enable runtime recording in GCT

To inspect runtime observations, add or update this block in your normal GCT
configuration, preserving its other settings:

```json
{
  "diagnostics": {
    "enabled": true
  }
}
```

Recording takes effect when you next start GCT. It saves bounded JSON records in
the `diagnostics` folder inside GCT's data directory. Set the MCP
`--diagnostics-dir` argument to that same folder.

| Platform | Default GCT data directory |
| --- | --- |
| Linux and macOS | `$HOME/.gocryptotrader` |
| Windows | `%APPDATA%\GoCryptoTrader` |

If GCT uses a custom `--datadir`, use its `diagnostics` subfolder. Recording is
independent of MCP access: GCT produces the records, and the `diagnostics`
permission allows the AI to read them. These records do not need a history
database.

Ask the assistant to summarise the last recorded run. Expect capture times and
explicitly unavailable information. Saved observations can still be inspected
when GCT is stopped or its RPC connection is unavailable.

Recording can recover a previous recorder's lock on Linux, macOS and Windows
when the operating system confirms that the process has exited. If it cannot
confirm that, GCT keeps running and leaves recording unavailable. See
[recorder ownership and recovery](../../cmd/gctmcp/README.md#recorder-ownership-and-recovery)
before removing any lock file.

## Connect for market and account queries

Online queries use your running GCT instance with RPC enabled. RPC is GCT's
command interface, also used by `gctcli`. The same read-only policy permits public
market queries. To use this mode in Codex CLI,
use the following entry instead of the offline entry:

```toml
[mcp_servers.gct]
command = "/absolute/path/gocryptotrader/gctmcp"
args = [
  "--config", "/absolute/path/mcp-policy.json",
  "--rpchost", "localhost:9052",
  "--cert", "/absolute/path/gct-data/tls/cert.pem",
  "--diagnostics-dir", "/absolute/path/gct-data/diagnostics"
]
env_vars = ["GCT_RPC_PASSWORD"]
```

For Codex CLI, provide `GCT_RPC_PASSWORD` in the environment of the terminal that
starts Codex. The `env_vars` entry passes it to `gctmcp`. Other apps can use their
private environment or secret settings. This is GCT's RPC password, not an
exchange API key.
The username defaults to `admin`; for another username, add `--rpcuser` and your
username to `args`. The certificate must match your GCT RPC instance. Keep
credentials out of chat messages and committed configuration files.

After reconnecting, try the cached ticker request above. Ask the assistant to
report an unavailable market or failed connection explicitly. GCT's configured
exchange, asset and pair must support the request.

## Choose what the AI can access

The [permission reference](../../cmd/gctmcp/README.md#configuration) describes all
five permissions. Keep `mode` as `readonly` for inspection. Enabling `accountData`
permits private account queries, and their results are sent to your chosen AI
client and may be processed by its AI provider.

Changing `mode` to `execution` permits state changes only when the relevant
permission is also enabled. Order management, withdrawals and system control
are separate choices. These grants authorise matching tool calls; MCP does not
require a separate human confirmation for each call or impose spending limits.
Your AI client's approval controls are an additional setting.

Reconnect the MCP connection after changing its policy. GCT continues using its
normal configuration and exchange permissions.

## If a request does not work

| What you see | What to check |
| --- | --- |
| No GCT tools in the AI app | Check the executable's full path, the policy file path and the client's connection log. The executable must be available on the computer running the MCP process. |
| MCP is disabled | Check `remoteControl.mcp.enabled` in the policy selected by `--config`. Reconnect after editing it. |
| A saved record is unavailable | Check that GCT recording is enabled, GCT has run with that setting, and `--diagnostics-dir` points to its records. The MCP permission alone does not create recordings. |
| Recording is unavailable after a restart | Check for another GCT process using the same diagnostic directory. Follow the recorder recovery guidance above if a previous process has exited; old observations alone do not justify removing its lock. |
| A live query cannot connect | Check that GCT is running with RPC enabled, then check the RPC address, certificate, username and password supplied to the MCP process. |
| A command is denied | Check the policy's mode and permissions. Some operations also need order-management permission because they affect managed orders or account connections. |
| Results look old or incomplete | Check source and capture times. Ask which evidence is missing and whether a response was truncated. |

The [command reference](../../cmd/gctmcp/README.md) tables every additional MCP
command and explains refreshes, output limits and the optional exit recorder.
For the underlying RPC command options, use the
[GCT CLI reference](../../cmd/gctcli/README.md).
