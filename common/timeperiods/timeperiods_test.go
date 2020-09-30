package timeperiods

import (
	"testing"
	"time"
)

func TestTDDTimeRangeCombiner(t *testing.T) {
	// validation issues
	ranges, err := CalculateDataTimeRanges(
		time.Time{},
		time.Time{},
		0,
		nil,
	)
	if err != nil && err.Error() != "invalid start time, invalid end time, invalid period" {
		t.Fatal(err)
	}
	// empty trade times
	searchStartTime := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)
	searchEndTime := time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)
	var tradeTimes []time.Time
	ranges, err = CalculateDataTimeRanges(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	if err != nil {
		t.Error(err)
	}
	if len(ranges) != 1 {
		t.Errorf("expected 1 time range, received %v", len(ranges))
	}
	// 1 trade with 3 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC))
	ranges, err = CalculateDataTimeRanges(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	if err != nil {
		t.Error(err)
	}
	if len(ranges) != 3 {
		t.Errorf("expected 3 time ranges, received %v", len(ranges))
	}
	// 2 trades with 3 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 3, 0, 0, 0, time.UTC))
	ranges, err = CalculateDataTimeRanges(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	if err != nil {
		t.Error(err)
	}
	if len(ranges) != 3 {
		t.Errorf("expected 3 time ranges, received %v", len(ranges))
	}
	// 3 trades with 5 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 5, 0, 0, 0, time.UTC))
	ranges, err = CalculateDataTimeRanges(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	if err != nil {
		t.Error(err)
	}
	if len(ranges) != 5 {
		t.Errorf("expected 5 time ranges, received %v", len(ranges))
	}
	// 4 trades with 5 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 6, 0, 0, 0, time.UTC))
	ranges, err = CalculateDataTimeRanges(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	if err != nil {
		t.Error(err)
	}
	if len(ranges) != 5 {
		t.Errorf("expected 5 time ranges, received %v", len(ranges))
	}
	// 5 trades with 7 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC))
	ranges, err = CalculateDataTimeRanges(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	if err != nil {
		t.Error(err)
	}
	if len(ranges) != 7 {
		t.Errorf("expected 7 time ranges, received %v", len(ranges))
	}
}

func TestCalculateTimePeriodsInRange(t *testing.T) {
	// validation issues
	intervals, err := CalculateTimePeriodsInRange(time.Time{}, time.Time{}, 0)
	if err != nil && err.Error() != "invalid start time, invalid end time, invalid period" {
		t.Fatal(err)
	}
	// start after end
	intervals, err = CalculateTimePeriodsInRange(time.Now(), time.Now().Add(-time.Hour), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 0 {
		t.Errorf("expected 0 interval(s), received %v", len(intervals))
	}
	// 1 interval
	intervals, err = CalculateTimePeriodsInRange(time.Now().Add(-time.Hour), time.Now(), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 1 {
		t.Errorf("expected 1 interval(s), received %v", len(intervals))
	}
	// multiple intervals
	intervals, err = CalculateTimePeriodsInRange(time.Now().Add(-time.Hour*24), time.Now(), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 24 {
		t.Errorf("expected 24 interval(s), received %v", len(intervals))
	}
	// odd times
	intervals, err = CalculateTimePeriodsInRange(time.Now().Add(-(time.Hour*24)-(time.Minute*16)), time.Now(), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 25 {
		t.Errorf("expected 25 interval(s), received %v", len(intervals))
	}
	// truncate always goes to zero, no mid rounding
	intervals, err = CalculateTimePeriodsInRange(time.Now().Add(-time.Minute*46), time.Now(), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 1 {
		t.Errorf("expected 1 interval(s), received %v", len(intervals))
	}
	// interval too large
	intervals, err = CalculateTimePeriodsInRange(time.Now().Add(-time.Hour), time.Now(), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 1 {
		t.Errorf("expected 1 interval(s), received %v", len(intervals))
	}
}

func TestValidateCalculatePeriods(t *testing.T) {
	var lol TimePeriodCalculator
	lol.calculatePeriods()
	if len(lol.TimePeriods) > 0 {
		t.Error("validation has been removed")
	}
}
