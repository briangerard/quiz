[![Go Report Card](https://goreportcard.com/badge/github.com/briangerard/quiz)](https://goreportcard.com/report/github.com/briangerard/quiz)
[![Build Status](https://travis-ci.org/briangerard/quiz.svg?branch=master)](https://travis-ci.org/briangerard/quiz)

# Solution to the NodePrime Quiz

First, I'd just like to say thanks for a really fun programming puzzle!  This was really
enjoyable to think through and implement, and I learned a good bit in the process.

Another thing to note; this was clearly not all done in a single commit.  The progressive
work that led up to this point is in a private Bitbucket repository, and I can bring it
over to github so that the history is viewable if anyone would like to see it.  Just let
me know.

---

## The Short Version

Compilation can be as simple as running `make` in the current directory.  That will put an
executable called `compound` in `$GOPATH/bin`.  Running that program with no arguments
provides usage.

---

## The Long-Winded Explanation

### Building

If make isn't working, or you'd rather run the steps manually, you can recreate what make
would do by running:
```
bash$ go build -o ${GOPATH}/bin/compound -i compound.go
```

You can also run the tests via `make test` or manually:
```
bash$ go test -v
```

### Running

Running the executable without arguments gives usage info.

In the following command examples, I am assuming that `compound` is in your `$PATH`.

To run on a file of words, use:
```
bash$ compound word.list
antidisestablishmentarianisms = antidisestablishmentarian + isms
```

The output will be of the form "fullword = component + component + ...".

To run on multiple files, use:
```
bash$ compound words0 words1 words2
antidisestablishmentarianisms = antidisestablishmentarian + isms
```

To run on a stream of words, use:
```
bash$ cat word.list | compound -
antidisestablishmentarianisms = antidisestablishmentarian + isms
```
---

## Performance

The guidance in the original README was that this should run in under an hour.  On a
quad-core i5-2400 with 16GiB of RAM, my solution processes the original word.list in under
one second, and on a laptop with an i3 and 4 GiB of RAM, it does so in about two seconds.

---

I pulled down other files to test, and these are the times I got on the i5 mentioned
above:

| Description | File Size (MiB) | Word Count | Time to Process (sec) |
| :---------- | --------------: | ---------: | --------------------: |
| Original word.list | 2.6 | 263K | 0.9 |
| english.dic, from a password cracking site| 32 | 3.1M | 11.7 |
| Much larger GDict_v2.txt, from a similar source | 267 | 21.6M | 93.7 |

Just going by these numbers, performance **appears** to be roughly linear on file size,
but I haven't rigorously profiled or proven that by any means.

---

Performance will be impacted by at least the following factors:

1. Unsorted data
  * The algorithm I designed requires the input to be in ascending order, so prior to
    actually looking for words-within-words, a sort has to be done.  In tests with the
    original word.list that didn't seem to incur more than about a 10% penalty when I
    randomly mixed it up, but larger inputs may be more heavily impacted.
2. Lists with many small words
  * With smaller, especially single- or double-character "words", there are many more
    combinations to check.

---

## The Algorithm

I construct a graph of bytes to contain the list of words, and walk it in order to
determine if one word is a "subword" of another.

Using byte slices instead of strings allows for the ability to very quickly make
substrings out of full words, without incurring the cost of string manipulations.

It also enables me to very quickly determine if a word begins with another word on the
graph.  If I am looking for the word "food" on the graph and "foo" is also a word from the
list, then as I traverse the graph from 'f' to 'o' to 'o', there will be a marker there
which indicates that a word ends at that second 'o'.  I can continue the traversal to 'd',
but in the process I've discovered that "food" may be a compound word by virtue of the
fact that it begins with another.

It also means that I can quickly tell if a word is **not** a potential compound word, by
the contrapositive of that reasoning.

Building on that, and examining words in reverse order of their lengths made for a very
fast solution.

The comments in the code will provide a good deal more explanation of what's going on, but
that is the quick summary.

---
