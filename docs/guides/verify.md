# Verifying Wording Against Audio

The `stt/verify` package checks wording attributed to a recording, such as a
quotation from a podcast or the script of generated narration, by comparing
what several speech-to-text engines hear in a clip of it. Everything except
the engines' transcriptions is deterministic: the same hearings always give
the same result.

One engine's transcript is a weak check. Engines mishear names, new terms and
numbers, and a single transcript can't tell you which words it got wrong.
Several independent engines hearing the same short clip make the errors
visible: where they all hear the quoted words, the quotation stands; where
they disagree on a word that matters, a person or a judge has to listen.

## How a Quotation Is Checked

1. **Locate.** Find the quotation in a timed transcript of the whole
   recording, allowing for words that transcript misheard.
2. **Clip.** Cut the passage, with 1.5 seconds of padding on each side, as
   16 kHz mono WAV (`audio/clip`, which runs ffmpeg).
3. **Hear.** Have every engine transcribe the clip at once.
4. **Normalize.** Turn the quotation and each hearing into comparable
   tokens: lower case, no punctuation, contractions expanded ("don't" →
   "do not"), informal forms written out ("gonna" → "going to", "kinda" →
   "kind of"), number words as digits ("twenty five" → "25"), "%" as
   "percent", and fillers such as "um" dropped.
5. **Align.** Line up each hearing against the quotation word by word.
6. **Decide** by fixed rules.

## Decisions

| Decision | When |
|---|---|
| `agreed` | Every engine heard every quoted word, apart from function words where most engines agree with the quotation |
| `corrected` | The engines agree with each other but not with the quotation; `Corrected` holds what they heard |
| `disputed` | The engines disagree on a substantive word, or no engine heard the quotation's first or last words |
| `notFound` | The quotation could not be located in the recording |
| `insufficient` | Fewer engines than `MinEngines` (default 3) produced a transcript |

Each token has a kind, and the kind decides whether a difference matters:

| Kind | Examples | Substantive |
|---|---|---|
| `number` | 25, 10, 80 | Yes |
| `negation` | not, never, no, without | Yes |
| `term` | Terms you list, such as product names | Yes |
| `name` | Capitalized words not starting a sentence | Yes |
| `content` | Other words | Yes |
| `function` | the, a, of, is, and | No |

A substantive word has to be heard the same way by every engine. A simple
majority is not enough: two engines can share a weakness, such as a fast
phrase at the start of a clip, and outvote the one that heard it right.

## Alignment

Alignment is Gotoh's affine-gap method, with costs chosen for quotations:

- Words the speaker said that the quotation leaves out cost one edit to skip
  as a run, plus a little per word, because quotations often drop a phrase.
  Words heard across a marked elision ("…") cost next to nothing: less than
  any edit, but enough that the shortest matching stretch wins. A quotation
  with an elision therefore can't match a stretch that runs on to a later
  repeat of its last words.
- A misheard word costs less when the two words are spelled alike ("doing"
  and "do") than when they are not.
- Leaving out a quoted word costs most, because the quotation's words are
  what is being checked.
- A word said twice in a row ("to, to build") is a stutter and is ignored.
- A word an engine splits in two ("work flow", "a synchronous") is joined
  when that spells a quoted word, and a word an engine runs together
  ("speedups") is split when it spells two quoted words in a row.
- A word broken off as the speaker restarted, which engines write with a
  trailing hyphen ("ha- at heart"), is dropped unless it is a quoted word.

When no engine hears the quotation's first or last words, they may lie just
outside the clip. The verifier widens that edge and listens again, up to two
times, before reporting the quotation as disputed.

## Using the Package

```go
import (
    "github.com/plexusone/omnivoice-core/stt"
    "github.com/plexusone/omnivoice-core/stt/verify"
)

t, err := stt.LoadTranscript("episode.json") // a timed transcript of the whole recording
if err != nil {
    return err
}

v := verify.Verifier{
    Engines: []verify.Engine{
        {Name: "deepgram", Provider: deepgram},
        {Name: "elevenlabs", Provider: elevenlabs},
        {Name: "openai", Provider: openai, NoKeywords: true},
    },
    Options: verify.Options{
        Keywords: []string{"Bedrock", "Mantle"},
    },
}

report, err := v.VerifyQuotes(ctx, "episode.mp3", verify.WordsFromTranscript(t), "deepgram/nova-3",
    []verify.Quote{{ID: "q1", Text: "we can have the model run all the tests"}})
```

A quotation with `Start` and `End` set is clipped there instead of being
located.

### Key Words

Key words help engines with names and terms. Two rules keep them from
deciding the result:

- Never pass a quotation's own wording as key words.
- At least one engine runs without them. `TranscribeAll` returns
  `ErrAllEnginesUseKeywords` if key words are given and no engine has
  `NoKeywords` set.

### Names, Titles and Dates

Engines often get names wrong in the same way, so agreement on a name is
weak evidence. Confirm names, titles and dates from a written source.

## Comparing Without Audio

`Compare` works on text alone, for hearings you already have:

```go
c := verify.Compare("AI assisted agentic coding", []verify.Hearing{
    {Engine: "deepgram", Text: "AI assisted agentic coding"},
    {Engine: "elevenlabs", Text: "A.I. assisted agentic coding."},
    {Engine: "openai", Text: "ai-assisted agentic coding"},
}, verify.CompareOptions{})
// c.Decision == verify.DecisionAgreed
```

`CompareOptions.Aliases` accepts other ways of saying a term, so a script's
"UTM" is satisfied by a hearing of "U T M".

## Searching for What a Recording Doesn't Say

Before saying a recording doesn't mention something, search several
independent transcripts of it:

```go
res := verify.Search([]verify.IndexedTranscript{
    {Engine: "deepgram", Words: deepgramWords},
    {Engine: "openai", Words: openaiWords},
}, []string{"run the whole stack locally", "on my laptop"}, verify.AbsenceOptions{})
fmt.Println(res.Statement)
// None of the 2 phrases was found in 2 independent transcripts.
```

The result never says a phrase is absent from the recording, only whether
each transcript has it.

## Reports

`VerifyQuotes` returns a `Report`:

- the recording's path and SHA-256;
- the index transcript's engine;
- each engine's name, model and whether it had key words;
- the options;
- one result per quotation: where it was found, the clip's times, how often
  the clip was widened, and the comparison with every engine's hearing and
  each difference;
- counts per decision.

The report's JSON Schema is `schema.VerifyReportV1Schema`, generated from the
Go types. Times are in milliseconds.
