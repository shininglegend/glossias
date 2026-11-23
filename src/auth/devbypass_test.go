package auth

import "testing"

func TestCheckDevBypass(t *testing.T) {
	cases := []struct {
		devUser, appURL string
		ok              bool
	}{
		{"", "", true},
		{"", "https://app.example.com", true},
		{"user_1", "http://localhost:5173", true},
		{"user_1", "http://127.0.0.1:5173", true},
		{"user_1", "http://[::1]:5173", true},
		{"user_1", "http://glossias.localhost", true},
		{"user_1", "", false},
		{"user_1", "https://app.example.com", false},
		{"user_1", "https://localhost.example.com", false},
		{"user_1", "http://10.0.0.5:5173", false},
		{"user_1", "not a url", false},
	}
	for _, c := range cases {
		err := CheckDevBypass(c.devUser, c.appURL)
		if (err == nil) != c.ok {
			t.Errorf("CheckDevBypass(%q, %q) = %v, want ok=%v", c.devUser, c.appURL, err, c.ok)
		}
	}
}
