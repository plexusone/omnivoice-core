package tts

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Pronunciation tells a provider how to say a term.
type Pronunciation struct {
	// Term is the word or phrase as written in the text. It matches whole
	// words, ignoring case.
	Term string

	// IPA is the pronunciation in the International Phonetic Alphabet,
	// without slashes or brackets, such as "lamborˈɡiːni". Optional.
	IPA string

	// Respelling is text to speak instead of Term, such as
	// "lambor gheeny", for providers and models that read text only.
	// Optional.
	Respelling string
}

// ApplyPronunciations rewrites text before it's sent to a provider. With
// phonemes true, a term with IPA becomes an SSML phoneme element, such as
// <phoneme alphabet="ipa" ph="ˈmjura">Miura</phoneme>, keeping the text's
// own spelling. Otherwise, or when a term has no IPA, a term with a
// Respelling is replaced by it. Terms with neither are left as written.
//
// Longer terms win over shorter ones they contain, and rewritten text is
// never matched again. Only the inserted element is XML-escaped; the rest
// of the text is passed through as it is.
func ApplyPronunciations(text string, ps []Pronunciation, phonemes bool) string {
	byTerm := map[string]int{}
	var entries []Pronunciation
	for _, p := range ps {
		p.Term = strings.TrimSpace(p.Term)
		key := strings.ToLower(p.Term)
		if key == "" || (p.IPA == "" && p.Respelling == "") || (!phonemes && p.Respelling == "") {
			continue
		}
		if i, ok := byTerm[key]; ok {
			entries[i] = p
			continue
		}
		byTerm[key] = len(entries)
		entries = append(entries, p)
	}
	if len(entries) == 0 {
		return text
	}
	sort.SliceStable(entries, func(i, j int) bool { return len(entries[i].Term) > len(entries[j].Term) })

	// One group per term, longest first: alternatives are tried in order,
	// and the group that matched names the term, so case folding never has
	// to be repeated to find it.
	groups := make([]string, len(entries))
	for i, e := range entries {
		groups[i] = "(" + regexp.QuoteMeta(e.Term) + ")"
	}
	re := regexp.MustCompile(`(?i)` + strings.Join(groups, "|"))
	// anchored[i] matches entries[i] at the start of a string, for trying
	// shorter terms where a longer one isn't a whole word.
	anchored := make([]*regexp.Regexp, len(entries))
	anchoredAt := func(i int) *regexp.Regexp {
		if anchored[i] == nil {
			anchored[i] = regexp.MustCompile(`(?i)\A(?:` + regexp.QuoteMeta(entries[i].Term) + `)`)
		}
		return anchored[i]
	}

	var b strings.Builder
	last, pos := 0, 0
	for pos < len(text) {
		m := re.FindStringSubmatchIndex(text[pos:])
		if m == nil {
			break
		}
		start := pos + m[0]
		matched, end := -1, 0
		if startEdge(text, start) {
			for i := range entries {
				if m[2+2*i] >= 0 {
					matched, end = i, pos+m[1]
					break
				}
			}
			// The longest term here isn't a whole word, as "New York" in
			// "New Yorkshire"; try the shorter ones, as "New".
			for !endEdge(text, end) && matched >= 0 {
				next := -1
				for i := matched + 1; i < len(entries); i++ {
					if loc := anchoredAt(i).FindStringIndex(text[start:]); loc != nil {
						next, end = i, start+loc[1]
						break
					}
				}
				matched = next
			}
		}
		if matched < 0 {
			_, size := utf8.DecodeRuneInString(text[start:])
			pos = start + size
			continue
		}
		p, written := entries[matched], text[start:end]
		b.WriteString(text[last:start])
		if phonemes && p.IPA != "" {
			b.WriteString(`<phoneme alphabet="ipa" ph="`)
			b.WriteString(escapeXML(p.IPA))
			b.WriteString(`">`)
			b.WriteString(escapeXML(written))
			b.WriteString(`</phoneme>`)
		} else {
			b.WriteString(p.Respelling)
		}
		last, pos = end, end
	}
	b.WriteString(text[last:])
	return b.String()
}

// startEdge reports whether text[start:] doesn't begin inside a word: no
// letter, digit or mark, in any script, comes before it.
func startEdge(text string, start int) bool {
	if start == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(text[:start])
	return !isWordRune(r)
}

// endEdge reports whether text[:end] doesn't end inside a word.
func endEdge(text string, end int) bool {
	if end >= len(text) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[end:])
	return !isWordRune(r)
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
}

var xmlEscaper = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;", `'`, "&apos;")

func escapeXML(s string) string { return xmlEscaper.Replace(s) }
