package main

import (
	"fmt"
	"slices"

	"github.com/Di0ks/go-programming-labs/lab3/mathutils"
	"github.com/Di0ks/go-programming-labs/lab3/stringutils"
	"github.com/Di0ks/go-programming-labs/utils"
)

// 1. и 2. Вычисления факториала числа через функцию пакета
func factorial() {
	fmt.Println("Задания 1. и 2.")

	num := utils.ReadSingleInt("Введите число для расчёта факториала: ")

	fmt.Printf("%d! = %d\n", num, mathutils.Factorial(num))

	fmt.Println()
}

// 3. Разворот строки
func reverse() {
	fmt.Println("Задание 3.")

	s := utils.ReadLine("Строка для разворота\n>> ")
	fmt.Println(stringutils.Reverse(s))

	fmt.Println()
}

// 4. Массив из 5 целых чисел
func array_of_5() []int {
	fmt.Println("Задание 4.")

	const LEN int = 5
	var nums [LEN]int

	fmt.Print()
	nums_read := utils.ReadInts(fmt.Sprintf("Введите %d чисел для массива (через пробел): ", LEN))

	if len(nums_read) != LEN {
		panic("Неверное количество чисел")
	}
	copy(nums[:], nums_read)

	fmt.Println("Массив:", nums)

	fmt.Println()

	return nums[:]
}

// 5. Операции добавления и удаления элементов в срезе
func modify_slice(s []int) {
	fmt.Println("Задание 5.")

	// добавление
	len := len(s)
	insert_i := utils.ReadSingleInt(fmt.Sprintf("Введите индекс для добавления элемента в срез (максимум %d): ", len))
	if insert_i > len {
		panic(fmt.Sprintf("Индекс должен быть меньше длины либо равен ей (%d)", len))
	}
	val := utils.ReadSingleInt("Какой элемент добавить (введите значение): ")

	s = slices.Insert(s, insert_i, val)
	len++
	fmt.Println("Добавление выполнено успешно:", s)

	// удаление
	remove_i := utils.ReadSingleInt(fmt.Sprintf("Выберите индекс для удаления элемента из среза (максимум %d): ", len-1))
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

	length := utils.ReadSingleInt("Сколько строк создать для сравнения: ")
	if length <= 0 {
		panic("Количество строк должно быть положительное")
	}

	strings := make([]string, length)
	fmt.Println("Вводите строки поочерёдно")
	for i := range strings {
		strings[i] = utils.ReadLine(fmt.Sprintf("#%d>> ", i))
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
