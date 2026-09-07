package main

import (
	"fmt"
	"time"
)

// Назание месяца в родительном падеже на русском
func ru_month(month time.Month) string {
	switch month {
	case time.April:
		return "Апреля"
	case time.August:
		return "Августа"
	case time.December:
		return "Декабря"
	case time.February:
		return "Февраля"
	case time.January:
		return "Января"
	case time.July:
		return "Июля"
	case time.June:
		return "Июня"
	case time.March:
		return "Марта"
	case time.May:
		return "Мая"
	case time.November:
		return "Ноября"
	case time.October:
		return "Октября"
	case time.September:
		return "Сентября"
	default:
		panic(fmt.Sprintf("unexpected time.Month: %#v", month))
	}
}

// 1. Вывод текущих времени и даты
func show_date_time() {
	fmt.Println("Задание 1.")

	now := time.Now()
	year, month, day := now.Local().Date()
	hour, min := now.Local().Hour(), now.Local().Minute()
	fmt.Printf(
		"Время и дата сейчас: %02d:%02d %d %s %d года\n",
		hour,
		min,
		day,
		ru_month(month),
		year,
	)

	fmt.Println()
}

// 2. Работа с различными переменными
func various_vars() {
	fmt.Println("Задание 2.")

	const num int = 52
	const real float64 = 3.1415
	const weather string = "дождливо"
	const is_set bool = true
	fmt.Println("Целое число (int) =", num)
	fmt.Println("Число с плавающей запятой (float) =", real)
	fmt.Printf("Строка (string) = '%s'\n", weather)
	fmt.Println("Булево (bool) =", is_set)

	fmt.Println()
}

// 3. Краткая форма объявления переменных
func short_form() {
	fmt.Println("Задание 3.")

	item := 123
	fmt.Printf(
		"Переменная `item` типа '%T' объявлена с помощью короткой записи (`:=`) и равна %d\n",
		item,
		item,
	)

	fmt.Println()
}

// 4. Арифметические операции с целыми числами
func arith_ops() {
	fmt.Println("Задание 4.")

	a := 52
	b := 67

	fmt.Printf("Изначальные значения переменных: a=%d b=%d\n", a, b)
	fmt.Println("Вычитание: a - b =", a-b)
	fmt.Println("Сложение:  a + b =", a+b)
	fmt.Println("Умножение: a * b =", a*b)
	fmt.Println("Деление:   a / b =", a/b)
	fmt.Println("Остаток от деления: a % b =", a%b)

	fmt.Println()
}

// 5. Сумма и разность чисел с плавающей запятой
func sum_and_diff_float(a float32, b float32) {
	fmt.Println("Задание 5.")

	fmt.Printf("Переданные числа: a=%.2f b=%.2f\n", a, b)
	fmt.Printf("Сумма:    %.2f + %.2f = %.2f\n", a, b, a+b)
	fmt.Printf("Разность: %.2f - %.2f = %.2f\n", a, b, a-b)

	fmt.Println()
}

// 6. Среднее трёх чисел
func avg_three(a int, b int, c int) {
	fmt.Println("Задание 6.")

	fmt.Printf("Изначальные значения: a=%d b=%d c=%d\n", a, b, c)
	fmt.Println("Среднее:", float64(a+b+c)/3.)

	fmt.Println()
}

func main() {
	show_date_time()
	various_vars()
	short_form()
	arith_ops()
	sum_and_diff_float(6.66, 13.37)
	avg_three(21, -4, 123)
}
