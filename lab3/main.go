package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"

	"github.com/Di0ks/go-programming-labs/lab3/mathutils"
	"github.com/Di0ks/go-programming-labs/lab3/stringutils"
)

// Считывает строку со стандартного ввода до символа новой строки (символ новой
// строки не сохраняется)
func read_string() string {
	// нужен такой способ вместо обычного fmt.Scanln чтобы прочитать строку с пробелами
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil {
		panic(fmt.Sprintf("не удалось прочитать строку: %v", err))
	}
	// избавляемся от символа новой строки
	s = s[:len(s)-1]
	return s
}

// 1. и 2. Вычисления факториала числа через функцию пакета
func factorial() {
	fmt.Println("Задания 1. и 2.")

	var num int
	fmt.Print("Введите число для расчёта факториала: ")
	fmt.Scanln(&num)

	fmt.Printf("%d! = %d\n", num, mathutils.Factorial(num))

	fmt.Println()
}

// 3. Разворот строки
func reverse() {
	fmt.Println("Задание 3.")

	fmt.Print("Строка для разворота\n>> ")

	s := read_string()
	fmt.Println(stringutils.Reverse(s))

	fmt.Println()
}

// 4. Массив из 5 целых чисел
func array_of_5() []int {
	fmt.Println("Задание 4.")

	nums := [5]int{}

	fmt.Print("Введите 5 чисел для массива (через пробел): ")
	n, err := fmt.Scanln(
		&nums[0],
		&nums[1],
		&nums[2],
		&nums[3],
		&nums[4],
	)

	if n != 5 {
		panic(fmt.Sprintf("неверный ввод: %v", err))
	}

	fmt.Println("Массив:", nums)

	fmt.Println()

	return nums[:]
}

// 5. Операции добавления и удаления элементов в срезе
func modify_slice(s []int) {
	fmt.Println("Задание 5.")

	// добавление
	len := len(s)
	fmt.Printf("Введите индекс для добавления элемента в срез (максимум %d): ", len)
	var insert_i int
	n, err := fmt.Scanln(&insert_i)
	if n != 1 {
		panic(fmt.Sprintf("неверный ввод: %v", err))
	}
	if insert_i > len {
		panic(fmt.Sprintf("Индекс должен быть меньше длины либо равен ей (%d)", len))
	}
	fmt.Print("Какой элемент добавить (введите значение): ")
	var val int
	n, err = fmt.Scanln(&val)
	if n != 1 {
		panic(fmt.Sprintf("неверный ввод: %v", err))
	}

	s = slices.Insert(s, insert_i, val)
	len++
	fmt.Println("Добавление выполнено успешно:", s)

	// удаление
	fmt.Printf("Выберите индекс для удаления элемента из среза (максимум %d): ", len-1)
	var remove_i int
	n, err = fmt.Scanln(&remove_i)
	if n != 1 {
		panic(fmt.Sprintf("неверный ввод: %v", err))
	}
	if remove_i >= len {
		panic(fmt.Sprintf("Индекс должен быть строго меньше длины (%d)", len))
	}
	s = slices.Delete(s, remove_i, remove_i+1)
	fmt.Println("Удаление выполнено успешно:", s)

	fmt.Println()
}

// 6. Нахождение самой длинной строки
func longest_string() {
	fmt.Println("Задание 6.")

	fmt.Printf("Сколько строк создать для сравнения: ")
	var length int
	n, err := fmt.Scanln(&length)
	if n != 1 {
		panic(fmt.Sprintf("неверный ввод: %v", err))
	}
	if length <= 0 {
		panic("Количество строк должно быть положительное")
	}

	strings := make([]string, length)
	fmt.Println("Вводите строки поочерёдно")
	for i := range strings {
		fmt.Printf("#%d>> ", i)
		strings[i] = read_string()
	}

	max_len := 0
	max_i := 0
	for i, s := range strings {
		len := len(s)
		if len > max_len {
			max_len = len
			max_i = i
		}
	}
	fmt.Printf("Самая длинная строка (%d байта): %s\n", max_len, strings[max_i])

	fmt.Println()
}

func main() {
	factorial()
	reverse()
	slice := array_of_5()
	modify_slice(slice)
	longest_string()
}
