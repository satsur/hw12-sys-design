package main

import (
	"slices"
	"testing"
)

type WordTest struct {
	Name          string
	Answer        string
	Guess         string
	ExpectedError error
	Expected      []string
}

func TestProcessWord(t *testing.T) {

	tests := []WordTest{
		{"duplicate letters, one in right place", "HELLO", "LADLE", nil, []string{"present", "absent", "absent", "correct", "present"}},
		{"duplicate letters", "HELLO", "LABEL", nil, []string{"present", "absent", "absent", "present", "present"}},
		{"all present", "ABCDE", "ECDBA", nil, []string{"present", "present", "present", "present", "present"}},
		{"all wrong", "AAAAA", "BBBBB", nil, []string{"absent", "absent", "absent", "absent", "absent"}},
		{"all same letter", "AAAAA", "AAAAA", nil, []string{"correct", "correct", "correct", "correct", "correct"}},
		{"longer word than answer", "WORLD", "WORLDLY", ErrWordLength, []string{}},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			got, err := ProcessWord(test.Guess, test.Answer)
			assertError(t, err, test.ExpectedError)
			assertCorrectSlice(t, got, test.Expected)
		})
	}
}

func TestSplitWord(t *testing.T) {
	t.Run("split 5 letter word", func(t *testing.T) {
		word := "SLATE"
		got := SplitWord(word)
		want := []rune{'S', 'L', 'A', 'T', 'E'}

		assertCorrectSlice(t, got, want)
	})

	t.Run("split empty word", func(t *testing.T) {
		word := ""
		got := SplitWord(word)
		want := []rune{}

		assertCorrectSlice(t, got, want)
	})
}

func assertError(t testing.TB, err error, expectedErr error) {
	t.Helper()
	if err == nil {
		if expectedErr == nil {
			return
		}
		t.Errorf("Expected an error, did not receive one.")
	}
}

func assertCorrectSlice[T string | rune](t testing.TB, got, want []T) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
