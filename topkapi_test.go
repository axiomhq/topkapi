package topkapi

import (
	"bufio"
	"fmt"
	"io"
	"math"
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
		exactwc := float64(exact[w])
		lowerBound := uint64(math.Floor(exactwc * (1 - epsilon)))
		upperBound := uint64(math.Ceil(exactwc * (1 + epsilon)))

		if wc < lowerBound || wc > upperBound {
			numBad++
			//fmt.Printf("!! %s: %d not in range [%d, %d]\n", w, wc, lowerBound, upperBound)
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
		t.Errorf("Expected error rate <= %f. Found %f. Sketch size: %d", epsilon, effectiveEpsilon, len(sketch))
	}
}

func TestDeltaEpsilon(t *testing.T) {
	delta := 0.01
	epsilon := 0.05

	words := loadWords()

	// Words in prime index positions are copied
	for _, p := range []int{2, 3, 5, 7, 11, 13, 17, 23} {
		for i := p; i < len(words); i += p {
			words[i] = words[p]
		}
	}

	sketch, _ := NewTopK(10, uint64(len(words)), delta) //New(delta, epsilon)

	for _, w := range words {
		sketch.Insert(w, 1)
	}

	exact := exactCount(words)

	for _, res := range sketch.Result(1)[:10] {
		fmt.Printf("%s=%d (exact: %d)\n", res.Key, res.Count, exact[res.Key])
	}

	assertErrorRate(t, exact, sketch, delta, epsilon)
}
