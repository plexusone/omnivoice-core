// Package verify checks what speech-to-text engines heard against expected
// wording, such as a quote attributed to a recording or the script of
// generated narration.
//
// Everything in the package is deterministic: text is normalized into
// tokens, aligned word by word, and compared by fixed rules. Engines only
// supply the text they heard.
package verify

import (
	"strconv"
	"strings"
	"unicode"
)

// Kind classifies a token by how much a difference in it matters.
type Kind string

// Token kinds. Every kind except KindFunction is substantive.
const (
	KindNumber   Kind = "number"
	KindNegation Kind = "negation"
	KindTerm     Kind = "term"
	KindName     Kind = "name"
	KindFunction Kind = "function"
	KindContent  Kind = "content"
)

// Token is one normalized word.
type Token struct {
	// Text is the normalized form compared across texts.
	Text string `json:"text"`
	// Orig is the word as written, before normalization.
	Orig string `json:"orig"`
	// Word is the index of the original word the token came from. A word
	// can yield several tokens ("don't" yields "do" and "not") or none (a
	// filler).
	Word int `json:"word"`
	// Kind says how much a difference in the token matters.
	Kind Kind `json:"kind"`
	// GapAfter marks an elision ("…") after the token, where the recording
	// may hold words the text leaves out.
	GapAfter bool `json:"gap_after,omitempty"`

	// brk marks punctuation after the token, which ends a number phrase.
	brk bool
	// cut marks a word broken off as the speaker restarted, written with a
	// trailing hyphen ("ha-" before "heart").
	cut bool
	// end is the index of the last original word the token came from; it
	// differs from Word only for a number phrase spanning several words.
	end int
}

// Substantive reports whether a difference in the token changes meaning.
func (t Token) Substantive() bool {
	return t.Kind != KindFunction
}

// NormalizeOptions configures Normalize.
type NormalizeOptions struct {
	// Terms are names and terms to mark as KindTerm, such as product names.
	// Multi-word terms are matched as sequences.
	Terms []string
	// KeepFillers keeps filler sounds such as "um" and "uh".
	KeepFillers bool
}

// Normalize splits text into comparable tokens.
//
// It lower-cases, removes punctuation, expands contractions so negations are
// tokens, turns number words into digits, writes "%" as "percent", splits
// "10x" into "10" and "x", and drops fillers. Each token is given a Kind.
func Normalize(text string, opts NormalizeOptions) []Token {
	words := splitWords(text)
	for i := range words {
		words[i].src = i
	}
	return normalizeWords(words, opts)
}

// normalizeWords normalizes split words. A token's Word is its word's src,
// so words split from separate strings can keep their own indexes.
func normalizeWords(words []rawWord, opts NormalizeOptions) []Token {
	var toks []Token
	for _, w := range words {
		parts := expandWord(w.clean)
		for _, p := range parts {
			if p == "" {
				continue
			}
			if !opts.KeepFillers && fillers[p] {
				continue
			}
			toks = append(toks, Token{Text: p, Orig: w.orig, Word: w.src, end: w.src, cut: w.cut})
		}
		if w.elision && len(toks) > 0 {
			toks[len(toks)-1].GapAfter = true
		}
		if len(toks) > 0 && w.breakAfter {
			toks[len(toks)-1].brk = true
		}
	}
	toks = joinNumbers(toks)
	classify(toks, words, opts.Terms)
	return toks
}

type rawWord struct {
	src           int
	orig          string
	clean         string
	sentenceStart bool
	breakAfter    bool
	elision       bool
	cut           bool
}

func splitWords(text string) []rawWord {
	text = strings.NewReplacer(
		"’", "'", "‘", "'", "“", `"`, "”", `"`,
		"—", " ", "–", " ", "/", " ",
		"…", " … ", "...", " … ",
	).Replace(text)
	var out []rawWord
	start := true
	for _, f := range strings.Fields(text) {
		if f == "…" {
			if len(out) > 0 {
				out[len(out)-1].elision = true
				out[len(out)-1].breakAfter = true
			}
			continue
		}
		cut := strings.HasSuffix(strings.TrimRight(f, `"')],.;:!?`), "-")
		parts := splitHyphens(f)
		for k, h := range parts {
			c := cleanWord(h)
			if c == "" {
				continue
			}
			end := strings.TrimRight(h, `"')]`)
			w := rawWord{orig: h, clean: c, sentenceStart: start, cut: cut && k == len(parts)-1}
			last := ""
			if end != "" {
				last = end[len(end)-1:]
			}
			w.breakAfter = strings.ContainsAny(last, ",;:.!?")
			start = strings.ContainsAny(last, ".!?")
			out = append(out, w)
		}
	}
	return out
}

