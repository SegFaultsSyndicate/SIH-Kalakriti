// pkg/topics/topics_test.go
package topics

import "testing"

func TestAllTopicsAreUniqueAndKnown(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, len(All))
	for _, topic := range All {
		if topic == "" {
			t.Fatal("empty topic name in All")
		}
		if _, dup := seen[topic]; dup {
			t.Fatalf("duplicate topic %q in All", topic)
		}
		seen[topic] = struct{}{}

		if !IsKnown(topic) {
			t.Errorf("IsKnown(%q) = false, want true", topic)
		}
	}

	if got, want := len(All), 36; got != want {
		t.Errorf("len(All) = %d, want %d", got, want)
	}
}

func TestDLQ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		topic string
		want  string
	}{
		{name: "media", topic: MediaUploaded, want: "media.uploaded.dlq"},
		{name: "order lot", topic: OrderLotOffered, want: "order.lot.offered.dlq"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DLQ(tt.topic); got != tt.want {
				t.Errorf("DLQ(%q) = %q, want %q", tt.topic, got, tt.want)
			}
		})
	}
}

func TestIsKnownRejectsUnknown(t *testing.T) {
	t.Parallel()

	if IsKnown("order.lot.rejected") {
		t.Error(`IsKnown("order.lot.rejected") = true, want false`)
	}
}
