package topkapi

import (
	"errors"
	"math"
	"sort"

	"github.com/dgryski/go-metro"
)

type Sketch struct {
	delta   float64
	epsilon float64
	l       uint64 // number of rows
	b       uint64 // think of this as the k
	cms     [][]uint64
	words   [][]string
}

// New creates a new Topkapi Sketch with given error rate and confidence.
// Accuracy guarantees will be made in terms of a pair of user specified parameters,
// ε and δ, meaning that the error in answering a query is within a factor of ε with
// probability δ
func New(delta, epsilon float64) (*Sketch, error) {
	if epsilon <= 0 || epsilon >= 1 {
		return nil, errors.New("topkapi: value of epsilon should be in range of (0, 1)")
	}
	if delta <= 0 || delta >= 1 {
		return nil, errors.New("topkapi: value of delta should be in range of (0, 1)")
	}

	var (
		b     = uint64(math.Ceil(1 / delta))
		l     = uint64(math.Log(2 / epsilon))
		cms   = make([][]uint64, l)
		words = make([][]string, l)
	)

	for i := range cms {
		cms[i] = make([]uint64, b)
		words[i] = make([]string, b)
	}

	return &Sketch{
		delta:   delta,
		epsilon: epsilon,
		l:       l,
		b:       b,
		cms:     cms,
		words:   words,
	}, nil
}

func (sk *Sketch) Update(key string, count uint64) {
	hsum := metro.Hash64Str(key, 1337)
	h1 := uint32(hsum & 0xffffffff)
	h2 := uint32((hsum >> 32) & 0xffffffff)

	for i := range sk.cms {
		hi := uint64((h1 + uint32(i)*h2)) % sk.b
		if sk.words[i][hi] == key {
			sk.cms[i][hi]++
		} else {
			sk.cms[i][hi]--
			if sk.cms[i][hi] <= 0 {
				sk.words[i][hi] = key
				sk.cms[i][hi] = 1
			}
		}
	}
}

func (sk *Sketch) Result(threshold uint64) []LocalHeavyHitter {
	var (
		seen = make(map[string]int)
		cs   = make([]LocalHeavyHitter, sk.b)
	)

	for i := range sk.words {
		for j, word := range sk.words[i] {
			count := sk.cms[i][j]
			idx, ok := seen[word]
			if !ok {
				cs = append(cs, LocalHeavyHitter{
					key:   word,
					count: count,
				})
				idx = len(cs)
				seen[word] = idx
				continue
			}
			if count > cs[idx].count {
				cs[idx].count = count
			}
		}
	}

	sort.Slice(cs, func(a, b int) bool {
		return cs[a].count < cs[b].count
	})

	return cs
}
