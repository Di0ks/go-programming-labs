package main

import (
	"crypto"
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"hash"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Di0ks/go-programming-labs/lab10/restapi"
	"github.com/Di0ks/go-programming-labs/lab10/tcptls"
	tcptlsclient "github.com/Di0ks/go-programming-labs/lab10/tcptls-client"
	"github.com/Di0ks/go-programming-labs/utils"
)

func hashing() {
	fmt.Println("Хэш-утилита\nhelp для вывод доступных команд")
outer:
	for {
		input := strings.TrimSpace(utils.ReadLine("\n>> "))
		if len(input) == 0 {
			continue
		}
		args := strings.Fields(input)
		switch args[0] {
		case "help":
			fmt.Println(`Хэш-утилита
Доступные команды:
    exit - завершение работы
    help - показывает это сообщение
    hash <algo> <text> - хэшировать <text> использую алгоритмы <algo>; доступные алгоритмы:
        md5
        sha-256
        sha-512
    check <text> <hash> - проверка целостности строки <text> по её хэшу <hash>`)
		case "hash":
			if len(args) != 3 {
				fmt.Println("неверное использование hash, ожидалось: hash <algo> <text>")
			}

			algo := args[1]
			text := args[2]
			var hasher hash.Hash
			switch algo {
			case "md5":
				hasher = md5.New()
			case "sha-256":
				hasher = sha256.New()
			case "sha-512":
				hasher = sha512.New()
			default:
				fmt.Println("Неизвестный алгоритм хэширования")
				continue
			}

			hasher.Write([]byte(text))
			hash := hasher.Sum(nil)
			fmt.Printf("%x\n", hash)
		case "check":
			if len(args) != 3 {
				fmt.Println("неверное использование check, ожидалось: check <text> <hash>")
			}

			text := args[1]
			hex_hash := args[2]

			raw_hash, err := hex.DecodeString(hex_hash)
			if err != nil {
				fmt.Println("Не удалось прочитать хэш:", err)
				continue
			}

			var hasher hash.Hash
			switch len(raw_hash) {
			case 16:
				hasher = md5.New()
			case 32:
				hasher = sha256.New()
			case 64:
				hasher = sha512.New()
			default:
				fmt.Printf(
					"Некорректная длина хэша %d. Допустимые размеры: 16 для md5, 32 для sha-256, 64 для sha-512\n",
					len(raw_hash),
				)
				continue
			}
			hasher.Write([]byte(text))
			hashed := hasher.Sum(nil)

			if slices.Equal(hashed, raw_hash) {
				fmt.Println("Проверка пройдена")
			} else {
				fmt.Println("Хэш не соответствует тексту!")
			}
		case "exit":
			break outer
		default:
			fmt.Println("Неизвестная команда, введите help")
		}
	}
}

func sym_enc() {
	fmt.Println("AES шифрование\nhelp для вывод доступных команд")
outer:
	for {
		input := strings.TrimSpace(utils.ReadLine("\n>> "))
		if len(input) == 0 {
			continue
		}
		args := strings.Fields(input)
		switch args[0] {
		case "help":
			fmt.Println(`AES шифрование
Доступные команды:
    exit - завершение работы
    help - показывает это сообщение
    enc <text> <key> - зашифровать сообщение <text> ключом <key>
    dec <ctx> <key> - расшифровать <ctx> ключом <key>`)
		case "enc":
			if len(args) != 3 {
				fmt.Println("неверное использование enc, ожидалось: enc <text> <key>")
			}

			text := []byte(args[1])
			key := sha256.Sum256([]byte(args[2]))
			if len(text)%aes.BlockSize != 0 {
				pad := aes.BlockSize - (len(text) % aes.BlockSize)
				padded := make([]byte, len(text)+pad)
				copy(padded, text)
				text = padded
			}

			enc, _ := aes.NewCipher(key[:])
			enc.Encrypt(text, text)
			fmt.Println(hex.EncodeToString(text))
		case "dec":
			if len(args) != 3 {
				fmt.Println("неверное использование dec, ожидалось: dec <ctx> <key>")
			}

			ctx, err := hex.DecodeString(args[1])
			if err != nil {
				fmt.Println("Некорректный шифротекст:", err)
				continue
			}

			if len(ctx)%aes.BlockSize != 0 {
				fmt.Printf(
					"Длина введённого шифротекста не соответствует размеру блока (%d)\n",
					aes.BlockSize,
				)
				continue
			}
			key := sha256.Sum256([]byte(args[2]))

			enc, _ := aes.NewCipher(key[:])
			enc.Decrypt(ctx, ctx)

			fmt.Println(string(ctx))
		case "exit":
			break outer
		default:
			fmt.Println("Неизвестная команда, введите help")
		}
	}
}

