// The 'compound' utility takes either a list of filenames, or the single
// character '-', and returns the longest word which is entirely composed of
// other words from the provided list.  If run with '-', input will be read
// from STDIN.  If run with both files and '-', the word list will be the
// combined contents of the files and whatever is read from STDIN.
//
// Running it without arguments or with '-h' provides the following usage
// info:
//
// ---
//
// Usage: compound < -h | - | filename [filename ...] >
//
// Where:
//        -h : Prints this message.
//         - : Indicates that words should be read from STDIN.
//  filename : Specifies a file containing a list of words to read in.
//             Specifying multiple files will cause compound to read them
//             all in and work on the aggregate list.
//             Specifying both filename(s) and "-" will combine the contents
//             of the file(s) and whatever is passed in via STDIN.
//
// Whether in a stream or in file(s), words are expected to be given one per line.
//
// ---
//
// The basic approach to the problem that is implemented here is as follows:
//
//  1) A graph is constructed of the constituent bytes which make up each
//     word.  At the end of a word on this graph, there is an "end of word"
//     marker.
//      * This means that if one word begins with another, the smaller
//        word will be entirely on the path through the graph where the
//        larger word is found.
//      * See the declaration of type bytegraph, and the makegraph() function
//        in the source for more details.
//
//  2) Only words which begin with other words according to the graph are
//     examined more closely to see if they are compound words.  A word
//     which does *not* begin with another word on the graph *cannot* be
//     a compound word (at least with respect to the current word list).
//
//  3) Compound words are searched for in reverse order of size, so that
//     the first word that is found which is a compound word ends the run.
//
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const (
	maxInt = int(^uint(0) >> 1)
)

// I got tired of typing brackets pretty early on.
type word []byte
type words []word

// Len, Less, and Swap make 'words' a sort.Interface, allowing the
// use of sort.Sort() on a list of words.
func (ws words) Len() int {
	return len(ws)
}

// This is technically LessThanOrEqualTo, but that won't change
// the validity of the test where Sort is concerned.
func (ws words) Less(i, j int) bool {
BYTE:
	for k := range ws[i] {
		if k >= len(ws[j]) || ws[i][k] > ws[j][k] {
			return false
		}
		if ws[i][k] < ws[j][k] {
			break BYTE
		}
	}
	return true
}

func (ws words) Swap(i, j int) {
	ws[i], ws[j] = ws[j], ws[i]
}

// A bytegraph allows for quick determination of whether or not
// a slice of bytes constitutes a word from the list, without having
// to maintain a map of words or mess with a bunch of string splits
// to do so.
//
// If the word list contains "foo", "foody", and "foe", the resulting
// bytegraph should partially consist of something like this:
// 'f' -> { endOfWord:false
//          next: {
//            'o' -> { endOfWord:false
//                     next: {
//                       'e' -> { endOfWord:true
//                                next:nil }
//                       'o' -> { endOfWord:true
//                                next: {
//                                  'd' -> { endOfWord:false
//                                           next: {
//                                             'y' -> { endOfWord:true
//                                                      next:nil
// } } } } } } } } }
//
// ...and so on, as more words are added.
//
// The main benefit of this over a map, however, is that it enables
// me to quickly determine whether or not a word begins with other
// words.  Traversing the graph above, if you're checking if 'foody'
// is a word, it's easy to see that 'foo' is a word along the graph.
// This becomes an important factor in finding out what words *might*
// be compound words.
//
type bytegraph struct {
	endOfWord bool
	next      map[byte]bytegraph
}

// makegraph takes a word and a pointer to a pre-existing bytegraph
// (populated or not), and populates the bytegraph accordingly (see
// example above).  Note that accurately determining whether or not
// a word has prefixes is dependent on the bytegraph already containing
// those prefixes.  That is the reason the main bytegraph must be
// populated from a sorted list of words.
func makegraph(w word, g *bytegraph) (hasPrefixes bool) {
	if len(w) > 0 {
		hasPrefixes = g.endOfWord
		b := w[0]
		ng, exists := g.next[b]
		if !exists {
			ng = bytegraph{}
			ng.next = make(map[byte]bytegraph)
		}
		hasPrefixes = makegraph(w[1:], &ng) || hasPrefixes
		g.next[b] = ng
	} else {
		g.endOfWord = true
	}
	return
}

