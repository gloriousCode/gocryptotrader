package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli"
)

var (
	guidExample            = "deadbeef-dead-beef-dead-beef13371337"
	specificJobSubCommands = []cli.Flag{
		cli.StringFlag{
			Name:  "id",
			Usage: guidExample,
		},
		cli.StringFlag{
			Name:  "nickname",
			Usage: "binance-spot-btc-usdt-2019-trades",
		},
	}
	fullJobSubCommands = []cli.Flag{
		cli.StringFlag{
			Name:     "nickname",
			Usage:    "binance-spot-btc-usdt-2019-trades",
			Required: true,
		},
		cli.StringFlag{
			Name:  "exchange",
			Usage: "binance",
		},
		cli.StringFlag{
			Name:  "asset",
			Usage: "spot",
		},
		cli.StringFlag{
			Name:  "pair",
			Usage: "btc-usdt",
		},
		cli.StringFlag{
			Name:        "start_date",
			Usage:       "formatted as: 2006-01-02 15:04:05",
			Value:       time.Now().AddDate(-1, 0, 0).Format(common.SimpleTimeFormat),
			Destination: &startTime,
		},
		cli.StringFlag{
			Name:        "end_date",
			Usage:       "formatted as: 2006-01-02 15:04:05",
			Value:       time.Now().AddDate(0, -1, 0).Format(common.SimpleTimeFormat),
			Destination: &endTime,
		},
		cli.StringFlag{
			Name:  "interval",
			Usage: klineMessage,
		},
		cli.StringFlag{
			Name:  "request_size_limit",
			Usage: "500 - will only retrieve 500 candles per API request",
		},
		cli.StringFlag{
			Name:  "data_type",
			Usage: "0 for candles, 1 for trades",
		},
		cli.StringFlag{
			Name:  "max_retry_attempts",
			Usage: "3 - the maximum retry attempts for an interval period before giving up",
		},
		cli.StringFlag{
			Name:  "batch_size",
			Usage: "3 - the amount of API calls to make per run",
		},
	}
)

var dataHistoryCommands = cli.Command{
	Name:      "datahistory",
	Usage:     "manage data history jobs to retrieve historic trade or candle data over time",
	ArgsUsage: "<command> <args>",
	Subcommands: []cli.Command{
		{
			Name:   "getactivejobs",
			Usage:  "returns all jobs that are currently active",
			Flags:  []cli.Flag{},
			Action: getActiveDataHistoryJobs,
		},
		{
			Name:  "getjobsbetweendates",
			Usage: "returns all jobs with creation dates between the two provided dates",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "start_date",
					Usage:       "formatted as: 2006-01-02 15:04:05",
					Value:       time.Now().AddDate(0, -1, 0).Format(common.SimpleTimeFormat),
					Destination: &startTime,
				},
				cli.StringFlag{
					Name:        "end_date",
					Usage:       "formatted as: 2006-01-02 15:04:05",
					Value:       time.Now().Format(common.SimpleTimeFormat),
					Destination: &endTime,
				},
			},
			Action: getDataHistoryJobsBetween,
		},
		{
			Name:        "getajob",
			Usage:       "returns a job by either its id or nickname",
			Description: "na-na, why don't you get a job?",
			ArgsUsage:   "<id> or <nickname>",
			Action:      getDataHistoryJob,
			Flags:       specificJobSubCommands,
		},
		{
			Name:        "getjobwithdetailedresults",
			Usage:       "returns a job by either its nickname along with all its data retrieval results",
			Description: "results may be large",
			ArgsUsage:   "<nickname>",
			Action:      getDataHistoryJob,
			Flags: []cli.Flag{cli.StringFlag{
				Name:  "nickname",
				Usage: "binance-spot-btc-usdt-2019-trades",
			},
			}},
		{
			Name:      "getjobstatussummary",
			Usage:     "returns a job with human readable summary of its status",
			ArgsUsage: "<nickname>",
			Action:    getDataHistoryJobSummary,
			Flags: []cli.Flag{cli.StringFlag{
				Name:  "nickname",
				Usage: "binance-spot-btc-usdt-2019-trades",
			}},
		},
		{
			Name:   "addnewjob",
			Usage:  "creates a new data history job",
			Flags:  fullJobSubCommands,
			Action: upsertDataHistoryJob,
		},
		{
			Name:   "upsertjob",
			Usage:  "adds a new job, or updates an existing one if it matches jobid OR nickname",
			Flags:  fullJobSubCommands,
			Action: upsertDataHistoryJob,
		},
		{
			Name:      "deletejob",
			Usage:     "sets a jobs status to deleted so it no longer is processed",
			ArgsUsage: "<id> or <nickname>",
			Flags:     specificJobSubCommands,
			Action:    deleteDataHistoryJob,
		},
	},
}

func getDataHistoryJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var id string
	if c.IsSet("id") {
		id = c.String("id")
	}
	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	}

	if nickname != "" && id != "" {
		return errors.New("can only set 'id' OR 'nickname'")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.GetDataHistoryJobDetailsRequest{
		Id:       id,
		Nickname: nickname,
	}
	if strings.EqualFold(c.Command.Name, "getjobwithdetailedresults") {
		request.FullDetails = true
	}

	result, err := client.GetDataHistoryJobDetails(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getActiveDataHistoryJobs(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetActiveDataHistoryJobs(context.Background(),
		&gctrpc.GetInfoRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func upsertDataHistoryJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var (
		err                                                               error
		nickname, exchange, assetType, pair                               string
		interval, requestSizeLimit, dataType, maxRetryAttempts, batchSize int64
	)
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	} else {
		nickname = c.Args().First()
	}

	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().Get(1)
	}
	if !validExchange(exchange) {
		return errInvalidExchange
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("pair") {
		pair = c.String("pair")
	} else {
		pair = c.Args().Get(3)
	}
	if !validPair(pair) {
		return errInvalidPair
	}
	p, err := currency.NewPairDelimiter(pair, pairDelimiter)
	if err != nil {
		return fmt.Errorf("cannot process pair: %w", err)
	}

	if c.IsSet("start_date") {
		startTime = c.String("start_date")
	}
	if c.IsSet("end_date") {
		endTime = c.String("end_date")
	}

	var s, e time.Time
	s, err = time.Parse(common.SimpleTimeFormat, startTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.Parse(common.SimpleTimeFormat, endTime)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if c.IsSet("interval") {
		interval = c.Int64("interval")
	} else {
		interval, err = convert.Int64FromString(c.Args().Get(6))
		if err != nil {
			return fmt.Errorf("cannot process interval: %w", err)
		}
	}
	candleInterval := time.Duration(interval) * time.Second

	if c.IsSet("request_size_limit") {
		requestSizeLimit = c.Int64("request_size_limit")
	} else {
		requestSizeLimit, err = convert.Int64FromString(c.Args().Get(7))
		if err != nil {
			return fmt.Errorf("cannot process request_size_limit: %w", err)
		}
	}

	if c.IsSet("data_type") {
		dataType = c.Int64("data_type")
	} else {
		dataType, err = convert.Int64FromString(c.Args().Get(8))
		if err != nil {
			return fmt.Errorf("cannot process data_type: %w", err)
		}
	}

	if c.IsSet("max_retry_attempts") {
		maxRetryAttempts = c.Int64("max_retry_attempts")
	} else {
		maxRetryAttempts, err = convert.Int64FromString(c.Args().Get(9))
		if err != nil {
			return fmt.Errorf("cannot process max_retry_attempts: %w", err)
		}
	}

	if c.IsSet("batch_size") {
		batchSize = c.Int64("batch_size")
	} else {
		batchSize, err = convert.Int64FromString(c.Args().Get(10))
		if err != nil {
			return fmt.Errorf("cannot process batch_size: %w", err)
		}
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.UpsertDataHistoryJobRequest{
		Nickname: nickname,
		Exchange: exchange,
		Asset:    assetType,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		StartDate:        negateLocalOffset(s),
		EndDate:          negateLocalOffset(e),
		Interval:         int64(candleInterval),
		RequestSizeLimit: requestSizeLimit,
		DataType:         dataType,
		MaxRetryAttempts: maxRetryAttempts,
		BatchSize:        batchSize,
	}
	if strings.EqualFold(c.Command.Name, "addnewjob") {
		request.InsertOnly = true
	}

	result, err := client.UpsertDataHistoryJob(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getDataHistoryJobsBetween(c *cli.Context) error {
	if c.IsSet("start_date") {
		startTime = c.String("start_date")
	}
	if c.IsSet("end_date") {
		endTime = c.String("end_date")
	}
	s, err := time.Parse(common.SimpleTimeFormat, startTime)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.Parse(common.SimpleTimeFormat, endTime)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	if e.Before(s) {
		return errors.New("start cannot be after end")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetDataHistoryJobsBetween(context.Background(),
		&gctrpc.GetDataHistoryJobsBetweenRequest{
			StartDate: negateLocalOffset(s),
			EndDate:   negateLocalOffset(e),
		})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func deleteDataHistoryJob(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var id string
	if c.IsSet("id") {
		id = c.String("id")
	}
	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	}

	if nickname != "" && id != "" {
		return errors.New("can only set 'id' OR 'nickname'")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.GetDataHistoryJobDetailsRequest{
		Id:       id,
		Nickname: nickname,
	}

	result, err := client.DeleteDataHistoryJob(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getDataHistoryJobSummary(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowCommandHelp(c, c.Command.Name)
	}

	var nickname string
	if c.IsSet("nickname") {
		nickname = c.String("nickname")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()
	client := gctrpc.NewGoCryptoTraderClient(conn)
	request := &gctrpc.GetDataHistoryJobDetailsRequest{
		Nickname: nickname,
	}

	result, err := client.GetDataHistoryJobSummary(context.Background(), request)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
