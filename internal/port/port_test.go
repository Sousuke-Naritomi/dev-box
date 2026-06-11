package port

import "testing"

func TestFindAvailable_ValidPort(t *testing.T) {
	p, err := FindAvailable(10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p < 10000 || p > MaxPort {
		t.Errorf("expected port in range [10000, %d], got %d", MaxPort, p)
	}
}

func TestFindAvailable_InvalidStart_Zero(t *testing.T) {
	_, err := FindAvailable(0)
	if err == nil {
		t.Error("expected error for startPort 0")
	}
}

func TestFindAvailable_InvalidStart_Negative(t *testing.T) {
	_, err := FindAvailable(-1)
	if err == nil {
		t.Error("expected error for negative startPort")
	}
}

func TestFindAvailable_InvalidStart_TooHigh(t *testing.T) {
	_, err := FindAvailable(MaxPort + 1)
	if err == nil {
		t.Error("expected error for startPort > MaxPort")
	}
}

func TestFindAvailable_AtMaxPort(t *testing.T) {
	// MaxPort (65535) itself should be a valid startPort — it may or may not be available
	_, err := FindAvailable(MaxPort)
	// We don't care if it succeeds or fails due to the port being in use,
	// but it should not return an invalid-range error.
	if err != nil && err.Error() == "start port 65535 is out of valid range (1-65535)" {
		t.Error("MaxPort should be a valid start port")
	}
}
