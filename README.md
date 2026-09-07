# Лабораторные работы по прикладному программирования на Go

В этом репозитории представлены лабораторные работы, каждая в соответствующей папке.

Информацию о каждой лабораторной работе можно посмотреть в `README.md` соответствущей лабораторной.

## Запуск

Запускать лабораторные можно двумя способами — из склонированного репозитория или установив через `go install`.

Из репозитория:

```bash
git clone https://github.com/Di0ks/go-programming-labs
cd go-programming-labs/lab1
go run .
```

С помощью уставощика:

```bash
mkdir labs
cd labs
GOBIN=$(pwd) go install github.com/Di0ks/go-programming-labs/lab1@latest
./lab1
```
