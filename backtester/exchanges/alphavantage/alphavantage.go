package alphavantage

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Response struct {
	GlobalQuote struct {
		Symbol           string  `json:"01. symbol"`
		Open             float64 `json:"02. open"`
		High             float64 `json:"03. high"`
		Low              float64 `json:"04. low"`
		Price            float64 `json:"05. price"`
		Volume           int     `json:"06. volume"`
		LatestTradingDay string  `json:"07. latest trading day"`
		PreviousClose    float64 `json:"08. previous close"`
		Change           float64 `json:"09. change"`
		ChangePercent    float64 `json:"10. change percent"`
	} `json:"Global Quote"`
}

func main() {
	// Replace YOUR_API_KEY with your actual API key
	response, err := http.Get("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=MSFT&apikey=YOUR_API_KEY")
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return
	}
	defer response.Body.Close()

	data, _ := ioutil.ReadAll(response.Body)
	var quote Response
	json.Unmarshal(data, &quote)

	fmt.Printf("Stock quote for %s:\n", quote.GlobalQuote.Symbol)
	fmt.Printf("Open: %f\n", quote.GlobalQuote.Open)
	fmt.Printf("High: %f\n", quote.GlobalQuote.High)
	fmt.Printf("Low: %f\n", quote.GlobalQuote.Low)
	fmt.Printf("Price: %f\n", quote.GlobalQuote.Price)
	fmt.Printf("Volume: %d\n", quote.GlobalQuote.Volume)
	fmt.Printf("Latest trading day: %s\n", quote.GlobalQuote.LatestTradingDay)
	fmt.Printf("Previous close: %f\n", quote.GlobalQuote.PreviousClose)
	fmt.Printf("Change: %f\n", quote.GlobalQuote.Change)
	fmt.Printf("Change percent: %f\n", quote.GlobalQuote.ChangePercent)
}

func (l *LemonExchange) SendRequest(request *Request, resp interface{}, endpoint, apiKey string) error {
	jsonValue, _ := json.Marshal(request)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, baseURL+endpoint+"?search=tesla", strings.NewReader(string(jsonValue)))
	if err != nil {
		return err
	}

	// Set the API key in the request headers
	req.Header.Set("Authorization", "Bearer "+apiKey)

	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, resp)
	if err != nil {
		return err
	}

	return nil
}