// A 'potential' struct is used to hold a word once it has been
// determined that it is possible for that word to be compound.
type potential struct {
	whole      word
	prefixes   words
	components words
}
type potentials []potential

// isCompound is the entry point for the code that determines the central
// question - whether or not a word is a compound word.
func (p *potential) isCompound(g bytegraph, minLen int) bool {
	for _, pfx := range p.prefixes {
		parts := subWords(p.whole[len(pfx):], g, minLen)
		if parts != nil {
			p.components = make(words, 0)
			p.components = append(append(p.components, pfx), parts...)
			return true
		}
	}
	return false
}

// subWords takes a word or partial word and returns all the words that
// go together to make it up, but only if the word *can* be decomposed
// into other words.  If w cannot be decomposed, ws will be nil.
func subWords(w word, g bytegraph, minLen int) (ws words) {
	// Obviously, if this is a word to start with, just return it.
	if isWord(w, g) {
		return append(ws, w)
	}

	// Otherwise, we check all the substrings of length at least minLen
	// to see if *they* are words.
PRE:
	for i := len(w) - minLen; i >= minLen; i-- {
		pre, rest := w[:i], w[i:]
		// If the prefix is a word...
		if isWord(pre, g) {
			// ...then we check the remainder...
			if isWord(rest, g) {
				// ...and if they're both words, we're done.
				ws = append(ws, pre, rest)
				break PRE
			} else {
				// If the remainder is not a word on its own, check
				// and see if it is composed of other words.
				moar := subWords(rest, g, minLen)
				if moar != nil {
					// And again, if it is, we have our answer.
					ws = append(append(ws, pre), moar...)
					break PRE
				}
			}
		}
	}

	// ws is only populated if the *entire* word was able to be split
	// into a combination of other words - it never contains just a
	// partial list, in other words, so this should be a safe return.
	return
}

// Walk the graph and see if w is a word.
func isWord(w word, g bytegraph) bool {
	for _, b := range w {
		next, exists := g.next[b]
		if exists {
			g = next
		} else {
			return false
		}
	}
	return g.endOfWord
}

// Returns either:
//   foobar = foo + bar
// - or -
//   foobar [NOT COMPOUND]
func (p potential) String() string {
	s := string(p.whole)
	if len(p.components) > 0 {
		s += " = "
		for i := range p.components {
			s += string(p.components[i])
			if i < len(p.components)-1 {
				s += " + "
			}
		}
	} else {
		s += " [NOT COMPOUND]"
	}

	return s
}

// loadWordsFrom takes a stream of words and populates a simple list
// of words.  It returns the length of the shortest word it sees.
func loadWordsFrom(r io.Reader, wordlist *words) (minLen int) {
	wordloader := bufio.NewScanner(r)
	minLen = maxInt

	for wordloader.Scan() {
		nw := make(word, len(wordloader.Bytes()))
		copy(nw, wordloader.Bytes())
		*wordlist = append(*wordlist, nw)
		if len(nw) < minLen {
			minLen = len(nw)
		}
	}

	return
}

func loadAllTheWords(wordlist *words) (minLen int) {
	// Setting this initially to the maximum possible so
	// anything returned by loadWordsFrom() will be less.
	minLen = maxInt

	for _, arg := range os.Args[1:] {
		var file *os.File
		var err error

		if arg == "-" {
			file = os.Stdin
		} else {
			file, err = os.Open(arg)
			if err != nil {
				panic(err)
			}
		}

		minLength := loadWordsFrom(file, wordlist)
		if minLength < minLen {
			minLen = minLength
		}

		if file != os.Stdin {
			err = file.Close()
			if err != nil {
				panic(err)
			}
		}
	}

	return
}

