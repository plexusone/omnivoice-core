// Package clip cuts a stretch of a recording into audio suitable for
// speech-to-text: 16 kHz mono WAV.
//
// It runs ffmpeg, which must be on PATH.
package clip

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// SampleRate is the sample rate of clips, in Hz.
const SampleRate = 16000

// ErrNoFFmpeg is returned when ffmpeg is not on PATH.
var ErrNoFFmpeg = errors.New("clip: ffmpeg not found on PATH")

// Clip returns the audio of path from start to end as 16 kHz mono WAV.
// A start before zero is treated as zero.
func Clip(ctx context.Context, path string, start, end time.Duration) ([]byte, error) {
	if start < 0 {
		start = 0
	}
	if end <= start {
		return nil, fmt.Errorf("clip: end %v is not after start %v", end, start)
	}
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, ErrNoFFmpeg
	}
	args := []string{
		"-v", "error", "-nostdin",
		"-ss", seconds(start),
		"-i", path,
		"-t", seconds(end - start),
		"-ac", "1", "-ar", strconv.Itoa(SampleRate),
		"-map_metadata", "-1", "-fflags", "+bitexact",
		"-f", "wav", "pipe:1",
	}
	var out, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bin, args...) //nolint:gosec // fixed binary; path is passed as one argument, never through a shell
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("clip: ffmpeg: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	if out.Len() <= 44 {
		return nil, fmt.Errorf("clip: no audio between %v and %v in %s", start, end, path)
	}
	wav := out.Bytes()
	if err := fixWAVSizes(wav); err != nil {
		return nil, err
	}
	return wav, nil
}

// fixWAVSizes writes the real RIFF and data chunk sizes into a WAV header.
// ffmpeg can't seek back on a pipe, so it leaves both as 0xFFFFFFFF, which
// some speech-to-text services reject as an unrecognized format.
func fixWAVSizes(wav []byte) error {
	if len(wav) < 12 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return errors.New("clip: ffmpeg did not write a WAV file")
	}
	binary.LittleEndian.PutUint32(wav[4:8], uint32(len(wav)-8)) //nolint:gosec // a clip is far below 4 GiB
	for i := 12; i+8 <= len(wav); {
		id := string(wav[i : i+4])
		size := int(binary.LittleEndian.Uint32(wav[i+4 : i+8]))
		if id == "data" {
			binary.LittleEndian.PutUint32(wav[i+4:i+8], uint32(len(wav)-i-8)) //nolint:gosec // as above
			return nil
		}
		i += 8 + size + size%2
	}
	return errors.New("clip: WAV file has no data chunk")
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}
