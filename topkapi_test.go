package topkapi

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func loadWords() []string {
	f, _ := os.Open("testdata/words.txt")
	defer f.Close()
	r := bufio.NewReader(f)

	res := make([]string, 0, 1024)
	for i := 0; ; i++ {
		if l, err := r.ReadString('\n'); err != nil {
			if err == io.EOF {
				return res
			}
			panic(err)
		} else {
			l = strings.Trim(l, "\r\n ")
			if len(l) > 0 {
				res = append(res, l)
			}
		}
	}

	return res
}

func exactCount(words []string) map[string]uint64 {
	m := make(map[string]uint64, len(words))
	for _, w := range words {
		if _, ok := m[w]; ok {
			m[w]++
		} else {
			m[w] = 1
		}
	}

	return m
}

// Just a sanity check that counts are == 1 when running on a small sample set
func TestSingleWordsInSmallSample(t *testing.T) {
	top10, _ := NewK(10)
	words := loadWords()[:100]

	for _, w := range words {
		top10.Insert(w, 1)
	}

	for _, res := range top10.Result(1) {
		if res.Count != 1 {
			t.Errorf("Bad count for '%s'. Expected %d found %d", res.Key, 1, res.Count)
		}
	}
}

func TestDoublePrimeIndexWords(t *testing.T) {
	top10, _ := NewK(10)
	words := loadWords()

	for _, p := range []int{2, 3, 5, 7, 11, 13, 17, 23} {
		for i := p; i < len(words); i += p {
			words[i] = words[p]
		}
	}

	for _, w := range words {
		top10.Insert(w, 1)
	}

	exact := exactCount(words)

	for _, res := range top10.Result(0)[:10] {
		if res.Count != exact[res.Key] {
			t.Errorf("Bad count for '%s'. Expected %d found %d", res.Key, exact[res.Key], res.Count)
		}
		fmt.Println(res)
	}
}
