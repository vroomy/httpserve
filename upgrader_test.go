package apex

import (
	"net/http/httptest"
	"testing"
)

func TestUpgraderRedirects(t *testing.T) {
	var upgrader, err = NewUpgrader(443,
		VirtualHost{Domain: "proxy.example", Port: 8443},
		VirtualHost{Domain: "same.example", Port: 443},
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if closeErr := upgrader.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	var cases = []struct {
		name string
		url  string
		want string
	}{
		{"default", "http://example.com/path?key=value", "https://example.com:443/path?key=value"},
		{"replace HTTP port", "http://example.com:8080/", "https://example.com:443/"},
		{"virtual host", "http://proxy.example:8080/a%20b?q=a%2Fb", "https://proxy.example:8443/a%20b?q=a%2Fb"},
		{"same port", "http://same.example/path", "https://same.example:443/path"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var request = httptest.NewRequest("GET", test.url, nil)
			var original = *request.URL
			var recorder = httptest.NewRecorder()
			upgrader.s.g.r.ServeHTTP(recorder, request)
			if recorder.Code != 301 || recorder.Header().Get("Location") != test.want || recorder.Body.Len() != 0 {
				t.Fatalf("redirect = %d %v %q", recorder.Code, recorder.Header(), recorder.Body.String())
			}

			if *request.URL != original {
				t.Fatal("redirect mutated the request URL")
			}
		})
	}
}
