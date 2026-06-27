package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/plexusone/omnivoice-core/audio"
	"github.com/plexusone/omnivoice-core/config"
	"github.com/plexusone/omnivoice-core/voices"
	"github.com/spf13/cobra"
)

var (
	// voiceAnalyzeDuration is the target segment duration
	voiceAnalyzeDuration float64

	// voiceAnalyzeTopN is the number of segments to show
	voiceAnalyzeTopN int

	// voiceExtractOutput is the output file path
	voiceExtractOutput string

	// voiceExtractStart is the start time for extraction
	voiceExtractStart float64

	// voiceExtractDuration is the duration for extraction
	voiceExtractDuration float64

	// voiceExtractBest extracts the best segment automatically
	voiceExtractBest bool

	// voiceCreateSlug is the profile slug for creation
	voiceCreateSlug string

	// voiceCreateTranscript is the transcript text
	voiceCreateTranscript string

	// voiceCreateTranscriptFile is the transcript file path
	voiceCreateTranscriptFile string
)

// voiceCmd is the parent command for voice operations
var voiceCmd = &cobra.Command{
	Use:   "voice",
	Short: "Voice profile management",
	Long: `Manage voice profiles for F5-TTS voice cloning.

Voice profiles are stored in ~/.plexusone/omnivoice/voices/ by default.
Each profile contains a reference.wav and reference.txt file.`,
}

// voiceAnalyzeCmd analyzes a WAV file for voice cloning
var voiceAnalyzeCmd = &cobra.Command{
	Use:   "analyze <file.wav>",
	Short: "Analyze audio to find best segments for voice cloning",
	Long: `Analyze a WAV file and identify the best segments for F5-TTS voice cloning.

The analysis scores segments based on:
  - Speech activity ratio (looking for consistent speech)
  - Energy consistency (avoiding volume swings)
  - Silence gaps (avoiding long pauses)
  - Clipping detection (avoiding distorted audio)

Examples:
  # Analyze a 3-minute recording
  omnictl voice analyze recording.wav

  # Find best 10-second segments
  omnictl voice analyze recording.wav --duration 10

  # Show top 10 segments
  omnictl voice analyze recording.wav --top 10`,
	Args: cobra.ExactArgs(1),
	RunE: runVoiceAnalyze,
}

// voiceExtractCmd extracts a segment from a WAV file
var voiceExtractCmd = &cobra.Command{
	Use:   "extract <file.wav>",
	Short: "Extract a segment from a WAV file",
	Long: `Extract a segment from a WAV file for use as voice cloning reference.

Examples:
  # Extract best 15-second segment automatically
  omnictl voice extract recording.wav --best -o reference.wav

  # Extract specific time range
  omnictl voice extract recording.wav --start 30.5 --duration 15 -o reference.wav`,
	Args: cobra.ExactArgs(1),
	RunE: runVoiceExtract,
}

// voiceListCmd lists voice profiles
var voiceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available voice profiles",
	Long: `List all voice profiles in the voice library.

Voice profiles are stored in ~/.plexusone/omnivoice/voices/ by default.
Set OMNIVOICE_VOICES_DIR to use a different location.`,
	RunE: runVoiceList,
}

// voiceCreateCmd creates a new voice profile
var voiceCreateCmd = &cobra.Command{
	Use:   "create <file.wav>",
	Short: "Create a new voice profile",
	Long: `Create a new voice profile from a WAV file and transcript.

Examples:
  # Create profile with transcript text
  omnictl voice create reference.wav --slug john-narrator \
    --transcript "This is what I said in the recording."

  # Create profile with transcript file
  omnictl voice create reference.wav --slug john-narrator \
    --transcript-file transcript.txt

  # Analyze, extract best segment, and create profile
  omnictl voice extract long-recording.wav --best -o /tmp/ref.wav
  omnictl voice create /tmp/ref.wav --slug john-narrator \
    --transcript "..."`,
	Args: cobra.ExactArgs(1),
	RunE: runVoiceCreate,
}

