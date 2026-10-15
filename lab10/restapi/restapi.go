package restapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Di0ks/go-programming-labs/lab7/httpserv"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed users.sql
var usersSchema string

type User struct {
	id      int
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Comment *string `json:"comment"`
	Email   string  `json:"email"`
	Phone   *string `json:"phone"`
}

type database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Close()
}

var db database
var dbcfg *DbConfig

const (
	sessionCookieName = "session_id"
	sessionLifetime   = 24 * time.Hour
)

type authPrincipal struct {
	UserID        int
	Role          string
	SessionIDHash string
	CSRFToken     string
}

type principalContextKey struct{}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func hashSessionID(sessionID string) string {
	hash := sha256.Sum256([]byte(sessionID))
	return hex.EncodeToString(hash[:])
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func createSession(w http.ResponseWriter, r *http.Request, userID int) bool {
	if !executeQuery(w, r, "clean expired sessions", "DELETE FROM sessions WHERE expires_at <= NOW()", nil, nil) {
		return false
	}
	sessionID, err := randomToken()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Не удалось создать сессию", "")
		return false
	}
	csrfToken, err := randomToken()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Не удалось создать CSRF-токен", "")
		return false
	}
	expiresAt := time.Now().Add(sessionLifetime)
	query := "INSERT INTO sessions (session_id_hash, user_id, csrf_token, expires_at) VALUES ($1, $2, $3, $4) RETURNING session_id_hash"
	if !executeQuery(w, r, "create session", query, []any{hashSessionID(sessionID), userID, csrfToken, expiresAt}, func(rows pgx.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("session insert returned no result")
		}
		var insertedHash string
		return rows.Scan(&insertedHash)
	}) {
		return false
	}
	setSessionCookie(w, r, sessionID, expiresAt)
	return true
}

func authenticate(requireAdmin, requireCSRF bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeAPIError(w, http.StatusUnauthorized, "Требуется сессия", "")
			return
		}

		principal := authPrincipal{SessionIDHash: hashSessionID(cookie.Value)}
		found := false
		query := "SELECT s.user_id, u.role, s.csrf_token FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.session_id_hash = $1 AND s.expires_at > NOW()"
		if !executeQuery(w, r, "authenticate session", query, []any{principal.SessionIDHash}, func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			found = true
			return rows.Scan(&principal.UserID, &principal.Role, &principal.CSRFToken)
		}) {
			return
		}
		if !found {
			clearSessionCookie(w, r)
			writeAPIError(w, http.StatusUnauthorized, "Сессия недействительна или истекла", "")
			return
		}
		if requireAdmin && principal.Role != "admin" {
			writeAPIError(w, http.StatusForbidden, "Недостаточно прав", "")
			return
		}
		if requireCSRF && subtle.ConstantTimeCompare([]byte(principal.CSRFToken), []byte(r.Header.Get("X-CSRF-Token"))) != 1 {
			writeAPIError(w, http.StatusForbidden, "Недействительный или отсутствующий CSRF-токен", "")
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), principalContextKey{}, principal)))
	}
}

func requireJSONBody(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeAPIError(w, http.StatusUnsupportedMediaType, "Ожидается Content-Type: application/json", "")
		return false
	}
	return true
}

