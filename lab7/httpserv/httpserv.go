package httpserv

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

// Функция для использования в качестве middleware для сервера.
//
// Аргументы:
//
// r *http.Request: Обрабатываемый запрос
//
// accept chan bool: Канал для отправки разрешения обработки запроса (при false
// отправляется 403)
//
// status chan int: Канал сигнализирующий завершение обработки запроса cо статусом
// (отправляется в том числе при accept=false, после отправки 403)
type MiddlewareFunc func(
	r *http.Request,
	accept chan bool,
	status chan int,
)

type middlewareResponseWriter struct {
	writer http.ResponseWriter
	status *int
}

func (m middlewareResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := m.writer.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hj.Hijack()
}

func (m middlewareResponseWriter) Header() http.Header {
	return m.writer.Header()
}

func (m middlewareResponseWriter) Write(data []byte) (int, error) {
	return m.writer.Write(data)
}

func (m middlewareResponseWriter) WriteHeader(statusCode int) {
	*m.status = statusCode
	m.writer.WriteHeader(statusCode)
}

func newMiddlewareWriter(writer http.ResponseWriter) middlewareResponseWriter {
	return middlewareResponseWriter{
		writer: writer,
		status: new(http.StatusOK),
	}
}

func wrapHandler(fn http.HandlerFunc, middle MiddlewareFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writer := newMiddlewareWriter(w)
		var accept chan bool = nil
		var done chan int = nil
		if middle != nil {
			accept = make(chan bool)
			done = make(chan int)
			signal_done := func() {
				done <- *writer.status
			}
			defer signal_done()
			go middle(r, accept, done)

			if !<-accept {
				// статус нужно выставить до отправки ошибки, чтобы
				// middleware залогировал реальный код ответа
				*writer.status = http.StatusForbidden
				http.Error(w, "Not allowed", http.StatusForbidden)
				return
			}
		}
		fn(writer, r)
	}
}

// Карта паттернов и маршрутов
type Routes map[string]http.HandlerFunc

type ServerConfig struct {
	// IP-адрес для использования серевером
	Ip string
	// Порт для использования сервером
	Port int
	// Карта с хэндлерами на каждый шаблон маршрута (шаблоны идентичны
	// этому https://pkg.go.dev/net/http#hdr-Patterns-ServeMux)
	Routes
	// Middleware для дополнительной обработки запросов
	Middleware MiddlewareFunc
}

// Создаёт конфигурацию для HTTP сервера со значениями по умолчанию
func New() ServerConfig {
	return ServerConfig{
		Ip:         "127.0.0.1",
		Port:       8080,
		Middleware: nil,
		Routes:     make(Routes),
	}
}

// Создаёт хэндлер для предоставления содержимого директории path рекурсивно.
//
// Аргумент pattern_matcher позволяет указать шаблон, который использовался
// в маршрутизации, если такой был, например "/img/{imgpath}".
// Для использования всего URI запроса нужно указать пустую строку "".
//
// Файлы отсылаются напрямую в виде байтов, не как вложение.
// Если файл не найден (или возникла какая-либо другая ошибка в процессе),
// то возвращается 404.
// Если запрошенный путь является директорией, то так же возвращается 404.
// Пути, выходящие за пределы указанной директории (например, содержащие
// ..), не обслуживаются.
//
// Паникует, если указанный путь не является корректным
// путём файловой системы (существование директории при этом не проверяется)
func ServeDir(pattern_matcher, dir_path string) http.HandlerFunc {
	// путь может быть и абсолютным, поэтому используем Abs для проверки
	abs_dir_path, err := filepath.Abs(filepath.Clean(dir_path))
	if err != nil {
		panic(fmt.Sprintln("Некорректный путь к директории:", dir_path))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var rel_path string
		if pattern_matcher == "" {
			// Clean от корня "/" убирает выход за пределы ("../")
			rel_path = filepath.Clean("/" + r.URL.Path)
		} else {
			rel_path = r.PathValue(pattern_matcher)
		}
		// Clean убирает exit-последовательности вида "../", поэтому
		// обращение через GetFilesystem не требуется
		rel_path = filepath.Clean("/" + rel_path)
		if rel_path == "/" {
			http.NotFound(w, r)
			return
		}
		rel_path = rel_path[1:]
		full_path := filepath.Join(abs_dir_path, rel_path)
		data, err := os.ReadFile(full_path)
		if err != nil {
			log.Printf(
				"[ERR] Не удалось прочитать файл по пути %s: %v\n",
				full_path,
				err,
			)
			http.NotFound(w, r)
			return
		}
		write_file_data(w, r, full_path, data)
	}
}

// Создаёт хэндлер для простой отсылки файла по указанному пути
// (абсолютному или относительному).
//
// Файл отсылается напрямую в виде байтов, не как вложение.
// Если файл не найден (или возникла какая-либо другая ошибка в процессе),
// то возвращается 404.
//
// Паникует, если указанный путь не является корректным
// путём файловой системы (существование файла при этом не проверяется)
func ServeFile(path string) http.HandlerFunc {
	abs_path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		panic(fmt.Sprintln("Некорректный путь к файлу:", path))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(abs_path)
		if err != nil {
			log.Printf(
				"[ERR] Не удалось прочитать файл по пути %s: %v\n",
				abs_path,
				err,
			)
			http.NotFound(w, r)
			return
		}
		write_file_data(w, r, abs_path, data)
	}
}

// Отправляет содержимое data как файл по пути full_path (используется только
// для логирования).
//
// При ошибке записи отправляется 404.
func write_file_data(w http.ResponseWriter, r *http.Request, full_path string, data []byte) {
	written, err := w.Write(data)
	if err != nil {
		log.Printf(
			"[ERR] Не удалось записать файл по пути %s: %v\n",
			full_path,
			err,
		)
		http.NotFound(w, r)
		return
	}
	if written < len(data) {
		log.Printf(
			"[WARN] Не все данные были отправлены (%d < %d)\n",
			written,
			len(data),
		)
	}
}

func get_routing(routes Routes, middleware MiddlewareFunc) *http.ServeMux {
	mux := http.NewServeMux()

	for k, v := range routes {
		handler := wrapHandler(v, middleware)
		mux.HandleFunc(k, handler)
	}

	return mux
}

func (cfg ServerConfig) Run() error {
	err := http.ListenAndServe(
		fmt.Sprintf("%s:%d", cfg.Ip, cfg.Port),
		get_routing(cfg.Routes, cfg.Middleware),
	)
	return err
}
