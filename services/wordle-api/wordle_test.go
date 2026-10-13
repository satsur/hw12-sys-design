package main

import (
	"slices"
	"testing"
)

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

func assertCorrectSlice(t testing.TB, got, want []rune) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
