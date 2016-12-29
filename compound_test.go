package main

import (
	"bytes"
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
				t.Errorf("Swap - Exchanging %d<->%d; expected:\n",
					"\t%q\nBut got\n\t%q", i, j, expected, actual)
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
		byte('a'): bytegraph{endOfWord: true,
			next: map[byte]bytegraph{
				byte('b'): bytegraph{endOfWord: true,
					next: map[byte]bytegraph{
						byte('c'): bytegraph{endOfWord: false,
							next: map[byte]bytegraph{
								byte('d'): bytegraph{endOfWord: true,
									next: map[byte]bytegraph{},
								}}}}}}}}}

func TestMakegraph(t *testing.T) {
	testgraph := bytegraph{}
	testgraph.next = make(map[byte]bytegraph)
	for _, w := range shortWords {
		_ = makegraph(w, &testgraph)
	}

	if !reflect.DeepEqual(testgraph, shortGraph) {
		t.Errorf("makegraph - Expected:\n\t%q\nBut got\n\t%q", shortGraph, testgraph)
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
		{word("fooartartfulbar"), words{word("foo"), word("art"), word("artful"), word("bar")}},
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
		potential{whole: word("quartsplat"),
			prefixes:   words{word("qu"), word("quart")},
			components: words{word("quart"), word("splat")}},
		potential{whole: word("quartfulsquish"),
			prefixes:   words{word("qu"), word("quart")},
			components: words{word("qu"), word("artful"), word("squish")}},
		potential{whole: word("quartfulsquishy"),
			prefixes:   words{word("qu"), word("quart")},
			components: nil},
		potential{whole: word("nosuchword"),
			prefixes:   nil,
			components: nil},
	}

	for _, p := range testPotentials {
		if !strings.Contains(p.String(), string(p.whole)) {
			t.Errorf("String - Representation of \"%s\" does not contain the word itself: %q\n",
				string(p.whole), p.String())
		}
	}
}

func TestLoadWordsFrom(t *testing.T) {
	// Making a fake file out of the testWords.  No need to rely on an
	// actual file on disk when bytes.NewReader will give me what I need.
	var fakeFile []byte
	for _, w := range testWords {
		fakeFile = append(fakeFile, w...)
		fakeFile = append(fakeFile, '\n')
	}
	source := bytes.NewReader(fakeFile)

	testMinLen := int(^uint(0) >> 1)
	for _, w := range testWords {
		if len(w) < testMinLen {
			testMinLen = len(w)
		}
	}

	actualWords := make(words, 0)
	actualMinLen := loadWordsFrom(source, &actualWords)

	if actualMinLen != testMinLen {
		t.Errorf("loadWordsFrom - MinLen mismatch: expected: %d, got: %d\n",
			testMinLen, actualMinLen)
	}

	if !reflect.DeepEqual(testWords, actualWords) {
		t.Errorf("loadWordsFrom - Word list mismatch.\n"+
			"Expected:\n\t%q\nActual:\n\t%q\n", testWords, actualWords)
	}
}

// Yeah, not all that robust a test, but beyond this, there's not
// much more that can be reasonably asserted.
func TestUsage(t *testing.T) {
	usageMsg := usage()

	if !strings.Contains(usageMsg, "Usage") {
		t.Errorf("usage - Message does not contain \"Usage\"\n")
	}
}