func init() {
	rootCmd.AddCommand(voiceCmd)
	voiceCmd.AddCommand(voiceAnalyzeCmd)
	voiceCmd.AddCommand(voiceExtractCmd)
	voiceCmd.AddCommand(voiceListCmd)
	voiceCmd.AddCommand(voiceCreateCmd)

	// Analyze flags
	voiceAnalyzeCmd.Flags().Float64Var(&voiceAnalyzeDuration, "duration", 15.0, "target segment duration in seconds")
	voiceAnalyzeCmd.Flags().IntVar(&voiceAnalyzeTopN, "top", 5, "number of top segments to show")

	// Extract flags
	voiceExtractCmd.Flags().StringVarP(&voiceExtractOutput, "output", "o", "", "output file path (required)")
	voiceExtractCmd.Flags().Float64Var(&voiceExtractStart, "start", 0, "start time in seconds")
	voiceExtractCmd.Flags().Float64Var(&voiceExtractDuration, "duration", 15.0, "duration in seconds")
	voiceExtractCmd.Flags().BoolVar(&voiceExtractBest, "best", false, "automatically extract the best segment")
	voiceExtractCmd.MarkFlagRequired("output")

	// Create flags
	voiceCreateCmd.Flags().StringVar(&voiceCreateSlug, "slug", "", "profile slug/ID (required)")
	voiceCreateCmd.Flags().StringVar(&voiceCreateTranscript, "transcript", "", "transcript text")
	voiceCreateCmd.Flags().StringVar(&voiceCreateTranscriptFile, "transcript-file", "", "path to transcript file")
	voiceCreateCmd.MarkFlagRequired("slug")
}

func runVoiceAnalyze(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	fmt.Printf("Analyzing: %s\n", inputPath)
	fmt.Printf("Target duration: %.1fs\n\n", voiceAnalyzeDuration)

	// Read WAV file
	header, samples, err := audio.ReadWAV(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read WAV: %w", err)
	}

	totalDuration := float64(len(samples)) / float64(header.SampleRate)
	fmt.Printf("Audio: %.1fs @ %dHz, %d-bit\n\n", totalDuration, header.SampleRate, header.BitsPerSample)

	// Find best segments
	config := audio.DefaultAnalyzeConfig()
	config.TargetDuration = voiceAnalyzeDuration
	config.TopN = voiceAnalyzeTopN

	segments, err := audio.FindBestSegments(samples, header.SampleRate, config)
	if err != nil {
		return err
	}

	fmt.Printf("Top %d segments for voice cloning:\n\n", len(segments))
	fmt.Println("Rank  Start     End       Score   Speech%  Energy   Gaps")
	fmt.Println("----  --------  --------  ------  -------  -------  ----")

	for i, seg := range segments {
		fmt.Printf("%2d    %s  %s  %6.1f  %5.0f%%   %.3f    %d\n",
			i+1,
			formatTime(seg.StartTime),
			formatTime(seg.EndTime),
			seg.Score,
			seg.Metrics.SpeechRatio*100,
			seg.Metrics.MeanEnergy,
			seg.Metrics.SilenceGaps,
		)
	}

	if len(segments) > 0 {
		best := segments[0]
		fmt.Printf("\nRecommended: %s - %s (score: %.1f)\n",
			formatTime(best.StartTime),
			formatTime(best.EndTime),
			best.Score,
		)
		fmt.Printf("\nTo extract: omnictl voice extract %s --start %.1f --duration %.1f -o reference.wav\n",
			inputPath, best.StartTime, voiceAnalyzeDuration)
		fmt.Printf("Or use:     omnictl voice extract %s --best -o reference.wav\n", inputPath)
	}

	return nil
}

func runVoiceExtract(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Read WAV file
	header, samples, err := audio.ReadWAV(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read WAV: %w", err)
	}

	var startSample, endSample int

	if voiceExtractBest {
		// Find best segment
		config := audio.DefaultAnalyzeConfig()
		config.TargetDuration = voiceExtractDuration
		config.TopN = 1

		segments, err := audio.FindBestSegments(samples, header.SampleRate, config)
		if err != nil {
			return err
		}
		if len(segments) == 0 {
			return fmt.Errorf("no suitable segments found")
		}

		best := segments[0]
		startSample = best.StartSample
		endSample = best.EndSample
		fmt.Printf("Selected best segment: %s - %s (score: %.1f)\n",
			formatTime(best.StartTime),
			formatTime(best.EndTime),
			best.Score,
		)
	} else {
		// Use specified time range
		startSample = int(voiceExtractStart * float64(header.SampleRate))
		endSample = startSample + int(voiceExtractDuration*float64(header.SampleRate))

		if endSample > len(samples) {
			return fmt.Errorf("segment extends beyond audio end")
		}
	}

	// Extract segment
	segment := audio.ExtractSegment(samples, startSample, endSample)

	// Resample to 24kHz if needed
	outputSampleRate := uint32(24000)
	if header.SampleRate != outputSampleRate {
		fmt.Printf("Resampling from %dHz to %dHz...\n", header.SampleRate, outputSampleRate)
		segment = resample(segment, header.SampleRate, outputSampleRate)
	}

	// Write output
	if err := audio.WriteWAV(voiceExtractOutput, segment, outputSampleRate); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	duration := float64(len(segment)) / float64(outputSampleRate)
	fmt.Printf("Wrote %.1fs to %s (%dHz, 16-bit mono)\n", duration, voiceExtractOutput, outputSampleRate)

	return nil
}

