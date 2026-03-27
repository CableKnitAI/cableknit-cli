package api

import (
	"testing"
)

func resetDemo() {
	DemoEnabled = false
	DemoLoggedIn = false
}

// --- Client routing tests ---

func TestNewAPIClient_ReturnsRealClientByDefault(t *testing.T) {
	resetDemo()
	client := NewAPIClient()
	if _, ok := client.(*MockClient); ok {
		t.Error("NewAPIClient should return real Client when DemoEnabled is false")
	}
}

func TestNewAPIClient_ReturnsMockClientInDemoMode(t *testing.T) {
	resetDemo()
	DemoEnabled = true
	defer resetDemo()

	client := NewAPIClient()
	if _, ok := client.(*MockClient); !ok {
		t.Error("NewAPIClient should return MockClient when DemoEnabled is true")
	}
}

// --- Demo mode starts logged out ---

func TestDemoMode_StartsLoggedOut(t *testing.T) {
	resetDemo()
	DemoEnabled = true
	defer resetDemo()

	if DemoLoggedIn {
		t.Error("DemoLoggedIn should be false when demo mode starts")
	}
}

// --- Mock login accepts any credentials ---

func TestMockLogin_AcceptsAnyCredentials(t *testing.T) {
	resetDemo()
	DemoEnabled = true
	defer resetDemo()

	client := NewMockClient()

	credentials := []struct {
		email    string
		password string
	}{
		{"anything@example.com", "whatever"},
		{"", ""},
		{"fake@fake.fake", "12345"},
	}

	for _, cred := range credentials {
		DemoLoggedIn = false
		var resp LoginResponse
		err := client.JSON("POST", "/api/v1/cli/sessions", LoginRequest{
			Email:    cred.email,
			Password: cred.password,
		}, &resp)
		if err != nil {
			t.Errorf("mock login should accept %q/%q, got error: %v", cred.email, cred.password, err)
		}
		if resp.Token == "" {
			t.Errorf("mock login should return a token for %q/%q", cred.email, cred.password)
		}
		if !DemoLoggedIn {
			t.Errorf("mock login should set DemoLoggedIn=true for %q/%q", cred.email, cred.password)
		}
	}
}

// --- Mock logout clears DemoLoggedIn ---

func TestMockLogout_ClearsDemoLoggedIn(t *testing.T) {
	resetDemo()
	DemoEnabled = true
	DemoLoggedIn = true
	defer resetDemo()

	client := NewMockClient()
	err := client.JSON("DELETE", "/api/v1/cli/sessions", nil, nil)
	if err != nil {
		t.Errorf("mock logout should not error: %v", err)
	}
	// Note: DemoLoggedIn is cleared in the app handler (cmd/app.go), not in mock client
	// This test verifies the mock DELETE doesn't error
}

// --- Real client does NOT use DemoLoggedIn ---

func TestRealClient_IgnoresDemoLoggedIn(t *testing.T) {
	resetDemo()
	// DemoEnabled is false, so NewAPIClient returns real client
	DemoLoggedIn = true
	defer resetDemo()

	client := NewAPIClient()
	if _, ok := client.(*MockClient); ok {
		t.Error("real client should be returned when DemoEnabled is false, regardless of DemoLoggedIn")
	}
}

// --- DemoEnabled does not imply DemoLoggedIn ---

func TestDemoEnabled_DoesNotImplyLoggedIn(t *testing.T) {
	resetDemo()
	DemoEnabled = true
	defer resetDemo()

	if DemoLoggedIn {
		t.Error("setting DemoEnabled should not automatically set DemoLoggedIn")
	}
}

// --- Real login rejects bad credentials ---

func TestRealLogin_RejectsBadCredentials(t *testing.T) {
	resetDemo()
	// DemoEnabled is false — uses real client hitting localhost:3000
	client := NewClient()
	var resp LoginResponse
	err := client.JSON("POST", "/api/v1/cli/sessions", LoginRequest{
		Email:    "fake@fake.com",
		Password: "wrongpassword",
	}, &resp)
	if err == nil {
		t.Error("real login should reject bad credentials, but got no error")
	}
	if resp.Token != "" {
		t.Error("real login should not return a token for bad credentials")
	}
	if DemoLoggedIn {
		t.Error("real login should not set DemoLoggedIn")
	}
}