func assym_enc() {
	msgid := 0
	saved_base := "rsa"
	fmt.Println("Асимметричное шифрование RSA\nhelp для вывод доступных команд")
outer:
	for {
		input := strings.TrimSpace(utils.ReadLine(fmt.Sprintf("\n%s>> ", saved_base)))
		if len(input) == 0 {
			continue
		}
		args := strings.Fields(input)
		switch args[0] {
		case "help":
			fmt.Println(`Асимметричное шифрование RSA
Доступные команды:
    exit - завершение работы
    help - показывает это сообщение
    switch <filename> - переключает на пару ключей <filename>.pub и <filename>.key
    save <filename> - генерирует пару ключей и сохраняет в файлы <filename>.pub и <filename>.key
    sign <msg> - подписывает сообщение приватным ключом из файла
    check <msg> <sig> - проверка cообщения <msg> подписью <sig> и публичным ключом из файла
    check <msgid> - проверка сообщения и подписи из файлов msg<msgid>.txt и msg<msgid>.sig`)
		case "switch":
			if len(args) != 2 {
				fmt.Println("неверное использование switch, ожидалось: switch <filename>")
				continue
			}
			saved_base = args[1]
		case "save":
			if len(args) != 2 {
				fmt.Println("неверное использование save, ожидалось: save <filename>")
				continue
			}

			base := args[1]

			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				fmt.Println("Не удалось сгенерировать пару ключей:", err)
			}
			pub := key.Public()

			keyPEM := pem.EncodeToMemory(
				&pem.Block{
					Type:  "RSA PRIVATE KEY",
					Bytes: x509.MarshalPKCS1PrivateKey(key),
				},
			)

			pubPEM := pem.EncodeToMemory(
				&pem.Block{
					Type:  "RSA PUBLIC KEY",
					Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
				},
			)

			if err := os.WriteFile(base+".key", keyPEM, 0600); err != nil {
				fmt.Println("Не удалось сохранить приватный ключ:", err)
				continue
			}
			fmt.Printf("Приватный ключ сохранён в %s.key\n", base)

			if err := os.WriteFile(base+".pub", pubPEM, 0655); err != nil {
				fmt.Println("Не удалось сохранить публичный ключ:", err)
				continue
			}
			fmt.Printf("Публичный ключ сохранён в %s.pub\n", base)
			saved_base = base
		case "sign":
			if len(args) != 2 {
				fmt.Println("неверное использование sign, ожидалось: sign <msg>")
				continue
			}

			msg := []byte(args[1])

			keyPEM, err := os.ReadFile(saved_base + ".key")
			if err != nil {
				fmt.Println("Не удалось прочитать файл приватного ключа:", err)
				continue
			}

			block, _ := pem.Decode(keyPEM)
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				fmt.Println("Не удалось прочитать приватный ключ RSA:", err)
				continue
			}
			hashed := sha256.Sum256(msg)
			sig, err := rsa.SignPKCS1v15(
				rand.Reader,
				key,
				crypto.SHA256,
				hashed[:],
			)
			if err != nil {
				fmt.Println("Не удалось подписать сообщение:", err)
				continue
			}

			fmt.Printf("Подпись:\n%x\n", sig)
			if err := os.WriteFile("msg"+fmt.Sprint(msgid)+".txt", msg, 0655); err != nil {
				fmt.Println("Не удалось сохранить сообщение:", err)
				continue
			}
			if err := os.WriteFile("msg"+fmt.Sprint(msgid)+".sig", sig, 0655); err != nil {
				fmt.Println("Не удалось сохранить подпись:", err)
				continue
			}
			fmt.Printf(
				"Сообщение и подпись сохранены в файлы msg%d.txt и msg%d.sig соответственно",
				msgid,
				msgid,
			)
			msgid++
		case "check":
			var msg []byte
			var sig []byte
			switch len(args) {
			case 2:
				msgid, err := strconv.Atoi(args[1])
				if err != nil {
					fmt.Println("Не удалось прочитать msgid:", err)
					continue
				}

				msg, err = os.ReadFile("msg" + fmt.Sprint(msgid) + ".txt")
				if err != nil {
					fmt.Println("Не удалось прочитать сообщение:", err)
					continue
				}

				sig, err = os.ReadFile("msg" + fmt.Sprint(msgid) + ".sig")
				if err != nil {
					fmt.Println("Не удалось прочитать подпись:", err)
					continue
				}
			case 3:
				msg = []byte(args[1])
				var err error
				sig, err = hex.DecodeString(args[2])
				if err != nil {
					fmt.Println("Не удалось прочитать подпись:", err)
					continue
				}
			default:
				fmt.Println("неверное использование check, ожидалось: check <msg> <sig> или check <msgid>")
				continue
			}

			pubPEM, err := os.ReadFile(saved_base + ".pub")
			if err != nil {
				fmt.Println("Не удалось прочитать файл публичного ключа:", err)
				continue
			}

			block, _ := pem.Decode(pubPEM)
			key, err := x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				fmt.Println("Не удалось прочитать публичный ключ RSA:", err)
				continue
			}
			hashed := sha256.Sum256(msg)
			err = rsa.VerifyPKCS1v15(
				key,
				crypto.SHA256,
				hashed[:],
				sig,
			)
			if err != nil {
				fmt.Println("Проверка не пройдена:", err)
			} else {
				fmt.Println("Проверка пройдена")
			}
		case "exit":
			break outer
		default:
			fmt.Println("Неизвестная команда, введите help")
		}
	}
}

