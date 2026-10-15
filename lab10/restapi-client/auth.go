package main

import (
	"fmt"
	"net/http"
	"strings"
)

type Authenticator struct {
	apiClient *APIClient
}

type Session struct {
	Username string
	Role     string
	Client   *APIClient
}

func NewAuthenticator(apiClient *APIClient) *Authenticator {
	return &Authenticator{apiClient: apiClient}
}

func (authenticator *Authenticator) Authenticate(email, password string) (*Session, error) {
	client, err := authenticator.apiClient.NewSessionClient()
	if err != nil {
		return nil, err
	}
	var response struct {
		Role string `json:"role"`
	}
	if _, err := client.request(http.MethodPost, "/auth/login", nil, map[string]string{
		"email":    strings.TrimSpace(email),
		"password": password,
	}, &response); err != nil {
		return nil, err
	}
	return authenticator.session(email, response.Role, client)
}

func (authenticator *Authenticator) Register(user User, password string) (*Session, error) {
	client, err := authenticator.apiClient.NewSessionClient()
	if err != nil {
		return nil, err
	}
	payload := struct {
		Name     string  `json:"name"`
		Age      int     `json:"age"`
		Comment  *string `json:"comment"`
		Email    string  `json:"email"`
		Phone    *string `json:"phone"`
		Password string  `json:"password"`
	}{user.Name, user.Age, user.Comment, strings.TrimSpace(user.Email), user.Phone, password}
	var response struct {
		Role string `json:"role"`
	}
	if _, err := client.request(http.MethodPost, "/auth/register", nil, payload, &response); err != nil {
		return nil, err
	}
	return authenticator.session(user.Email, response.Role, client)
}

func (authenticator *Authenticator) session(email, role string, client *APIClient) (*Session, error) {
	if role != "user" && role != "admin" {
		return nil, fmt.Errorf("сервер вернул неизвестную роль %q", role)
	}
	if err := client.fetchCSRFToken(); err != nil {
		return nil, fmt.Errorf("не удалось получить CSRF-токен сессии: %w", err)
	}
	return &Session{Username: strings.TrimSpace(email), Role: role, Client: client}, nil
}
