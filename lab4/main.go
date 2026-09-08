package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Di0ks/go-programming-labs/utils"
)

// 1. Создание карты людей и их возрастов, добавление нового
func make_map() map[string]int {
	fmt.Println("Задание 1.")

	people := map[string]int{
		"Иван":     21,
		"Игорь":    33,
		"Василиса": 19,
		"Евгения":  44,
	}

	name := utils.ReadLine("Введите имя человека для добавления: ")
	name, _, _ = strings.Cut(name, " ")

	if _, ok := people[name]; ok {
		panic("Человек с таким именем уже существует")
	}

	age := utils.ReadSingleInt("Введите возраст этого человека: ")
	if age <= 0 {
		panic("Возраст должен быть положительным")
	}

	people[name] = age
	fmt.Println("Запись была добавлена успешно")
	fmt.Println("Карта людей:", people)

	fmt.Println()

	return people
}

// 2. Рассчитывает средний возраст людей в карте
func avg_age(people map[string]int) int {
	sum := 0
	count := 0
	for _, age := range people {
		sum += age
		count++
	}
	return sum / count
}

// 3. Удаление записи по имени
func del_by_name(people map[string]int) {
	fmt.Println("Задание 3.")

	name := utils.ReadLine("Введите имя человека для удаления из карты: ")
	name, _, _ = strings.Cut(name, " ")

	if _, ok := people[name]; !ok {
		panic("В карте нет такого имени")
	}
	delete(people, name)

	fmt.Println("Удалено успешно")
	fmt.Println("Теперь карта выглядит так:", people)

	fmt.Println()
}

// 4. Строка в верхний регистр
func read_upper() {
	fmt.Println("Задание 4.")

	s := utils.ReadLine("Введите строку для перевода в верхний регистр\n>> ")
	s = strings.Map(unicode.ToUpper, s)
	fmt.Println(s)

	fmt.Println()
}

// 5. Сумма чисел введённых пользователем
func sum_nums() {
	fmt.Println("Задание 5.")

	nums := utils.ReadInts("Введите любое количество целых чисел для суммы: ")
	fmt.Println("Чисел успешно считано:", len(nums))
	sum := 0
	for _, num := range nums {
		sum += num
	}

	fmt.Println("Сумма равна", sum)

	fmt.Println()
}

// 6. Массив чисел в обратном порядке
func rev_array() {
	fmt.Println("Задание 6.")

	const LEN int = 6
	var arr [LEN]int

	nums := utils.ReadInts(fmt.Sprintf("Введите %d целых чисел для разворота: ", LEN))
	if len(nums) != LEN {
		panic("Введено неверное количество чисел")
	}

	for i, num := range nums {
		arr[LEN-i-1] = num
	}

	fmt.Println("Числа в обратном порядке:", arr)

	fmt.Println()
}

func main() {
	people := make_map()

	fmt.Println("Задание 2.")
	avg := avg_age(people)
	fmt.Println("Средний возраст людей в карте:", avg)
	fmt.Println()

	del_by_name(people)
	read_upper()
	sum_nums()
	rev_array()
}
