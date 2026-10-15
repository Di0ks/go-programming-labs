package restapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticationRequiresSession(t *testing.T) {
	mux := http.NewServeMux()
	for pattern, handler := range apiRoutes() {
		mux.HandleFunc(pattern, handler)
	}

	for _, request := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/users", ""},
		{http.MethodGet, "/users/1", ""},
		{http.MethodPost, "/users", `{}`},
		{http.MethodGet, "/admin/info", ""},
		{http.MethodGet, "/auth/csrf", ""},
	} {
		response := serveRawRequest(mux, request.method, request.path, request.body)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session: status=%d, want %d", request.method, request.path, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestAuthEndpointsRequireJSON(t *testing.T) {
	mux := http.NewServeMux()
	for pattern, handler := range apiRoutes() {
		mux.HandleFunc(pattern, handler)
	}

	request := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("POST /auth/login with text/plain: status=%d, want %d", response.Code, http.StatusUnsupportedMediaType)
	}
}
