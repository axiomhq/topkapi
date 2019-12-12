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

// epsilon: count should be within exact*epsilon range
// returns: probability that a sample in the sketch lies outside the error range (delta)
func errorRate(epsilon float64, exact, sketch map[string]uint64) float64 {
	var numOk, numBad int

	for w, wc := range sketch {
		exactwc, wcf := float64(exact[w]), float64(wc)
		lowerBound := exactwc * (1 - epsilon)
		upperBound := exactwc * (1 + epsilon)

		if wcf < lowerBound || wcf > upperBound {
			numBad++
			fmt.Printf("!! %s: %d not in range [%f, %f]\n", w, wc, lowerBound, upperBound)
		} else {
			numOk++
		}
	}

	return float64(numBad) / float64(len(sketch))
}

func sketchToMap(sk *Sketch) map[string]uint64 {
	res := make(map[string]uint64, 1024)
	for _, lhh := range sk.Result(1) {
		res[lhh.Key] = lhh.Count
	}

	return res
}

func assertErrorRate(t *testing.T, exact map[string]uint64, sk *Sketch, delta, epsilon float64) {
	sketch := sketchToMap(sk)
	effectiveEpsilon := errorRate(epsilon, exact, sketch)
	if effectiveEpsilon >= epsilon {
		t.Errorf("Expected error rate <= %f. Found %f", epsilon, effectiveEpsilon)
	}
}

// Just a sanity check that counts are == 1 when running on a small sample set without duplicates
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

func TestDeltaEpsilon(t *testing.T) {
	delta := 0.01
	epsilon := 0.05

	sketch, _ := New(delta, epsilon)
	words := loadWords()

	// Words in prime index positions are copied
	for _, p := range []int{2, 3, 5, 7, 11, 13, 17, 23} {
		for i := p; i < len(words); i += p {
			words[i] = words[p]
		}
	}

	for _, w := range words {
		sketch.Insert(w, 1)
	}

	exact := exactCount(words)

	for _, res := range sketch.Result(1)[:10] {
		fmt.Printf("%s=%d (exact: %d)\n", res.Key, res.Count, exact[res.Key])
	}

	assertErrorRate(t, exact, sketch, delta, epsilon)
}
