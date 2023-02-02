# GoCryptoTrader Backtester: Config package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/config)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This config package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Config package overview
This readme contains details for both the GoCryptoTrader Backtester config structure along with the strategy config structure

## GoCryptoTrader Backtester Config overview
Below are the details for the GoCryptoTrader Backtester _application_ config. Strategy config overview is below this section

| Key                     | Description                                                                                                                                              | Example                          |
|-------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------|
| print-logo              | Whether to print the GoCryptoTrader Backtester logo on startup. Recommended because it looks good                                                        | `true`                           |
| verbose                 | Whether to receive verbose output. If running a GRPC server, it outputs to the server, not to the client                                                 | `false`                          |
| log-subheaders          | Whether log output contains a descriptor of what area the log is coming from, for example `STRATEGY`. Helpful for debugging                              | `true`                           |
| stop-all-tasks-on-close | When closing the application, the Backtester will attempt to stop all active tasks                                                                       | `true`                           |
| plugin-path             | When using custom strategy plugins, you can enter the path here to automatically load the plugin                                                         | `true`                           |
| report                  | Contains details on the output report after a successful backtesting run                                                                                 | See Report table below           |
| grpc                    | Contains GRPC server details                                                                                                                             | See GRPC table below             |
| use-cmd-colours         | If enabled, will output pretty colours of your choosing when running the application                                                                     | `true`                           |
| cmd-colours             | Contains details on what the colour definitions are                                                                                                      | See Colours table below          |

### Backtester Config Report overview

| Key            | Description                                                          | Example                         |
|----------------|----------------------------------------------------------------------|---------------------------------|
| output-report  | Whether or not to output a report after a successful backtesting run | `true`                          |
| template-path  | The path for the template to use when generating a report            | `/backtester/report/tpl.gohtml` |
| output-path    | The path where report output is saved                                | `/backtester/results`           |
| dark-mode      | Whether or not the report defaults to using dark mode                | `true`                          |

### Backtester Config GRPC overview

| Key                    | Description                                                                                                                                 | Example                        |
|------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| username               | Your username to negotiate a successful connection with the server                                                                          | `rpcuser`                      |
| password               | Your password to negotiate a successful connection with the server                                                                          | `helloImTheDefaultPassword`    |
| enabled                | Whether the server is enabled. Setting this to `false` and `SingleRun` to `false` would be inadvisable                                      | `true`                         |
| listenAddress          | The listen address for the GRPC server                                                                                                      | `localhost:9054`               |
| grpcProxyEnabled       | If enabled, creates a proxy server to interact with the GRPC server via HTTP commands                                                       | `true`                         |
| grpcProxyListenAddress | The address for the proxy to listen on                                                                                                      | `localhost:9053`               |
| tls-dir                | The directory for holding your TLS certifications to make connections to the server. Will be generated by default on startup if not present | `/backtester/config/location/` |


### Backtester Config Colours overview

| Key      | Description                                                         | Example        |
|----------|---------------------------------------------------------------------|----------------|
| Default  | The colour definition for default text output                       |`[0m`        |
| Green    | The colour definition for when green is warranted, such as the logo |`[38;5;157m` |
| White    | The colour definition for when white is warranted such as the logo  |`[38;5;255m` |
| Grey     | The colour definition for grey                                      | `[38;5;240m`|
| DarkGrey | The colour definition for dark grey                                 | `[38;5;243m`|
| H1       | The colour definition for main headers                              | `[38;5;33m` |
| H2       | The colour definition for sub headers                               | `[38;5;39m` |
| H3       | The colour definition for sub sub headers                           | `[38;5;45m` |
| H4       | The colour definition for sub sub sub headers                       | `[38;5;51m` |
| Success  | The colour definition for successful operations                     | `[38;5;40m` |
| Info     | The colour definition for when informing you of something           | `[32m`      |
| Debug    | The colour definition for debug output such as verbose              | `[34m`      |
| Warn     | The colour definition for when a warning occurs                     | `[33m`      |
| Error    | The colour definition for when an error occurs                      | `[38;5;196m`|



### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-corp/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
