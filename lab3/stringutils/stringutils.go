package stringutils

import (
	"unicode/utf8"
)

func Reverse(s string) string {
	// длина в рунах, не в байтах
	len := utf8.RuneCountInString(s)
	new_s := make([]rune, len)
	for i, c := range []rune(s) {
		new_s[len-i-1] = c
	}
	return string(new_s)
}
