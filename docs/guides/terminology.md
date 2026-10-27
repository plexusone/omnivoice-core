# Terminology

Package `terminology` adapts the canonical term IR from
[`terminology-spec`](https://github.com/plexusone/terminology-spec) into the two
text-transform primitives every TTS/subtitle consumer of OmniVoice needs. It is
the **single shared implementation** of that substitution engine — consumers
(videoascode and others) should use it instead of reimplementing a regex-based
term matcher.

## The two primitives

| Primitive | When it runs | What it fixes |
|-----------|--------------|---------------|
| **`Pronouncer`** | Before TTS synthesis | The *spoken* form of a term, per BCP-47 locale — so the audio says "AAuth" as "ay auth" |
| **`CaseCorrector`** | After STT (subtitle text) | The *written/displayed* form of a term, independent of locale — so subtitles read "PlexusOne", not "plexus one" |

The two are deliberately opposite: `Pronouncer` changes what a term *sounds*
like without touching the displayed spelling; `CaseCorrector` changes what a
term *looks* like in transcribed text.

## Term sources

```go
import "github.com/plexusone/omnivoice-core/terminology"

// Embedded generic-industry terms (~237: AI/ML, companies, dev tools,
// languages, frameworks, cloud/infra) — no filesystem access required.
terms, _ := terminology.Builtin()

// Hand-authored canonical terms with real pronunciation/translation data
// (e.g. protocol/standard names like "AAuth", "OAuth").
curated, _ := terminology.Curated()

// Builtin merged with Curated (Curated wins on ID collision).
all, _ := terminology.All()

// Or load a project's own canonical terms/*.json directory.
custom, _ := terminology.LoadDir("terms")
```

## Usage

```go
// Pronunciation: rewrite spoken forms before sending text to a TTS provider.
p := terminology.NewPronouncer(all)
spoken := p.Apply("Sign in with AAuth", "en-US") // -> "Sign in with ay auth"

// Case correction: fix displayed forms in STT-derived subtitle text.
c := terminology.NewCaseCorrector(all)
fixed := c.Apply("sign in with aauth")           // -> "sign in with AAuth"
```

`Term` is re-exported from `terminology-spec`, so consumers don't need a second
direct import just to name the type.

## Matching semantics

- Case-insensitive with word boundaries.
- Multi-word terms are matched before shorter overlapping ones, so a longer term
  isn't shadowed by a shorter one nested inside it.
- Terms with no entry for the requested locale are left unchanged.

## See also

- [Local TTS/STT Providers](local-tts.md) — where pronunciation is applied on the
  synthesis side.
- videoascode's Pronunciation guide — the config-driven consumer of `Pronouncer`.
