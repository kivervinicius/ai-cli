package web

import "testing"

func TestExchangeBootstrapTokenReusableOnLoopback(t *testing.T) {
	auth, token, err := NewAuthManager("127.0.0.1", "3000")
	if err != nil {
		t.Fatal(err)
	}

	first, ok := auth.ExchangeBootstrapToken(token)
	if !ok || first == nil {
		t.Fatal("first loopback bootstrap exchange must succeed")
	}
	second, ok := auth.ExchangeBootstrapToken(token)
	if !ok || second == nil {
		t.Fatal("loopback bootstrap token must remain reusable for local re-auth")
	}
	if first.ID == second.ID {
		t.Fatal("each exchange must mint a distinct session")
	}
	if auth.usedBootstrap {
		t.Fatal("loopback listen must not mark bootstrap as consumed")
	}
}

func TestExchangeBootstrapTokenOneTimeOnPrivateBind(t *testing.T) {
	auth, token, err := NewAuthManager("192.168.1.10", "3000")
	if err != nil {
		t.Fatal(err)
	}

	first, ok := auth.ExchangeBootstrapToken(token)
	if !ok || first == nil {
		t.Fatal("first private-bind bootstrap exchange must succeed")
	}
	if !auth.usedBootstrap {
		t.Fatal("private/remote bind must consume bootstrap after first use")
	}
	second, ok := auth.ExchangeBootstrapToken(token)
	if ok || second != nil {
		t.Fatal("private/remote bootstrap must be one-time")
	}
}

func TestIsLoopbackListen(t *testing.T) {
	cases := map[string]bool{
		"":          true,
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
		"10.0.0.5":  false,
		"0.0.0.0":   false,
	}
	for host, want := range cases {
		auth := &AuthManager{listenHost: host}
		if got := auth.isLoopbackListen(); got != want {
			t.Fatalf("host %q: got %v want %v", host, got, want)
		}
	}
}
