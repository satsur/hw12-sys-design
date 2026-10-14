package main

import (
	"slices"
	"testing"
)

type Wordle struct {
	answer string
	guess  string
	want   string
}

func TestProcessWord(t *testing.T) {

	t.Run("duplicate letters, one in right place", func(t *testing.T) {
		answer := "HELLO"
		word := "LADLE"

		got, _ := ProcessWord(word, answer)
		want := []string{"present", "absent", "absent", "correct", "present"}

		assertCorrectSlice(t, got, want)
	})

	t.Run("duplicate letters", func(t *testing.T) {
		answer := "HELLO"
		word := "LABEL"

		got, _ := ProcessWord(word, answer)
		want := []string{"present", "absent", "absent", "present", "present"}

		assertCorrectSlice(t, got, want)
	})

	t.Run("all present", func(t *testing.T) {
		answer := "ABCDE"
		word := "ECDBA"

		got, _ := ProcessWord(word, answer)
		want := []string{"present", "present", "present", "present", "present"}

		assertCorrectSlice(t, got, want)
	})

	t.Run("all wrong", func(t *testing.T) {
		answer := "AAAAA"
		word := "BBBBB"

		got, _ := ProcessWord(word, answer)
		want := []string{"absent", "absent", "absent", "absent", "absent"}

		assertCorrectSlice(t, got, want)
	})
	t.Run("all same letter", func(t *testing.T) {
		answer := "AAAAA"
		word := "AAAAA"

		got, _ := ProcessWord(word, answer)
		want := []string{"correct", "correct", "correct", "correct", "correct"}

		assertCorrectSlice(t, got, want)
	})

	t.Run("word WORLD answer HELLO", func(t *testing.T) {
		answer := "HELLO"
		word := "WORLD"

		got, _ := ProcessWord(word, answer)
		want := []string{"absent", "present", "absent", "correct", "absent"}

		assertCorrectSlice(t, got, want)
	})

	t.Run("longer word than answer", func(t *testing.T) {
		answer := "WORLD"
		word := "WORLDLY"

		got, err := ProcessWord(word, answer)
		want := []string{}

		assertError(t, err)
		assertCorrectSlice(t, got, want)
	})
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

func assertError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected an error, did not receive one.")
	}
}

func assertCorrectSlice[T string | rune](t testing.TB, got, want []T) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
