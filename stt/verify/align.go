package verify

import "math"

// Op is one step of an alignment.
type Op string

// Alignment operations.
const (
	OpMatch      Op = "match"
	OpSubstitute Op = "substitute"
	OpDelete     Op = "delete" // a reference token the hypothesis lacks
	OpInsert     Op = "insert" // a hypothesis token the reference lacks
)

// Pair is one aligned position: a reference token, a hypothesis token, or
// both. An index of -1 means the side has no token at this position.
type Pair struct {
	Ref int `json:"ref"`
	Hyp int `json:"hyp"`
	Op  Op  `json:"op"`
}

// Alignment costs, in tenths of an edit. A run of extra hypothesis words
// costs one edit to open and a little for each further word, because a
// quotation often leaves out a phrase: five quoted words should not be
// matched against five wrong words just to avoid skipping a gap. A
// substitution costs less when the two words are spelled alike ("doing" and
// "do"), as when a word is misheard, and more when they are not. Leaving out
// a quoted word costs most, since the quotation's words are what is being
// checked.
const (
	costSubSimilar = 6
	costSub        = 14
	costDel        = 20
	costInsOpen    = 10
	costInsExtend  = 3
)

// subCost is the cost of aligning reference word a with hypothesis word b.
func subCost(a, b string) int {
	switch {
	case a == b:
		return 0
	case similar(a, b):
		return costSubSimilar
	default:
		return costSub
	}
}

// similar reports whether two different words are spelled alike: one starts
// with the other (at least two letters), or they differ in at most a third
// of the longer word's letters.
func similar(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	short, long := ra, rb
	if len(short) > len(long) {
		short, long = long, short
	}
	if len(short) >= 2 && string(long[:len(short)]) == string(short) {
		return true
	}
	return charDistance(ra, rb)*3 <= len(long)
}

func charDistance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			c := 1
			if a[i-1] == b[j-1] {
				c = 0
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+c)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// Align aligns two token sequences word by word at the lowest cost.
//
// Ties are broken in a fixed order (match or substitute, then delete, then
// insert), so the same input always gives the same alignment.
func Align(ref, hyp []Token) []Pair {
	pairs, _ := align(ref, hyp, false)
	return pairs
}

// AlignWithin aligns ref against the best-matching stretch of hyp, ignoring
// hypothesis tokens before and after it. Use it when the hypothesis covers
// more than the reference, such as a padded clip around a quote. It returns
// the alignment of that stretch and its number of edits: substitutions,
// deletions and runs of insertions, where words heard across a marked
// elision cost nothing.
func AlignWithin(ref, hyp []Token) ([]Pair, int) {
	return align(ref, hyp, true)
}

const (
	stM = iota // last step matched or substituted
	stD        // last step deleted a reference token
	stI        // last step inserted a hypothesis token
)

const inf = math.MaxInt32 / 4

