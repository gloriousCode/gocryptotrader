package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type DummyConnection struct {
	Connection
	ch chan []byte
	u  string
}

func (d *DummyConnection) ReadMessage() Response {
	return Response{Raw: <-d.ch}
}

func (d *DummyConnection) Push(data []byte) {
	d.ch <- data
}

func (d *DummyConnection) GetURL() string {
	if d.u != "" {
		return d.u
	}
	return "ws://test"
}

func ProcessWithSomeSweetLag(context.Context, Connection, []byte) error {
	time.Sleep(time.Millisecond)
	return nil
}

func TestDefaultProcessReporter(t *testing.T) {
	t.Parallel()
	w := &Manager{}
	reporterManager := defaultProcessReporterManager{period: time.Millisecond * 10}
	w.SetProcessReportManager(&reporterManager)
	conn := &DummyConnection{ch: make(chan []byte)}
	w.Wg.Add(1)
	go w.Reader(t.Context(), conn, ProcessWithSomeSweetLag)

	for range 100 {
		conn.Push([]byte("test"))
	}
	conn.Push(nil)
}

func TestDefaultProcessReporterManagerConnectionIDs(t *testing.T) {
	t.Parallel()

	reporterManager := &defaultProcessReporterManager{period: time.Millisecond * 10}
	first := reporterManager.New(&DummyConnection{u: "ws://same"})
	second := reporterManager.New(&DummyConnection{u: "ws://same"})
	third := reporterManager.New(&DummyConnection{u: "ws://other"})

	firstReporter, ok := first.(*defaultProcessReporter)
	require.True(t, ok, "first reporter type assertion failed")
	secondReporter, ok := second.(*defaultProcessReporter)
	require.True(t, ok, "second reporter type assertion failed")
	thirdReporter, ok := third.(*defaultProcessReporter)
	require.True(t, ok, "third reporter type assertion failed")
	require.Equal(t, int64(1), firstReporter.connectionID, "unexpected first reporter connection id")
	require.Equal(t, int64(2), secondReporter.connectionID, "unexpected second reporter connection id")
	require.Equal(t, int64(3), thirdReporter.connectionID, "unexpected third reporter connection id")

	first.Close()
	second.Close()
	third.Close()
}
