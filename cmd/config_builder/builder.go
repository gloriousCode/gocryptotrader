package main

import (
	"context"
	"log"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func main() {
	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		log.Fatalf("Failed to initialise engine. Err: %s", err)
	}

	log.Println("Loading exchanges..")
	var wg sync.WaitGroup
	for i := range exchange.Exchanges {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err = engine.Bot.LoadExchange(name); err != nil {
				log.Println("Failed to load exchange " + name + ". Err: " + err.Error())
			}
		}(exchange.Exchanges[i])
	}
	wg.Wait()
	log.Println("Done.")

	exchanges := engine.Bot.GetExchanges()
	cfgs := make([]config.Exchange, 0, len(exchanges))
	for x := range exchanges {
		var cfg *config.Exchange
		cfg, err = exchange.GetDefaultConfig(context.Background(), exchanges[x])
		if err != nil {
			log.Println("Failed to get exchanges default config. Err: " + err.Error())
			continue
		}
		log.Println("Adding " + exchanges[x].GetName())
		cfgs = append(cfgs, *cfg)
	}

	data, err := json.MarshalIndent(cfgs, "", " ")
	if err != nil {
		log.Fatalf("Unable to marshal cfgs. Err: %s", err)
	}

	log.Println(string(data))
}
