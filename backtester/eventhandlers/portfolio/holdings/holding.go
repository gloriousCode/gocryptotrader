package holdings

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetLatestSnapshot returns the last holding snapshot
func (s *Snapshots) GetLatestSnapshot() Holding {
	if len(s.Holdings) == 0 {
		return Holding{}
	}
	return s.Holdings[len(s.Holdings)-1]
}

// GetLatestSnapshot returns a specific snapshot at a given time
func (s *Snapshots) GetSnapshotAtTimestamp(t time.Time) Holding {
	for i := range s.Holdings {
		if t.Equal(s.Holdings[i].Timestamp) {
			return s.Holdings[i]
		}
	}
	return Holding{}
}

// Create takes a fill event and creates a new holding for the exchange, asset, pair
func Create(f fill.Event, initialFunds, riskFreeRate float64) (Holding, error) {
	if f == nil {
		return Holding{}, errors.New("nil event received")
	}
	if initialFunds <= 0 {
		return Holding{}, errors.New("initial funds <= 0")
	}
	h := Holding{
		Pair:           f.Pair(),
		Asset:          f.GetAssetType(),
		Exchange:       f.GetExchange(),
		Timestamp:      f.GetTime(),
		InitialFunds:   initialFunds,
		RemainingFunds: initialFunds,
		RiskFreeRate:   riskFreeRate,
	}
	h.update(f)

	return h, nil
}

// Update calculates holding statistics for the events time
func (h *Holding) Update(f fill.Event) {
	h.Timestamp = f.GetTime()
	h.update(f)
}

// UpdateValue calculates the holding's value for a data event's time and price
func (h *Holding) UpdateValue(d common.DataEventHandler) {
	h.Timestamp = d.GetTime()
	latest := d.Price()
	h.updateValue(latest)
}

func (h *Holding) update(f fill.Event) {
	direction := f.GetDirection()
	o := f.GetOrder()
	switch direction {
	case order.Buy:
		h.PositionsSize += o.Amount
		h.PositionsValue += o.Amount * o.Price
		h.RemainingFunds -= (o.Amount * o.Price) + o.Fee
		h.TotalFees += o.Fee
		h.BoughtAmount += o.Amount
		h.BoughtValue += o.Amount * o.Price
	case order.Sell:
		h.PositionsSize -= o.Amount
		h.PositionsValue -= o.Amount * o.Price
		h.RemainingFunds += (o.Amount * o.Price) - o.Fee
		h.TotalFees += o.Fee
		h.SoldAmount += o.Amount
		h.SoldValue += o.Amount * o.Price
	case common.DoNothing, common.CouldNotSell, common.CouldNotBuy, common.MissingData, "":
	}
	h.TotalValueLostToSlippage += (f.GetVolumeAdjustedPrice() - f.GetPurchasePrice()) * f.GetAmount()
	h.TotalValueLostToVolumeSizing += (f.GetClosePrice() - f.GetVolumeAdjustedPrice()) * f.GetAmount()
	h.updateValue(f.GetClosePrice())
}

func (h *Holding) updateValue(l float64) {
	origPosValue := h.PositionsValue
	origBoughtValue := h.PositionsValue
	origSoldValue := h.SoldValue
	origTotalValue := h.TotalValue
	h.PositionsValue = h.PositionsSize * l
	h.BoughtValue = h.BoughtAmount * l
	h.SoldValue = h.SoldAmount * l
	h.TotalValue = h.PositionsValue + h.RemainingFunds

	h.TotalValueDifference = h.TotalValue - origTotalValue
	h.BoughtValueDifference = h.BoughtValue - origBoughtValue
	h.PositionsValueDifference = h.PositionsValue - origPosValue
	h.SoldValueDifference = h.SoldValue - origSoldValue

	if origTotalValue != 0 {
		h.ChangeInTotalValuePercent = (h.TotalValue - origTotalValue) / origTotalValue
	}
	h.ExcessReturnPercent = h.ChangeInTotalValuePercent - h.RiskFreeRate
}
