package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/shlex"
	"golang.org/x/term"
)

type App struct {
	apiClient     *APIClient
	authenticator *Authenticator
	sessions      map[string]*Session
	activeUser    string
	input         *bufio.Reader
	output        io.Writer
	listOptions   ListOptions
}

func NewApp(apiClient *APIClient, authenticator *Authenticator, input io.Reader, output io.Writer) *App {
	return &App{
		apiClient:     apiClient,
		authenticator: authenticator,
		sessions:      make(map[string]*Session),
		input:         bufio.NewReader(input),
		output:        output,
		listOptions:   ListOptions{Limit: 20},
	}
}

func (app *App) Run() {
	app.printMainHelp()
	for {
		prompt := "auth> "
		if app.activeUser != "" {
			prompt = app.activeUser + "> "
		}
		line, err := app.readLine(prompt)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintf(app.output, "Ошибка чтения: %v\n", err)
			}
			return
		}
		args, err := parseCommand(line)
		if err != nil {
			fmt.Fprintf(app.output, "Ошибка разбора команды: %v\n", err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		if app.activeUser == "" {
			app.runUnauthenticatedCommand(args)
			if args[0] == "quit" || args[0] == "exit" {
				return
			}
			continue
		}
		if app.runAuthenticatedCommand(args) {
			return
		}
	}
}

func (app *App) runUnauthenticatedCommand(args []string) {
	switch args[0] {
	case "help":
		app.printMainHelp()
	case "login":
		if len(args) != 1 {
			fmt.Fprintln(app.output, "Использование: login")
			return
		}
		if err := app.login(); err != nil {
			app.printError(err)
		}
	case "reg":
		if len(args) != 1 {
			fmt.Fprintln(app.output, "Использование: reg")
			return
		}
		if err := app.register(); err != nil {
			app.printError(err)
		}
	case "sessions":
		app.printSessions()
	case "switch":
		if err := app.switchUser(args); err != nil {
			app.printError(err)
		}
	case "logout":
		fmt.Fprintln(app.output, "Нет активной сессии.")
	case "quit", "exit":
		return
	default:
		fmt.Fprintf(app.output, "Команда %q недоступна без входа. Используйте login или reg.\n", args[0])
	}
}

func (app *App) runAuthenticatedCommand(args []string) bool {
	switch args[0] {
	case "help":
		app.printMainHelp()
	case "list":
		if err := app.listMenu(); err != nil {
			app.printError(err)
		}
	case "login":
		if len(args) != 1 {
			fmt.Fprintln(app.output, "Использование: login")
			return false
		}
		if err := app.login(); err != nil {
			app.printError(err)
		}
	case "reg":
		if len(args) != 1 {
			fmt.Fprintln(app.output, "Использование: reg")
			return false
		}
		if err := app.register(); err != nil {
			app.printError(err)
		}
	case "sessions":
		app.printSessions()
	case "switch":
		if err := app.switchUser(args); err != nil {
			app.printError(err)
		}
	case "logout":
		if len(args) != 1 {
			fmt.Fprintln(app.output, "Использование: logout")
			return false
		}
		app.logout()
	case "add":
		if len(args) != 1 {
			fmt.Fprintln(app.output, "Использование: add")
			return false
		}
		if err := app.requireAdmin(); err != nil {
			app.printError(err)
		} else if err := app.addUser(); err != nil {
			app.printError(err)
		}
	case "edit":
		id, ok := app.commandID(args, "edit")
		if ok {
			if err := app.requireAdmin(); err != nil {
				app.printError(err)
			} else if err := app.editUser(id); err != nil {
				app.printError(err)
			}
		}
	case "del":
		id, ok := app.commandID(args, "del")
		if ok {
			if err := app.requireAdmin(); err != nil {
				app.printError(err)
			} else if err := app.deleteUser(id); err != nil {
				app.printError(err)
			}
		}
	case "get":
		id, ok := app.commandID(args, "get")
		if ok {
			user, err := app.currentSession().Client.GetUser(id)
			if err != nil {
				app.printError(err)
			} else {
				app.printUser(user)
			}
		}
	case "info":
		if err := app.requireAdmin(); err != nil {
			app.printError(err)
		} else if err := app.showAdminInfo(); err != nil {
			app.printError(err)
		}
	case "quit", "exit":
		return true
	default:
		fmt.Fprintf(app.output, "Неизвестная команда %q. Введите help.\n", args[0])
	}
	return false
}

