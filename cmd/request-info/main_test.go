package main

import "testing"

func TestPortFromEnvDefaultsTo8080(t *testing.T) {
	t.Parallel()

	port, err := portFromEnv("")
	if err != nil {
		t.Fatalf("portFromEnv returned error: %v", err)
	}
	if port != "8080" {
		t.Fatalf("port = %q, want 8080", port)
	}
}

func TestPortFromEnvUsesConfiguredPort(t *testing.T) {
	t.Parallel()

	port, err := portFromEnv("9090")
	if err != nil {
		t.Fatalf("portFromEnv returned error: %v", err)
	}
	if port != "9090" {
		t.Fatalf("port = %q, want 9090", port)
	}
}

func TestPortFromEnvRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	_, err := portFromEnv("not-a-port")
	if err == nil {
		t.Fatal("portFromEnv returned nil error, want error")
	}
}

func TestPortFromEnvRejectsOutOfRangePort(t *testing.T) {
	t.Parallel()

	_, err := portFromEnv("70000")
	if err == nil {
		t.Fatal("portFromEnv returned nil error, want error")
	}
}
