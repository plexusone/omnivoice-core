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
	byTerm := map[string]Pronunciation{}
	var terms []string
	for _, p := range ps {
		key := strings.ToLower(strings.TrimSpace(p.Term))
		if key == "" || (p.IPA == "" && p.Respelling == "") || (!phonemes && p.Respelling == "") {
			continue
		}
		if _, ok := byTerm[key]; !ok {
			terms = append(terms, key)
		}
		byTerm[key] = p
	}
	if len(terms) == 0 {
		return text
	}
	sort.SliceStable(terms, func(i, j int) bool { return len(terms[i]) > len(terms[j]) })
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = regexp.QuoteMeta(t)
	}
	// Alternatives are tried in order, so the longest term matches first.
	re := regexp.MustCompile(`(?i)` + strings.Join(quoted, "|"))

	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringIndex(text, -1) {
		start, end := m[0], m[1]
		if !wordEdge(text, start, end) {
			continue
		}
		written := text[start:end]
		p := byTerm[strings.ToLower(written)]
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
		last = end
	}
	b.WriteString(text[last:])
	return b.String()
}

// wordEdge reports whether text[start:end] isn't joined to a letter or
// digit on either side, in any script.
func wordEdge(text string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:start])
		if isWordRune(r) {
			return false
		}
	}
	if end < len(text) {
		r, _ := utf8.DecodeRuneInString(text[end:])
		if isWordRune(r) {
			return false
		}
	}
	return true
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r)
}

var xmlEscaper = strings.NewReplacer(`&`, "&amp;", `<`, "&lt;", `>`, "&gt;", `"`, "&quot;", `'`, "&apos;")

func escapeXML(s string) string { return xmlEscaper.Replace(s) }
