package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Struct to unmarshal the API response
type KlineResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Symbol             string `json:"symbol"`
		Interval           string `json:"interval"`
		OpenTime           int64  `json:"openTime"`
		CloseTime          int64  `json:"closeTime"`
		Open               string `json:"open"`
		High               string `json:"high"`
		Low                string `json:"low"`
		Close              string `json:"close"`
		Volume             string `json:"volume"`
		NumberOfTrades     int    `json:"numberOfTrades"`
		QuoteAssetVolume   string `json:"quoteAssetVolume"`
		TakerBuyBaseAsset  string `json:"takerBuyBaseAsset"`
		TakerBuyQuoteAsset string `json:"takerBuyQuoteAsset"`
	} `json:"data"`
}

// Function to retrieve Kline data for a specific contract
func GetContractKline(symbol string, interval string, startTime int64, endTime int64, limit int) (*KlineResponse, error) {
	baseURL := "https://fapi.binance.com/fapi/v1/historicalKlines"

	// Prepare parameters
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	params.Set("startTime", strconv.FormatInt(startTime, 10))
	params.Set("endTime", strconv.FormatInt(endTime, 10))
	params.Set("limit", strconv.Itoa(limit))

	// Prepare request
	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	// Execute request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	// Decode response body
	var klineResp KlineResponse
	err = json.NewDecoder(resp.Body).Decode(&klineResp)
	if err != nil {
		return nil, err
	}

	return &klineResp, nil
}

// Function to calculate Binance Futures quarterly contract expiration dates for a range of months
func GetExpiredContractDates(startDate time.Time, numMonths int) []time.Time {
	var expirationDates []time.Time

	// Loop through the specified number of months
	for i := 0; i < numMonths; i++ {
		// Calculate the expiration date for the current month
		expirationDate := startDate.AddDate(0, i+3, 0)

		// Append the expiration date to the slice
		expirationDates = append(expirationDates, expirationDate)
	}

	return expirationDates
}

func main() {
	// Example start date (January 1, 2024)
	startDate := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	numMonths := 6   // Example number of months
	interval := "1d" // Example interval (daily)
	limit := 100     // Example limit

	// Get the range of expired contract dates
	expiredContractDates := GetExpiredContractDates(startDate, numMonths)

	// Loop through each expired contract date
	for _, date := range expiredContractDates {
		// Calculate contract symbol
		contractSymbol := fmt.Sprintf("BTCUSD_%02d%02d%02d", date.Year()%100, date.Month(), date.Day())

		// Calculate start and end times for Kline data retrieval
		startTime := date.AddDate(0, 0, -30) // 30 days before expiration
		endTime := date                      // Expired contract date

		// Retrieve Kline data for the expired contract
		klineResp, err := GetContractKline(contractSymbol, interval, startTime.Unix()*1000, endTime.Unix()*1000, limit)
		if err != nil {
			fmt.Printf("Error retrieving Kline data for %s: %v\n", contractSymbol, err)
			continue
		}

		// Print Kline data
		fmt.Printf("Kline data for %s:\n", contractSymbol)
		for _, kline := range klineResp.Data {
			fmt.Printf("Open Time: %d, Close Time: %d, Open: %s, Close: %s\n", kline.OpenTime, kline.CloseTime, kline.Open, kline.Close)
		}
	}
}
