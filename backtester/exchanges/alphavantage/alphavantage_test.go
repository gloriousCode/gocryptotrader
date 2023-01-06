package alphavantage

import "testing"

func TestSendRequest(t *testing.T) {
	t.Parallel()
	l := LemonExchange{}
	var resp InstrumentsResponse
	err := l.SendRequest(nil,
		&resp,
		instrumentEndpoint,
		"")
	if err != nil {
		t.Error(err)
	}
	t.Log(resp)
}