func (app *App) login() error {
	email, err := app.readLine("Email: ")
	if err != nil {
		return err
	}
	password, err := app.readPassword("Пароль: ")
	if err != nil {
		return err
	}
	defer func() { password = "" }()

	session, err := app.authenticator.Authenticate(strings.TrimSpace(email), password)
	if err != nil {
		return err
	}
	app.addSession(session)
	fmt.Fprintf(app.output, "Сессия пользователя %q добавлена.\n", session.Username)
	app.printMainHelp()
	return nil
}

func (app *App) register() error {
	name, err := app.readLine("Имя: ")
	if err != nil {
		return err
	}
	age, err := app.readInt("Возраст: ")
	if err != nil {
		return err
	}
	email, err := app.readLine("Email: ")
	if err != nil {
		return err
	}
	comment, err := app.readOptional("Комментарий (Enter — пропустить): ")
	if err != nil {
		return err
	}
	phone, err := app.readOptional("Телефон (Enter — пропустить): ")
	if err != nil {
		return err
	}
	password, err := app.readPassword("Пароль: ")
	if err != nil {
		return err
	}
	defer func() { password = "" }()
	confirmation, err := app.readPassword("Повторите пароль: ")
	if err != nil {
		return err
	}
	defer func() { confirmation = "" }()
	if password != confirmation {
		return errors.New("пароли не совпадают")
	}
	session, err := app.authenticator.Register(User{
		Name: name, Age: age, Email: strings.TrimSpace(email), Comment: comment, Phone: phone,
	}, password)
	if err != nil {
		return err
	}
	app.addSession(session)
	fmt.Fprintf(app.output, "Пользователь %q зарегистрирован; сессия добавлена.\n", session.Username)
	app.printMainHelp()
	return nil
}

func (app *App) addSession(session *Session) {
	app.sessions[session.Username] = session
	if app.activeUser == "" {
		app.activeUser = session.Username
	}
}

func (app *App) switchUser(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("использование: switch <имя пользователя>")
	}
	username := strings.TrimSpace(args[1])
	if _, exists := app.sessions[username]; !exists {
		return fmt.Errorf("для пользователя %q нет сохранённой сессии; сначала выполните login или reg", username)
	}
	app.activeUser = username
	fmt.Fprintf(app.output, "Активная сессия: %s\n", username)
	app.printMainHelp()
	return nil
}

func (app *App) logout() {
	username := app.activeUser
	if session := app.sessions[username]; session != nil {
		if err := session.Client.Logout(); err != nil {
			app.printError(fmt.Errorf("не удалось завершить серверную сессию: %w", err))
		}
	}
	delete(app.sessions, username)
	app.activeUser = ""
	fmt.Fprintf(app.output, "Сессия пользователя %q завершена.\n", username)
	app.printMainHelp()
}

func (app *App) printSessions() {
	if len(app.sessions) == 0 {
		fmt.Fprintln(app.output, "Сохранённых сессий нет.")
		return
	}
	usernames := make([]string, 0, len(app.sessions))
	for username := range app.sessions {
		usernames = append(usernames, username)
	}
	sort.Strings(usernames)
	fmt.Fprintln(app.output, "Сессии:")
	for _, username := range usernames {
		marker := " "
		if username == app.activeUser {
			marker = "*"
		}
		fmt.Fprintf(app.output, " %s %s\n", marker, username)
	}
}

func (app *App) currentSession() *Session {
	return app.sessions[app.activeUser]
}

