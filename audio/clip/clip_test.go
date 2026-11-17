package clip

import (
	"bytes"
	"context"
	"encoding/binary"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestClip(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH")
	}
	src := filepath.Join(t.TempDir(), "tone.mp3")
	gen := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=3", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate: %v: %s", err, out)
	}
	wav, err := Clip(context.Background(), src, time.Second, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("not a WAV file")
	}
	if got := binary.LittleEndian.Uint32(wav[4:8]); int(got) != len(wav)-8 {
		t.Errorf("RIFF size %d, want %d", got, len(wav)-8)
	}
	if i := bytes.Index(wav, []byte("data")); i < 0 || int(binary.LittleEndian.Uint32(wav[i+4:i+8])) != len(wav)-i-8 {
		t.Errorf("data chunk size not set")
	}
	// One second of 16-bit mono audio at 16 kHz, plus the header.
	if want := SampleRate * 2; len(wav) < want || len(wav) > want+200 {
		t.Errorf("clip is %d bytes, want about %d", len(wav), want)
	}
}

func TestClipRejectsEmptyRange(t *testing.T) {
	if _, err := Clip(context.Background(), "x.mp3", 2*time.Second, time.Second); err == nil {
		t.Fatal("want an error for end before start")
	}
}

func TestFixWAVSizes(t *testing.T) {
	wav := []byte("RIFF\xff\xff\xff\xffWAVEfmt \x02\x00\x00\x00abdata\xff\xff\xff\xff1234")
	if err := fixWAVSizes(wav); err != nil {
		t.Fatal(err)
	}
	if int(binary.LittleEndian.Uint32(wav[4:8])) != len(wav)-8 || binary.LittleEndian.Uint32(wav[26:30]) != 4 {
		t.Errorf("sizes not fixed: % x", wav[:30])
	}
	if err := fixWAVSizes([]byte("not a wav file")); err == nil {
		t.Error("want an error for a non-WAV file")
	}
}