func graphAndFindCandidates(wordlist words) (g bytegraph, pm map[int]potentials) {
	g = bytegraph{}
	g.next = make(map[byte]bytegraph)

	pm = make(map[int]potentials)

	for i, thisword := range wordlist {

		// The only words we're really interested in examining further
		// are those that begin with another word from the list.  No
		// others can possibly be compound words.
		hasPrefixes := makegraph(thisword, &g)
		if hasPrefixes {
			np := potential{}
			np.whole = make(word, len(thisword))
			copy(np.whole, thisword)

			// This determines which other words from the list begin the current
			// word.  If the current word is "foodie", and "foo" and "food" are
			// on the list, they will be added to "foodie"'s prefix list.
			np.prefixes = make(words, 0)
		PREFIX:
			for j := 1; j <= i; j++ {
				if bytes.HasPrefix(wordlist[i], wordlist[i-j]) {
					np.prefixes = append(np.prefixes, make(word, len(wordlist[i-j])))
					copy(np.prefixes[len(np.prefixes)-1], wordlist[i-j])
				} else {
					// Since the word list is sorted, as soon as a word is reached
					// that is NOT a prefix of the current word, no further words
					// need to be looked at.  All prefixes for a given word MUST
					// immediately precede that word in the list.
					break PREFIX
				}
			}

			_, exists := pm[len(np.whole)]
			if !exists {
				pm[len(np.whole)] = make(potentials, 0)
			}
			pm[len(np.whole)] = append(pm[len(np.whole)], np)
		}
	}

	return
}

//////////////
//
// And now, without any further ado...
//
func main() {

	// We do need *something* to work with.
	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "-h") {
		fmt.Fprintf(os.Stderr, usage())
		os.Exit(0)
	}

	// First, load up whatever words are to be processed.
	allwords := make(words, 0)

	// Recording the minimum word length makes the subword search a
	// bit more efficient.  If the smallest word is three characters,
	// there's no need to go looking for a two character word, for
	// instance.
	minWordLength := loadAllTheWords(&allwords)

	// The words must be sorted in order for the algorithm to work.
	sort.Sort(allwords)

	// chargraph is the main bytegraph, which allows for a very rapid
	// determination of composite words.
	//
	// candidatesByLength is pretty much what it sounds like.  Potential
	// compound words, indexed by word length.
	//
	// Since the purpose is finding the *longest* compound word, it
	// makes sense to be able to look at candidate words in descending
	// order of length.  We can stop at the first one found since it
	// will by definition be the longest.
	chargraph, candidatesByLength := graphAndFindCandidates(allwords)

	var descendingLengths []int
	for l := range candidatesByLength {
		descendingLengths = append(descendingLengths, l)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(descendingLengths)))

POSSIBLE:
	for _, l := range descendingLengths {
		for _, w := range candidatesByLength[l] {
			if (&w).isCompound(chargraph, minWordLength) {
				fmt.Println(w)
				break POSSIBLE
			}
		}
	}
}

// exitUsage - what it says on the tin.  Just print the basic usage, and
// exit gracefully.
func usage() (u string) {
	programName := filepath.Base(os.Args[0])

	u = "Usage: " + programName + " < -h | - | filename [filename ...] >\n" +
		"\tWhere:\n" +
		"\t\t      -h : Prints this message.\n" +
		"\t\t       - : Indicates that words should be read from STDIN.\n" +
		"\t\tfilename : Specifies a file containing a list of words to read in.\n" +
		"\t\t           Specifying multiple files will cause " + programName + " to read " +
		"them all in\n" +
		"\t\t           and work on the aggregate list.\n" +
		"\t\t           Specifying both filename(s) and \"-\" will combine the contents of\n" +
		"\t\t           the file(s) and whatever is passed in via STDIN.\n" +
		"\n" +
		"Whether in a stream or in file(s), words are expected to be given one per line.\n" +
		"\n"

	return
}
