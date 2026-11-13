package terminology

import "testing"

func TestPronouncer_Apply(t *testing.T) {
	p := NewPronouncer([]Term{
		{CanonicalForm: "AAuth", Pronunciations: map[string]string{"en-US": "ay auth"}},
	})

	got := p.Apply("Welcome to AAuth today.", "en-US")
	want := "Welcome to ay auth today."
	if got != want {
		t.Errorf("Apply() = %q, want %q", got, want)
	}
}

func TestPronouncer_NoEntryForLocale(t *testing.T) {
	p := NewPronouncer([]Term{
		{CanonicalForm: "AAuth", Pronunciations: map[string]string{"en-US": "ay auth"}},
	})

	text := "Bienvenue à AAuth."
	if got := p.Apply(text, "fr-FR"); got != text {
		t.Errorf("Apply() with no fr-FR entry = %q, want unchanged %q", got, text)
	}
}

func TestPronouncer_NilReceiver(t *testing.T) {
	var p *Pronouncer
	text := "unchanged"
	if got := p.Apply(text, "en-US"); got != text {
		t.Errorf("nil Pronouncer.Apply() = %q, want %q", got, text)
	}
}

func TestCaseCorrector_Correct(t *testing.T) {
	cc := NewCaseCorrector([]Term{
		{CanonicalForm: "OpenAI"},
		{CanonicalForm: "AAuth", Aliases: []string{"A-Auth"}},
	})

	got := cc.Correct("openai builds tools; a-auth is a protocol.")
	want := "OpenAI builds tools; AAuth is a protocol."
	if got != want {
		t.Errorf("Correct() = %q, want %q", got, want)
	}
}

func TestCaseCorrector_NilReceiver(t *testing.T) {
	var cc *CaseCorrector
	text := "unchanged"
	if got := cc.Correct(text); got != text {
		t.Errorf("nil CaseCorrector.Correct() = %q, want %q", got, text)
	}
}

func TestBuiltinCorrections(t *testing.T) {
	corrections, err := BuiltinCorrections()
	if err != nil {
		t.Fatalf("BuiltinCorrections() error = %v", err)
	}
	if corrections["openai"] != "OpenAI" {
		t.Errorf(`corrections["openai"] = %q, want "OpenAI"`, corrections["openai"])
	}
}

func TestMerge_ReExport(t *testing.T) {
	base := []Term{{ID: "x", CanonicalForm: "X"}}
	override := []Term{{ID: "x", CanonicalForm: "X2"}}
	merged := Merge(base, override)
	if len(merged) != 1 || merged[0].CanonicalForm != "X2" {
		t.Errorf("Merge() = %+v, want single term with CanonicalForm X2", merged)
	}
}

func TestPronunciationsFor(t *testing.T) {
	terms := []Term{
		{CanonicalForm: "Lamborghini", Aliases: []string{"Lambo"}, IPA: map[string]string{"en-US": "lamborˈɡiːni"}, Pronunciations: map[string]string{"en-US": "lambor gheeny"}},
		{CanonicalForm: "AAuth", Pronunciations: map[string]string{"fr-FR": "a auth"}},
	}
	got := PronunciationsFor(terms, "en-US")
	if len(got) != 2 || got[0].Term != "Lamborghini" || got[1].Term != "Lambo" ||
		got[1].IPA != "lamborˈɡiːni" || got[1].Respelling != "lambor gheeny" {
		t.Errorf("PronunciationsFor = %+v", got)
	}
}
