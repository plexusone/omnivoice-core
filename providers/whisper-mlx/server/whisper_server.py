#!/usr/bin/env python3
"""
Whisper MLX gRPC Server for OmniVoice.

This server provides speech-to-text transcription using OpenAI's Whisper model
optimized for Apple Silicon via MLX.

Usage:
    python whisper_server.py [--socket /tmp/omnivoice-whisper.sock] [--model large-v3-turbo]
"""

import argparse
import importlib.metadata
import io
import logging
import os
import platform
import signal
import sys
import time
from concurrent import futures
from typing import Optional

import grpc
import numpy as np
import soundfile as sf

# Import generated protobuf modules
import localstt_pb2
import localstt_pb2_grpc

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
)
logger = logging.getLogger(__name__)

# Available Whisper models with descriptions
WHISPER_MODELS = {
    "tiny": ("Tiny model (~39M params)", 75),
    "base": ("Base model (~74M params)", 145),
    "small": ("Small model (~244M params)", 465),
    "medium": ("Medium model (~769M params)", 1500),
    "large": ("Large model (~1.5B params)", 2900),
    "large-v2": ("Large v2 model (~1.5B params)", 2900),
    "large-v3": ("Large v3 model (~1.5B params)", 2900),
    "large-v3-turbo": ("Large v3 Turbo - fastest large model", 1600),
}

# Default model
DEFAULT_MODEL = "large-v3-turbo"


