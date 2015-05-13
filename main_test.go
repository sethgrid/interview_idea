package main

import "testing"

func TestDeduplication(t *testing.T) {
	t.Log("test that duplicate characters in a string are removed")

	cases := []struct {
		input  string
		output string
	}{
		{input: "apple", output: "aple"},
		{input: "eeeeeee", output: "e"},
		{input: "NKíjé", output: "NKíjé"},
		{input: "a", output: "a"},
	}

	for _, c := range cases {
		if got, want := deduplicate(c.input), c.output; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}
