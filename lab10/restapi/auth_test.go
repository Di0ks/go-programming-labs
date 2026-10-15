package restapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthenticationAndAuthorization(t *testing.T) {
	mux := http.NewServeMux()
	for pattern, handler := range apiRoutes() {
		mux.HandleFunc(pattern, handler)
	}

	assertStatus := func(method, path, role string, want int) {
		t.Helper()
		var response *httptest.ResponseRecorder
		if role == "" {
			response = serveRawRequest(mux, method, path, "")
		} else {
			response = serveRequestAs(mux, method, path, "", role)
		}
		if response.Code != want {
			t.Errorf("%s %s as %q: status=%d, want %d", method, path, role, response.Code, want)
		}
	}

	assertStatus(http.MethodGet, "/users", "", http.StatusUnauthorized)
	assertStatus(http.MethodGet, "/users", "user", http.StatusInternalServerError)
	assertStatus(http.MethodPost, "/users", "user", http.StatusForbidden)
	assertStatus(http.MethodGet, "/admin/info", "user", http.StatusForbidden)
	assertStatus(http.MethodGet, "/admin/info", "admin", http.StatusInternalServerError)

	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token: status=%d, want %d", response.Code, http.StatusUnauthorized)
	}

	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims{
		UserID: 1,
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}).SignedString(jwtSigningKey)
	if err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("Authorization", "Bearer "+expired)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expired token: status=%d, want %d", response.Code, http.StatusUnauthorized)
	}
}
