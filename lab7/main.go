package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Di0ks/go-programming-labs/lab7/httpserv"
	"github.com/Di0ks/go-programming-labs/utils"
	"golang.org/x/net/websocket"
)

func handle_conn(c net.Conn) {
	addr := c.RemoteAddr().String()
	data := make([]byte, 64)
message_loop:
	for {
		read_sum := 0
		// если данных слишком много, то они могут не вместиться, и может
		// потребоваться увеличить размер среза
		for start_i := 0; ; {
			read, err := c.Read(data[start_i:])
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					log.Printf("%s закрыл соединение\n", addr)
					break message_loop
				}
				// соединение конкретного клиента не должно ронять весь сервер
				log.Printf(
					"%s: не удалось прочитать сообщение: %v\n",
					addr,
					err,
				)
				break message_loop
			}
			read_sum += read
			if read+start_i == len(data) {
				data = slices.Grow(data, len(data)*2)
				data = data[:cap(data)]
				start_i += read
			} else {
				break
			}
		}

		message := string(data[:read_sum])
		log.Printf(
			"сообщение от %s: %s\n",
			addr,
			message,
		)

		_, err := fmt.Fprintf(c, "принято байт: %d\n", read_sum)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Printf("%s закрыл соединение\n", addr)
				break message_loop
			}
			log.Printf(
				"%s: не удалось отправить сообщение: %v\n",
				addr,
				err,
			)
			break message_loop
		}
	}

	err := c.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf(
			"%s: не удалось завершить подключение: %v\n",
			addr,
			err,
		)
	}
}

func Serve(ctx context.Context, l net.Listener, wg *sync.WaitGroup) {
	log.Printf("Сервер запущен на %s\n", l.Addr().String())
	// активные соединения, чтобы можно было их закрыть при завершении
	var conns_mu sync.Mutex
	conns := make(map[net.Conn]struct{})

	// приёмник подключений
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					break
				}
				log.Println("не удалось принять подключение:", err)
				break
			}
			log.Printf("Принято подключение от %s\n", conn.RemoteAddr().String())
			conns_mu.Lock()
			conns[conn] = struct{}{}
			conns_mu.Unlock()
			wg.Go(func() {
				defer func() {
					conns_mu.Lock()
					delete(conns, conn)
					conns_mu.Unlock()
				}()
				handle_conn(conn)
			})
		}
	}()

	<-ctx.Done()

	log.Println("Сервер завершает работу")
	// закрытие листенера остановит Accept, закрытие активных соединений
	// прервёт их обработку
	err := l.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		log.Println("не удалось корректно завершить работу сервера:", err)
	}
	conns_mu.Lock()
	for conn := range conns {
		conn.Close()
	}
	conns_mu.Unlock()
}

// 1. и 3. TCP сервер
func tcp_serv() {
	fmt.Println("Задания 1. и 3.")

	// контекст для прерывания выполнения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// группа ожидания, чтобы можно было дождаться окончания всех подключений
	var wg sync.WaitGroup

	// для обработки завершения сервера (при ctrl+c или подобном)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	addr := utils.ReadLine("Адрес для сервера, включая IP и порт (например 127.0.0.1:8080): ")
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicln("не удалось запустить сервер:", err)
	}
	wg.Go(func() { Serve(ctx, listener, &wg) })

	<-sig
	log.Println("Получен сигнал о завершении сервера")
	cancel()

	wg.Wait()

	fmt.Println()
}

// 2. TCP клиент
func tcp_client() {
	fmt.Println("Задание 2.")

	addr := utils.ReadLine("Адрес для подключения, включая IP и порт (например 127.0.0.1:8080): ")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Panicln("не удалось подключиться к серверу:", err)
	}

	message := utils.ReadLine("Введите сообщение для отправки\n>> ")
	sent, err := conn.Write([]byte(message))
	if err != nil {
		log.Panicln("не удалось отправить сообщение:", err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Panicln("не удалось получить ответ:", err)
	}
	var read int
	parsed, err := fmt.Sscanf(response, "принято байт: %d\n", &read)
	if parsed != 1 || err != nil {
		log.Panicln("не удалось обработать ответ сервера:", err)
	}

	if read != sent {
		log.Printf(
			"Сервер доложил получение другого количества байт (отправлено клиентом %d, получено сервером %d)\n",
			sent,
			read,
		)
	} else {
		log.Println("Сервер корректно принял сообщение")
	}

	log.Println("Завершаем работу клиента")
	err = conn.Close()
	if err != nil {
		log.Panicln("не удалось корректно завершить работу клиента:", err)
	}

	fmt.Println()
}

