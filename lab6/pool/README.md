# Пул воркеров

Данный пакет предоставляет реализацию пула воркеров, позволяющего равномерно разделить крупную работу на несколько потоков.

## Публичные идентификаторы
### `JobFunc[Arg, Out any]`

Функция передаваемая воркерам для выполнения. Параметризуется по `Arg` и `Out`. `Arg` — тип аргумента, `Out` — тип возвращаемого значения функции.

### `func Spawn[Arg, Out any](worker_count int, data []Arg,	fn JobFunc[Arg, Out], trace bool) chan Out`

Запускает пул воркеров с указанными параметрами (смотирте документацию функции и параметров для подробностей).

## Пример

Поиск хэша в несколько потоков:

```go
package main

import (
    "fmt"
	"crypto/sha256"
	"encoding/hex"
	"encoding/binary"
)

func main() {
	const message string = "some message text to hash"
	const iters int = 1000000
	salts := make([]int, 0, iters)
	for i := range iters {
		salts = append(salts, i)
	}

	type Result struct {
		// равен `nil` если хэш не соответствует условию
		hash []byte
		salt int
	}

	fn := func(salt int) Result {
		hasher := sha256.New()
		hasher.Write([]byte(message))
		var salt_bytes [4]byte
		binary.LittleEndian.PutUint32(salt_bytes[:], uint32(salt))
		hasher.Write(salt_bytes[:])
		hash := hasher.Sum(nil)
		// нужен хэш с первыми 6 ноликами (это 3 нулевых байта)
		for i := range 3 {
			if hash[i] != 0 {
				return Result{nil, 0}
			}
		}
		return Result{hash, salt}
	}

	for res := range pool.Spawn(16, salts, fn, false) {
		if res.hash != nil {
			fmt.Printf("Найден хэш: %d -> %s\n", res.salt, hex.EncodeToString(res.hash))
			break
		}
	}
}
```

Результат:
```
2026/09/10 15:39:47 Создаем 16 воркеров...
Найден хэш: 497787 -> 000000c8119bc7e1231ce2ede35546dd745d8d40aaa571a3c96f4efd3a39ff92
```