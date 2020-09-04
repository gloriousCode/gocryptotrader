package traders

import two_outta_three "github.com/thrasher-corp/gocryptotrader/traders/two-outta-three"

func LoadTraders(names []string ) []ITrader {
	var resp []ITrader
	for i := range names {
		switch names[i] {
		case two_outta_three.Name:
			resp = append(resp, new(two_outta_three.TOT))
		}
	}
	return resp
}