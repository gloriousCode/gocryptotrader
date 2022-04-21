package okx

type GeneralResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

type PlaceOrderRequest struct {
	InstId  string `json:"instId"`
	TdMode  string `json:"tdMode"`
	ClOrdId string `json:"clOrdId"`
	Side    string `json:"side"`
	OrdType string `json:"ordType"`
	Px      string `json:"px"`
	Sz      string `json:"sz"`
}

type PlaceOrderResponse struct {
	ClOrdId string `json:"clOrdId"`
	OrdId   string `json:"ordId"`
	Tag     string `json:"tag"`
	SCode   string `json:"sCode"`
	SMsg    string `json:"sMsg"`
}

type CancelOrderRequest struct {
	OrdId  string `json:"ordId"`
	InstId string `json:"instId"`
}

type CancelOrderResponse struct {
	ClOrdId string `json:"clOrdId"`
	OrdId   string `json:"ordId"`
	SCode   string `json:"sCode"`
	SMsg    string `json:"sMsg"`
}
