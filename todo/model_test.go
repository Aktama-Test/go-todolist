package todo

import "testing"

func TestPriorityValid(t *testing.T) {
	for _, p := range []Priority{PriorityLow, PriorityMedium, PriorityHigh} {
		if !p.Valid() {
			t.Errorf("Priority(%q).Valid() = false, want true", p)
		}
	}
	if Priority("urgent").Valid() {
		t.Error(`Priority("urgent").Valid() = true, want false`)
	}
}
