package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
)

const defaultSleepTime = time.Second * 30

func containsOTP(cfg *config.Config) bool {
	for x := range cfg.Exchanges {
		if cfg.Exchanges[x].API.Credentials.OTPSecret != "" {
			return true
		}
	}
	return false
}

func main() {
	var cfgFile, code string
	var single bool
	var err error

	flag.StringVar(&cfgFile, "config", config.DefaultFilePath(), "The config input file to process.")
	flag.BoolVar(&single, "single", false, "prompt for single use OTP code gen")
	flag.Parse()

	log.Println("GoCryptoTrader: OTP code generator tool.")
	log.Println(core.Copyright)

	// Handle single use OTP code gen
	if single {
		var input string
		for {
			log.Println("Please enter in your OTP secret:")
			if _, err = fmt.Scanln(&input); err != nil {
				log.Println("Failed to read input. Err: " + err.Error())
				continue
			}
			if input != "" {
				break
			}
		}

		for {
			code, err = totp.GenerateCode(input, time.Now())
			if err != nil {
				log.Fatalf("Unable to generate OTP code. Err: %s", err)
			}
			log.Println("OTP code: " + code)
			time.Sleep(defaultSleepTime)
		}
	}

	// Otherwise default to loading the config file and generating OTP codes from it
	var cfg config.Config
	err = cfg.LoadConfig(cfgFile, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Loaded config file.")

	if !containsOTP(&cfg) {
		log.Fatal("No exchanges with OTP code stored. Exiting.")
	}

	for {
		for x := range cfg.Exchanges {
			if cfg.Exchanges[x].API.Credentials.OTPSecret != "" {
				code, err = totp.GenerateCode(cfg.Exchanges[x].API.Credentials.OTPSecret, time.Now())
				if err != nil {
					log.Println("Exchange " + cfg.Exchanges[x].Name + ": Failed to generate OTP code. Err: " + err.Error())
					continue
				}
				log.Println(cfg.Exchanges[x].Name + ": " + code)
			}
		}
		time.Sleep(defaultSleepTime)
	}
}
