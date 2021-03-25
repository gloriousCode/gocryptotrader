package tradetracker

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Config struct {
	m                sync.Mutex
	Synchronisations []SynchronisationConfig
	Validations      []SynchronisationConfig
	MaxWorkers       uint32
}

type SynchronisationConfig struct {
	RequestsPerJob uint32
	Enabled        bool
	StartDate      time.Time
	EndDate        time.Time
	Currency       string
	Asset          string
	DataType       string
	Exchange       string
	Interval       time.Duration
	RetriesAllowed uint32
}

type tracker struct {
	enabled        bool
	startDate      time.Time
	endDate        time.Time
	currency       currency.Pair
	asset          asset.Item
	exchange       exchange.IBotExchange
	dataType       string
	interval       kline.Interval
	retriesAllowed uint32
	lastUpdated    time.Time
	running        uint32
	requestsPerJob uint32
}

type butterino struct {
	synchronisers []*tracker
	validators    []*tracker
	maxworkers    uint32
	running       uint32
}

func (b *butterino) canWork() bool {
	var runners uint32
	for i := range b.synchronisers {
		if atomic.LoadUint32(&b.synchronisers[i].running) == 1 {
			runners++
		}
	}
	for i := range b.validators {
		if atomic.LoadUint32(&b.validators[i].running) == 1 {
			runners++
		}
	}
	return runners < b.maxworkers
}

func Setup(cfg *Config) error {
	cfg.m.Lock()
	defer cfg.m.Unlock()

	b := butterino{
		maxworkers: cfg.MaxWorkers,
	}

	for i := range cfg.Synchronisations {
		t := &tracker{
			enabled:        cfg.Synchronisations[i].Enabled,
			startDate:      cfg.Synchronisations[i].StartDate,
			endDate:        cfg.Synchronisations[i].EndDate,
			currency:       cfg.Synchronisations[i].Currency,
			asset:          cfg.Synchronisations[i].Asset,
			exchange:       cfg.Synchronisations[i].Exchange,
			interval:       cfg.Synchronisations[i].Interval,
			retriesAllowed: cfg.Synchronisations[i].RetriesAllowed,
			dataType:       cfg.Synchronisations[i].DataType,
		}
		b.synchronisers = append(b.synchronisers, t)
	}

	for i := range cfg.Validations {
		t := &tracker{
			enabled:        cfg.Validations[i].Enabled,
			startDate:      cfg.Validations[i].StartDate,
			endDate:        cfg.Validations[i].EndDate,
			currency:       cfg.Validations[i].Currency,
			asset:          cfg.Validations[i].Asset,
			exchange:       cfg.Validations[i].Exchange,
			interval:       cfg.Validations[i].Interval,
			retriesAllowed: cfg.Validations[i].RetriesAllowed,
			dataType:       cfg.Validations[i].DataType,
		}
		b.validators = append(b.validators, t)
	}
	return nil
}

func (t *tracker) runSynchronise() {

}

func (t *tracker) runValidation() {

}
