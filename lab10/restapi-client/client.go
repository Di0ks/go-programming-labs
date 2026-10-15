package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type User struct {
	ID      int     `json:"-"`
	Name    string  `json:"name"`
	Age     int     `json:"age"`
	Comment *string `json:"comment"`
	Email   string  `json:"email"`
	Phone   *string `json:"phone"`
}

type ListOptions struct {
	Limit  int
	Offset int
	Name   string
	Email  string
	Phone  string
	MinAge *int
	MaxAge *int
}

type ListResult struct {
	Users  []User
	Total  int
	Limit  int
	Offset int
}

type APIClient struct {
	baseURL    string
	httpClient *http.Client
	csrfToken  string
}

type AdminInfo struct {
	GoVersion      string `json:"go_version"`
	Goroutines     int    `json:"goroutines"`
	Database       string `json:"database"`
	DatabaseStatus string `json:"database_status"`
	UserCount      int64  `json:"user_count"`
}

type APIError struct {
	StatusCode int
	Message    string
	Field      string
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
	if e.Field != "" {
		message += " (поле: " + e.Field + ")"
	}
	return message
}

func NewAPIClient(address string, httpClient *http.Client) (*APIClient, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		address = "127.0.0.1:8080"
	}
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}

	parsedURL, err := url.Parse(address)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("некорректный адрес сервера %q: укажите host:port или URL с http/https", address)
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &APIClient{baseURL: strings.TrimRight(parsedURL.String(), "/"), httpClient: httpClient}, nil
}

func (client *APIClient) NewSessionClient() (*APIClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать cookie jar: %w", err)
	}
	httpClient := &http.Client{
		Transport:     client.httpClient.Transport,
		CheckRedirect: client.httpClient.CheckRedirect,
		Jar:           jar,
		Timeout:       client.httpClient.Timeout,
	}
	return &APIClient{baseURL: client.baseURL, httpClient: httpClient}, nil
}

func (client *APIClient) ListUsers(options ListOptions) (ListResult, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(options.Limit))
	query.Set("offset", strconv.Itoa(options.Offset))
	if options.Name != "" {
		query.Set("name", options.Name)
	}
	if options.Email != "" {
		query.Set("email", options.Email)
	}
	if options.Phone != "" {
		query.Set("phone", options.Phone)
	}
	if options.MinAge != nil {
		query.Set("min_age", strconv.Itoa(*options.MinAge))
	}
	if options.MaxAge != nil {
		query.Set("max_age", strconv.Itoa(*options.MaxAge))
	}

	var usersByID map[string]User
	response, err := client.request(http.MethodGet, "/users", query, nil, &usersByID)
	if err != nil {
		return ListResult{}, err
	}

	result := ListResult{Users: make([]User, 0, len(usersByID))}
	for idText, user := range usersByID {
		id, err := strconv.Atoi(idText)
		if err != nil {
			return ListResult{}, fmt.Errorf("сервер вернул некорректный ID %q: %w", idText, err)
		}
		user.ID = id
		result.Users = append(result.Users, user)
	}
	sort.Slice(result.Users, func(i, j int) bool {
		return result.Users[i].ID < result.Users[j].ID
	})
	result.Total, err = parseResponseInt(response, "X-Total-Count")
	if err != nil {
		return ListResult{}, err
	}
	result.Limit, err = parseResponseInt(response, "X-Limit")
	if err != nil {
		return ListResult{}, err
	}
	result.Offset, err = parseResponseInt(response, "X-Offset")
	if err != nil {
		return ListResult{}, err
	}
	return result, nil
}

func (client *APIClient) GetUser(id int) (User, error) {
	var user User
	_, err := client.request(http.MethodGet, "/users/"+strconv.Itoa(id), nil, nil, &user)
	user.ID = id
	return user, err
}

func (client *APIClient) CreateUser(user User) error {
	_, err := client.request(http.MethodPost, "/users", nil, user, nil)
	return err
}

func (client *APIClient) UpdateUser(id int, fields map[string]any) error {
	_, err := client.request(http.MethodPut, "/users/"+strconv.Itoa(id), nil, fields, nil)
	return err
}

func (client *APIClient) DeleteUser(id int) error {
	_, err := client.request(http.MethodDelete, "/users/"+strconv.Itoa(id), nil, nil, nil)
	return err
}

func (client *APIClient) GetAdminInfo() (AdminInfo, error) {
	var info AdminInfo
	_, err := client.request(http.MethodGet, "/admin/info", nil, nil, &info)
	return info, err
}

func (client *APIClient) fetchCSRFToken() error {
	var response struct {
		Token string `json:"csrf_token"`
	}
	if _, err := client.request(http.MethodGet, "/auth/csrf", nil, nil, &response); err != nil {
		return err
	}
	if response.Token == "" {
		return fmt.Errorf("сервер не вернул CSRF-токен")
	}
	client.csrfToken = response.Token
	return nil
}

func (client *APIClient) Logout() error {
	_, err := client.request(http.MethodPost, "/auth/logout", nil, nil, nil)
	return err
}

func (client *APIClient) request(method, path string, query url.Values, body any, target any) (*http.Response, error) {
	requestURL := client.baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("не удалось подготовить запрос: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, requestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать HTTP-запрос: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if client.csrfToken != "" && method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
		request.Header.Set("X-CSRF-Token", client.csrfToken)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка соединения с сервером: %w", err)
	}
	defer response.Body.Close()

	responseData, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать ответ сервера: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var apiError struct {
			Error string `json:"error"`
			Field string `json:"field"`
		}
		if err := json.Unmarshal(responseData, &apiError); err != nil || apiError.Error == "" {
			apiError.Error = strings.TrimSpace(string(responseData))
		}
		if apiError.Error == "" {
			apiError.Error = http.StatusText(response.StatusCode)
		}
		return nil, &APIError{StatusCode: response.StatusCode, Message: apiError.Error, Field: apiError.Field}
	}
	if target != nil && len(responseData) > 0 {
		if err := json.Unmarshal(responseData, target); err != nil {
			return nil, fmt.Errorf("сервер вернул некорректный JSON: %w", err)
		}
	}
	return response, nil
}

func parseResponseInt(response *http.Response, header string) (int, error) {
	value := response.Header.Get(header)
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("сервер вернул некорректный заголовок %s: %q", header, value)
	}
	return parsed, nil
}
