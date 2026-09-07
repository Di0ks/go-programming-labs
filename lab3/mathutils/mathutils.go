package mathutils

var results []int = []int{1, 1, 2, 6}

func Factorial(n int) int {
	if n == 0 {
		return 1
	}

	len := len(results)
	if n >= len {
		new_len := n + 1
		new_results := make([]int, new_len)
		copy(new_results, results)

		for i := len; i < new_len; i++ {
			new_results[i] = new_results[i-1] * i
		}
		results = new_results
	}

	return results[n]
}
