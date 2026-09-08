package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Считывает строку со стандартного ввода до символа новой строки (символ новой
// строки не сохраняется).
//
// Вызывает панику при неверном вводе или ошибке.
func ReadLine(prompt string) string {
	fmt.Print(prompt)

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

// Считывает одно целое число со стандартного ввода (до символа новой строки).
//
// Вызывает панику если ввод не является целым числом или из-за внутренней ошибки.
func ReadSingleInt(prompt string) int {
	fmt.Print(prompt)

	var res int
	n, err := fmt.Scanln(&res)
	if n != 1 {
		panic(fmt.Sprintf("не удалось прочитать целое число: %v", err))
	}

	return res
}

// Считывает любое количество целых чисел разделённых пробелом из стандартного ввода.
//
// Останавливается на первом некорректном символе или символе новой строки.
//
// Паникует при внутренних ошибках ввода.
func ReadInts(prompt string) []int {
	s := ReadLine(prompt)
	items := strings.Fields(s)
	res := make([]int, 0, len(items))
	for _, v := range items {
		num, err := strconv.Atoi(v)
		if err != nil {
			break
		}

		res = append(res, num)
	}

	return res
}
