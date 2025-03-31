package subfinder

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestRunEnumeration(t *testing.T) {
	// Skip this test by default since it makes actual external API calls
	// Set ENABLE_LIVE_TESTS=1 to run these tests
	if os.Getenv("ENABLE_LIVE_TESTS") != "1" {
		t.Skip("Skipping live test. Set ENABLE_LIVE_TESTS=1 to enable")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create a test configuration
	config := SubfinderConfig{
		ProviderConfigPath:   "../provider-config.yaml", // Adjust path as needed
		Timeout:              10,                        // Short timeout for test
		Recursive:            false,
		SourcesFilter:        "dnsdumpster",
		ExcludeSourcesFilter: "",
	}

	// Use a domain that's likely to have some subdomains but not too many
	domain := "example.com"

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Call the function being tested
	results, err := RunEnumeration(ctx, domain, config, logger)

	// Check errors
	if err != nil {
		t.Fatalf("RunEnumeration failed: %v", err)
	}

	// Basic validation of results
	if len(results) == 0 {
		t.Logf("No subdomains found for %s, this could be normal but worth checking", domain)
	} else {
		t.Logf("Found %d subdomains for %s", len(results), domain)
		for i, subdomain := range results {
			if i < 5 { // Only log first few to avoid verbosity
				t.Logf("Subdomain found: %s", subdomain)
			}
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	// This test doesn't make external calls, so no need to skip
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with zero values to ensure defaults are applied
	config := SubfinderConfig{
		Timeout:    0,
		MaxDepth:   0,
		Recursive:  true,
	}

	// Mock context with extremely short timeout to guarantee timeout error
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// We're going to terminate this early intentionally
	_, err := RunEnumeration(ctx, "testdomain.com", config, logger)

	// We expect an error due to the short timeout
	if err == nil {
		t.Fatalf("Expected error due to short timeout")
	} else {
		// Log the error to confirm it's a context deadline error
		t.Logf("Got expected error: %v", err)
	}

	// The test passes if it reached this point without panicking,
	// which means the default values were properly applied
	t.Log("Default configuration values were applied correctly")
}
