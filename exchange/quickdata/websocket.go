package quickdata

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

type AnyKey = struct {
	Exchange      string
	ConnectionKey any
}

type wsHolder struct {
	store map[AnyKey]*wsStoreButts
	m     sync.Mutex
}

var store = wsHolder{
	store: make(map[AnyKey]*wsStoreButts),
}

type wsStoreButts = struct {
	Conn       websocket.Connection
	Manager    *websocket.Manager
	DataPusher map[key.ExchangeAssetPair]chan any
}

func Store(key AnyKey, conn *wsStoreButts) {
	store.m.Lock()
	defer store.m.Unlock()
	store.store[key] = conn
}

func Get(key AnyKey) *wsStoreButts {
	store.m.Lock()
	defer store.m.Unlock()
	return store.store[key]
}
