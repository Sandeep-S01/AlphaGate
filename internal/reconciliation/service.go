package reconciliation

import (
	"context"
	"fmt"
	"math"
	"time"
)

type SnapshotReader interface {
	Snapshot(ctx context.Context) (Snapshot, error)
}

type Store interface {
	Save(ctx context.Context, run Run) (Run, error)
}

type Dependencies struct {
	Internal SnapshotReader
	External SnapshotReader
	Store    Store
	Now      func() time.Time
}

type Service struct {
	deps Dependencies
}

func NewService(deps Dependencies) *Service {
	if deps.Now == nil {
		deps.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{deps: deps}
}

func (s *Service) Run(ctx context.Context) (Run, error) {
	internal, err := s.deps.Internal.Snapshot(ctx)
	if err != nil {
		return Run{}, fmt.Errorf("read internal snapshot: %w", err)
	}
	external, err := s.deps.External.Snapshot(ctx)
	if err != nil {
		return Run{}, fmt.Errorf("read external snapshot: %w", err)
	}
	run := Run{
		Status:     StatusMatched,
		Mismatches: compareSnapshots(internal, external),
		CreatedAt:  s.deps.Now(),
	}
	if len(run.Mismatches) > 0 {
		run.Status = StatusMismatch
	}
	if s.deps.Store == nil {
		return run, nil
	}
	saved, err := s.deps.Store.Save(ctx, run)
	if err != nil {
		return Run{}, fmt.Errorf("save reconciliation run: %w", err)
	}
	return saved, nil
}

func compareSnapshots(internal Snapshot, external Snapshot) []Mismatch {
	mismatches := []Mismatch{}
	internalBalances := map[string]float64{}
	for _, balance := range internal.Balances {
		internalBalances[balance.Asset] = balance.Free
	}
	externalBalances := map[string]float64{}
	for _, balance := range external.Balances {
		externalBalances[balance.Asset] = balance.Free
	}
	for asset, internalValue := range internalBalances {
		externalValue := externalBalances[asset]
		if math.Abs(internalValue-externalValue) > 0.00000001 {
			mismatches = append(mismatches, Mismatch{
				Kind:          MismatchBalance,
				Key:           asset,
				InternalValue: fmt.Sprintf("%.8f", internalValue),
				ExternalValue: fmt.Sprintf("%.8f", externalValue),
				Severity:      "warning",
			})
		}
	}
	internalOrders := map[string]Order{}
	for _, order := range internal.Orders {
		internalOrders[order.ClientOrderID] = order
	}
	externalOrders := map[string]Order{}
	for _, order := range external.Orders {
		externalOrders[order.ClientOrderID] = order
	}
	for clientOrderID, internalOrder := range internalOrders {
		externalOrder, found := externalOrders[clientOrderID]
		if !found {
			mismatches = append(mismatches, Mismatch{
				Kind:          MismatchOrder,
				Key:           clientOrderID,
				InternalValue: internalOrder.Status,
				ExternalValue: "missing",
				Severity:      "critical",
			})
			continue
		}
		if internalOrder.Status != externalOrder.Status {
			mismatches = append(mismatches, Mismatch{
				Kind:          MismatchOrder,
				Key:           clientOrderID,
				InternalValue: internalOrder.Status,
				ExternalValue: externalOrder.Status,
				Severity:      "warning",
			})
		}
	}
	return mismatches
}