// splitHyphens splits "well-known" into words, keeping a leading minus sign
// on a number ("-5").
func splitHyphens(s string) []string {
	if !strings.Contains(s, "-") {
		return []string{s}
	}
	neg := negativeNumber(s)
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' })
	if neg && len(parts) > 0 {
		parts[0] = "-" + parts[0]
	}
	return parts
}

// negativeNumber reports whether s starts with a minus sign and a digit.
func negativeNumber(s string) bool {
	return len(s) > 1 && s[0] == '-' && s[1] >= '0' && s[1] <= '9'
}

// cleanWord lower-cases a word and strips surrounding punctuation, keeping
// apostrophes, decimal points and percent signs that belong to it.
func cleanWord(s string) string {
	s = strings.ToLower(s)
	neg := negativeNumber(strings.TrimLeft(s, `"'([`))
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '%'
	})
	if neg {
		s = "-" + s
	}
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '%':
			b.WriteRune(r)
		case r == '\'':
			b.WriteRune(r)
		case r == '-' && i == 0:
			b.WriteRune(r) // a minus sign kept above
		case (r == '.' || r == ',') && i > 0 && i < len(rs)-1 && unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1]):
			if r == '.' {
				b.WriteRune(r)
			} // a thousands comma is dropped: "1,000" becomes "1000"
		}
	}
	return b.String()
}

var fillers = map[string]bool{
	"um": true, "uh": true, "er": true, "erm": true, "ah": true,
	"hmm": true, "mm": true, "mhm": true, "uhm": true, "umm": true,
}

var contractions = map[string][]string{
	"won't": {"will", "not"}, "can't": {"can", "not"}, "cannot": {"can", "not"},
	"shan't": {"shall", "not"}, "ain't": {"is", "not"}, "let's": {"let", "us"},
	"i'm": {"i", "am"},
	// Informal spoken forms. Engines differ in whether they write what was
	// said ("gonna") or the standard form ("going to").
	"gonna": {"going", "to"}, "wanna": {"want", "to"}, "gotta": {"got", "to"},
	"kinda": {"kind", "of"}, "sorta": {"sort", "of"}, "ok": {"okay"},
}

var isWords = map[string]bool{
	"it": true, "that": true, "there": true, "here": true, "what": true,
	"who": true, "he": true, "she": true, "where": true, "how": true,
}

// expandWord splits one cleaned word into normalized parts.
func expandWord(w string) []string {
	if parts, ok := contractions[w]; ok {
		return parts
	}
	switch {
	case strings.HasSuffix(w, "n't"):
		return []string{strings.TrimSuffix(w, "n't"), "not"}
	case strings.HasSuffix(w, "'re"):
		return []string{strings.TrimSuffix(w, "'re"), "are"}
	case strings.HasSuffix(w, "'ve"):
		return []string{strings.TrimSuffix(w, "'ve"), "have"}
	case strings.HasSuffix(w, "'ll"):
		return []string{strings.TrimSuffix(w, "'ll"), "will"}
	case strings.HasSuffix(w, "'d"):
		return []string{strings.TrimSuffix(w, "'d"), "would"}
	case strings.HasSuffix(w, "'s"):
		base := strings.TrimSuffix(w, "'s")
		if isWords[base] {
			return []string{base, "is"}
		}
		return []string{base} // possessive
	}
	w = strings.ReplaceAll(w, "'", "")
	if strings.HasSuffix(w, "%") {
		return []string{strings.TrimSuffix(w, "%"), "percent"}
	}
	if w == "%" {
		return []string{"percent"}
	}
	// "10x" and "4.6x" split into the number and "x".
	if strings.HasSuffix(w, "x") && len(w) > 1 {
		if _, err := strconv.ParseFloat(strings.TrimSuffix(w, "x"), 64); err == nil {
			return []string{strings.TrimSuffix(w, "x"), "x"}
		}
	}
	return []string{w}
}

var units = map[string]int{
	"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6,
	"seven": 7, "eight": 8, "nine": 9, "ten": 10, "eleven": 11, "twelve": 12,
	"thirteen": 13, "fourteen": 14, "fifteen": 15, "sixteen": 16,
	"seventeen": 17, "eighteen": 18, "nineteen": 19,
}

var tens = map[string]int{
	"twenty": 20, "thirty": 30, "forty": 40, "fifty": 50, "sixty": 60,
	"seventy": 70, "eighty": 80, "ninety": 90,
}

var scales = map[string]int{"hundred": 100, "thousand": 1000, "million": 1000000, "billion": 1000000000}

func isNumberWord(s string) bool {
	_, u := units[s]
	_, t := tens[s]
	_, sc := scales[s]
	return u || t || sc
}

