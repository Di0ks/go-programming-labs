package main

import (
	"cmp"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Di0ks/go-programming-labs/lab3/mathutils"
	"github.com/Di0ks/go-programming-labs/lab6/pool"
	"github.com/Di0ks/go-programming-labs/utils"
)

// Обертка вокруг функции факториала для демонстрации параллельного исполнения
func factorial_wrap(iters int, sleep time.Duration) {
	fmt.Printf(
		"Запущены факториалы чисел на %d итерацией с задержкой %d миллисекунд\n",
		iters,
		sleep.Milliseconds(),
	)
	for i := range iters {
		num := mathutils.Factorial(i)
		fmt.Printf("#%d: Факториал числа %d\n", i, num)
		time.Sleep(sleep)
	}
}

// Обертка вокруг функции нахождения простого числа для демонстрации параллельного исполнения
func prime_wrap(iters int, sleep time.Duration) {
	fmt.Printf(
		"Запущены простые числа на %d итерацией с задержкой %d миллисекунд\n",
		iters,
		sleep.Milliseconds(),
	)
	for i := range iters {
		num := mathutils.Prime(i)
		fmt.Printf("#%d: Простое число %d\n", i, num)
		time.Sleep(sleep)
	}
}

// Обертка вокруг функции получения случайного числа для демонстрации параллельного исполнения
func rand_wrap(iters int, sleep time.Duration) {
	fmt.Printf(
		"Запущены случайные числа на %d итерацией с задержкой %d миллисекунд\n",
		iters,
		sleep.Milliseconds(),
	)
	for i := range iters {
		num := rand.Intn(1000)
		fmt.Printf("#%d: Случайное число %d\n", i, num)
		time.Sleep(sleep)
	}
}

// 1. Параллельное выполнение трёх горутин
func par_3() {
	fmt.Println("Задание 1.")

	iters := 10
	long_sleep := 500 * time.Millisecond
	medium_sleep := 300 * time.Millisecond
	short_sleep := 100 * time.Millisecond

	go rand_wrap(iters, long_sleep)
	go prime_wrap(iters, medium_sleep)
	go factorial_wrap(iters, short_sleep)

	time.Sleep(time.Duration(iters) * long_sleep)

	fmt.Println()
}

// 2. Каналы для передачи данных
func channel_usage() {
	fmt.Println("Задание 2.")

	fib_gen := func(out chan int) {
		for i := range 10 {
			out <- mathutils.Fibonacci(i)
		}
		close(out)
	}

	fib_print := func(in chan int, exit chan bool) {
		fmt.Print("Числа Фибоначчи: ")
		for num := range in {
			fmt.Printf("%d ", num)
		}
		fmt.Println()
		exit <- true
		close(exit)
	}

	nums := make(chan int)
	exit := make(chan bool)
	go fib_gen(nums)
	go fib_print(nums, exit)

	_, ok := <-exit
	if !ok {
		panic("Канал закрыт до получения флага о выходе")
	}

	fmt.Println()
}

// 3. Использование `select`
func select_usage() {
	fmt.Println("Задание 3.")

	rand_gen := func(out chan int, next_allowed chan struct{}) {
		for range next_allowed {
			out <- rand.Intn(1000)
		}
	}

	is_even := func(in chan int, out chan bool) {
		for num := range in {
			out <- num%2 == 0
		}
	}

	// ручная синхронизация, так как `select` выбирает случайным
	// образом и может 2 раза выбрать канал `rand_nums`, который
	// генерирует числа неограниченно, забивая `reqs_even`
	next_allowed := make(chan struct{})

	rand_nums := make(chan int)
	reqs_even := make(chan int)
	results_even := make(chan bool)
	go rand_gen(rand_nums, next_allowed)
	go is_even(reqs_even, results_even)
	next_allowed <- struct{}{}
	for range 10 {
		select {
		case even := <-results_even:
			var res string
			if even {
				res = "чётное"
			} else {
				res = "нечётное"
			}
			fmt.Println(res)
			next_allowed <- struct{}{}
		case num := <-rand_nums:
			fmt.Printf("Получили число %d, отправляем на проверку... ", num)
			reqs_even <- num
		}
	}

	fmt.Println()
}

// 4. Синхронизация с помощью мьютексов
func mutex_sync() {
	fmt.Println("Задание 4.")

	shared_cnt := struct {
		n  int
		mu sync.Mutex
	}{
		n: 0,
	}

	finish := make(chan struct{})

	increment := func(id int, iter int) {
		for range iter {
			shared_cnt.mu.Lock()
			new_value := shared_cnt.n + 1
			fmt.Printf("Горутина #%d: %d + 1 = %d\n", id, shared_cnt.n, new_value)
			shared_cnt.n = new_value
			shared_cnt.mu.Unlock()
		}
		finish <- struct{}{}
	}
	increment_no_sync := func(id int, iter int) {
		for range iter {
			new_value := shared_cnt.n + 1
			fmt.Printf("Горутина #%d: %d + 1 = %d\n", id, shared_cnt.n, new_value)
			shared_cnt.n = new_value
		}
		finish <- struct{}{}
	}

	const iter = 3
	const go_cnt = 3
	const target = iter * go_cnt
	fmt.Println("Запускаются синхронизированные счётчики...")
	for i := range go_cnt {
		go increment(i, iter)
	}

	for range go_cnt {
		<-finish
	}

	if shared_cnt.n == target {
		fmt.Printf("Получился верный результат: %d == %d\n", shared_cnt.n, target)
	} else {
		fmt.Printf("Результат неверный! %d != %d\n", shared_cnt.n, target)
	}

	shared_cnt.n = 0

	fmt.Println("Запускаются несинхронизированные счётчики...")
	for i := range go_cnt {
		go increment_no_sync(i, iter)
	}

	for range iter {
		<-finish
	}

	if shared_cnt.n == target {
		fmt.Printf("Получился верный результат: %d == %d\n", shared_cnt.n, target)
	} else {
		fmt.Printf("Результат неверный! %d != %d\n", shared_cnt.n, target)
	}

	fmt.Println()
}

