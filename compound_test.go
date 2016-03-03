package main

import (
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
)

//////////////
//
//  Test Data
//

// Basic list of words
var testWords = words{
	word("foo"), word("bar"), word("quux"), word("foobar"),
	word("barfooquux"), word("qu"), word("splat"), word("artful"),
	word("splatter"), word("squish"), word("quart"), word("art"),
}

var notWords = words{
	word("fibble"), word("squadoosh"), word("foobary"),
	word("quartfulbarqufoosquis"),
}

// ...and the same list, sorted.  Populated in init().
var sortedTestWords words

// A bytegraph populated from the above words.  Generated in init().
var testGraph bytegraph

// There's a little bit of a chicken-and-egg going on here.  I'm
// relying on makegraph, Len, Less, and Swap to all function
// correctly in order to populate sortedTestWords and testGraph.
//
// This is predicated on the hypothesis that those functions' tests
// *should* catch any bugs in them.  In other words, assuming their
// tests pass, I'm going on the presumption that this usage will be
// more or less safe.  Generating a correct bytegraph of this size
// by hand would be tedious at best, and a fairly error-prone
// endeavor regardless.
func init() {
	sortedTestWords = make(words, len(testWords))
	copy(sortedTestWords, testWords)
	sort.Sort(sortedTestWords)

	testGraph.next = make(map[byte]bytegraph)
	for _, w := range sortedTestWords {
		_ = makegraph(w, &testGraph)
	}
}

func TestLen(t *testing.T) {
	expected := len(testWords)
	actual := testWords.Len()
	if expected != actual {
		t.Errorf("Len: Expected %q but got %q", expected, actual)
	}
}

// Note that Less is really LessThanOrEqualTo, hence the <= in
// the string comparison.
func TestLess(t *testing.T) {
	for i := range testWords {
		for j := range testWords {
			expected := string(testWords[i]) <= string(testWords[j])
			actual := testWords.Less(i, j)
			if expected != actual {
				t.Errorf("Less: %q < %q : expected %v but got %v",
					testWords[i], testWords[j], expected, actual)
			}
		}
	}
}

func TestSwap(t *testing.T) {
	for i := range testWords {
		for j := range testWords {
			expected := make(words, len(testWords))
			copy(expected, testWords)
			expected[i], expected[j] = expected[j], expected[i]

			actual := make(words, len(testWords))
			copy(actual, testWords)
			actual.Swap(i, j)

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("Swap - Exchanging %d<->%d; expected:\n\t%q\nBut got\n\t%q", i, j, expected, actual)
			}
		}
	}
}

// A much more manageable word list to test makegraph().
// NOTE: This list MUST be sorted for the test to be valid.
var shortWords = words{word("a"), word("ab"), word("abcd")}

// ...and the resulting also-much-more-manageable bytegraph
// that comes from it.  ...and by "manageable" I mean "easier
// to generate by hand".
var shortGraph = bytegraph{endOfWord: false,
	next: map[byte]bytegraph{
		byte('a'): {endOfWord: true,
			next: map[byte]bytegraph{
				byte('b'): {endOfWord: true,
					next: map[byte]bytegraph{
						byte('c'): {endOfWord: false,
							next: map[byte]bytegraph{
								byte('d'): {endOfWord: true,
									next: map[byte]bytegraph{},
								}}}}}}}}}

func TestMakegraph(t *testing.T) {
	testgraph := bytegraph{}
	testgraph.next = make(map[byte]bytegraph)
	for _, w := range shortWords {
		_ = makegraph(w, &testgraph)
	}

	if !reflect.DeepEqual(testgraph, shortGraph) {
		t.Errorf("makegraph - Expected:\n\t%v\nBut got\n\t%v", shortGraph, testgraph)
	}
}

func TestIsWord(t *testing.T) {
	for _, w := range testWords {
		if !isWord(w, testGraph) {
			t.Errorf("isWord - %s should be a word", string(w))
		}
	}
	for _, w := range notWords {
		if isWord(w, testGraph) {
			t.Errorf("isWord - %s should NOT be a word", string(w))
		}
	}
}

