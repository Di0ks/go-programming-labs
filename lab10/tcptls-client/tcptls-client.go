package tcptlsclient

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"github.com/Di0ks/go-programming-labs/utils"
)

func RunClient(addr, certPath, keyPath string) {
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Panicln("не удалось загрузить сертификат клиента:", err)
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		log.Panicln("не удалось прочитать сертификат сервера:", err)
	}
	serverCAs := x509.NewCertPool()
	if !serverCAs.AppendCertsFromPEM(certPEM) {
		log.Panicln("не удалось добавить сертификат сервера в список доверенных")
	}
	trustedCertificate, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		log.Panicln("не удалось разобрать сертификат сервера:", err)
	}

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: true,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return fmt.Errorf("сервер не предоставил сертификат")
			}
			if state.PeerCertificates[0].Subject.CommonName != trustedCertificate.Subject.CommonName {
				return fmt.Errorf("сертификат сервера не соответствует доверенному")
			}
			_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
				Roots:     serverCAs,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})
			return err
		},
	})
	if err != nil {
		log.Panicln("не удалось установить защищённое подключение:", err)
	}
	fmt.Printf(
		"Сертификат проверен\nLocality: %s\nCN: %s\nOU: %s\n",
		trustedCertificate.Issuer.Locality[0],
		trustedCertificate.Issuer.CommonName,
		trustedCertificate.Issuer.OrganizationalUnit[0],
	)

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
