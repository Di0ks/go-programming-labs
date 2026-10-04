package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const defaultCredentialsPath = "users.json"

type credentialFile struct {
	Users []credential `json:"users"`
}

type credential struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

type Authenticator struct {
	path string
	mu   sync.Mutex
}

type Session struct {
	Username string
	token    string
	Client   UserAPI
}

var errInvalidCredentials = errors.New("неверное имя пользователя или пароль")

func NewAuthenticator(path string) *Authenticator {
	if path == "" {
		path = defaultCredentialsPath
	}
	return &Authenticator{path: path}
}

func (authenticator *Authenticator) Authenticate(username, password string) (*Session, error) {
	authenticator.mu.Lock()
	users, err := authenticator.loadUsers()
	authenticator.mu.Unlock()
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Username == username && bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil {
			tokenBytes := make([]byte, 32)
			if _, err := rand.Read(tokenBytes); err != nil {
				return nil, fmt.Errorf("не удалось создать токен сессии: %w", err)
			}
			return &Session{
				Username: username,
				token:    base64.RawURLEncoding.EncodeToString(tokenBytes),
			}, nil
		}
	}
	return nil, errInvalidCredentials
}

func (authenticator *Authenticator) Register(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("имя пользователя не должно быть пустым")
	}
	if password == "" {
		return errors.New("пароль не должен быть пустым")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("не удалось захэшировать пароль: %w", err)
	}

	authenticator.mu.Lock()
	defer authenticator.mu.Unlock()
	users, err := authenticator.loadUsers()
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.Username == username {
			return fmt.Errorf("пользователь %q уже зарегистрирован", username)
		}
	}
	users = append(users, credential{Username: username, PasswordHash: string(passwordHash)})
	return authenticator.saveUsers(users)
}

func (authenticator *Authenticator) loadUsers() ([]credential, error) {
	file, err := os.Open(authenticator.path)
	if errors.Is(err, os.ErrNotExist) {
		return []credential{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл учётных записей %q: %w", authenticator.path, err)
	}
	defer file.Close()

	var data credentialFile
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл учётных записей %q: %w", authenticator.path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("файл учётных записей %q содержит более одного JSON-объекта", authenticator.path)
		}
		return nil, fmt.Errorf("ошибка в конце файла учётных записей %q: %w", authenticator.path, err)
	}

	seen := make(map[string]struct{}, len(data.Users))
	for index := range data.Users {
		user := &data.Users[index]
		user.Username = strings.TrimSpace(user.Username)
		if user.Username == "" || user.PasswordHash == "" {
			return nil, fmt.Errorf("пользователь #%d должен иметь непустые username и password_hash", index+1)
		}
		if _, exists := seen[user.Username]; exists {
			return nil, fmt.Errorf("имя пользователя %q указано несколько раз", user.Username)
		}
		if _, err := bcrypt.Cost([]byte(user.PasswordHash)); err != nil {
			return nil, fmt.Errorf("некорректный bcrypt-хэш для пользователя %q: %w", user.Username, err)
		}
		seen[user.Username] = struct{}{}
	}
	return data.Users, nil
}

func (authenticator *Authenticator) saveUsers(users []credential) error {
	directory := filepath.Dir(authenticator.path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("не удалось создать каталог учётных записей: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".users-*.tmp")
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл учётных записей: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("не удалось защитить временный файл учётных записей: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(credentialFile{Users: users}); err != nil {
		temporary.Close()
		return fmt.Errorf("не удалось записать файл учётных записей: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("не удалось сохранить файл учётных записей: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("не удалось закрыть файл учётных записей: %w", err)
	}
	if err := os.Rename(temporaryPath, authenticator.path); err != nil {
		return fmt.Errorf("не удалось обновить файл учётных записей %q: %w", authenticator.path, err)
	}
	return nil
}