func home_handler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

var chat_page = httpserv.ServeFile("chat.html")

func chat_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	chat_page(w, r)
}

func get_hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<html><h2>привет</h2>
<img src="http://loverust.space:8000/fish-spinning.gif"/></html>`))
}

func post_data(w http.ResponseWriter, r *http.Request) {
	var json_data json.RawMessage
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&json_data)
	if err != nil {
		log.Printf("[ERR] не удалось считать json: %v\n", err)
		http.Error(w, "failed to read json", http.StatusBadRequest)
		return
	}
	// RawMessage является срезом байт, поэтому печатается как строка
	log.Println(string(json_data))
	w.Write([]byte("accepted."))
}

type ClientId int

type Client struct {
	name string
	tx   chan Message
}

type Message struct {
	author, msg string
	cid         ClientId
	time        time.Time
}

// Размер буфера канала клиента. Медленные клиенты будут пропускать
// сообщения, если буфер переполнен.
const client_tx_buffer = 64

type WsServer struct {
	next_id ClientId
	msgs    []Message
	// к клиентам
	clients map[ClientId]Client
	mu      sync.Mutex
	// от клиентов
	rx chan Message
}

// Пытается отправить сообщение клиенту без блокировки. Если буфер канала
// переполнен (клиент не успевает читать), сообщение пропускается.
func try_send(tx chan Message, msg Message) {
	select {
	case tx <- msg:
	default:
	}
}

func (ws *WsServer) handle_incoming() {
	for msg := range ws.rx {
		ws.mu.Lock()
		ws.msgs = append(ws.msgs, msg)
		// каналы собираются под мьютексом, а отправка происходит после
		// его отпускания, чтобы медленный клиент не блокировал всю
		// рассылку и операции с клиентами
		chans := make([]chan Message, 0, len(ws.clients))
		for _, client := range ws.clients {
			chans = append(chans, client.tx)
		}
		ws.mu.Unlock()
		for _, ch := range chans {
			try_send(ch, msg)
		}
	}
}

func (ws *WsServer) add_client(name string) (ClientId, chan Message) {
	ch := make(chan Message, client_tx_buffer)
	ws.mu.Lock()
	id := ws.next_id
	ws.next_id++
	ws.clients[id] = Client{
		name: name,
		tx:   ch,
	}
	msgs_snapshot := make([]Message, len(ws.msgs))
	copy(msgs_snapshot, ws.msgs)
	ws.mu.Unlock()

	// история отправляется синхронно, поэтому порядок сообщений
	// относительно новых не перемешается
	for _, msg := range msgs_snapshot {
		ch <- msg
	}

	return id, ch
}

func (ws *WsServer) remove_client(id ClientId) {
	ws.mu.Lock()
	if client, ok := ws.clients[id]; ok {
		close(client.tx)
		delete(ws.clients, id)
	}
	ws.mu.Unlock()
}

func init_websock_chat() func(w http.ResponseWriter, r *http.Request) {
	st := new(WsServer{
		next_id: 0,
		msgs:    []Message{},
		clients: map[ClientId]Client{},
		mu:      sync.Mutex{},
		rx:      make(chan Message),
	})

	go st.handle_incoming()

	// websocket.Handler требует заголовок Origin и отдаёт 403 без него,
	// что блокирует CLI-клиенты; поэтому используется Server с кастомным
	// хендшейком, который проверяет Origin только если он указан
	serv := websocket.Server{
		Handshake: func(config *websocket.Config, req *http.Request) error {
			// допустим корректный Origin или его отсутствие (CLI-клиенты
			// его не шлют), Config.Origin используется в RemoteAddr, поэтому
			// его нужно выставить
			origin, err := url.ParseRequestURI(req.Header.Get("Origin"))
			if err != nil {
				config.Origin = &url.URL{}
			} else {
				config.Origin = origin
			}
			return nil
		},
		Handler: websocket.Handler(func(c *websocket.Conn) {
			addr := c.RemoteAddr().String()
			defer c.Close()
			_, err := c.Write([]byte("Отправьте никнейм первым сообщением (максимум 64 байта)"))
			if err != nil {
				log.Println("[ERR] не удалось отправить сообщение по вебсокетам:", err)
				return
			}
			var name string
			if err := websocket.Message.Receive(c, &name); err != nil {
				log.Println("[ERR] не удалось получить сообщение по вебсокетам:", err)
				return
			}
			name = strings.TrimSpace(name)
			if name == "" {
				websocket.Message.Send(c, "Пустой никнейм недопустим")
				return
			}
			if len(name) > 64 {
				websocket.Message.Send(c, "Никнейм слишком длинный (максимум 64 байта)")
				return
			}
			id, rx := st.add_client(name)
			defer st.remove_client(id)
			tx := st.rx
			logerr := func(msg string, e error) {
				if e == nil {
					return
				}
				log.Printf("[ERR] от %s '%s' #%d: %s: %v\n", addr, name, id, msg, e)
			}

			go func(rx chan Message) {
				for msg := range rx {
					_, err := fmt.Fprintf(
						c,
						"[%s] %s: %s\n",
						msg.time.Local().Format(time.RFC822),
						msg.author,
						msg.msg,
					)
					if err != nil {
						logerr("не удалось отправить сообщение", err)
						c.Close()
						return
					}
				}
			}(rx)

			for {
				var message_text string
				// websocket.Message.Receive читает одно целое сообщение (фрейм)
				if err := websocket.Message.Receive(c, &message_text); err != nil {
					logerr("соединение закрыто", err)
					return
				}

				message := Message{
					author: name,
					msg:    message_text,
					cid:    id,
					time:   time.Now(),
				}

				tx <- message
			}
		}),
	}
	return serv.ServeHTTP
}

// 4. 5. 6. HTTP сервер (опционально с чатом на вебсокетах)
func http_serv(enable_chat bool) {
	fmt.Println("Задания 4, 5 и 6")

	server := httpserv.New()

	server.Routes = httpserv.Routes{
		"/":          home_handler,
		"GET /hello": get_hello,
		"POST /data": post_data,
	}

	if enable_chat {
		log.Println("Чат на основе веб-сокетов включён")
		server.Routes["GET /wschat"] = init_websock_chat()
		// страница-клиент для чата
		server.Routes["GET /chat"] = chat_handler
	}

	server.Middleware = func(r *http.Request, accept chan bool, status chan int) {
		start := time.Now()
		accept <- true
		code := <-status
		took := time.Since(start)
		log.Printf(
			"%s запрос от %s на URI %s; заняло %dµs -> %d %s",
			r.Method,
			r.RemoteAddr,
			r.URL.String(),
			took.Microseconds(),
			code,
			http.StatusText(code),
		)
	}
	log.Println("Добавлен middleware для логгирования запросов")

	log.Printf("HTTP-сервер запускается на http://%s:%d\n", server.Ip, server.Port)
	server.Run()
	log.Println("HTTP-сервер завершил работу")

	fmt.Println()
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	args := os.Args[1:]
	if len(args) != 1 {
		log.Fatal("Нужно указать 1 аргумент")
	}
	switch args[0] {
	case "tcp-serve":
		tcp_serv()
	case "tcp-client":
		tcp_client()
	case "http-serve":
		http_serv(false)
	case "sock-chat":
		http_serv(true)
	default:
		log.Fatal("В качестве аргумента нужно указать `tcp-serve`, `tcp-client`, `http-serve` или `sock-chat`")
	}
}