class WhisperMLXServicer(localstt_pb2_grpc.LocalSTTServicer):
    """gRPC servicer for Whisper MLX transcription."""

    def __init__(self, default_model: str = DEFAULT_MODEL):
        self._model = None
        self._model_name = None
        self._transcribe_fn = None
        self._default_model = default_model
        logger.info(f"WhisperMLXServicer initialized (default model: {default_model})")

    def _load_model(self, model_name: Optional[str] = None) -> bool:
        """Load the Whisper model."""
        model_name = model_name or self._default_model

        try:
            import mlx_whisper

            # Get the model path (downloads if needed)
            model_path = f"mlx-community/whisper-{model_name}"

            logger.info(f"Loading Whisper model: {model_path}")

            # Store reference to transcribe function
            self._transcribe_fn = mlx_whisper.transcribe
            self._model_name = model_name

            # Warm up with a tiny audio sample to ensure model is loaded
            logger.info("Warming up model...")
            _ = mlx_whisper.transcribe(
                np.zeros(16000, dtype=np.float32),  # 1 second of silence
                path_or_hf_repo=model_path,
            )

            logger.info(f"Model {model_name} loaded successfully")
            return True

        except Exception as e:
            logger.error(f"Failed to load model: {e}")
            return False

    def _get_model_path(self) -> str:
        """Get the HuggingFace model path for the current model."""
        return f"mlx-community/whisper-{self._model_name}"

    def Transcribe(self, request, context):
        """Transcribe audio to text."""
        if self._transcribe_fn is None:
            if not self._load_model():
                context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
                context.set_details("Model not loaded")
                return localstt_pb2.TranscribeResponse()

        try:
            start_time = time.time()

            # Convert audio bytes to numpy array
            audio_data = self._decode_audio(request.audio, request.config)

            # Build transcription options
            options = {}

            if request.config.language:
                options["language"] = request.config.language

            if request.config.task:
                options["task"] = request.config.task

            if request.config.initial_prompt:
                options["initial_prompt"] = request.config.initial_prompt

            # Enable word timestamps if requested
            if request.config.enable_word_timestamps:
                options["word_timestamps"] = True

            # Perform transcription
            result = self._transcribe_fn(
                audio_data,
                path_or_hf_repo=self._get_model_path(),
                **options,
            )

            processing_time = int((time.time() - start_time) * 1000)

            # Build response
            response = localstt_pb2.TranscribeResponse(
                text=result.get("text", "").strip(),
                language=result.get("language", ""),
                processing_time_ms=processing_time,
            )

            # Add segments
            segments = result.get("segments", [])
            for i, seg in enumerate(segments):
                pb_segment = localstt_pb2.Segment(
                    id=i,
                    text=seg.get("text", "").strip(),
                    start_ms=int(seg.get("start", 0) * 1000),
                    end_ms=int(seg.get("end", 0) * 1000),
                )

                # Add word-level timestamps if available
                words = seg.get("words", [])
                for word in words:
                    pb_word = localstt_pb2.Word(
                        text=word.get("word", ""),
                        start_ms=int(word.get("start", 0) * 1000),
                        end_ms=int(word.get("end", 0) * 1000),
                        confidence=word.get("probability", 0.0),
                    )
                    pb_segment.words.append(pb_word)

                response.segments.append(pb_segment)

            # Calculate duration from segments
            if segments:
                response.duration_ms = int(segments[-1].get("end", 0) * 1000)

            logger.info(
                f"Transcribed {len(request.audio)} bytes in {processing_time}ms: "
                f"{len(response.text)} chars, {len(segments)} segments"
            )

            return response

        except Exception as e:
            logger.error(f"Transcription failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return localstt_pb2.TranscribeResponse()

    def _decode_audio(
        self, audio_bytes: bytes, config: localstt_pb2.TranscriptionConfig
    ) -> np.ndarray:
        """Decode audio bytes to numpy array at 16kHz mono."""
        try:
            # Try to read with soundfile (handles WAV, FLAC, etc.)
            audio_io = io.BytesIO(audio_bytes)
            audio, sample_rate = sf.read(audio_io)

            # Convert to mono if stereo
            if len(audio.shape) > 1:
                audio = audio.mean(axis=1)

            # Resample to 16kHz if needed (Whisper requires 16kHz)
            if sample_rate != 16000:
                # Simple resampling - for production, use a proper resampler
                duration = len(audio) / sample_rate
                new_length = int(duration * 16000)
                audio = np.interp(
                    np.linspace(0, len(audio), new_length),
                    np.arange(len(audio)),
                    audio,
                )

            return audio.astype(np.float32)

        except Exception as e:
            # If soundfile fails, try raw PCM
            logger.warning(f"soundfile decode failed: {e}, trying raw PCM")

            if config.input_format and config.input_format.encoding:
                encoding = config.input_format.encoding.lower()
                sample_rate = config.input_format.sample_rate or 16000

                if encoding in ("pcm_s16le", "pcm"):
                    audio = np.frombuffer(audio_bytes, dtype=np.int16)
                    audio = audio.astype(np.float32) / 32768.0
                elif encoding == "pcm_f32le":
                    audio = np.frombuffer(audio_bytes, dtype=np.float32)
                else:
                    raise ValueError(f"Unsupported encoding: {encoding}")

                # Resample if needed
                if sample_rate != 16000:
                    duration = len(audio) / sample_rate
                    new_length = int(duration * 16000)
                    audio = np.interp(
                        np.linspace(0, len(audio), new_length),
                        np.arange(len(audio)),
                        audio,
                    )

                return audio.astype(np.float32)

            raise

    def TranscribeStream(self, request_iterator, context):
        """Streaming transcription (not yet implemented)."""
        context.set_code(grpc.StatusCode.UNIMPLEMENTED)
        context.set_details("Streaming transcription not yet implemented")
        return

    def Health(self, request, context):
        """Return health status."""
        return localstt_pb2.HealthResponse(
            healthy=True,
            model_loaded=self._transcribe_fn is not None,
            model_name=self._model_name or "",
            model_version="mlx",
            supported_languages=[
                "en",
                "zh",
                "de",
                "es",
                "ru",
                "ko",
                "fr",
                "ja",
                "pt",
                "tr",
                "pl",
                "ca",
                "nl",
                "ar",
                "sv",
                "it",
                "id",
                "hi",
                "fi",
                "vi",
                "he",
                "uk",
                "el",
                "ms",
                "cs",
                "ro",
                "da",
                "hu",
                "ta",
                "no",
                "th",
                "ur",
                "hr",
                "bg",
                "lt",
                "la",
                "mi",
                "ml",
                "cy",
                "sk",
                "te",
                "fa",
                "lv",
                "bn",
                "sr",
                "az",
                "sl",
                "kn",
                "et",
                "mk",
                "br",
                "eu",
                "is",
                "hy",
                "ne",
                "mn",
                "bs",
                "kk",
                "sq",
                "sw",
                "gl",
                "mr",
                "pa",
                "si",
                "km",
                "sn",
                "yo",
                "so",
                "af",
                "oc",
                "ka",
                "be",
                "tg",
                "sd",
                "gu",
                "am",
                "yi",
                "lo",
                "uz",
                "fo",
                "ht",
                "ps",
                "tk",
                "nn",
                "mt",
                "sa",
                "lb",
                "my",
                "bo",
                "tl",
                "mg",
                "as",
                "tt",
                "haw",
                "ln",
                "ha",
                "ba",
                "jw",
                "su",
            ],
        )

    def LoadModel(self, request, context):
        """Load the STT model."""
        start_time = time.time()

        model_name = request.model if request.model else self._default_model

        success = self._load_model(model_name)
        load_time_ms = int((time.time() - start_time) * 1000)

        # Estimate memory usage
        memory_mb = WHISPER_MODELS.get(model_name, ("", 0))[1]

        return localstt_pb2.LoadModelResponse(
            success=success,
            load_time_ms=load_time_ms,
            memory_used_mb=memory_mb,
            model_name=model_name if success else "",
            error_message=None if success else "Failed to load model",
        )

    def UnloadModel(self, request, context):
        """Unload the model from memory."""
        import gc

        memory_freed = WHISPER_MODELS.get(self._model_name, ("", 0))[1] if self._model_name else 0

        self._transcribe_fn = None
        self._model_name = None

        # Force garbage collection
        gc.collect()

        return localstt_pb2.UnloadModelResponse(
            success=True,
            memory_freed_mb=memory_freed,
        )

    def RuntimeInfo(self, request, context):
        """Return runtime environment information."""
        try:
            mlx_version = importlib.metadata.version("mlx")
        except importlib.metadata.PackageNotFoundError:
            mlx_version = "unknown"

        try:
            whisper_version = importlib.metadata.version("mlx-whisper")
        except importlib.metadata.PackageNotFoundError:
            whisper_version = "unknown"

        response = localstt_pb2.RuntimeInfoResponse(
            device_type="mlx",
            memory_used_mb=0,  # Could be computed
            memory_available_mb=0,  # Could be computed
            framework_version=f"mlx={mlx_version}, mlx-whisper={whisper_version}",
            python_version=platform.python_version(),
        )

        if self._model_name:
            model_info = WHISPER_MODELS.get(self._model_name, ("Unknown", 0))
            response.model_info.CopyFrom(
                localstt_pb2.ModelInfo(
                    name=self._model_name,
                    variant="mlx",
                    parameter_count=0,  # Could be computed
                    supported_languages=["en", "zh", "de", "es", "fr", "ja"],  # Subset
                )
            )

        return response

    def ListModels(self, request, context):
        """List available Whisper models."""
        models = []
        for name, (description, size_mb) in WHISPER_MODELS.items():
            models.append(
                localstt_pb2.AvailableModel(
                    name=name,
                    description=description,
                    size_mb=size_mb,
                    is_downloaded=False,  # Could check HF cache
                )
            )

        return localstt_pb2.ListModelsResponse(models=models)


def serve(socket_path: str, default_model: str):
    """Start the gRPC server."""
    # Remove existing socket file
    if os.path.exists(socket_path):
        os.remove(socket_path)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    localstt_pb2_grpc.add_LocalSTTServicer_to_server(
        WhisperMLXServicer(default_model=default_model),
        server,
    )

    server.add_insecure_port(f"unix://{socket_path}")
    server.start()

    logger.info(f"Whisper MLX server listening on unix://{socket_path}")
    logger.info(f"Default model: {default_model}")

    # Handle shutdown gracefully
    def shutdown(signum, frame):
        logger.info("Shutting down server...")
        server.stop(grace=5)
        if os.path.exists(socket_path):
            os.remove(socket_path)
        sys.exit(0)

    signal.signal(signal.SIGINT, shutdown)
    signal.signal(signal.SIGTERM, shutdown)

    server.wait_for_termination()


def main():
    parser = argparse.ArgumentParser(description="Whisper MLX gRPC Server")
    parser.add_argument(
        "--socket",
        default="/tmp/omnivoice-whisper.sock",
        help="Unix socket path (default: /tmp/omnivoice-whisper.sock)",
    )
    parser.add_argument(
        "--model",
        default=DEFAULT_MODEL,
        choices=list(WHISPER_MODELS.keys()),
        help=f"Default Whisper model (default: {DEFAULT_MODEL})",
    )
    args = parser.parse_args()

    serve(args.socket, args.model)


if __name__ == "__main__":
    main()
