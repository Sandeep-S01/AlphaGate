package indicator

import "testing"

func TestExponentialMovingAverageWeightsRecentCloses(t *testing.T) {
	value, err := ExponentialMovingAverage([]float64{10, 12, 14}, 3)
	if err != nil {
		t.Fatalf("ExponentialMovingAverage returned error: %v", err)
	}

	if value != 12.5 {
		t.Fatalf("expected EMA 12.5, got %f", value)
	}
}