// align runs Gotoh's affine-gap alignment. With free ends, leading and
// trailing hypothesis tokens cost nothing and are left out of the result.
func align(ref, hyp []Token, freeEnds bool) ([]Pair, int) {
	m, n := len(ref), len(hyp)
	var t [3][][]int
	for s := range t {
		t[s] = make([][]int, m+1)
		for i := range t[s] {
			t[s][i] = make([]int, n+1)
			for j := range t[s][i] {
				t[s][i][j] = inf
			}
		}
	}
	insCost := func(i int, extend bool) int {
		if i > 0 && ref[i-1].GapAfter {
			return 0 // words heard across a marked elision are expected
		}
		if extend {
			return costInsExtend
		}
		return costInsOpen
	}
	t[stM][0][0] = 0
	for j := 1; j <= n; j++ {
		if freeEnds {
			t[stM][0][j] = 0 // the stretch may start anywhere
		} else {
			t[stI][0][j] = minInt(t[stM][0][j-1]+insCost(0, false), t[stI][0][j-1]+insCost(0, true))
		}
	}
	for i := 1; i <= m; i++ {
		t[stD][i][0] = minInt(t[stM][i-1][0], t[stD][i-1][0]) + costDel
		for j := 1; j <= n; j++ {
			sub := subCost(ref[i-1].Text, hyp[j-1].Text)
			t[stM][i][j] = min3(t[stM][i-1][j-1], t[stD][i-1][j-1], t[stI][i-1][j-1]) + sub
			t[stD][i][j] = min3(t[stM][i-1][j], t[stD][i-1][j], t[stI][i-1][j]) + costDel
			t[stI][i][j] = min3(
				t[stM][i][j-1]+insCost(i, false),
				t[stI][i][j-1]+insCost(i, true),
				t[stD][i][j-1]+insCost(i, false),
			)
		}
	}
	best := func(i, j int) (int, int) {
		st, v := stM, t[stM][i][j]
		if t[stD][i][j] < v {
			st, v = stD, t[stD][i][j]
		}
		if t[stI][i][j] < v {
			st, v = stI, t[stI][i][j]
		}
		return st, v
	}
	endJ := n
	if freeEnds {
		// The lowest cost; on a tie, the latest end, which covers more of
		// the reference with matches instead of deleting its last tokens.
		_, bv := best(m, 0)
		endJ = 0
		for j := 1; j <= n; j++ {
			if _, v := best(m, j); v <= bv {
				bv, endJ = v, j
			}
		}
	}

	var rev []Pair
	i, j := m, endJ
	st, _ := best(i, j)
	for i > 0 || (j > 0 && !freeEnds) {
		switch st {
		case stM:
			op, sub := OpMatch, subCost(ref[i-1].Text, hyp[j-1].Text)
			if sub > 0 {
				op = OpSubstitute
			}
			rev = append(rev, Pair{Ref: i - 1, Hyp: j - 1, Op: op})
			v := t[stM][i][j] - sub
			i, j = i-1, j-1
			st = prevState(t, i, j, v, [3]int{})
		case stD:
			rev = append(rev, Pair{Ref: i - 1, Hyp: -1, Op: OpDelete})
			v := t[stD][i][j] - costDel
			i--
			st = prevState(t, i, j, v, [3]int{})
		default:
			rev = append(rev, Pair{Ref: -1, Hyp: j - 1, Op: OpInsert})
			v := t[stI][i][j]
			j--
			// Step costs by state: from a match or a deletion an insertion
			// opens a run; from an insertion it extends one.
			st = prevState(t, i, j, v, [3]int{insCost(i, false), insCost(i, false), insCost(i, true)})
		}
	}
	out := make([]Pair, len(rev))
	for k := range rev {
		out[k] = rev[len(rev)-1-k]
	}
	return out, countEdits(out, ref)
}

// prevState returns the state at (i, j) whose cost plus its step cost gives
// v, preferring match, then delete, then insert.
func prevState(t [3][][]int, i, j, v int, step [3]int) int {
	for _, st := range []int{stM, stD, stI} {
		if t[st][i][j] < inf && t[st][i][j]+step[st] == v {
			return st
		}
	}
	// Unreachable for a consistent table; fall back to the cheapest state.
	st, best := stM, t[stM][i][j]
	if t[stD][i][j] < best {
		st, best = stD, t[stD][i][j]
	}
	if t[stI][i][j] < best {
		st = stI
	}
	return st
}

// countEdits counts substitutions, deletions and runs of insertions, not
// counting insertions across a marked elision.
func countEdits(pairs []Pair, ref []Token) int {
	edits := 0
	inRun := false
	lastRef := -1
	for _, p := range pairs {
		switch p.Op {
		case OpSubstitute, OpDelete:
			edits++
			inRun = false
		case OpInsert:
			if lastRef >= 0 && ref[lastRef].GapAfter {
				continue
			}
			if !inRun {
				edits++
				inRun = true
			}
		default:
			inRun = false
		}
		if p.Ref >= 0 {
			lastRef = p.Ref
		}
	}
	return edits
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min3(a, b, c int) int {
	return minInt(minInt(a, b), c)
}