func TestSubWords(t *testing.T) {
	var swTests = []struct {
		w      word
		expect words
	}{
		{word("foobar"), words{word("foobar")}},
		{word("fooquux"), words{word("foo"), word("quux")}},
		{word("fooart"), words{word("foo"), word("art")}},
		{word("splatterart"), words{word("splatter"), word("art")}},
		{word("quartful"), words{word("qu"), word("artful")}},
		{word("foobarquu"), nil},
		{word("oobar"), nil},
		{word("bogus"), nil},
	}

	for _, tst := range swTests {
		actual := subWords(tst.w, testGraph, 2)
		if !reflect.DeepEqual(tst.expect, actual) {
			t.Errorf("subWords - Expected\n\t%q\nBut got\n\t%q", tst.expect, actual)
		}
	}
}

func TestIsCompound(t *testing.T) {
	var compTests = []struct {
		p      potential
		expect bool
	}{
		{p: potential{whole: word("quartsplat"),
			prefixes: words{word("qu"), word("quart")}}, expect: true},
		{p: potential{whole: word("quartfulsquish"),
			prefixes: words{word("qu"), word("quart")}}, expect: true},
		{p: potential{whole: word("quartfulsquishy"),
			prefixes: words{word("qu"), word("quart")}}, expect: false},
	}

	for _, tst := range compTests {
		actual := (&tst.p).isCompound(testGraph, 2)
		if actual != tst.expect {
			t.Errorf("isCompound - %s came back %v / expected %v",
				string(tst.p.whole), actual, tst.expect)
		}
	}
}

// NOTE: This only tests whether or not the String() method returns
// something which contains the original word.  Anything beyond that
// would just enforce some arbitrary string representation.
func TestString(t *testing.T) {
	var testPotentials = []potential{
		{whole: word("quartsplat"),
			prefixes:   words{word("qu"), word("quart")},
			components: words{word("quart"), word("splat")}},
		{whole: word("quartfulsquish"),
			prefixes:   words{word("qu"), word("quart")},
			components: words{word("qu"), word("artful"), word("squish")}},
		{whole: word("quartfulsquishy"),
			prefixes:   words{word("qu"), word("quart")},
			components: nil},
		{whole: word("nosuchword"),
			prefixes:   nil,
			components: nil},
	}

	for _, p := range testPotentials {
		if !strings.Contains(p.String(), string(p.whole)) {
			t.Errorf("String - Representation of %q does not contain the word itself: %q\n",
				p.whole, p.String())
		}
	}
}

// Concocting a fake file that can be used to test the loading
// function without having to rely on a file on disk for the
// test.
type fakeFile struct {
	ws     words
	cursor int
}

// The Read method that makes our fakeFile an io.Reader, so that
// loadWordsFrom can use it as a source of data.
func (f *fakeFile) Read(p []byte) (bytesRead int, err error) {
WORD:
	for {
		if f.cursor < len(f.ws) {
			thisLen := len(f.ws[f.cursor])

			// Strictly less than, since we need room for a newline.
			if thisLen < len(p)-bytesRead {
				copy(p[bytesRead:bytesRead+thisLen], f.ws[f.cursor])
				bytesRead += thisLen

				// A bufio.Scanner breaks on newlines by default; adding
				// them to the test words here, since our test word slice
				// doesn't include them (and doesn't need to).
				p[bytesRead] = byte('\n')
				bytesRead++

				f.cursor++
				continue WORD
			}
		} else {
			err = io.EOF
		}
		if bytesRead == 0 {
			f.cursor = 0
		}
		break WORD
	}
	return
}

func TestLoadWordsFrom(t *testing.T) {
	source := fakeFile{}
	source.ws = make(words, len(testWords))
	copy(source.ws, testWords)
	testMinLen := maxInt
	for _, w := range testWords {
		if len(w) < testMinLen {
			testMinLen = len(w)
		}
	}

	actualWords := make(words, 0)
	actualMinLen := loadWordsFrom(&source, &actualWords)

	if actualMinLen != testMinLen {
		t.Errorf("loadWordsFrom - MinLen mismatch: expected: %d, got: %d\n",
			testMinLen, actualMinLen)
	}

	if !reflect.DeepEqual(testWords, actualWords) {
		t.Errorf("loadWordsFrom - Word list mismatch.\n"+
			"Expected:\n\t%q\nActual:\n\t%q\n", testWords, actualWords)
	}
}
