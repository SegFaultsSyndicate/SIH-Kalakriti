// pkg/webhook/webhook_test.go
package webhook

import (
	"net/netip"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.0.0.5", "192.168.1.1", "172.16.0.1",
		"169.254.169.254", // cloud metadata endpoint
		"0.0.0.0",
		"::1", "fc00::1", "fe80::1",
	}
	for _, s := range blocked {
		ip, err := netip.ParseAddr(s)
		if err != nil {
			t.Fatalf("parsing %q: %v", s, err)
		}
		if !isBlockedIP(ip) {
			t.Errorf("isBlockedIP(%q) = false, want true", s)
		}
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "203.0.113.10"}
	for _, s := range allowed {
		ip, err := netip.ParseAddr(s)
		if err != nil {
			t.Fatalf("parsing %q: %v", s, err)
		}
		if isBlockedIP(ip) {
			t.Errorf("isBlockedIP(%q) = true, want false", s)
		}
	}
}

func TestValidateSubscriptionURL(t *testing.T) {
	bad := []string{
		"http://example.com/hook",       // not https
		"https://127.0.0.1/hook",        // loopback literal
		"https://169.254.169.254/hook",  // metadata literal
		"not-a-url",
		"https:///hook", // no host
	}
	for _, u := range bad {
		if err := validateSubscriptionURL(u); err == nil {
			t.Errorf("validateSubscriptionURL(%q) = nil error, want rejection", u)
		}
	}

	good := []string{"https://example.com/hook", "https://api.buyer.example.com:8443/hooks/kalakriti"}
	for _, u := range good {
		if err := validateSubscriptionURL(u); err != nil {
			t.Errorf("validateSubscriptionURL(%q) = %v, want nil", u, err)
		}
	}
}

func TestBackoffFor(t *testing.T) {
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{1, 1 * time.Minute},
		{2, 2 * time.Minute},
		{3, 4 * time.Minute},
		{7, 64 * time.Minute},  // 1<<6 = 64, exactly the cap
		{8, 64 * time.Minute},  // 1<<7 = 128, capped
		{20, 64 * time.Minute}, // stays capped, never overflows
	}
	for _, c := range cases {
		if got := backoffFor(c.attempts); got != c.want {
			t.Errorf("backoffFor(%d) = %v, want %v", c.attempts, got, c.want)
		}
	}
}
