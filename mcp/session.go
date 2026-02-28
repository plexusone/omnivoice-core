package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/plexusone/omnivoice-core/stt"
	"github.com/plexusone/omnivoice-core/transport"
	"github.com/plexusone/omnivoice-core/tts"
)

// Session errors
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionHungUp   = errors.New("session was hung up")
	ErrNoConnection    = errors.New("no active connection")
)

// Session represents an active voice call session.
type Session struct {
	ID        string
	StartTime time.Time

	config Config
	conn   transport.Connection

	// STT state
	sttWriter   io.WriteCloser
	sttEvents   <-chan stt.StreamEvent
	transcripts chan string

	// Conversation history
	history []Turn

	// State
	mu       sync.Mutex
	hungUp   bool
	cancelFn context.CancelFunc
}

// Turn represents a conversation turn.
type Turn struct {
	Speaker string // "claude" or "user"
	Message string
}

// SessionManager manages active voice sessions.
type SessionManager struct {
	config   Config
	sessions map[string]*Session
	mu       sync.RWMutex
	counter  int
}

// NewSessionManager creates a new session manager.
func NewSessionManager(config Config) *SessionManager {
	return &SessionManager{
		config:   config,
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new voice session.
func (m *SessionManager) CreateSession(ctx context.Context) (*Session, error) {
	m.mu.Lock()
	m.counter++
	id := fmt.Sprintf("call-%d-%d", m.counter, time.Now().UnixMilli())
	m.mu.Unlock()

	sessionCtx, cancel := context.WithCancel(ctx)

	session := &Session{
		ID:          id,
		StartTime:   time.Now(),
		config:      m.config,
		transcripts: make(chan string, 10),
		cancelFn:    cancel,
	}

	// Connect to transport
	conn, err := m.config.Transport.Connect(sessionCtx, m.config.UserPhoneNumber, transport.Config{
		SampleRate: 8000,
		Channels:   1,
		Encoding:   "mulaw",
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	session.conn = conn

	// Start STT stream
	sttConfig := stt.TranscriptionConfig{
		Model:      m.config.STTModel,
		Language:   m.config.STTLanguage,
		Encoding:   "mulaw",
		SampleRate: 8000,
		Channels:   1,
	}

	writer, events, err := m.config.STT.TranscribeStream(sessionCtx, sttConfig)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "failed to close connection during cleanup", "error", closeErr)
		}
		cancel()
		return nil, fmt.Errorf("failed to start STT: %w", err)
	}
	session.sttWriter = writer
	session.sttEvents = events

	// Start goroutines for audio streaming and event processing
	go session.streamAudioToSTT(sessionCtx)
	go session.processSTTEvents(sessionCtx)

	// Handle connection events
	go session.handleConnectionEvents(sessionCtx)

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID.
func (m *SessionManager) GetSession(id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// RemoveSession removes a session.
func (m *SessionManager) RemoveSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

// streamAudioToSTT reads audio from the connection and sends to STT.
func (s *Session) streamAudioToSTT(ctx context.Context) {
	defer func() {
		if s.sttWriter != nil {
			_ = s.sttWriter.Close()
		}
	}()

	audioOut := s.conn.AudioOut()
	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := audioOut.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				return
			}

			if n > 0 && s.sttWriter != nil {
				_, _ = s.sttWriter.Write(buf[:n])
			}
		}
	}
}

// processSTTEvents processes events from STT and extracts transcripts.
func (s *Session) processSTTEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-s.sttEvents:
			if !ok {
				return
			}

			if event.Type == stt.EventTranscript && event.IsFinal && event.Transcript != "" {
				select {
				case s.transcripts <- event.Transcript:
				default:
					// Channel full, drop
				}
			}
		}
	}
}

// handleConnectionEvents monitors connection events.
func (s *Session) handleConnectionEvents(ctx context.Context) {
	events := s.conn.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			if event.Type == transport.EventDisconnected {
				s.mu.Lock()
				s.hungUp = true
				s.mu.Unlock()
				return
			}
		}
	}
}

// Speak synthesizes and sends audio to the connection.
func (s *Session) Speak(ctx context.Context, text string) error {
	s.mu.Lock()
	if s.hungUp {
		s.mu.Unlock()
		return ErrSessionHungUp
	}
	s.mu.Unlock()

	if s.conn == nil {
		return ErrNoConnection
	}

	// Synthesize using streaming TTS
	synthConfig := tts.SynthesisConfig{
		VoiceID:      s.config.VoiceID,
		Model:        s.config.TTSModel,
		OutputFormat: "ulaw",
		SampleRate:   8000,
	}

	stream, err := s.config.TTS.SynthesizeStream(ctx, text, synthConfig)
	if err != nil {
		return fmt.Errorf("failed to synthesize: %w", err)
	}

	audioIn := s.conn.AudioIn()

	for chunk := range stream {
		if chunk.Error != nil {
			return chunk.Error
		}

		if len(chunk.Audio) > 0 {
			if _, err := audioIn.Write(chunk.Audio); err != nil {
				return fmt.Errorf("failed to write audio: %w", err)
			}
		}

		if chunk.IsFinal {
			break
		}
	}

	return nil
}

// Listen waits for user speech and returns the transcript.
func (s *Session) Listen(ctx context.Context, timeoutMs int) (string, error) {
	s.mu.Lock()
	if s.hungUp {
		s.mu.Unlock()
		return "", ErrSessionHungUp
	}
	s.mu.Unlock()

	timeout := time.Duration(timeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case transcript := <-s.transcripts:
		return transcript, nil
	}
}

// SpeakAndListen speaks a message and waits for a response.
func (s *Session) SpeakAndListen(ctx context.Context, text string, timeoutMs int) (string, error) {
	if err := s.Speak(ctx, text); err != nil {
		return "", err
	}

	// Small delay to let audio finish
	time.Sleep(200 * time.Millisecond)

	return s.Listen(ctx, timeoutMs)
}

// Close ends the session.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hungUp = true

	if s.cancelFn != nil {
		s.cancelFn()
	}

	if s.sttWriter != nil {
		_ = s.sttWriter.Close()
	}

	if s.conn != nil {
		return s.conn.Close()
	}

	return nil
}

// Duration returns the session duration.
func (s *Session) Duration() time.Duration {
	return time.Since(s.StartTime)
}

// AddHistory adds a turn to the conversation history.
func (s *Session) AddHistory(speaker, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, Turn{Speaker: speaker, Message: message})
}

// IsHungUp returns whether the session was hung up.
func (s *Session) IsHungUp() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hungUp
}
