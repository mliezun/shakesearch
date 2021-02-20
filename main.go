package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

const defaultLineOffset = 5
const maxResults = 2000

func main() {
	searcher := Searcher{}
	err := searcher.Load("completeworks.txt")
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch(searcher))
	http.HandleFunc("/load", handleLoad(searcher))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Listening on port %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

//Searcher stores search structures
type Searcher struct {
	SuffixArray *suffixarray.Index
}

//SearchOptions stores search options
type SearchOptions struct {
	MatchCase            bool
	MatchWholeWord       bool
	UseRegularExpression bool
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}
		if len(query[0]) < 4 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("query should have at least 4 characters"))
			return
		}
		searchOpts := SearchOptions{}
		opts, ok := r.URL.Query()["opts"]
		if ok {
			if err := json.Unmarshal([]byte(opts[0]), &searchOpts); err != nil {
				w.WriteHeader(http.StatusUnprocessableEntity)
				w.Write([]byte(err.Error()))
				return
			}
		}
		results, err := searcher.Search(query[0], &searchOpts)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(err.Error()))
			return
		}
		if len(results) > maxResults {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("query is too broad, try something more specific"))
			return
		}
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		if err := enc.Encode(results); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

func handleLoad(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		kind, ok := r.URL.Query()["k"]
		if !ok || len(kind[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing kind in URL params"))
			return
		}
		if kind[0] != "p" && kind[0] != "n" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("kind not supported"))
			return
		}
		ix, ok := r.URL.Query()["ix"]
		if !ok || len(ix[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing index in URL params"))
			return
		}
		i, err := strconv.Atoi(ix[0])
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(err.Error()))
			return
		}
		lineOffset := defaultLineOffset
		limit, ok := r.URL.Query()["limit"]
		if ok && len(limit) >= 1 {
			l, err := strconv.Atoi(limit[0])
			if err == nil {
				lineOffset = l
			}
		}
		var lines []*Line
		if kind[0] == "p" {
			lines = searcher.PreviousLines(i, lineOffset)
		} else {
			lines = searcher.NextLines(i, lineOffset)
		}
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		if err := enc.Encode(lines); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

//Load load file to searcher
func (s *Searcher) Load(filename string) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	s.SuffixArray = suffixarray.New(dat)
	return nil
}

//Search executes a query with the given options.
//Returns pairs of indices of matched byte slices.
func (s *Searcher) Search(query string, options *SearchOptions) ([]MatchedSearch, error) {
	generatedQuery := ""
	if !options.MatchCase {
		generatedQuery += "(?i)"
	}
	if !options.UseRegularExpression {
		query = regexp.QuoteMeta(query)
	}
	if options.MatchWholeWord {
		generatedQuery += fmt.Sprintf("\\b%s\\b", query)
	} else {
		generatedQuery += query
	}
	r, err := regexp.Compile(generatedQuery)
	if err != nil {
		return nil, err
	}
	ixsPairs := s.SuffixArray.FindAllIndex(r, -1)
	out := make([]MatchedSearch, len(ixsPairs))
	for i, pair := range ixsPairs {
		mline := &MatchedLine{
			Line:              s.ReadLine(pair[0]),
			MatchedStartIndex: pair[0],
			MatchedEndIndex:   pair[1],
		}
		out[i] = MatchedSearch{
			Previous: s.PreviousLines(mline.StartIndex, defaultLineOffset),
			Matched:  mline,
			Next:     s.NextLines(mline.EndIndex, defaultLineOffset),
		}
	}
	return out, nil
}

//PreviousLines returns "lineOffset" previous lines from startIx
func (s *Searcher) PreviousLines(startIx, lineOffset int) []*Line {
	previous := make([]*Line, 0)
	for i := 0; i < lineOffset; i++ {
		ixStart := startIx - 2
		if len(previous) != 0 {
			ixStart = previous[0].StartIndex - 2
		}
		if ixStart <= 0 {
			break
		}
		previous = append([]*Line{s.ReadLine(ixStart)}, previous...)
	}
	return previous
}

//NextLines returns "lineOffset" next lines from endIx
func (s *Searcher) NextLines(endIx, lineOffset int) []*Line {
	next := make([]*Line, 0)
	for i := 0; i < lineOffset; i++ {
		ixEnd := endIx
		if len(next) != 0 {
			ixEnd = next[len(next)-1].EndIndex + 1
		}
		if ixEnd >= len(s.SuffixArray.Bytes()) {
			break
		}
		next = append(next, s.ReadLine(ixEnd))
	}
	return next
}

//MatchedSearch stores information about a matched search
type MatchedSearch struct {
	Previous []*Line
	Matched  *MatchedLine
	Next     []*Line
}

//MatchedLine stores information about a matched line
type MatchedLine struct {
	*Line
	MatchedStartIndex int
	MatchedEndIndex   int
}

//Line stores line information
type Line struct {
	StartIndex int
	EndIndex   int
	Content    []byte
}

//ReadLine read entire line containing from ix
func (s *Searcher) ReadLine(ix int) *Line {
	body := s.SuffixArray.Bytes()
	ixLineStart := 0
	ixLineEnd := len(body)
	for i := ix; i > 0; i-- {
		if body[i] == '\n' {
			ixLineStart = i + 1
			break
		}
	}
	for i := ix; i < len(body); i++ {
		if body[i] == '\n' {
			ixLineEnd = i
			break
		}
	}
	ixLineStart = min(ixLineStart, ixLineEnd)
	ixLineEnd = max(ixLineStart, ixLineEnd)
	return &Line{
		StartIndex: ixLineStart,
		EndIndex:   ixLineEnd,
		Content:    body[ixLineStart:ixLineEnd],
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
