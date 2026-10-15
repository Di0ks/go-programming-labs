package tcptls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
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

func serve(ctx context.Context, l net.Listener, wg *sync.WaitGroup) {
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

func StartServer(addr, certPath, keyPath string) {
	// контекст для прерывания выполнения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// группа ожидания, чтобы можно было дождаться окончания всех подключений
	var wg sync.WaitGroup

	// для обработки завершения сервера (при ctrl+c или подобном)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Panicln("не удалось загрузить сертификат сервера:", err)
	}
	clientCAs := x509.NewCertPool()
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		log.Panicln("не удалось прочитать сертификат клиентов:", err)
	}
	if !clientCAs.AppendCertsFromPEM(certPEM) {
		log.Panicln("не удалось добавить сертификат клиента в список доверенных")
	}

	listener, err := tls.Listen("tcp", addr, &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
	})
	if err != nil {
		log.Panicln("не удалось запустить сервер:", err)
	}
	wg.Go(func() { serve(ctx, listener, &wg) })

	<-sig
	log.Println("Получен сигнал о завершении сервера")
	cancel()

	wg.Wait()
}
