package verify

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func texts(toks []Token) string {
	out := make([]string, len(toks))
	for i, t := range toks {
		out[i] = t.Text
	}
	return strings.Join(out, " ")
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Who cares if it takes twenty, thirty minutes?", "who cares if it takes 20 30 minutes"},
		{"twenty-five people, a hundred times", "25 people 100 times"},
		{"one hundred and five", "105"},
		{"We don't know, um, it's 10x or 11.3x.", "we do not know it is 10 x or 11.3 x"},
		{"close to 80% of the cycle", "close to 80 percent of the cycle"},
		{"eighty percent", "80 percent"},
		{"1,000 commits", "1000 commits"},
		{"the team's energy", "the team energy"},
		{"won't, can't, cannot", "will not can not can not"},
		{"agentic — coding… later", "agentic coding later"},
		{"you're gonna, kinda, OK", "you are going to kind of okay"},
	}
	for _, tt := range tests {
		if got := texts(Normalize(tt.in, NormalizeOptions{})); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeKinds(t *testing.T) {
	toks := Normalize("Then Joe said it's not 25 builds of Project Mantle.", NormalizeOptions{Terms: []string{"Project Mantle"}})
	got := map[string]Kind{}
	for _, tk := range toks {
		got[tk.Text] = tk.Kind
	}
	want := map[string]Kind{
		"then":    KindFunction, // sentence start, so not a name
		"joe":     KindName,
		"not":     KindNegation,
		"25":      KindNumber,
		"builds":  KindContent,
		"of":      KindFunction,
		"project": KindTerm,
		"mantle":  KindTerm,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("kind of %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestNormalizeElision(t *testing.T) {
	toks := Normalize("build and test … catch bugs", NormalizeOptions{})
	if !toks[2].GapAfter || toks[1].GapAfter {
		t.Fatalf("gap after = %v", toks)
	}
}

func TestAlignDeterministic(t *testing.T) {
	ref := Normalize("a b c d", NormalizeOptions{KeepFillers: true})
	hyp := Normalize("a x c d e", NormalizeOptions{KeepFillers: true})
	want := []Pair{
		{0, 0, OpMatch}, {1, 1, OpSubstitute}, {2, 2, OpMatch}, {3, 3, OpMatch}, {-1, 4, OpInsert},
	}
	for i := 0; i < 3; i++ {
		if got := Align(ref, hyp); !reflect.DeepEqual(got, want) {
			t.Fatalf("Align = %v, want %v", got, want)
		}
	}
}

func TestAlignPrefersGapOverWrongWords(t *testing.T) {
	ref := Normalize("before they impact other engineers doing testing", NormalizeOptions{})
	hyp := Normalize("before they get into into a beta environment where where they could impact other engineers who do in testing and so", NormalizeOptions{})
	pairs, _ := AlignWithin(ref, hyp)
	matched := map[string]bool{}
	for _, p := range pairs {
		if p.Op == OpMatch {
			matched[ref[p.Ref].Text] = true
		}
	}
	for _, w := range []string{"impact", "other", "engineers", "testing"} {
		if !matched[w] {
			t.Errorf("%q not matched: %v", w, pairs)
		}
	}
}

func TestSimilar(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"doing", "do", true}, {"care", "cares", true}, {"workflow", "workflows", true}, {"gentle", "agentic", false}, {"impact", "get", false}} {
		if got := similar(c.a, c.b); got != c.want {
			t.Errorf("similar(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

func TestAlignWithinIgnoresPadding(t *testing.T) {
	ref := Normalize("run all the tests", NormalizeOptions{})
	hyp := Normalize("and we can have the model run all the tests run all the builds", NormalizeOptions{})
	pairs, dist := AlignWithin(ref, hyp)
	if dist != 0 || len(pairs) != 4 {
		t.Fatalf("dist %d pairs %v", dist, pairs)
	}
}

func words(s string, step time.Duration) []TimedWord {
	var out []TimedWord
	for i, w := range strings.Fields(s) {
		out = append(out, TimedWord{Text: w, Start: time.Duration(i) * step, End: time.Duration(i)*step + step/2})
	}
	return out
}

func TestLocate(t *testing.T) {
	ws := words("so one of the things we worked on is how do we actually speed up this workflow to the point where we can get an answer in a few minutes", time.Second)
	span, ok := Locate(ws, "speed up this work flow to the point where we can get an answer in a few minutes", NormalizeOptions{})
	if !ok {
		t.Fatal("not located")
	}
	if span.FirstWord != 13 || span.LastWord != len(ws)-1 {
		t.Errorf("span words %d..%d", span.FirstWord, span.LastWord)
	}
	if span.Score <= 0.8 || span.Score >= 1 {
		t.Errorf("score %v", span.Score)
	}
	if span.Start.Duration() != 13*time.Second {
		t.Errorf("start %v", span.Start)
	}
}

func hearings(texts ...string) []Hearing {
	names := []string{"deepgram", "elevenlabs", "openai", "whisper"}
	var hs []Hearing
	for i, tx := range texts {
		hs = append(hs, Hearing{Engine: names[i], Text: tx})
	}
	return hs
}

func TestCompare(t *testing.T) {
	quote := "who cares if it takes twenty, thirty minutes to build and test and run all the integration tests?"
	tests := []struct {
		name      string
		expected  string
		hearings  []Hearing
		opts      CompareOptions
		decision  Decision
		corrected string
		subst     bool
	}{
		{
			name:     "all engines heard it, with padding",
			expected: quote,
			hearings: hearings(
				"Who cares if it takes twenty, thirty minutes to, to build and test and run all the integration tests? But",
				"who cares if it takes 20, 30 minutes to to build and test and run all the integration tests",
				"so who cares if it takes 20 30 minutes to build and test and run all the integration tests",
			),
			decision: DecisionAgreed,
		},
		{
			name:     "one engine differs on a function word",
			expected: "run all the tests",
			hearings: hearings("run all the tests", "run all a tests", "run all the tests"),
			decision: DecisionAgreed,
		},
		{
			name:      "every engine heard another word",
			expected:  "AI assisted gentle coding",
			hearings:  hearings("AI assisted agentic coding", "A.I. assisted agentic coding.", "ai-assisted agentic coding"),
			decision:  DecisionCorrected,
			corrected: "AI assisted agentic coding",
			subst:     true,
		},
		{
			name:     "engines disagree on a number",
			expected: "a 25% increase",
			hearings: hearings("a 25% increase", "a 25 percent increase", "a 35% increase"),
			decision: DecisionDisputed,
		},
		{
			name:     "engines disagree on a negation",
			expected: "it was not tested",
			hearings: hearings("it was not tested", "it was tested", "it wasn't tested"),
			decision: DecisionDisputed,
		},
		{
			name:     "an engine heard an extra negation",
			expected: "it was tested",
			hearings: hearings("it was tested", "it was not tested", "it was tested"),
			decision: DecisionDisputed,
		},
		{
			name:     "too few engines succeeded",
			expected: "run all the tests",
			hearings: append(hearings("run all the tests", "run all the tests"), Hearing{Engine: "openai", Err: "timeout"}),
			decision: DecisionInsufficient,
		},
		{
			name:     "words heard across an elision",
			expected: "build and test … catch integration bugs",
			hearings: hearings(
				"build and test and run all the integration tests, catch integration bugs",
				"build and test and run all the integration tests catch integration bugs",
				"build and test and run the integration tests catch integration bugs",
			),
			decision: DecisionAgreed,
		},
		{
			name:     "an alias satisfies a spelled-out term",
			expected: "tag every link with UTM parameters",
			hearings: hearings("tag every link with U T M parameters", "tag every link with UTM parameters", "tag every link with u.t.m. parameters"),
			opts:     CompareOptions{Aliases: map[string][]string{"UTM": {"U T M", "u.t.m."}}},
			decision: DecisionAgreed,
		},
		{
			name:     "a name all engines misheard is corrected, flagged substantive",
			expected: "I'm Alicia with you",
			hearings: hearings("I'm Simon Elisha with you", "I'm Simon Elisha with you", "I'm Simon Elisha with you"),
			decision: DecisionCorrected,
			subst:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Compare(tt.expected, tt.hearings, tt.opts)
			if c.Decision != tt.decision {
				t.Fatalf("decision %s, want %s; differences %+v", c.Decision, tt.decision, c.Differences)
			}
			if tt.corrected != "" && c.Corrected != tt.corrected {
				t.Errorf("corrected %q, want %q", c.Corrected, tt.corrected)
			}
			if c.Decision == DecisionCorrected && c.CorrectedSubstantive != tt.subst {
				t.Errorf("substantive %v, want %v", c.CorrectedSubstantive, tt.subst)
			}
		})
	}
}

func TestCompareEdgeWordsAreNotCorrections(t *testing.T) {
	// The clip ends before the quote's last words: no engine heard them.
	c := Compare("before they get into production, before they impact other engineers",
		hearings("before they get into production before they", "before they get into production before they", "before they get into production before they"),
		CompareOptions{})
	if c.Decision != DecisionDisputed || c.MissingEnd != 3 || c.Corrected != "" {
		t.Fatalf("decision %s missing end %d corrected %q", c.Decision, c.MissingEnd, c.Corrected)
	}
}

func TestCompareIsDeterministic(t *testing.T) {
	hs := hearings("a 25% increase", "a 25 percent increase", "a 35% increase")
	first := Compare("a 25% increase", hs, CompareOptions{})
	for i := 0; i < 20; i++ {
		// Reverse the input order; the result must not change.
		rev := []Hearing{hs[2], hs[1], hs[0]}
		if got := Compare("a 25% increase", rev, CompareOptions{}); !reflect.DeepEqual(got, first) {
			t.Fatalf("result changed with input order")
		}
	}
}

func TestLocateMissingEdges(t *testing.T) {
	ws := words("before they get into production source code before they get into a beta environment where they could impact other engineers who are doing testing", time.Second)
	span, ok := Locate(ws, "before they get into production source code, before they impact other engineers doing testing", NormalizeOptions{})
	if !ok {
		t.Fatal("not located")
	}
	if span.MissingEnd == 0 && span.LastWord < len(ws)-1 {
		t.Errorf("span ends early at word %d with nothing reported missing", span.LastWord)
	}
}

// Regression cases from review.

func TestAlignInsertionRunAfterMatch(t *testing.T) {
	ref := Normalize("cat end", NormalizeOptions{})
	hyp := Normalize("cat cats junk end", NormalizeOptions{})
	if _, edits := AlignWithin(ref, hyp); edits != 1 {
		t.Errorf("edits %d, want one insertion run", edits)
	}
}

func TestNumberNormalizationEdges(t *testing.T) {
	for in, want := range map[string]string{
		"use two and three bolts": "use 2 and 3 bolts",
		"one hundred and five":    "105",
		"two thousand and ten":    "2010",
		"it fell to -5 degrees":   "it fell to -5 degrees",
		"a well-known -5% drop":   "a well known -5 percent drop",
	} {
		if got := texts(Normalize(in, NormalizeOptions{})); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
	toks := Normalize("twenty five red cars", NormalizeOptions{})
	if toks[0].Text != "25" || toks[0].Orig != "twenty five" || toks[0].Kind != KindNumber {
		t.Errorf("number token %+v", toks[0])
	}
}

func TestCompareNegativeNumber(t *testing.T) {
	c := Compare("it fell to -5 degrees", hearings("it fell to 5 degrees", "it fell to 5 degrees", "it fell to 5 degrees"), CompareOptions{})
	if c.Decision == DecisionAgreed {
		t.Errorf("-5 heard as 5 was agreed")
	}
}

func TestCompareKeepsNumberPhraseInCorrection(t *testing.T) {
	c := Compare("twenty five red cars", hearings("25 blue cars", "25 blue cars", "25 blue cars"), CompareOptions{})
	if c.Decision != DecisionCorrected || c.Corrected != "twenty five blue cars" {
		t.Errorf("decision %s corrected %q", c.Decision, c.Corrected)
	}
}

func TestCompareCorrectsInsideContraction(t *testing.T) {
	c := Compare("we don't ship", hearings("we do ship", "we do ship", "we do ship"), CompareOptions{})
	if c.Decision != DecisionCorrected || c.Corrected != "we do ship" {
		t.Errorf("decision %s corrected %q", c.Decision, c.Corrected)
	}
}

func TestCompareNoMajorityOnFunctionWordIsDisputed(t *testing.T) {
	c := Compare("we run tests", hearings("they run tests", "you run tests", "i run tests"), CompareOptions{})
	if c.Decision != DecisionDisputed {
		t.Errorf("decision %s, want disputed", c.Decision)
	}
}

func TestCompareDuplicateEngineNamesKeepEveryVote(t *testing.T) {
	hs := []Hearing{{Engine: "openai", Text: "run 35 tests"}, {Engine: "openai", Text: "run 35 tests"}, {Engine: "openai", Text: "run 25 tests"}}
	c := Compare("run 25 tests", hs, CompareOptions{})
	if c.Decision == DecisionAgreed {
		t.Errorf("two of three engines heard 35, but the result was agreed")
	}
}

func TestCompareNotFoundHasEmptyEngines(t *testing.T) {
	c := Compare("", nil, CompareOptions{MinEngines: 1})
	if c.Engines == nil {
		t.Error("engines is nil; reports would write null")
	}
}

func TestLocateNumberAcrossWords(t *testing.T) {
	ws := words("we used twenty five bolts today", time.Second)
	span, ok := Locate(ws, "we used twenty five bolts", NormalizeOptions{})
	if !ok || span.Score != 1 || span.LastWord != 4 {
		t.Errorf("span %+v", span)
	}
}

func TestCompareInformalFormsAgree(t *testing.T) {
	c := Compare("you're gonna have a lot more bugs", hearings(
		"you're gonna have a lot more bugs",
		"you're gonna have a lot more bugs",
		"you're going to have a lot more bugs",
	), CompareOptions{})
	if c.Decision != DecisionAgreed {
		t.Errorf("decision %s, want agreed: %+v", c.Decision, c.Differences)
	}
}

func TestCompareJoinsSplitWordAfterExpectedWord(t *testing.T) {
	// "a" is also in the quote, but "synchronous" is not, so the two heard
	// words are joined.
	c := Compare("a surprise is that asynchronous nature", hearings(
		"a surprise is that asynchronous nature",
		"a surprise is that asynchronous nature",
		"a surprise is that a synchronous nature",
	), CompareOptions{})
	if c.Decision != DecisionAgreed {
		t.Errorf("decision %s, want agreed: %+v", c.Decision, c.Differences)
	}
}

func TestCompareKeepsTwoExpectedWordsApart(t *testing.T) {
	// "any" and "way" are both quoted words, so a hearing of "any way" is not
	// joined into "anyway".
	c := Compare("anyway any way", hearings("anyway any way", "anyway any way", "anyway any way"), CompareOptions{})
	if c.Decision != DecisionAgreed {
		t.Errorf("decision %s, want agreed", c.Decision)
	}
}

func TestLocateElisionPrefersShortestStretch(t *testing.T) {
	// The quote's last word is said again much later. Words skipped at the
	// elision must not let the match run on to that later repeat.
	text := "it was more as a gimmick rather than something that would make me more more effective " +
		strings.Repeat("filler words go here ", 50) + "and that was effective"
	ws := words(text, time.Second)
	span, ok := Locate(ws, "more as … a gimmick rather than something that would make me more … effective", NormalizeOptions{})
	if !ok {
		t.Fatal("not located")
	}
	if span.FirstWord != 2 || span.LastWord != 15 {
		t.Errorf("span words %d..%d, want 2..15", span.FirstWord, span.LastWord)
	}
}

func TestCompareDropsCutWord(t *testing.T) {
	c := Compare("I'm a builder at heart", hearings(
		"I'm a builder at heart",
		"I'm a builder at ha- at heart",
		"I'm a builder at heart",
	), CompareOptions{})
	if c.Decision != DecisionAgreed {
		t.Errorf("decision %s, want agreed: %+v", c.Decision, c.Differences)
	}
}

func TestCompareKeepsCutWordThatIsQuoted(t *testing.T) {
	// "of-" is broken off, but "of" is a quoted word, so it still counts.
	c := Compare("it might run out of context", hearings(
		"it might run out of context",
		"it might run out of- context",
		"it might run out of context",
	), CompareOptions{})
	if c.Decision != DecisionAgreed {
		t.Errorf("decision %s, want agreed: %+v", c.Decision, c.Differences)
	}
}

func TestCompareSplitsWordHeardAsOne(t *testing.T) {
	c := Compare("you're not getting quite the speed ups", hearings(
		"you're not getting quite the speed ups",
		"you're not getting quite the speed ups",
		"you're not getting quite the speedups",
	), CompareOptions{})
	if c.Decision != DecisionAgreed {
		t.Errorf("decision %s, want agreed: %+v", c.Decision, c.Differences)
	}
}
