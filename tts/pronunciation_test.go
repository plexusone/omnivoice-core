package tts

import "testing"

func TestApplyPronunciations(t *testing.T) {
	ps := []Pronunciation{
		{Term: "Lamborghini", IPA: "lamborˈɡiːni", Respelling: "lambor gheeny"},
		{Term: "Ferruccio Lamborghini", IPA: "ferˈruttʃo lamborˈɡiːni"},
		{Term: "Murciélago", IPA: "muɾˈθjelaɣo"},
		{Term: "Miura", Respelling: "mee-OO-rah"},
		{Term: "Countach"},
	}
	cases := []struct {
		name, text, want string
		phonemes         bool
	}{
		{"ipa", "Tractors made Lamborghini wealthy.", `Tractors made <phoneme alphabet="ipa" ph="lamborˈɡiːni">Lamborghini</phoneme> wealthy.`, true},
		{"respelling", "Tractors made Lamborghini wealthy.", "Tractors made lambor gheeny wealthy.", false},
		{"longest term first", "Welcome to Ferruccio Lamborghini.", `Welcome to <phoneme alphabet="ipa" ph="ferˈruttʃo lamborˈɡiːni">Ferruccio Lamborghini</phoneme>.`, true},
		{"keeps the text's case", "LAMBORGHINI", `<phoneme alphabet="ipa" ph="lamborˈɡiːni">LAMBORGHINI</phoneme>`, true},
		{"non-ASCII term", "The Murciélago, then.", `The <phoneme alphabet="ipa" ph="muɾˈθjelaɣo">Murciélago</phoneme>, then.`, true},
		{"respelling without IPA", "The Miura.", "The mee-OO-rah.", true},
		{"no IPA and no respelling", "The Countach.", "The Countach.", true},
		{"IPA only, text mode", "The Murciélago.", "The Murciélago.", false},
		{"whole words only", "Lamborghinis and xLamborghini", "Lamborghinis and xLamborghini", true},
		{"accented neighbour", "éLamborghini", "éLamborghini", true},
		{"not matched twice", "Lamborghini Lamborghini", "lambor gheeny lambor gheeny", false},
	}
	for _, tc := range cases {
		if got := ApplyPronunciations(tc.text, ps, tc.phonemes); got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

func TestApplyPronunciationsEscapes(t *testing.T) {
	got := ApplyPronunciations(`Say "Q&A"`, []Pronunciation{{Term: "Q&A", IPA: `kjuː"ænd"eɪ`}}, true)
	want := `Say "<phoneme alphabet="ipa" ph="kjuː&quot;ænd&quot;eɪ">Q&amp;A</phoneme>"`
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestApplyPronunciationsNone(t *testing.T) {
	if got := ApplyPronunciations("unchanged", nil, true); got != "unchanged" {
		t.Errorf("got %q", got)
	}
}

func TestApplyPronunciationsWordEdges(t *testing.T) {
	cases := []struct {
		name, text, want string
		ps               []Pronunciation
	}{
		{
			"shorter term where the longest isn't a whole word",
			"New Yorkshire and New York",
			"Noo Yorkshire and NY",
			[]Pronunciation{{Term: "New York", Respelling: "NY"}, {Term: "New", Respelling: "Noo"}},
		},
		{
			"no shorter term fits",
			"New Yorkshire",
			"New Yorkshire",
			[]Pronunciation{{Term: "New York", Respelling: "NY"}},
		},
		{
			"case folding beyond lowercase",
			"ΝΊΚΟΣ and Νίκος",
			"Nikos and Nikos",
			[]Pronunciation{{Term: "Νίκος", Respelling: "Nikos"}},
		},
		{
			"spacing vowel mark continues a word",
			"रामा and राम",
			"रामा and raam",
			[]Pronunciation{{Term: "राम", Respelling: "raam"}},
		},
		{
			"term ending in a vowel mark",
			"रामा said",
			"raamaa said",
			[]Pronunciation{{Term: "राम", Respelling: "raam"}, {Term: "रामा", Respelling: "raamaa"}},
		},
	}
	for _, tc := range cases {
		if got := ApplyPronunciations(tc.text, tc.ps, false); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}
}