func (app *App) listMenu() error {
	if err := app.showUsers(); err != nil {
		app.printError(err)
	}
	if app.isAdmin() {
		fmt.Fprintln(app.output, "Подменю list: show, filters, filter <поле> <значение>, limit <n>, offset <n>, add, edit <id>, del <id>, back, help")
	} else {
		fmt.Fprintln(app.output, "Подменю list: show, filters, filter <поле> <значение>, limit <n>, offset <n>, back, help")
	}
	for {
		line, err := app.readLine("list> ")
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		args, err := parseCommand(line)
		if err != nil {
			fmt.Fprintf(app.output, "Ошибка разбора команды: %v\n", err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "show", "refresh":
			if len(args) != 1 {
				fmt.Fprintln(app.output, "Использование: show")
				continue
			}
			if err := app.showUsers(); err != nil {
				app.printError(err)
			}
		case "filters":
			app.printFilters()
		case "filter":
			if err := app.setFilter(args); err != nil {
				app.printError(err)
			} else {
				app.printFilters()
			}
		case "limit", "offset":
			if len(args) != 2 {
				fmt.Fprintf(app.output, "Использование: %s <число>\n", args[0])
				continue
			}
			value, err := strconv.Atoi(args[1])
			if err != nil || value < 0 || (args[0] == "limit" && (value == 0 || value > 100)) {
				fmt.Fprintln(app.output, "Ожидается limit от 1 до 100 или неотрицательный offset")
				continue
			}
			if args[0] == "limit" {
				app.listOptions.Limit = value
			} else {
				app.listOptions.Offset = value
			}
			if err := app.showUsers(); err != nil {
				app.printError(err)
			}
		case "add":
			if len(args) != 1 {
				fmt.Fprintln(app.output, "Использование: add")
				continue
			}
			if err := app.requireAdmin(); err != nil {
				app.printError(err)
			} else if err := app.addUser(); err != nil {
				app.printError(err)
			} else if err := app.showUsers(); err != nil {
				app.printError(err)
			}
		case "edit":
			id, ok := app.commandID(args, "edit")
			if ok {
				if err := app.requireAdmin(); err != nil {
					app.printError(err)
				} else if err := app.editUser(id); err != nil {
					app.printError(err)
				} else if err := app.showUsers(); err != nil {
					app.printError(err)
				}
			}
		case "del":
			id, ok := app.commandID(args, "del")
			if ok {
				if err := app.requireAdmin(); err != nil {
					app.printError(err)
				} else if err := app.deleteUser(id); err != nil {
					app.printError(err)
				} else if err := app.showUsers(); err != nil {
					app.printError(err)
				}
			}
		case "help":
			if app.isAdmin() {
				fmt.Fprintln(app.output, "Команды list: show, filters, filter <поле> <значение>, limit <n>, offset <n>, add, edit <id>, del <id>, back")
			} else {
				fmt.Fprintln(app.output, "Команды list: show, filters, filter <поле> <значение>, limit <n>, offset <n>, back")
			}
		case "back", "quit", "exit":
			return nil
		default:
			fmt.Fprintf(app.output, "Неизвестная команда %q. Введите help.\n", args[0])
		}
	}
}

func (app *App) showUsers() error {
	result, err := app.currentSession().Client.ListUsers(app.listOptions)
	if err != nil {
		return err
	}
	if len(result.Users) == 0 {
		fmt.Fprintln(app.output, "Пользователи не найдены.")
	} else {
		for _, user := range result.Users {
			app.printUser(user)
		}
	}
	fmt.Fprintf(app.output, "Всего: %d; limit: %d; offset: %d\n", result.Total, result.Limit, result.Offset)
	return nil
}

func (app *App) showAdminInfo() error {
	info, err := app.currentSession().Client.GetAdminInfo()
	if err != nil {
		return err
	}
	fmt.Fprintf(app.output, "Go: %s | Goroutines: %d | БД: %s (%s) | Пользователей: %d\n",
		info.GoVersion, info.Goroutines, info.Database, info.DatabaseStatus, info.UserCount)
	return nil
}

func (app *App) isAdmin() bool {
	return app.currentSession() != nil && app.currentSession().Role == "admin"
}

func (app *App) requireAdmin() error {
	if !app.isAdmin() {
		return fmt.Errorf("операция доступна только администратору")
	}
	return nil
}

func (app *App) printUser(user User) {
	fmt.Fprintf(app.output, "ID: %d | Имя: %s | Возраст: %d | Email: %s | Комментарий: %s | Телефон: %s\n",
		user.ID, user.Name, user.Age, user.Email, displayOptional(user.Comment), displayOptional(user.Phone))
}

func (app *App) addUser() error {
	name, err := app.readLine("Имя: ")
	if err != nil {
		return err
	}
	age, err := app.readInt("Возраст: ")
	if err != nil {
		return err
	}
	email, err := app.readLine("Email: ")
	if err != nil {
		return err
	}
	comment, err := app.readOptional("Комментарий (Enter — пропустить): ")
	if err != nil {
		return err
	}
	phone, err := app.readOptional("Телефон (Enter — пропустить): ")
	if err != nil {
		return err
	}
	if err := app.currentSession().Client.CreateUser(User{Name: name, Age: age, Email: email, Comment: comment, Phone: phone}); err != nil {
		return err
	}
	fmt.Fprintln(app.output, "Пользователь добавлен.")
	return nil
}

func (app *App) editUser(id int) error {
	user, err := app.currentSession().Client.GetUser(id)
	if err != nil {
		return err
	}
	app.printUser(user)
	fmt.Fprintln(app.output, "Изменение: set <name|age|email|comment|phone> <значение>, clear <comment|phone>, show, save, cancel, help")
	changes := make(map[string]any)
	for {
		line, err := app.readLine("edit> ")
		if err != nil {
			return err
		}
		args, err := parseCommand(line)
		if err != nil {
			fmt.Fprintf(app.output, "Ошибка разбора команды: %v\n", err)
			continue
		}
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "show":
			if len(args) != 1 {
				fmt.Fprintln(app.output, "Использование: show")
			} else {
				app.printUser(app.userWithChanges(user, changes))
			}
		case "set":
			if len(args) != 3 {
				fmt.Fprintln(app.output, "Использование: set <поле> <значение>")
				continue
			}
			value, err := editValue(args[1], args[2])
			if err != nil {
				app.printError(err)
				continue
			}
			changes[args[1]] = value
			app.printUser(app.userWithChanges(user, changes))
		case "clear":
			if len(args) != 2 || (args[1] != "comment" && args[1] != "phone") {
				fmt.Fprintln(app.output, "Использование: clear <comment|phone>")
				continue
			}
			changes[args[1]] = nil
			app.printUser(app.userWithChanges(user, changes))
		case "save":
			if len(args) != 1 {
				fmt.Fprintln(app.output, "Использование: save")
				continue
			}
			if err := app.currentSession().Client.UpdateUser(id, changes); err != nil {
				return err
			}
			fmt.Fprintln(app.output, "Пользователь обновлён.")
			return nil
		case "cancel", "back", "quit", "exit":
			fmt.Fprintln(app.output, "Изменения отменены.")
			return nil
		case "help":
			fmt.Fprintln(app.output, "set <name|age|email|comment|phone> <значение>; clear <comment|phone>; show; save; cancel")
		default:
			fmt.Fprintf(app.output, "Неизвестная команда %q. Введите help.\n", args[0])
		}
	}
}