// 5. Многопоточный калькулятор
func concurrent_calc() {
	fmt.Println("Задание 5.")

	type Operation int
	const (
		// Сложение
		OperPlus = iota
		// Вычитание
		OperMinus
		// Умножение
		OperMul
		// Деление
		OperDiv
		// Остаток от деления
		OperRem
	)
	type Request struct {
		oper   Operation
		id     int
		a      int
		b      int
		result chan int
	}
	oper_char := map[Operation]rune{
		OperDiv:   '/',
		OperMinus: '-',
		OperMul:   '*',
		OperPlus:  '+',
		OperRem:   '%',
	}
	server := func(requests chan Request, finished chan struct{}) {
		handlers := struct {
			n int
			// condvar для грациозного завершения сервера
			cond sync.Cond
		}{
			n:    0,
			cond: *sync.NewCond(&sync.Mutex{}),
		}
		for req := range requests {
			fmt.Printf(
				"Получен запрос #%02d: %d %c %d\n",
				req.id,
				req.a,
				oper_char[req.oper],
				req.b,
			)
			// обработчик запросов
			go func(r Request) {
				handlers.cond.L.Lock()
				handlers.n++
				handlers.cond.L.Unlock()

				// случайная задержка для наглядности параллельности
				sleep := rand.Intn(2000) + 500
				time.Sleep(time.Millisecond * time.Duration(sleep))

				var res int
				switch r.oper {
				case OperDiv:
					res = r.a / r.b
				case OperMinus:
					res = r.a - r.b
				case OperMul:
					res = r.a * r.b
				case OperPlus:
					res = r.a + r.b
				case OperRem:
					res = r.a % r.b
				default:
					panic("Неизвестная операция!")
				}
				r.result <- res
				close(r.result)

				// оповещаем о завершении работы
				handlers.cond.L.Lock()
				handlers.n--
				handlers.cond.L.Unlock()
				handlers.cond.Broadcast()
			}(req)
		}

		// ждём пока все обработчики закончат
		handlers.cond.L.Lock()
		for handlers.n > 0 {
			handlers.cond.Wait()
		}

		finished <- struct{}{}
	}

	requests := make(chan Request, 10)
	finished := make(chan struct{})
	go server(requests, finished)

	// генератор случайных запросов
	rand_req := func(id int) (Request, chan int) {
		// [-10000, 10000]
		a := rand.Intn(20001) - 10000
		b := rand.Intn(20001) - 10000

		oper := Operation(rand.Intn(5))
		answer := make(chan int)
		req := Request{oper, id, a, b, answer}
		return req, answer
	}

	for i := range 20 {
		req, ans := rand_req(i)
		requests <- req
		go func() {
			result, ok := <-ans
			if ok {
				fmt.Printf("Запрос #%02d получил ответ: %d\n", i, result)
			} else {
				fmt.Printf("Запрос #%02d не получил ответа:(\n", i)
			}
		}()
	}
	close(requests)

	<-finished

	fmt.Println()
}

// 6. Пул воркеров (реализация в модуле)
func worker_pool() {
	fmt.Println("Задание 6.")

	log_level := utils.ReadSingleInt("Уровень логгирования пула (0=без логов; 1=некотрые логи; 2=подробно): ")
	if log_level < 0 || log_level > 2 {
	}
	var trace bool
	switch log_level {
	case 0:
		trace = false
		log.SetOutput(io.Discard)
	case 1:
		trace = false
		log.SetFlags(log.Ltime | log.Lshortfile)
	case 2:
		trace = true
		log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	default:
		panic("Неверный уровень логгирования, нужно число от 0 до 2 включительно")
	}

	file_path := utils.ReadLine("Путь до файла для реверсирования строк: ")
	data, err := os.ReadFile(file_path)
	if err != nil {
		panic(fmt.Sprintln("Не удалось прочитать файл:", err))
	}

	worker_count := utils.ReadSingleInt("Сколько воркеров запустить: ")
	if worker_count < 1 {
		panic("Должно быть положительно число воркеров")
	}

	var slow_down bool
	switch utils.ReadSingleInt("Имитировать замедление (0=нет; 1=да): ") {
	case 0:
		slow_down = false
	case 1:
		slow_down = true
	default:
		panic("Неверный ответ, нужно ввести 0 или 1")
	}

	// воркеры работают беспорядочно, так что нужен способ восстановить порядок
	// строк после реверсирования
	type Line struct {
		i    int
		line string
	}

	lines := make([]Line, 0, 10)
	for i, l := range strings.Split(string(data), "\n") {
		lines = append(lines, Line{i, l})
	}

	fn := func(l Line) Line {
		if slow_down {
			// случайная задержка для наглядности параллельной работы
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(2000)))
		}

		runes := []rune(l.line)
		slices.Reverse(runes)
		return Line{l.i, string(runes)}
	}

	reversed := make([]Line, 0, len(lines))
	for res := range pool.Spawn(worker_count, lines, fn, trace) {
		reversed = append(reversed, res)
	}

	slices.SortFunc(reversed, func(a Line, b Line) int {
		return cmp.Compare(a.i, b.i)
	})

	fmt.Println("Финальный результат:")
	for _, v := range reversed {
		fmt.Println(v.line)
	}

	fmt.Println()
}

func main() {
	par_3()
	channel_usage()
	select_usage()
	mutex_sync()
	concurrent_calc()
	worker_pool()
}