// joinNumbers replaces runs of number words with digits: "twenty five"
// becomes "25" and "a hundred" becomes "100". A run ends at punctuation, so
// "twenty, thirty" stays two numbers.
func joinNumbers(toks []Token) []Token {
	var out []Token
	for i := 0; i < len(toks); {
		j, val, ok := parseNumber(toks, i)
		if !ok {
			out = append(out, toks[i])
			i++
			continue
		}
		t := toks[i]
		t.Text = strconv.Itoa(val)
		t.Kind = KindNumber
		t.GapAfter = toks[j-1].GapAfter
		t.brk = toks[j-1].brk
		t.end = toks[j-1].end
		// Keep the whole phrase as written ("twenty five"), one copy per word.
		var orig []string
		last := -1
		for _, pt := range toks[i:j] {
			if pt.Word != last {
				orig = append(orig, pt.Orig)
				last = pt.Word
			}
		}
		t.Orig = strings.Join(orig, " ")
		out = append(out, t)
		i = j
	}
	return out
}

// parseNumber reads a number phrase starting at toks[i]. It returns the index
// after the phrase and its value.
func parseNumber(toks []Token, i int) (int, int, bool) {
	start := i
	if toks[i].Text == "a" && i+1 < len(toks) && scales[toks[i+1].Text] > 0 && !toks[i].brk {
		i++ // "a hundred"
	}
	if !isNumberWord(toks[i].Text) {
		return start, 0, false
	}
	total, current := 0, 0
	if toks[start].Text == "a" {
		current = 1
	}
	j := i
	for j < len(toks) {
		w := toks[j].Text
		switch {
		case units[w] > 0 || w == "zero":
			current += units[w]
		case tens[w] > 0:
			current += tens[w]
		case w == "hundred":
			if current == 0 {
				current = 1
			}
			current *= 100
		case scales[w] > 0:
			if current == 0 {
				current = 1
			}
			total += current * scales[w]
			current = 0
		case w == "and" && j > i && scales[toks[j-1].Text] > 0 && j+1 < len(toks) &&
			(units[toks[j+1].Text] > 0 || tens[toks[j+1].Text] > 0) && !toks[j-1].brk:
			// "one hundred and five"; "two and three" stays two numbers
		default:
			return j, total + current, true
		}
		j++
		if toks[j-1].brk {
			break
		}
		// Two units in a row ("five six") are separate numbers.
		if j < len(toks) && units[toks[j].Text] > 0 && units[toks[j-1].Text] > 0 {
			break
		}
		// A tens word after a unit ("one twenty") starts a new number.
		if j < len(toks) && tens[toks[j].Text] > 0 && (units[toks[j-1].Text] > 0 || tens[toks[j-1].Text] > 0) {
			break
		}
	}
	return j, total + current, true
}

var negations = map[string]bool{
	"not": true, "no": true, "never": true, "none": true, "nobody": true,
	"nothing": true, "neither": true, "nor": true, "without": true, "nowhere": true,
}

// functionWords are words whose substitution rarely changes meaning:
// articles, prepositions, conjunctions, auxiliaries, pronouns and discourse
// words. Quantifiers such as "more", "all" and "only" are not included.
var functionWords = toSet(`a an the and or but if then than so as because while
of in on at to for from by with about into onto over under up down out off through between after before during
is are was were be been being am do does did done have has had having
will would shall should can could may might must
i you he she it we they me him her us them my your his its our their mine yours ours theirs
this that these those there here what which who whom whose when where why how
just really very actually basically yeah yes oh okay ok well like kind sort`)

func toSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, f := range strings.Fields(s) {
		m[f] = true
	}
	return m
}

func classify(toks []Token, words []rawWord, terms []string) {
	for i := range toks {
		t := &toks[i]
		w := words[t.Word]
		switch {
		case isDigits(t.Text):
			t.Kind = KindNumber
		case negations[t.Text]:
			t.Kind = KindNegation
		case isName(w):
			t.Kind = KindName
		case functionWords[t.Text]:
			t.Kind = KindFunction
		default:
			t.Kind = KindContent
		}
	}
	for _, term := range terms {
		tt := Normalize(term, NormalizeOptions{KeepFillers: true})
		markSequence(toks, tt)
	}
}

func markSequence(toks []Token, seq []Token) {
	if len(seq) == 0 {
		return
	}
	for i := 0; i+len(seq) <= len(toks); i++ {
		match := true
		for k := range seq {
			if toks[i+k].Text != seq[k].Text {
				match = false
				break
			}
		}
		if match {
			for k := range seq {
				toks[i+k].Kind = KindTerm
			}
		}
	}
}

func isDigits(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isName(w rawWord) bool {
	if w.sentenceStart {
		return false
	}
	r := []rune(strings.TrimLeft(w.orig, `"'([`))
	if len(r) == 0 || !unicode.IsUpper(r[0]) {
		return false
	}
	low := strings.ToLower(w.clean)
	return low != "i" && !strings.HasPrefix(low, "i'")
}