func editValue(field, raw string) (any, error) {
	switch field {
	case "name", "email", "comment", "phone":
		return raw, nil
	case "age":
		age, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("возраст должен быть целым числом")
		}
		return age, nil
	default:
		return nil, fmt.Errorf("неизвестное поле %q", field)
	}
}

func (app *App) userWithChanges(user User, changes map[string]any) User {
	for field, value := range changes {
		switch field {
		case "name":
			user.Name = value.(string)
		case "age":
			user.Age = value.(int)
		case "email":
			user.Email = value.(string)
		case "comment":
			user.Comment = optionalString(value)
		case "phone":
			user.Phone = optionalString(value)
		}
	}
	return user
}

func optionalString(value any) *string {
	if value == nil {
		return nil
	}
	text := value.(string)
	return &text
}

func (app *App) deleteUser(id int) error {
	user, err := app.currentSession().Client.GetUser(id)
	if err != nil {
		return err
	}
	app.printUser(user)
	answer, err := app.readLine("Удалить этого пользователя? [y/N]: ")
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes", "д", "да":
		if err := app.currentSession().Client.DeleteUser(id); err != nil {
			return err
		}
		fmt.Fprintln(app.output, "Пользователь удалён.")
	default:
		fmt.Fprintln(app.output, "Удаление отменено.")
	}
	return nil
}

