package marketdata

import (
	"fmt"
	"sort"
	"time"
)

type Gap struct {
	From time.Time
	To   time.Time
}

func DetectGaps(from time.Time, to time.Time, interval string, existing []time.Time) ([]Gap, error) {
	step, err := IntervalDuration(interval)
	if err != nil {
		return nil, err
	}
	if !from.Before(to) {
		return nil, fmt.Errorf("from must be before to")
	}

	seen := make(map[time.Time]struct{}, len(existing))
	for _, openTime := range existing {
		seen[openTime.UTC()] = struct{}{}
	}

	var gaps []Gap
	var gapStart *time.Time
	for current := from.UTC(); current.Before(to); current = current.Add(step) {
		_, ok := seen[current]
		if !ok {
			if gapStart == nil {
				start := current
				gapStart = &start
			}
			continue
		}

		if gapStart != nil {
			gaps = append(gaps, Gap{From: *gapStart, To: current})
			gapStart = nil
		}
	}

	if gapStart != nil {
		gaps = append(gaps, Gap{From: *gapStart, To: to.UTC()})
	}

	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].From.Before(gaps[j].From)
	})
	return gaps, nil
}
