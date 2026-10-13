package main

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"slices"
	"strings"

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
    check <text> <hash> - проверка целостности строки <text> по её хэшу <hash>
`)
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

func main() {
	fmt.Println(`1 -> хэширование
2 -> симметричное шифрование
3 -> ассиметричное шифрование
4 -> TCP TLS
5 -> TLS REST API
`)
	choice := utils.ReadSingleInt("Выбор:\n> ")
	switch choice {
	case 1:
		hashing()
	case 2:
		println("todo")
	case 3:
		println("todo")
	case 4:
		println("todo")
	case 5:
		println("todo")
	default:
		fmt.Println("Неверный выбор")
	}
	fmt.Println("Завершаем работу")
}