func (app *App) setFilter(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("использование: filter <name|email|phone|min_age|max_age> <значение>; для очистки передайте \"\"")
	}
	field, value := args[1], args[2]
	switch field {
	case "name":
		app.listOptions.Name = value
	case "email":
		app.listOptions.Email = value
	case "phone":
		app.listOptions.Phone = value
	case "min_age", "max_age":
		if value == "" {
			if field == "min_age" {
				app.listOptions.MinAge = nil
			} else {
				app.listOptions.MaxAge = nil
			}
			return nil
		}
		age, err := strconv.Atoi(value)
		if err != nil || age < 0 || age > 150 {
			return fmt.Errorf("%s должен быть целым числом от 0 до 150", field)
		}
		if field == "min_age" {
			if app.listOptions.MaxAge != nil && age > *app.listOptions.MaxAge {
				return fmt.Errorf("min_age не может превышать max_age")
			}
			app.listOptions.MinAge = &age
		} else {
			if app.listOptions.MinAge != nil && age < *app.listOptions.MinAge {
				return fmt.Errorf("max_age не может быть меньше min_age")
			}
			app.listOptions.MaxAge = &age
		}
	default:
		return fmt.Errorf("неизвестный фильтр %q", field)
	}
	return nil
}

func (app *App) printFilters() {
	minAge, maxAge := "не задан", "не задан"
	if app.listOptions.MinAge != nil {
		minAge = strconv.Itoa(*app.listOptions.MinAge)
	}
	if app.listOptions.MaxAge != nil {
		maxAge = strconv.Itoa(*app.listOptions.MaxAge)
	}
	fmt.Fprintf(app.output, "limit=%d offset=%d name=%q email=%q phone=%q min_age=%s max_age=%s\n",
		app.listOptions.Limit, app.listOptions.Offset, app.listOptions.Name, app.listOptions.Email,
		app.listOptions.Phone, minAge, maxAge)
}

func (app *App) commandID(args []string, command string) (int, bool) {
	if len(args) != 2 {
		fmt.Fprintf(app.output, "Использование: %s <id>\n", command)
		return 0, false
	}
	id, err := strconv.Atoi(args[1])
	if err != nil || id <= 0 {
		fmt.Fprintln(app.output, "ID должен быть положительным целым числом")
		return 0, false
	}
	return id, true
}

func (app *App) readLine(prompt string) (string, error) {
	fmt.Fprint(app.output, prompt)
	line, err := app.input.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		return line, err
	}
	return line, nil
}

func (app *App) readPassword(prompt string) (string, error) {
	fmt.Fprint(app.output, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(app.output)
		return string(password), err
	}
	return app.readLine("")
}

func (app *App) readInt(prompt string) (int, error) {
	for {
		value, err := app.readLine(prompt)
		if err != nil {
			return 0, err
		}
		age, err := strconv.Atoi(value)
		if err == nil {
			return age, nil
		}
		fmt.Fprintln(app.output, "Введите целое число.")
	}
}

func (app *App) readOptional(prompt string) (*string, error) {
	value, err := app.readLine(prompt)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	return &value, nil
}

func parseCommand(line string) ([]string, error) {
	return shlex.Split(line)
}

func (app *App) printError(err error) {
	var apiError *APIError
	if errors.As(err, &apiError) {
		fmt.Fprintf(app.output, "Ошибка API: %s\n", apiError)
		return
	}
	fmt.Fprintf(app.output, "Ошибка: %v\n", err)
}

func (app *App) printMainHelp() {
	if app.activeUser == "" {
		fmt.Fprintln(app.output, "Без активной сессии: login, reg, switch <имя>, sessions, help, quit")
		return
	}
	if app.isAdmin() {
		fmt.Fprintf(app.output, "Активен %q (admin). Команды: list, get <id>, add, edit <id>, del <id>, info, login, reg, switch <email>, sessions, logout, help, quit\n", app.activeUser)
	} else {
		fmt.Fprintf(app.output, "Активен %q (user). Команды: list, get <id>, login, reg, switch <email>, sessions, logout, help, quit\n", app.activeUser)
	}
}

func displayOptional(value *string) string {
	if value == nil {
		return "—"
	}
	return *value
}

func readConsoleLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if errors.Is(err, io.EOF) && line != "" {
		return line, nil
	}
	return line, err
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	address, err := readConsoleLine(reader, "Введите адрес сервера (host:port или URL, Enter — http://127.0.0.1:8080): ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Не удалось прочитать адрес: %v\n", err)
		os.Exit(1)
	}
	address = strings.TrimSpace(address)
	apiClient, err := NewAPIClient(address, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	authenticator := NewAuthenticator(apiClient)
	app := NewApp(apiClient, authenticator, reader, os.Stdout)
	app.Run()
	fmt.Println("Завершаем работу")
}
