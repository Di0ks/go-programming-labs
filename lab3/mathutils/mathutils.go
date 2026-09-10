// Пакет предоставляет различные математические функции, например, получения числа из ряда.
//
// Все функции НЕ потоко-безопасные, если не сказано иного.
package mathutils

import "math"

var fac_cache []int = []int{1, 1, 2, 6}

// Возвращает факториал числа `n`
func Factorial(n int) int {
	if n == 0 {
		return 1
	}

	len := len(fac_cache)
	if n >= len {
		new_len := n + 1
		new_results := make([]int, new_len)
		copy(new_results, fac_cache)

		for i := len; i < new_len; i++ {
			new_results[i] = new_results[i-1] * i
		}
		fac_cache = new_results
	}

	return fac_cache[n]
}

var prime_cache []int = []int{2, 3, 5, 7}

// Проверка на простое число
func is_prime(num int) bool {
	max_factor := int(math.Sqrt(float64(num)))
	for factor := 2; factor <= max_factor; factor++ {
		if num%factor == 0 {
			return false
		}
	}
	return true
}

// Возвращает простое число из последовательности по индексом `i`
func Prime(i int) int {
	len := len(prime_cache)
	if i >= len {
		new_len := i + 1
		new_cache := make([]int, new_len)
		copy(new_cache, prime_cache)

		for i := len; i < new_len; i++ {
			for num := new_cache[i-1] + 1; ; num++ {
				if is_prime(num) {
					new_cache[i] = num
					break
				}
			}
		}
		prime_cache = new_cache
	}

	return prime_cache[i]
}

var fibonacci_cache []int = []int{0, 1, 1, 2}

// Возвращает число из последовательности Фибоначчи по индексом `i`
func Fibonacci(i int) int {
	len := len(fibonacci_cache)
	if i >= len {
		new_len := i + 1
		new_cache := make([]int, new_len)
		copy(new_cache, fibonacci_cache)

		for i := len; i < new_len; i++ {
			new_cache[i] = new_cache[i-1] + new_cache[i-2]
		}
		fibonacci_cache = new_cache
	}

	return fibonacci_cache[i]
}
