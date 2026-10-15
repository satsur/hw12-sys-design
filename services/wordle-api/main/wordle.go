package main

import (
	"errors"
	"strings"
)

const (
	correct = "correct"
	present = "present"
	absent  = "absent"
)

var ErrWordLength = errors.New("Guess length must match answer length")

func ProcessWord(word, answer string) ([]string, error) {
	word = strings.ToUpper(word)

	guessLetters := SplitWord(word)
	answerLetters := SplitWord(answer)

	if len(guessLetters) != len(answerLetters) {
		return nil, ErrWordLength
	}

	result := make([]string, len(answerLetters))

	// Tracks all letters that have been consumed (i.e. either matched to a correct guess letter or
	// matched to a "present" letter)
	consumed := make([]bool, len(answerLetters))

	// First iteration: set letters equal to absent or correct
	for i, letter := range guessLetters {
		if letter == answerLetters[i] {
			result[i] = correct
			consumed[i] = true
		} else {
			result[i] = absent
		}
	}

	// now iterate over all currently absent letters and check if should be present
	for i, guessLetter := range guessLetters {
		if result[i] == correct {
			continue
		}

		// Check inside word for presence
		for j, ansLetter := range answerLetters {
			if consumed[j] == false && ansLetter == guessLetter {
				result[i] = present
				consumed[j] = true
				break
			}
		}
	}
	return result, nil
}

func SplitWord(word string) []rune {
	return []rune(word)
}