func post_register(w http.ResponseWriter, r *http.Request) {
	if !requireJSONBody(w, r) {
		return
	}
	var input struct {
		Name     string  `json:"name"`
		Age      *int    `json:"age"`
		Comment  *string `json:"comment"`
		Email    string  `json:"email"`
		Phone    *string `json:"phone"`
		Password string  `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Некорректный JSON: "+err.Error(), "")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeAPIError(w, http.StatusBadRequest, "В запросе должно быть только одно JSON-значение", "")
		return
	}
	if len(input.Password) < 8 || len(input.Password) > 72 {
		writeAPIError(w, http.StatusUnprocessableEntity, "Пароль должен содержать от 8 до 72 байт", "password")
		return
	}
	user := User{Name: input.Name, Email: input.Email, Comment: input.Comment, Phone: input.Phone}
	if input.Age != nil {
		user.Age = *input.Age
	}
	if validationErr := validateUser(&user, input.Age != nil); validationErr != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, validationErr.Message, validationErr.Field)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Не удалось сохранить пароль", "")
		return
	}
	query := "INSERT INTO users (name, age, comment, email, phone, role, password_hash) VALUES ($1, $2, $3, $4, $5, 'user', $6) RETURNING id"
	var userID int
	if !executeQuery(w, r, "register user", query, []any{user.Name, user.Age, user.Comment, user.Email, user.Phone, string(hash)}, func(rows pgx.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("register did not return a user id")
		}
		return rows.Scan(&userID)
	}) {
		return
	}
	if !createSession(w, r, userID) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"role": "user"})
}

func post_login(w http.ResponseWriter, r *http.Request) {
	if !requireJSONBody(w, r) {
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Некорректный JSON: "+err.Error(), "")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeAPIError(w, http.StatusBadRequest, "В запросе должно быть только одно JSON-значение", "")
		return
	}
	var userID int
	var role, passwordHash string
	found := false
	if !executeQuery(w, r, "login user", "SELECT id, role, password_hash FROM users WHERE email = $1 ORDER BY id LIMIT 1", []any{strings.TrimSpace(input.Email)}, func(rows pgx.Rows) error {
		if !rows.Next() {
			return nil
		}
		found = true
		return rows.Scan(&userID, &role, &passwordHash)
	}) {
		return
	}
	if !found || (role != "user" && role != "admin") || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)) != nil {
		writeAPIError(w, http.StatusUnauthorized, "Неверный email или пароль", "")
		return
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if !executeQuery(w, r, "rotate session", "DELETE FROM sessions WHERE session_id_hash = $1", []any{hashSessionID(cookie.Value)}, nil) {
			return
		}
	}
	if !createSession(w, r, userID) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"role": role})
}

func get_csrf_token(w http.ResponseWriter, r *http.Request) {
	principal := r.Context().Value(principalContextKey{}).(authPrincipal)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": principal.CSRFToken})
}

func post_logout(w http.ResponseWriter, r *http.Request) {
	principal := r.Context().Value(principalContextKey{}).(authPrincipal)
	if !executeQuery(w, r, "delete session", "DELETE FROM sessions WHERE session_id_hash = $1", []any{principal.SessionIDHash}, nil) {
		return
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func get_admin_info(w http.ResponseWriter, r *http.Request) {
	var userCount int64
	if !executeQuery(w, r, "get admin server info", "SELECT COUNT(*) FROM users", nil, func(rows pgx.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("count query returned no result")
		}
		return rows.Scan(&userCount)
	}) {
		return
	}
	info := struct {
		GoVersion      string `json:"go_version"`
		Goroutines     int    `json:"goroutines"`
		Database       string `json:"database"`
		DatabaseStatus string `json:"database_status"`
		UserCount      int64  `json:"user_count"`
	}{runtime.Version(), runtime.NumGoroutine(), dbcfg.DbName, "connected", userCount}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

const userColumns = "id, name, age, comment, email, phone"

const (
	defaultUsersLimit = 20
	maxUsersLimit     = 100
)

type apiError struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func writeAPIError(w http.ResponseWriter, status int, message, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(apiError{Error: message, Field: field}); err != nil {
		log.Println("Не удалось отправить ответ об ошибке:", err)
	}
}

// executeQuery is the common entry point for API database queries. It owns
// the rows lifecycle, checks iteration errors, and writes the same safe 500
// response for every database failure.
func executeQuery(
	w http.ResponseWriter,
	r *http.Request,
	operation,
	query string,
	args []any,
	consume func(pgx.Rows) error,
) bool {
	queryFailed := func(err error) bool {
		log.Printf("%s: %v", operation, err)
		writeAPIError(w, http.StatusInternalServerError, "Ошибка при работе с базой данных", "")
		return false
	}

	if db == nil {
		return queryFailed(fmt.Errorf("database is not initialized"))
	}

	rows, err := db.Query(r.Context(), query, args...)
	if err != nil {
		return queryFailed(err)
	}
	defer rows.Close()

	if consume != nil {
		if err := consume(rows); err != nil {
			return queryFailed(err)
		}
	}
	if err := rows.Err(); err != nil {
		return queryFailed(err)
	}
	return true
}

func scanUser(rows pgx.Rows) (User, error) {
	var user User
	err := rows.Scan(&user.id, &user.Name, &user.Age, &user.Comment, &user.Email, &user.Phone)
	return user, err
}

func get_users(w http.ResponseWriter, r *http.Request) {
	options, validationErr := parseUserListOptions(r.URL.Query())
	if validationErr != nil {
		writeAPIError(w, http.StatusBadRequest, validationErr.Message, validationErr.Field)
		return
	}

	where, filterArgs := buildUserFilters(options)
	var total int64
	if !executeQuery(w, r, "count users", "SELECT COUNT(*) FROM users"+where, filterArgs, func(rows pgx.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("count query returned no result")
		}
		return rows.Scan(&total)
	}) {
		return
	}

	users := make(map[int]User)
	args := append([]any(nil), filterArgs...)
	args = append(args, options.Limit, options.Offset)
	query := fmt.Sprintf(
		"SELECT %s FROM users%s ORDER BY id LIMIT $%d OFFSET $%d",
		userColumns,
		where,
		len(args)-1,
		len(args),
	)
	if !executeQuery(w, r, "get users", query, args, func(rows pgx.Rows) error {
		for rows.Next() {
			user, err := scanUser(rows)
			if err != nil {
				return err
			}
			users[user.id] = user
		}
		return nil
	}) {
		return
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	w.Header().Set("X-Limit", strconv.Itoa(options.Limit))
	w.Header().Set("X-Offset", strconv.Itoa(options.Offset))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Println("Не удалось кодировать пользователей как JSON:", err)
	}
}

func make_get_user(wildcard string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := parseUserID(r, wildcard)
		if !ok {
			writeAPIError(w, http.StatusBadRequest, "Некорректный идентификатор пользователя", "id")
			return
		}

		var user User
		found := false
		if !executeQuery(w, r, "get user", "SELECT "+userColumns+" FROM users WHERE id = $1", []any{key}, func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			var scanErr error
			user, scanErr = scanUser(rows)
			found = scanErr == nil
			return scanErr
		}) {
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "Пользователь не найден", "id")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(user); err != nil {
			log.Printf("Не удалось кодировать пользователя `%v` как JSON: %v", user, err)
		}
	}
}

func post_user(w http.ResponseWriter, r *http.Request) {
	user, ok := decodeAndValidateUser(w, r)
	if !ok {
		return
	}

	query := "INSERT INTO users (name, age, comment, email, phone) VALUES ($1, $2, $3, $4, $5) RETURNING id"
	executeQuery(w, r, "create user", query, []any{user.Name, user.Age, user.Comment, user.Email, user.Phone}, func(rows pgx.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("insert did not return a user id")
		}
		return rows.Scan(&user.id)
	})
}

func make_put_user(wildcard string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := parseUserID(r, wildcard)
		if !ok {
			writeAPIError(w, http.StatusBadRequest, "Некорректный идентификатор пользователя", "id")
			return
		}

		update, err := decodeUserUpdate(r)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "Некорректный JSON: "+err.Error(), "")
			return
		}
		if validationErr := validateUserUpdate(&update); validationErr != nil {
			writeAPIError(w, http.StatusUnprocessableEntity, validationErr.Message, validationErr.Field)
			return
		}

		query, args := buildUserUpdateQuery(update, key)
		found := false
		if len(update.setFields()) == 0 {
			if !executeQuery(w, r, "check user before empty update", query, args, func(rows pgx.Rows) error {
				if !rows.Next() {
					return nil
				}
				found = true
				var existingID int
				return rows.Scan(&existingID)
			}) {
				return
			}
		} else if !executeQuery(w, r, "update user", query, args, func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			found = true
			var updatedID int
			return rows.Scan(&updatedID)
		}) {
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "Пользователь не найден", "id")
		}
	}
}

func make_delete_user(wildcard string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := parseUserID(r, wildcard)
		if !ok {
			writeAPIError(w, http.StatusBadRequest, "Некорректный идентификатор пользователя", "id")
			return
		}

		found := false
		if !executeQuery(w, r, "delete user", "DELETE FROM users WHERE id = $1 RETURNING id", []any{key}, func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			found = true
			var deletedID int
			return rows.Scan(&deletedID)
		}) {
			return
		}
		if !found {
			writeAPIError(w, http.StatusNotFound, "Пользователь не найден", "id")
		}
	}
}

type validationError struct {
	Field   string
	Message string
}

type userListOptions struct {
	Limit  int
	Offset int
	Name   string
	Email  string
	Phone  string
	MinAge *int
	MaxAge *int
}

func parseUserListOptions(query url.Values) (userListOptions, *validationError) {
	options := userListOptions{Limit: defaultUsersLimit}
	var validationErr *validationError
	if options.Limit, validationErr = parseBoundedQueryInt(query, "limit", defaultUsersLimit, 1, maxUsersLimit); validationErr != nil {
		return userListOptions{}, validationErr
	}
	if options.Offset, validationErr = parseBoundedQueryInt(query, "offset", 0, 0, int(^uint(0)>>1)); validationErr != nil {
		return userListOptions{}, validationErr
	}

	options.Name = strings.TrimSpace(query.Get("name"))
	options.Email = strings.TrimSpace(query.Get("email"))
	options.Phone = strings.TrimSpace(query.Get("phone"))
	if options.MinAge, validationErr = parseOptionalAge(query, "min_age"); validationErr != nil {
		return userListOptions{}, validationErr
	}
	if options.MaxAge, validationErr = parseOptionalAge(query, "max_age"); validationErr != nil {
		return userListOptions{}, validationErr
	}
	if options.MinAge != nil && options.MaxAge != nil && *options.MinAge > *options.MaxAge {
		return userListOptions{}, &validationError{Field: "min_age", Message: "min_age не может быть больше max_age"}
	}
	return options, nil
}

func parseBoundedQueryInt(query url.Values, field string, defaultValue, minValue, maxValue int) (int, *validationError) {
	rawValues, exists := query[field]
	if !exists {
		return defaultValue, nil
	}
	if len(rawValues) != 1 {
		return 0, &validationError{Field: field, Message: field + " должен быть указан один раз"}
	}
	value, err := strconv.Atoi(rawValues[0])
	if err != nil || value < minValue || value > maxValue {
		return 0, &validationError{
			Field:   field,
			Message: fmt.Sprintf("%s должен быть целым числом от %d до %d", field, minValue, maxValue),
		}
	}
	return value, nil
}

func parseOptionalAge(query url.Values, field string) (*int, *validationError) {
	_, exists := query[field]
	if !exists {
		return nil, nil
	}
	value, validationErr := parseBoundedQueryInt(query, field, 0, 0, 150)
	if validationErr != nil {
		return nil, validationErr
	}
	return &value, nil
}

func buildUserFilters(options userListOptions) (string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)
	addTextFilter := func(column, value string) {
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("STRPOS(LOWER(%s), LOWER($%d)) > 0", column, len(args)))
	}

	addTextFilter("name", options.Name)
	addTextFilter("email", options.Email)
	addTextFilter("phone", options.Phone)
	if options.MinAge != nil {
		args = append(args, *options.MinAge)
		conditions = append(conditions, fmt.Sprintf("age >= $%d", len(args)))
	}
	if options.MaxAge != nil {
		args = append(args, *options.MaxAge)
		conditions = append(conditions, fmt.Sprintf("age <= $%d", len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

type optionalField[T any] struct {
	Present bool
	Null    bool
	Value   T
}

type userUpdate struct {
	Name    optionalField[string]
	Age     optionalField[int]
	Comment optionalField[string]
	Email   optionalField[string]
	Phone   optionalField[string]
}

func decodeUserUpdate(r *http.Request) (userUpdate, error) {
	fields, err := decodeJSONObject(r)
	if err != nil {
		return userUpdate{}, err
	}

	allowed := map[string]bool{
		"name": true, "age": true, "comment": true, "email": true, "phone": true,
	}
	for name := range fields {
		if !allowed[name] {
			return userUpdate{}, fmt.Errorf("неизвестное поле `%s`", name)
		}
	}

	var update userUpdate
	if update.Name, err = decodeOptionalField[string](fields["name"]); err != nil {
		return userUpdate{}, fmt.Errorf("поле `name`: %w", err)
	}
	if update.Age, err = decodeOptionalField[int](fields["age"]); err != nil {
		return userUpdate{}, fmt.Errorf("поле `age`: %w", err)
	}
	if update.Comment, err = decodeOptionalField[string](fields["comment"]); err != nil {
		return userUpdate{}, fmt.Errorf("поле `comment`: %w", err)
	}
	if update.Email, err = decodeOptionalField[string](fields["email"]); err != nil {
		return userUpdate{}, fmt.Errorf("поле `email`: %w", err)
	}
	if update.Phone, err = decodeOptionalField[string](fields["phone"]); err != nil {
		return userUpdate{}, fmt.Errorf("поле `phone`: %w", err)
	}
	return update, nil
}

func decodeJSONObject(r *http.Request) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(r.Body)
	var body json.RawMessage
	if err := decoder.Decode(&body); err != nil {
		return nil, err
	}
	trimmedBody := bytes.TrimSpace(body)
	if len(trimmedBody) == 0 || trimmedBody[0] != '{' {
		return nil, fmt.Errorf("ожидается JSON-объект пользователя")
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("в запросе должно быть только одно JSON-значение")
		}
		return nil, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmedBody, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func decodeOptionalField[T any](raw json.RawMessage) (optionalField[T], error) {
	if len(raw) == 0 {
		return optionalField[T]{}, nil
	}

	field := optionalField[T]{Present: true}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		field.Null = true
		return field, nil
	}
	if err := json.Unmarshal(raw, &field.Value); err != nil {
		return optionalField[T]{}, err
	}
	return field, nil
}

func validateUserUpdate(update *userUpdate) *validationError {
	if update.Name.Present {
		if update.Name.Null {
			return &validationError{Field: "name", Message: "Имя не может быть null"}
		}
		name, validationErr := validateName(update.Name.Value)
		if validationErr != nil {
			return validationErr
		}
		update.Name.Value = name
	}

	if update.Age.Present {
		if update.Age.Null {
			return &validationError{Field: "age", Message: "Возраст не может быть null"}
		}
		if validationErr := validateAge(update.Age.Value); validationErr != nil {
			return validationErr
		}
	}

	if update.Email.Present {
		if update.Email.Null {
			return &validationError{Field: "email", Message: "Email не может быть null"}
		}
		email, validationErr := validateEmail(update.Email.Value)
		if validationErr != nil {
			return validationErr
		}
		update.Email.Value = email
	}

	if update.Phone.Present && !update.Phone.Null {
		phone, validationErr := validatePhone(update.Phone.Value)
		if validationErr != nil {
			return validationErr
		}
		update.Phone.Value = phone
	}

	return nil
}

func (u userUpdate) setFields() []string {
	fields := make([]string, 0, 5)
	if u.Name.Present {
		fields = append(fields, "name")
	}
	if u.Age.Present {
		fields = append(fields, "age")
	}
	if u.Comment.Present {
		fields = append(fields, "comment")
	}
	if u.Email.Present {
		fields = append(fields, "email")
	}
	if u.Phone.Present {
		fields = append(fields, "phone")
	}
	return fields
}

func buildUserUpdateQuery(update userUpdate, id int) (string, []any) {
	fields := update.setFields()
	if len(fields) == 0 {
		return "SELECT id FROM users WHERE id = $1", []any{id}
	}

	args := make([]any, 0, len(fields)+1)
	assignments := make([]string, 0, len(fields))
	appendAssignment := func(column string, value any) {
		args = append(args, value)
		assignments = append(assignments, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if update.Name.Present {
		appendAssignment("name", update.Name.Value)
	}
	if update.Age.Present {
		appendAssignment("age", update.Age.Value)
	}
	if update.Comment.Present {
		var value any = update.Comment.Value
		if update.Comment.Null {
			value = nil
		}
		appendAssignment("comment", value)
	}
	if update.Email.Present {
		appendAssignment("email", update.Email.Value)
	}
	if update.Phone.Present {
		var value any = update.Phone.Value
		if update.Phone.Null {
			value = nil
		}
		appendAssignment("phone", value)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d RETURNING id", strings.Join(assignments, ", "), len(args))
	return query, args
}

func parseUserID(r *http.Request, wildcard string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(wildcard))
	return id, err == nil && id > 0
}

func decodeAndValidateUser(w http.ResponseWriter, r *http.Request) (User, bool) {
	user, ageProvided, err := decodeUser(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Некорректный JSON: "+err.Error(), "")
		return User{}, false
	}
	if validationErr := validateUser(&user, ageProvided); validationErr != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, validationErr.Message, validationErr.Field)
		return User{}, false
	}
	return user, true
}

func decodeUser(r *http.Request) (User, bool, error) {
	var input struct {
		Name    string  `json:"name"`
		Age     *int    `json:"age"`
		Comment *string `json:"comment"`
		Email   string  `json:"email"`
		Phone   *string `json:"phone"`
	}
	decoder := json.NewDecoder(r.Body)
	var body json.RawMessage
	if err := decoder.Decode(&body); err != nil {
		return User{}, false, err
	}
	trimmedBody := bytes.TrimSpace(body)
	if len(trimmedBody) == 0 || trimmedBody[0] != '{' {
		return User{}, false, fmt.Errorf("ожидается JSON-объект пользователя")
	}

	inputDecoder := json.NewDecoder(bytes.NewReader(trimmedBody))
	inputDecoder.DisallowUnknownFields()
	if err := inputDecoder.Decode(&input); err != nil {
		return User{}, false, err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return User{}, false, fmt.Errorf("в запросе должно быть только одно JSON-значение")
		}
		return User{}, false, err
	}

	user := User{
		Name:    input.Name,
		Comment: input.Comment,
		Email:   input.Email,
		Phone:   input.Phone,
	}
	if input.Age != nil {
		user.Age = *input.Age
	}
	return user, input.Age != nil, nil
}

func validateUser(user *User, ageProvided bool) *validationError {
	name, validationErr := validateName(user.Name)
	if validationErr != nil {
		return validationErr
	}
	user.Name = name

	if !ageProvided {
		return &validationError{Field: "age", Message: "Возраст обязателен для заполнения"}
	}
	if validationErr := validateAge(user.Age); validationErr != nil {
		return validationErr
	}

	email, validationErr := validateEmail(user.Email)
	if validationErr != nil {
		return validationErr
	}
	user.Email = email

	if user.Phone != nil {
		phone, validationErr := validatePhone(*user.Phone)
		if validationErr != nil {
			return validationErr
		}
		user.Phone = &phone
	}

	return nil
}

func validateName(value string) (string, *validationError) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", &validationError{Field: "name", Message: "Имя обязательно для заполнения"}
	}
	return name, nil
}

func validateAge(age int) *validationError {
	if age < 0 || age > 150 {
		return &validationError{Field: "age", Message: "Возраст должен быть от 0 до 150 лет"}
	}
	return nil
}

func validateEmail(value string) (string, *validationError) {
	email := strings.TrimSpace(value)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", &validationError{Field: "email", Message: "Укажите корректный адрес электронной почты"}
	}
	return email, nil
}

func validatePhone(value string) (string, *validationError) {
	phone := strings.TrimSpace(value)
	if !validPhone(phone) {
		return "", &validationError{Field: "phone", Message: "Телефон должен содержать от 7 до 15 цифр и допустимые разделители"}
	}
	return phone, nil
}

func validPhone(phone string) bool {
	digits := 0
	for index, char := range phone {
		switch {
		case char >= '0' && char <= '9':
			digits++
		case char == '+' && index == 0:
		case char == ' ' || char == '-' || char == '(' || char == ')' || char == '.':
		default:
			return false
		}
	}
	return digits >= 7 && digits <= 15
}

type DbConfig struct {
	Ip          string
	Port        int
	RetryCount  int
	RetryPeriod time.Duration
	DbName      string
	DbUser      string
	DbPwd       *string
}

func NewDbConfig() DbConfig {
	return DbConfig{
		Ip:          "127.0.0.1",
		Port:        5432,
		RetryCount:  10,
		RetryPeriod: time.Millisecond * 200,
		DbName:      "goserv",
		DbUser:      "postgres",
		DbPwd:       nil,
	}
}

func init_db(cfg *DbConfig) error {
	if cfg == nil {
		return fmt.Errorf("database config is nil")
	}
	if cfg.Port <= 0 {
		return fmt.Errorf("database port must be positive")
	}

	userInfo := url.User(cfg.DbUser)
	if cfg.DbPwd != nil {
		userInfo = url.UserPassword(cfg.DbUser, *cfg.DbPwd)
	}
	dsn := (&url.URL{
		Scheme: "postgresql",
		User:   userInfo,
		Host:   net.JoinHostPort(cfg.Ip, strconv.Itoa(cfg.Port)),
		Path:   cfg.DbName,
	}).String()

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return err
	}

	retries := max(cfg.RetryCount, 1)

	var pingErr error
	for attempt := 0; attempt < retries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pingErr = pool.Ping(ctx)
		cancel()
		if pingErr == nil {
			break
		}
		if attempt+1 < retries && cfg.RetryPeriod > 0 {
			time.Sleep(cfg.RetryPeriod)
		}
	}
	if pingErr != nil {
		pool.Close()
		return fmt.Errorf("could not connect to database: %w", pingErr)
	}

	for _, statement := range strings.Split(usersSchema, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			pool.Close()
			return fmt.Errorf("could not initialize users schema: %w", err)
		}
	}

	if db != nil {
		db.Close()
	}
	db = pool
	cfgCopy := *cfg
	dbcfg = &cfgCopy
	return nil
}

func RunApi(ip string, port int, dbconf *DbConfig) error {
	if dbconf == nil {
		return fmt.Errorf("database config is nil")
	}

	serv := httpserv.New()
	serv.Ip = ip
	serv.Port = port
	serv.Routes = apiRoutes()

	serv.Middleware = func(r *http.Request, accept chan bool, status chan int) {
		start := time.Now()
		accept <- true
		code := <-status
		took := time.Since(start)
		var took_str string
		if took.Microseconds() < 2000 {
			took_str = fmt.Sprintf("%dµs", took.Microseconds())
		} else {
			took_str = fmt.Sprintf("%dms", took.Milliseconds())
		}
		log.Printf(
			"[%s] %s %s -> %d %s %s",
			r.RemoteAddr,
			r.Method,
			r.URL.String(),
			code,
			http.StatusText(code),
			took_str,
		)
	}

	log.Printf(
		"Подключаемся к базе данных `%s` по адресу %s:%d как пользователь `%s`",
		dbconf.DbName,
		dbconf.Ip,
		dbconf.Port,
		dbconf.DbUser,
	)
	if err := init_db(dbconf); err != nil {
		log.Println("[ERR] Не удалось подключиться к базе данных:", err)
		return err
	}
	defer db.Close()

	log.Printf("Starting REST API server on %s:%d\n", serv.Ip, serv.Port)
	return serv.Run()
}

func apiRoutes() httpserv.Routes {
	return httpserv.Routes{
		"POST /auth/register": post_register,
		"POST /auth/login":    post_login,
		"GET /auth/csrf":      authenticate(false, false, get_csrf_token),
		"POST /auth/logout":   authenticate(false, true, post_logout),
		"GET /users":          authenticate(false, false, get_users),
		"GET /users/{id}":     authenticate(false, false, make_get_user("id")),
		"POST /users":         authenticate(true, true, post_user),
		"PUT /users/{id}":     authenticate(true, true, make_put_user("id")),
		"DELETE /users/{id}":  authenticate(true, true, make_delete_user("id")),
		"GET /admin/info":     authenticate(true, false, get_admin_info),
	}
}
