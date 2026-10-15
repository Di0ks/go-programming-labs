package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUsersRoutes(t *testing.T) {
	mux := setupIntegrationMux(t)
	prefix := fmt.Sprintf("restapi-test-%d", time.Now().UnixNano())

	t.Run("registration, login, and admin diagnostics", func(t *testing.T) {
		registration := fmt.Sprintf(`{"name":"Auth Test","age":30,"email":%q,"password":"test-password"}`, prefix+"-auth@example.test")
		response := serveRawRequest(mux, http.MethodPost, "/auth/register", registration)
		if response.Code != http.StatusCreated {
			t.Fatalf("POST /auth/register status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body)
		}
		var registered struct {
			Role string `json:"role"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &registered); err != nil || registered.Role != "user" {
			t.Fatalf("unexpected registration response: %+v; error: %v", registered, err)
		}
		userCookie := findSessionCookie(t, response)
		if userCookie.HttpOnly != true || userCookie.Secure || userCookie.SameSite != http.SameSiteLaxMode || userCookie.Path != "/" {
			t.Fatalf("unexpected HTTP session cookie attributes: %+v", userCookie)
		}
		csrfToken := getCSRFToken(t, mux, userCookie)
		if got := serveRequestWithSession(mux, http.MethodGet, "/users", "", userCookie, "").Code; got != http.StatusOK {
			t.Fatalf("registered user GET /users status = %d, want %d", got, http.StatusOK)
		}
		if got := serveRequestWithSession(mux, http.MethodPost, "/users", `{}`, userCookie, csrfToken).Code; got != http.StatusForbidden {
			t.Fatalf("registered user POST /users status = %d, want %d", got, http.StatusForbidden)
		}

		if got := serveRequestWithSession(mux, http.MethodPost, "/users", `{}`, testAdminCookie, "").Code; got != http.StatusForbidden {
			t.Fatalf("admin POST /users without CSRF token status = %d, want %d", got, http.StatusForbidden)
		}
		if got := serveRequestWithSession(mux, http.MethodPost, "/users", `{}`, testAdminCookie, "wrong-token").Code; got != http.StatusForbidden {
			t.Fatalf("admin POST /users with invalid CSRF token status = %d, want %d", got, http.StatusForbidden)
		}

		login := fmt.Sprintf(`{"email":%q,"password":"test-password"}`, prefix+"-auth@example.test")
		response = serveRawRequest(mux, http.MethodPost, "/auth/login", login)
		if response.Code != http.StatusOK {
			t.Fatalf("POST /auth/login status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body)
		}
		loginCookie := findSessionCookie(t, response)
		loginCSRF := getCSRFToken(t, mux, loginCookie)
		if loginCSRF == csrfToken {
			t.Fatal("login reused the previous session's CSRF token")
		}
		response = serveRawRequest(mux, http.MethodPost, "/auth/login", strings.Replace(login, "test-password", "wrong-password", 1))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("login with wrong password status = %d, want %d", response.Code, http.StatusUnauthorized)
		}

		response = serveRequest(mux, http.MethodGet, "/admin/info", "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET /admin/info status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body)
		}

		response = serveRequestWithSession(mux, http.MethodPost, "/auth/logout", "", userCookie, csrfToken)
		if response.Code != http.StatusNoContent {
			t.Fatalf("POST /auth/logout status = %d, want %d", response.Code, http.StatusNoContent)
		}
		if got := serveRequestWithSession(mux, http.MethodGet, "/users", "", userCookie, "").Code; got != http.StatusUnauthorized {
			t.Fatalf("request after logout status = %d, want %d", got, http.StatusUnauthorized)
		}
	})

	t.Run("GET /users filters and paginates", func(t *testing.T) {
		users := []User{
			{Name: prefix + " Alice", Age: 30, Email: prefix + "-alice@example.test", Phone: stringPointer("+1234567890")},
			{Name: prefix + " Alicia", Age: 41, Email: prefix + "-alicia@example.test", Phone: stringPointer("+4444444444")},
			{Name: prefix + " Bob", Age: 22, Email: prefix + "-bob@sample.test"},
		}
		ids := make([]int, 0, len(users))
		for _, user := range users {
			ids = append(ids, createTestUser(t, mux, user))
		}

		target := "/users?" + url.Values{
			"name":    {prefix},
			"min_age": {"30"},
			"max_age": {"45"},
			"limit":   {"1"},
			"offset":  {"1"},
		}.Encode()
		response := serveRequest(mux, http.MethodGet, target, "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET /users status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body)
		}
		if response.Header().Get("X-Total-Count") != "2" || response.Header().Get("X-Limit") != "1" || response.Header().Get("X-Offset") != "1" {
			t.Fatalf("unexpected pagination headers: %v", response.Header())
		}
		result := decodeUsers(t, response)
		if len(result) != 1 || result[fmt.Sprint(ids[1])].Name != users[1].Name {
			t.Fatalf("GET /users returned %v, want only user %d", result, ids[1])
		}

		target = "/users?" + url.Values{
			"email": {"EXAMPLE.TEST"},
			"phone": {"12345"},
			"name":  {prefix},
		}.Encode()
		response = serveRequest(mux, http.MethodGet, target, "")
		result = decodeUsers(t, response)
		if response.Code != http.StatusOK || len(result) != 1 || result[fmt.Sprint(ids[0])].Name != users[0].Name {
			t.Fatalf("email/phone filters returned status=%d users=%v", response.Code, result)
		}

		response = serveRequest(mux, http.MethodGet, "/users?name="+url.QueryEscape(prefix), "")
		if response.Code != http.StatusOK || response.Header().Get("X-Limit") != "20" || response.Header().Get("X-Offset") != "0" {
			t.Fatalf("default pagination: status=%d headers=%v", response.Code, response.Header())
		}
		assertAPIError(t, serveRequest(mux, http.MethodGet, "/users?limit=0", ""), http.StatusBadRequest, "limit")
	})

	t.Run("GET /users/{id}", func(t *testing.T) {
		user := User{Name: prefix + " Dana", Age: 34, Email: prefix + "-dana@example.test"}
		id := createTestUser(t, mux, user)

		response := serveRequest(mux, http.MethodGet, fmt.Sprintf("/users/%d", id), "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET /users/%d status = %d, want %d", id, response.Code, http.StatusOK)
		}
		var got User
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode user: %v", err)
		}
		if got.Name != user.Name || got.Age != user.Age || got.Email != user.Email {
			t.Fatalf("GET /users/%d returned %+v, want %+v", id, got, user)
		}

		assertAPIError(t, serveRequest(mux, http.MethodGet, "/users/2147483647", ""), http.StatusNotFound, "id")
		assertAPIError(t, serveRequest(mux, http.MethodGet, "/users/not-an-id", ""), http.StatusBadRequest, "id")
	})

	t.Run("POST /users", func(t *testing.T) {
		user := User{Name: prefix + " Eve", Age: 26, Email: prefix + "-eve@example.test", Phone: stringPointer("+1234567890")}
		id := createTestUser(t, mux, user)
		response := serveRequest(mux, http.MethodGet, fmt.Sprintf("/users/%d", id), "")
		if response.Code != http.StatusOK {
			t.Fatalf("created user %d could not be read: status=%d", id, response.Code)
		}

		assertAPIError(t, serveRequest(mux, http.MethodPost, "/users", `{"name":`), http.StatusBadRequest, "")
		assertAPIError(t, serveRequest(mux, http.MethodPost, "/users", `{"name":"  ","age":26,"email":"eve@example.test"}`), http.StatusUnprocessableEntity, "name")
	})

	t.Run("PUT /users/{id} updates supplied fields only", func(t *testing.T) {
		user := User{
			Name: prefix + " Frank", Age: 38, Email: prefix + "-frank@example.test",
			Comment: stringPointer("keep this"), Phone: stringPointer("+1234567890"),
		}
		id := createTestUser(t, mux, user)
		response := serveRequest(mux, http.MethodPut, fmt.Sprintf("/users/%d", id), `{"age":39,"comment":null}`)
		if response.Code != http.StatusOK {
			t.Fatalf("PUT /users/%d status = %d, want %d; body: %s", id, response.Code, http.StatusOK, response.Body)
		}

		response = serveRequest(mux, http.MethodGet, fmt.Sprintf("/users/%d", id), "")
		var updated User
		if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
			t.Fatalf("decode updated user: %v", err)
		}
		if updated.Name != user.Name || updated.Age != 39 || updated.Email != user.Email || updated.Comment != nil || updated.Phone == nil || *updated.Phone != *user.Phone {
			t.Fatalf("partial PUT returned unexpected user: %+v", updated)
		}

		response = serveRequest(mux, http.MethodPut, fmt.Sprintf("/users/%d", id), `{}`)
		if response.Code != http.StatusOK {
			t.Fatalf("empty PUT status = %d, want %d", response.Code, http.StatusOK)
		}
		assertAPIError(t, serveRequest(mux, http.MethodPut, fmt.Sprintf("/users/%d", id), `{"email":"not-an-email"}`), http.StatusUnprocessableEntity, "email")
		assertAPIError(t, serveRequest(mux, http.MethodPut, "/users/2147483647", `{"age":39}`), http.StatusNotFound, "id")
	})

	t.Run("DELETE /users/{id}", func(t *testing.T) {
		id := createTestUser(t, mux, User{Name: prefix + " Grace", Age: 29, Email: prefix + "-grace@example.test"})
		response := serveRequest(mux, http.MethodDelete, fmt.Sprintf("/users/%d", id), "")
		if response.Code != http.StatusOK {
			t.Fatalf("DELETE /users/%d status = %d, want %d", id, response.Code, http.StatusOK)
		}
		assertAPIError(t, serveRequest(mux, http.MethodGet, fmt.Sprintf("/users/%d", id), ""), http.StatusNotFound, "id")
		assertAPIError(t, serveRequest(mux, http.MethodDelete, fmt.Sprintf("/users/%d", id), ""), http.StatusNotFound, "id")
	})
}

func setupIntegrationMux(t *testing.T) *http.ServeMux {
	t.Helper()
	previousDB := db
	cfg := NewDbConfig()
	cfg.RetryCount = 1
	cfg.RetryPeriod = 0
	if err := init_db(&cfg); err != nil {
		db = previousDB
		t.Skipf("PostgreSQL integration tests require %s:%d/%s: %v", cfg.Ip, cfg.Port, cfg.DbName, err)
	}

	pool, ok := db.(*pgxpool.Pool)
	if !ok {
		t.Fatal("init_db did not install a PostgreSQL pool")
	}
	tx, err := pool.Begin(context.Background())
	if err != nil {
		pool.Close()
		db = previousDB
		t.Fatalf("begin isolated test transaction: %v", err)
	}
	db = &transactionalTestDatabase{Tx: tx, pool: pool}
	t.Cleanup(func() {
		db.Close()
		db = previousDB
	})

	mux := http.NewServeMux()
	for pattern, handler := range apiRoutes() {
		mux.HandleFunc(pattern, handler)
	}

	adminEmail := fmt.Sprintf("session-test-admin-%d@example.test", time.Now().UnixNano())
	registration := fmt.Sprintf(`{"name":"Session Test Admin","age":40,"email":%q,"password":"admin-password"}`, adminEmail)
	response := serveRawRequest(mux, http.MethodPost, "/auth/register", registration)
	if response.Code != http.StatusCreated {
		t.Fatalf("create integration admin: status=%d body=%s", response.Code, response.Body)
	}
	testAdminCookie = findSessionCookie(t, response)
	rows, err := db.Query(context.Background(), "UPDATE users SET role = 'admin' WHERE email = $1 RETURNING id", adminEmail)
	if err != nil {
		t.Fatalf("promote integration admin: %v", err)
	}
	if !rows.Next() {
		rows.Close()
		t.Fatal("promote integration admin returned no user")
	}
	rows.Close()
	testAdminCSRFToken = getCSRFToken(t, mux, testAdminCookie)
	return mux
}

type transactionalTestDatabase struct {
	pgx.Tx
	pool *pgxpool.Pool
}

func (db *transactionalTestDatabase) Close() {
	_ = db.Tx.Rollback(context.Background())
	db.pool.Close()
}

func createTestUser(t *testing.T, mux *http.ServeMux, user User) int {
	t.Helper()
	body, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("encode test user: %v", err)
	}
	response := serveRequest(mux, http.MethodPost, "/users", string(body))
	if response.Code != http.StatusOK {
		t.Fatalf("POST /users status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body)
	}

	rows, err := db.Query(context.Background(), "SELECT id FROM users WHERE email = $1", user.Email)
	if err != nil {
		t.Fatalf("find inserted test user: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("POST /users did not insert %q", user.Email)
	}
	var id int
	if err := rows.Scan(&id); err != nil {
		t.Fatalf("read inserted user id: %v", err)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read inserted user rows: %v", err)
	}
	return id
}

var testAdminCookie *http.Cookie
var testAdminCSRFToken string

func serveRequest(mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	return serveRequestWithSession(mux, method, target, body, testAdminCookie, testAdminCSRFToken)
}

func serveRequestWithSession(mux *http.ServeMux, method, target, body string, cookie *http.Cookie, csrfToken string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrfToken != "" {
		request.Header.Set("X-CSRF-Token", csrfToken)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func serveRawRequest(mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	return serveRequestWithSession(mux, method, target, body, nil, "")
}

func findSessionCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie
		}
	}
	t.Fatalf("response has no %q cookie", sessionCookieName)
	return nil
}

func getCSRFToken(t *testing.T, mux *http.ServeMux, cookie *http.Cookie) string {
	t.Helper()
	response := serveRequestWithSession(mux, http.MethodGet, "/auth/csrf", "", cookie, "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /auth/csrf status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("GET /auth/csrf Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
	var result struct {
		Token string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Token == "" {
		t.Fatalf("invalid CSRF-token response: %+v; error: %v", result, err)
	}
	return result.Token
}

func decodeUsers(t *testing.T, response *httptest.ResponseRecorder) map[string]User {
	t.Helper()
	var users map[string]User
	if err := json.Unmarshal(response.Body.Bytes(), &users); err != nil {
		t.Fatalf("decode users response: %v; body: %s", err, response.Body)
	}
	return users
}

func assertAPIError(t *testing.T, response *httptest.ResponseRecorder, status int, field string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("response status = %d, want %d; body: %s", response.Code, status, response.Body)
	}
	var body apiError
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v; body: %s", err, response.Body)
	}
	if body.Error == "" || body.Field != field {
		t.Fatalf("error response = %+v, want a message and field %q", body, field)
	}
}

//go:fix inline
func stringPointer(value string) *string { return new(value) }

var _ database = (*transactionalTestDatabase)(nil)
