package orderbook

// RetrieveLimited captures bounded levels and their source timestamps under one lock.
func (d *Depth) RetrieveLimited(limit int) (*Book, error) {
	if limit <= 0 {
		return nil, errInvalidBookDepth
	}
	d.m.RLock()
	defer d.m.RUnlock()
	if d.validationError != nil {
		return nil, d.validationError
	}
	return &Book{
		Bids:         d.bidLevels.retrieve(limit),
		Asks:         d.askLevels.retrieve(limit),
		Exchange:     d.exchange,
		Pair:         d.pair,
		Asset:        d.asset,
		LastUpdated:  d.lastUpdated,
		LastPushed:   d.lastPushed,
		InsertedAt:   d.insertedAt,
		LastUpdateID: d.lastUpdateID,
		RestSnapshot: d.restSnapshot,
	}, nil
}
