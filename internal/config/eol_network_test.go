//go:build eol

package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Run with: go test -tags eol -run TestPinnedComponentsAreSupported ./internal/config
// It needs network access, so it is not part of the default test run; a weekly
// workflow runs it.
func TestPinnedComponentsAreSupported(t *testing.T) {
	const margin = 45 * 24 * time.Hour
	now := time.Now().UTC()
	client := &http.Client{Timeout: 20 * time.Second}

	type cycle struct {
		Cycle string `json:"cycle"`
		EOL   any    `json:"eol"`
	}
	fetch := func(product string) ([]cycle, error) {
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			resp, err := client.Get("https://endoflife.date/api/" + product + ".json")
			if err != nil {
				lastErr = err
				time.Sleep(2 * time.Second)
				continue
			}
			var cycles []cycle
			err = json.NewDecoder(resp.Body).Decode(&cycles)
			status := resp.StatusCode
			_ = resp.Body.Close()
			if status != http.StatusOK {
				return nil, fmt.Errorf("HTTP %d for product %q", status, product)
			}
			if err != nil {
				return nil, fmt.Errorf("decoding %q: %w", product, err)
			}
			return cycles, nil
		}
		return nil, lastErr
	}

	checked := 0
	for _, c := range trackedComponents {
		cycles, err := fetch(c.Product)
		if err != nil {
			if checked == 0 {
				t.Skipf("endoflife.date is not reachable, nothing was checked: %v", err)
			}
			t.Errorf("%s: %v", c.Constant, err)
			continue
		}
		line := releaseLine(c.Version)
		var match *cycle
		// The list is newest first, so a floating tag such as "7" resolves to its newest minor.
		for i := range cycles {
			if cycles[i].Cycle == line || strings.HasPrefix(cycles[i].Cycle, line+".") {
				match = &cycles[i]
				break
			}
		}
		if match == nil {
			t.Errorf("%s: release line %q of %s is not listed on endoflife.date", c.Constant, line, c.Product)
			continue
		}
		checked++
		ok, detail := supportedUntil(match.EOL, now, margin)
		t.Logf("%-16s %-14s %-18s %s", c.Constant, c.Product+" "+match.Cycle, c.Version, detail)
		if !ok {
			t.Errorf("%s = %q (%s %s): %s. Update it in versions.go, then regenerate the sample project and verify the stacks (see the sample test).", c.Constant, c.Version, c.Product, match.Cycle, detail)
		}
	}
}
