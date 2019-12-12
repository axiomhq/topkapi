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

func resultToMap(result []LocalHeavyHitter) map[string]uint64 {
	res := make(map[string]uint64, len(result))
	for _, lhh := range result {
		res[lhh.Key] = lhh.Count
	}

	return res
}

func assertErrorRate(t *testing.T, exact map[string]uint64, result []LocalHeavyHitter, delta, epsilon float64) {
	t.Helper() // Indicates to the testing framework that this is a helper func to skip in stack traces
	sketch := resultToMap(result)
	effectiveEpsilon := errorRate(epsilon, exact, sketch)
	if effectiveEpsilon >= epsilon {
		t.Errorf("Expected error rate <= %f. Found %f. Sketch size: %d", epsilon, effectiveEpsilon, len(sketch))
	}
}

func TestDeltaEpsilon(t *testing.T) {
	delta := 0.01
	epsilon := 0.05
	topK := uint64(10)

	words := loadWords()

	// Words in prime index positions are copied
	for _, p := range []int{2, 3, 5, 7, 11, 13, 17, 23} {
		for i := p; i < len(words); i += p {
			words[i] = words[p]
		}
	}

	sketch, _ := NewTopK(topK, uint64(len(words)), delta)

	for _, w := range words {
		sketch.Insert(w, 1)
	}

	exact := exactCount(words)

	assertErrorRate(t, exact, sketch.Result(1), delta, epsilon)
	//assertErrorRate(t, exact, sketch.Result(1)[:topK], delta, epsilon) // We would LOVE this to pass!
}

func TestMerge2(t *testing.T) {
	delta := 0.01
	epsilon := 0.05
	topK := uint64(20)

	words := loadWords()

	// Words in prime index positions are copied
	for _, p := range []int{2, 3, 5, 7, 11, 13, 17, 23} {
		for i := p; i < len(words); i += p {
			words[i] = words[p]
		}
	}

	sketch1, _ := NewTopK(topK, uint64(len(words)), delta) //New(delta, epsilon)
	sketch2, _ := NewTopK(topK, uint64(len(words)), delta) //New(delta, epsilon)

	split := len(words) / 2
	words1 := words[:split]
	words2 := words[split:]

	for _, w := range words1 {
		sketch1.Insert(w, 1)
	}

	for _, w := range words2 {
		sketch2.Insert(w, 1)
	}

	exact1 := exactCount(words1)
	exact2 := exactCount(words2)
	exactAll := exactCount(words)

	assertErrorRate(t, exact1, sketch1.Result(1), delta, epsilon)
	assertErrorRate(t, exact2, sketch2.Result(1), delta, epsilon)
	assertErrorRate(t, exactAll, sketch2.Result(1), delta, epsilon) // This should NOT PASS but it does
	//assertErrorRate(t, exact1, sketch1.Result(1)[:topK], delta, epsilon) // We would LOVE this to pass!
	//assertErrorRate(t, exact2, sketch2.Result(1)[:topK], delta, epsilon) // We would LOVE this to pass!

	if err := sketch1.Merge(sketch2); err != nil {
		t.Error("Merge failed")
	}

	for _, res := range sketch1.Result(1)[:topK] {
		fmt.Printf("%s=%d (exact: %d)\n", res.Key, res.Count, exactAll[res.Key])
	}

	assertErrorRate(t, exactAll, sketch1.Result(1), delta, epsilon)        // Should pass according to article, but does not
	assertErrorRate(t, exactAll, sketch1.Result(1)[:topK], delta, epsilon) // We would LOVE this to pass!
}
