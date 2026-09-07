package main

import (
	"bufio"
	"fmt"
	"os"
)

// Считывает одно целое число из стандартного ввода
func read_single_int(prompt string) int {
	fmt.Print(prompt)

	var res int
	n, err := fmt.Scanln(&res)

	if n != 1 {
		panic(fmt.Sprintf("не удалось прочитать ввод: %v", err))
	}

	return res
}

// 1. Чётное или нечётное
func even_odd() {
	fmt.Println("Задание 1.")

	num := read_single_int("Введите одно целое число: ")
	s := ""
	if num%2 == 0 {
		s = "чётное"
	} else {
		s = "нечётное"
	}

	fmt.Println("Число", num, s)
	fmt.Println()
}

// 2. Положительное, отрицательное или ноль
func sign_of_number(num int) {
	s := ""
	if num < 0 {
		s = "отрицательное"
	} else if num > 0 {
		s = "положительное"
	} else {
		s = "это ноль"
	}

	fmt.Println("Число", num, s)
}

// 3. От 1 до 10 с помощью for
func count_to_10() {
	fmt.Println("Задание 3.")

	for i := 1; i <= 10; i++ {
		fmt.Println(i)
	}

	fmt.Println()
}

// 4. Длина строки
func strlen(s string) int {
	// простой вариант:
	// return len(s)

	len := 0
	for range s {
		len++
	}
	return len
}

// Прямоугольник
type Rectangle struct {
	length, width int
}

// Вычисляет площадь прямоугольника
func (rect Rectangle) Area() int {
	return rect.length * rect.width
}

// 5. Структура и вызов метода площади
func task5() {
	fmt.Println("Задание 5.")

	length := read_single_int("Длина прямоугольника: ")
	if length <= 0 {
		panic("Длина должна быть положительной")
	}
	width := read_single_int("Ширина прямоугольника: ")
	if width <= 0 {
		panic("Ширина должна быть положительной")
	}
	rect := Rectangle{length, width}
	fmt.Println("Площадь равна", rect.Area())

	fmt.Println()
}

// 6. Среднее значение из двух чисел
func avg_two(a int, b int) float64 {
	return float64(a+b) / 2.0
}

func main() {
	even_odd()

	fmt.Println("Задание 2.")
	num := read_single_int("Введите одно целое число: ")
	sign_of_number(num)
	fmt.Println()

	count_to_10()

	fmt.Println("Задание 4.")
	fmt.Print("Введите любую строку\n>> ")
	// нужен такой способ вместо обычного fmt.Scanln чтобы прочитать строку с пробелами
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil {
		panic(fmt.Sprintf("не удалось прочитать строку: %v", err))
	}
	// избавляемся от символа новой строки
	s = s[:len(s)-1]
	fmt.Printf("Длина строки '%s' равна %d\n", s, strlen(s))
	fmt.Println()

	task5()

	fmt.Println("Задание 6.")
	first := read_single_int("Первое число для расчёта среднего: ")
	second := read_single_int("Второе число для расчёта среднего: ")
	avg := avg_two(first, second)
	fmt.Println("Среднее равно", avg)
	fmt.Println()
}
