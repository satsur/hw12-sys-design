package main

import "strings"

func SplitWord(word string) []rune {
	letters := strings.Split(word, "")
	runes := make([]rune, len(letters))

	for i, letter := range letters {
		// In case letter is multi-byte, we just grab first "rune"
		for _, r := range letter {
			runes[i] = r
			break
		}
	}

	return runes
}
