package topkapi

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
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

func exactTop(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(a, b int) bool {
		return m[keys[a]] > m[keys[b]]
	})

	return keys
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
			//fmt.Printf("!! %s: %d not in range [%d, %d], epsilon=%f\n", w, wc, lowerBound, upperBound, epsilon)
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
	effectiveDelta := errorRate(epsilon, exact, sketch)
	if effectiveDelta >= delta {
		t.Errorf("Expected error rate <= %f. Found %f. Sketch size: %d", delta, effectiveDelta, len(sketch))
	}
}

func TestSingle(t *testing.T) {
	delta := 0.05
	topK := uint64(100)

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
	top := exactTop(exact)

	assertErrorRate(t, exact, sketch.Result(1), delta, sketch.Epsilon())
	//assertErrorRate(t, exact, sketch.Result(1)[:topK], delta, epsilon) // We would LOVE this to pass!

	// Assert order of heavy hitters in sub-sketch is as expected
	// TODO: by way of construction of test set we have pandemonium after #8, would like to check top[:topk]
	skTop := sketch.Result(1)
	for i, w := range top[:8] {
		if w != skTop[i].Key && exact[w] != skTop[i].Count {
			fmt.Println("key", w, exact[w])
			t.Errorf("Expected top %d/%d to be '%s'(%d) found '%s'(%d)", i, topK, w, exact[w], skTop[i].Key, skTop[i].Count)
		}
	}
}

func TestMerge2(t *testing.T) {
	delta := 0.01 // FIXME: tests fail for 0.03
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

	slices := split(words, 2)
	words1 := slices[0]
	words2 := slices[1]

	for _, w := range words1 {
		sketch1.Insert(w, 1)
	}

	for _, w := range words2 {
		sketch2.Insert(w, 1)
	}

	exact1 := exactCount(words1)
	exact2 := exactCount(words2)
	exactAll := exactCount(words)

	assertErrorRate(t, exact1, sketch1.Result(1), sketch1.Delta(), sketch1.Epsilon())
	assertErrorRate(t, exact2, sketch2.Result(1), sketch2.Delta(), sketch2.Epsilon())
	assertErrorRate(t, exactAll, sketch2.Result(1), sketch2.Delta(), sketch2.Epsilon()) // This should NOT PASS but it does
	//assertErrorRate(t, exact1, sketch1.Result(1)[:topK], sketch1.Delta(), sketch1.Epsilon()) // We would LOVE this to pass!
	//assertErrorRate(t, exact2, sketch2.Result(1)[:topK], sketch2.Delta(), sketch2.Epsilon()) // We would LOVE this to pass!

	if err := sketch1.Merge(sketch2); err != nil {
		t.Error("Merge failed")
	}

	for _, res := range sketch1.Result(1)[:topK] {
		fmt.Printf("%s=%d (%d)\n", res.Key, res.Count, exactAll[res.Key])
	}

	assertErrorRate(t, exactAll, sketch1.Result(1), sketch1.Delta(), sketch1.Epsilon()) // Should pass according to article, but does not
	//assertErrorRate(t, exactAll, sketch1.Result(1)[:topK], sketch1.Delta(), sketch1.Epsilon()) // We would LOVE this to pass!
}

func TestTheShebang(t *testing.T) {
	words := loadWords()

	// Words in prime index positions are copied
	for _, p := range []int{2, 3, 5, 7, 11, 13, 17, 23} {
		for i := p; i < len(words); i += p {
			words[i] = words[p]
		}
	}

	cases := []struct {
		name   string
		slices [][]string
		delta  float64
		topk   int
	}{
		{
			name:   "Single slice top20 d=0.01",
			slices: split(words, 1),
			delta:  0.01,
			topk:   20,
		},
		{
			name:   "Two slices top20 d=0.01",
			slices: split(words, 2),
			delta:  0.01,
			topk:   20,
		},
		{
			name:   "Three slices top20 d=0.01",
			slices: split(words, 3),
			delta:  0.01,
			topk:   20,
		},
		{
			name:   "100 slices top20 d=0.01",
			slices: split(words, 100),
			delta:  0.01,
			topk:   20,
		},
	}

	for _, cas := range cases {
		t.Run(cas.name, func(t *testing.T) {
			caseRunner(t, cas.slices, uint64(cas.topk), cas.delta)
		})
	}
}

func caseRunner(t *testing.T, slices [][]string, topk uint64, delta float64) {
	var sketches []*Sketch
	var corpusSize int

	// Find corpus size
	for _, slice := range slices {
		corpusSize += len(slice)
	}

	// Build sketches for each slice
	for _, slice := range slices {
		sk, err := NewTopK(topk, uint64(corpusSize), delta)
		if err != nil {
			panic(err)
		}
		for _, w := range slice {
			sk.Insert(w, 1)
		}
		exact := exactCount(slice)
		top := exactTop(exact)
		skTop := sk.Result(1)

		assertErrorRate(t, exact, sk.Result(1), sk.Delta(), sk.Epsilon())

		// Assert order of heavy hitters in sub-sketch is as expected
		// TODO: by way of construction of test set we have pandemonium after #8, would like to check top[:topk]
		for i, w := range top[:8] {
			if w != skTop[i].Key && exact[w] != skTop[i].Count {
				fmt.Println("key", w, exact[w])
				t.Errorf("Expected top %d/%d to be '%s'(%d) found '%s'(%d)", i, topk, w, exact[w], skTop[i].Key, skTop[i].Count)
			}
		}

		sketches = append(sketches, sk)
	}

	if len(slices) == 1 {
		return
	}

	// Compute exact stats for entire corpus
	var allSlice []string
	for _, slice := range slices {
		allSlice = append(allSlice, slice...)
	}
	exactAll := exactCount(allSlice)

	// Merge all sketches
	mainSketch := sketches[0]
	for _, sk := range sketches[1:] {
		mainSketch.Merge(sk)
		// TODO: it would be nice to incrementally check the error rates
	}
	assertErrorRate(t, exactAll, mainSketch.Result(1), mainSketch.Delta(), mainSketch.Epsilon())

	// Assert order of heavy hitters in final result is as expected
	// TODO: by way of construction of test set we have pandemonium after #8, would like to check top[:topk]
	top := exactTop(exactAll)
	skTop := mainSketch.Result(1)
	for i, w := range top[:8] {
		if w != skTop[i].Key {
			t.Errorf("Expected top %d/%d to be '%s'(%d) found '%s'(%d)", i, topk, w, exactAll[w], skTop[i].Key, skTop[i].Count)
		}
	}
}

// split and array of strings into n slices
func split(words []string, splits int) [][]string {
	l := len(words)
	step := l / splits

	slices := make([][]string, 0, splits)
	for i := 0; i < splits; i++ {
		if i == splits-1 {
			slices = append(slices, words[i*step:])
		} else {
			slices = append(slices, words[i*step:i*step+step])
		}
	}

	sanityCheck := 0
	for _, slice := range slices {
		sanityCheck += len(slice)
	}
	if sanityCheck != l {
		panic("Internal error")
	}
	if len(slices) != splits {
		panic(fmt.Sprintf("Num splits mismatch %d/%d", len(slices), splits))
	}

	return slices
}
