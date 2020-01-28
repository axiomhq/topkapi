package msgp

//go:generate msgp

// Sketch ...
type Sketch struct {
	L      uint64 // number of rows
	B      uint64 // think of this as the k
	Sparse map[string]uint64
	CMS    [][]uint64
	Counts [][]int64
	Words  [][]string
}