func runVoiceList(cmd *cobra.Command, args []string) error {
	library := voices.NewLibrary("")

	fmt.Printf("Voice library: %s\n\n", library.Dir())

	profiles, err := library.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		fmt.Println("No voice profiles found.")
		fmt.Printf("\nCreate one with: omnictl voice create <reference.wav> --slug <name> --transcript \"...\"\n")
		return nil
	}

	fmt.Printf("Found %d voice profile(s):\n\n", len(profiles))
	fmt.Println("Slug                  Name                  Language  Description")
	fmt.Println("--------------------  --------------------  --------  -----------")

	for _, p := range profiles {
		name := p.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		lang := p.Language
		if lang == "" {
			lang = "-"
		}
		desc := p.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		if desc == "" {
			desc = "-"
		}

		fmt.Printf("%-20s  %-20s  %-8s  %s\n", p.Slug, name, lang, desc)
	}

	return nil
}

func runVoiceCreate(cmd *cobra.Command, args []string) error {
	audioPath := args[0]

	// Get transcript
	var transcript string
	if voiceCreateTranscriptFile != "" {
		data, err := os.ReadFile(voiceCreateTranscriptFile)
		if err != nil {
			return fmt.Errorf("failed to read transcript file: %w", err)
		}
		transcript = strings.TrimSpace(string(data))
	} else if voiceCreateTranscript != "" {
		transcript = voiceCreateTranscript
	} else {
		return fmt.Errorf("either --transcript or --transcript-file is required")
	}

	// Read audio
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return fmt.Errorf("failed to read audio file: %w", err)
	}

	// Ensure voices directory exists
	if err := config.EnsureVoicesDir(); err != nil {
		return fmt.Errorf("failed to create voices directory: %w", err)
	}

	// Create profile
	library := voices.NewLibrary("")

	if library.Exists(voiceCreateSlug) {
		return fmt.Errorf("profile already exists: %s", voiceCreateSlug)
	}

	if err := library.Create(voiceCreateSlug, audioData, transcript, nil); err != nil {
		return err
	}

	profileDir := filepath.Join(library.Dir(), voiceCreateSlug)
	fmt.Printf("Created voice profile: %s\n", voiceCreateSlug)
	fmt.Printf("Location: %s\n", profileDir)
	fmt.Printf("Files:\n")
	fmt.Printf("  - reference.wav (%d bytes)\n", len(audioData))
	fmt.Printf("  - reference.txt (%d chars)\n", len(transcript))

	return nil
}

func formatTime(seconds float64) string {
	mins := int(seconds) / 60
	secs := seconds - float64(mins*60)
	return fmt.Sprintf("%02d:%05.2f", mins, secs)
}

// resample performs simple linear interpolation resampling.
// For production use, consider a proper resampling library.
func resample(samples []float64, fromRate, toRate uint32) []float64 {
	if fromRate == toRate {
		return samples
	}

	ratio := float64(fromRate) / float64(toRate)
	newLen := int(float64(len(samples)) / ratio)
	result := make([]float64, newLen)

	for i := 0; i < newLen; i++ {
		srcIdx := float64(i) * ratio
		srcIdxInt := int(srcIdx)
		frac := srcIdx - float64(srcIdxInt)

		if srcIdxInt+1 < len(samples) {
			result[i] = samples[srcIdxInt]*(1-frac) + samples[srcIdxInt+1]*frac
		} else if srcIdxInt < len(samples) {
			result[i] = samples[srcIdxInt]
		}
	}

	return result
}
