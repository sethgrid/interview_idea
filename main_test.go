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

func TestIntersection(t *testing.T) {
	t.Log("test two strings' intersection and duplicates droped")

	cases := []struct {
		a      string
		b      string
		output string
	}{
		{a: "apple", b: "pie", output: "pe"},
		{a: "eeeeeee", b: "e", output: "e"},
		{a: "NKíjé", b: "NKíjé", output: "NKíjé"},
		{a: "a", b: "b", output: ""},
		{a: "a", b: "", output: ""},
		{a: "", b: "b", output: ""},
		{a: "", b: "", output: ""},
	}

	for _, c := range cases {
		if got, want := intersection(c.a, c.b), c.output; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func TestUnion(t *testing.T) {
	t.Log("test two strings' union and duplicates droped")

	cases := []struct {
		a      string
		b      string
		output string
	}{
		{a: "apple", b: "pie", output: "aplei"},
		{a: "eeeeeee", b: "e", output: "e"},
		{a: "NKíjé", b: "NKíjé", output: "NKíjé"},
		{a: "a", b: "b", output: "ab"},
		{a: "a", b: "", output: "a"},
		{a: "", b: "b", output: "b"},
		{a: "", b: "", output: ""},
	}

	for _, c := range cases {
		if got, want := union(c.a, c.b), c.output; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func TestUnionSort(t *testing.T) {
	t.Log("test two strings' union and duplicates droped")

	cases := []struct {
		a      string
		b      string
		output string
	}{
		{a: "apple", b: "pie", output: "aeilp"},
		{a: "eeeeeee", b: "e", output: "e"},
		{a: "NKíjé", b: "NKíjé", output: "éíjKN"},
		{a: "b", b: "a", output: "ab"},
		{a: "a", b: "", output: "a"},
		{a: "", b: "b", output: "b"},
		{a: "", b: "", output: ""},
	}

	for _, c := range cases {
		if got, want := unionSort(c.a, c.b), c.output; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func TestMangle(t *testing.T) {
	t.Log("test two strings' mangle and duplicates droped")

	cases := []struct {
		a      string
		b      string
		output string
	}{
		{a: "aaaaaa", b: "bbbbbb", output: "ab"},
		{a: "abcdef", b: "abcdef", output: "abcdef"},
		{a: "NKíklé", b: "NKíklé", output: "NKíklé"},
		{a: "abcdef", b: "fedcba", output: "aec"},
		{a: "abcd", b: "wxyz", output: "axcz"},
	}

	for _, c := range cases {
		if got, want := mangle(c.a, c.b), c.output; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}