func tcp_tls() {
	ans := utils.ReadSingleInt("Запустить сервер или клиент?\n1. Сервер\n2. Клиент\n>> ")
	switch ans {
	case 1:
		addr := utils.ReadLine("Адрес для сервера, включая IP и порт (например 127.0.0.1:8080): ")
		tcptls.StartServer(addr, "certs/certificate.pem", "certs/privatekey.pem")
	case 2:
		addr := utils.ReadLine("Адрес для подключения, включая IP и порт (например 127.0.0.1:8080): ")
		tcptlsclient.RunClient(addr, "certs/certificate.pem", "certs/privatekey.pem")
	}
}

func restapi_with_auth() {
	cfg := restapi.NewDbConfig()
	restapi.RunApi("127.0.0.1", 8080, &cfg)
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	fmt.Println(`1 -> хэширование
2 -> симметричное шифрование
3 -> асимметричное шифрование
4 -> TCP TLS
5 -> REST API с аутентификацией`)
	choice := utils.ReadSingleInt("Выбор:\n> ")
	switch choice {
	case 1:
		hashing()
	case 2:
		sym_enc()
	case 3:
		assym_enc()
	case 4:
		tcp_tls()
	case 5:
		restapi_with_auth()
	default:
		fmt.Println("Неверный выбор")
	}
	fmt.Println("Завершаем работу")
}
