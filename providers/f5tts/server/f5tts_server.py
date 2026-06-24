#!/usr/bin/env python3
"""
F5-TTS MLX gRPC Server

This server provides a gRPC interface to F5-TTS MLX for local text-to-speech
synthesis with voice cloning support.

Usage:
    python f5tts_server.py [--socket /tmp/omnivoice-f5tts.sock] [--auto-load]

The server listens on a Unix Domain Socket for low-latency local communication.
"""

import argparse
import io
import logging
import os
import sys
import time
from concurrent import futures
from typing import Optional

import grpc
import numpy as np
import soundfile as sf

# Add parent directory to path for proto imports
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

# Import generated proto modules
import localvoice_pb2 as pb
import localvoice_pb2_grpc as pb_grpc

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


class F5TTSServicer(pb_grpc.LocalVoiceServicer):
    """gRPC servicer for F5-TTS MLX."""

    def __init__(self, auto_load: bool = False):
        self.model = None
        self.model_loaded = False
        self.model_name = "f5-tts-mlx"
        self.model_version = "1.0.0"
        self.voice_profiles: dict[str, dict] = {}
        self._load_time_ms = 0
        self._memory_used_mb = 0

        if auto_load:
            self._load_model()

    def _load_model(self, model_path: Optional[str] = None) -> bool:
        """Load the F5-TTS model."""
        try:
            start_time = time.time()
            logger.info("Loading F5-TTS MLX model...")

            # Import here to avoid slow startup if model not needed
            from f5_tts_mlx import F5TTS

            self.model = F5TTS()
            self.model_loaded = True
            self._load_time_ms = int((time.time() - start_time) * 1000)

            # Estimate memory usage (rough approximation)
            self._memory_used_mb = 2000  # ~2GB for F5-TTS

            logger.info(
                f"Model loaded in {self._load_time_ms}ms, "
                f"estimated memory: {self._memory_used_mb}MB"
            )
            return True

        except ImportError as e:
            logger.error(f"Failed to import F5-TTS: {e}")
            logger.error("Install with: pip install f5-tts-mlx")
            return False
        except Exception as e:
            logger.error(f"Failed to load model: {e}")
            return False

    def _synthesize_audio(
        self,
        text: str,
        voice_id: str = "default",
        speed: float = 1.0,
        reference_audio: Optional[bytes] = None,
        reference_text: Optional[str] = None,
    ) -> tuple[np.ndarray, int]:
        """
        Synthesize audio from text.

        Returns:
            Tuple of (audio_array, sample_rate)
        """
        if not self.model_loaded:
            raise RuntimeError("Model not loaded")

        # Check for cached voice profile
        if voice_id in self.voice_profiles and reference_audio is None:
            profile = self.voice_profiles[voice_id]
            reference_audio = profile.get("audio")
            reference_text = profile.get("text")

        # Generate audio
        if reference_audio is not None and reference_text is not None:
            # Voice cloning mode
            # Convert reference audio bytes to numpy array
            ref_audio_array, ref_sr = sf.read(io.BytesIO(reference_audio))
            if ref_audio_array.ndim > 1:
                ref_audio_array = ref_audio_array[:, 0]  # Mono

            audio = self.model.generate(
                text=text,
                ref_audio=ref_audio_array,
                ref_text=reference_text,
                speed=speed,
            )
        else:
            # Default voice mode
            audio = self.model.generate(
                text=text,
                speed=speed,
            )

        # F5-TTS outputs at 24kHz
        sample_rate = 24000

        return audio, sample_rate

    def _audio_to_wav_bytes(
        self, audio: np.ndarray, sample_rate: int
    ) -> bytes:
        """Convert numpy audio array to WAV bytes."""
        buffer = io.BytesIO()
        sf.write(buffer, audio, sample_rate, format="WAV", subtype="PCM_16")
        buffer.seek(0)
        return buffer.read()

    def _audio_to_pcm_bytes(self, audio: np.ndarray) -> bytes:
        """Convert numpy audio array to PCM S16LE bytes."""
        # Normalize to int16 range
        audio_int16 = (audio * 32767).astype(np.int16)
        return audio_int16.tobytes()

    def Synthesize(self, request, context):
        """Synthesize speech from text."""
        if not self.model_loaded:
            context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
            context.set_details("Model not loaded. Call LoadModel first.")
            return

        try:
            speed = request.speed if request.speed else 1.0
            audio, sample_rate = self._synthesize_audio(
                text=request.text,
                voice_id=request.voice_id or "default",
                speed=speed,
            )

            # Convert to requested format
            if request.format == pb.AUDIO_FORMAT_PCM_S16LE:
                audio_bytes = self._audio_to_pcm_bytes(audio)
                format_type = pb.AUDIO_FORMAT_PCM_S16LE
            else:
                # Default to WAV
                audio_bytes = self._audio_to_wav_bytes(audio, sample_rate)
                format_type = pb.AUDIO_FORMAT_WAV

            # Stream in chunks
            chunk_size = 8192
            first_chunk = True

            for i in range(0, len(audio_bytes), chunk_size):
                chunk_data = audio_bytes[i : i + chunk_size]
                is_final = i + chunk_size >= len(audio_bytes)

                chunk = pb.AudioChunk(
                    data=chunk_data,
                    is_final=is_final,
                )

                if first_chunk:
                    chunk.metadata.CopyFrom(
                        pb.AudioMetadata(
                            format=format_type,
                            sample_rate=sample_rate,
                            channels=1,
                            bit_depth=16,
                        )
                    )
                    first_chunk = False

                yield chunk

        except Exception as e:
            logger.error(f"Synthesis failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))

    def SynthesizeWithReference(self, request, context):
        """Synthesize speech using reference audio for voice cloning."""
        if not self.model_loaded:
            context.set_code(grpc.StatusCode.FAILED_PRECONDITION)
            context.set_details("Model not loaded. Call LoadModel first.")
            return

        try:
            speed = request.speed if request.speed else 1.0
            audio, sample_rate = self._synthesize_audio(
                text=request.text,
                reference_audio=request.reference_audio,
                reference_text=request.reference_text,
                speed=speed,
            )

            # Convert to requested format
            if request.format == pb.AUDIO_FORMAT_PCM_S16LE:
                audio_bytes = self._audio_to_pcm_bytes(audio)
                format_type = pb.AUDIO_FORMAT_PCM_S16LE
            else:
                audio_bytes = self._audio_to_wav_bytes(audio, sample_rate)
                format_type = pb.AUDIO_FORMAT_WAV

            # Stream in chunks
            chunk_size = 8192
            first_chunk = True

            for i in range(0, len(audio_bytes), chunk_size):
                chunk_data = audio_bytes[i : i + chunk_size]
                is_final = i + chunk_size >= len(audio_bytes)

                chunk = pb.AudioChunk(
                    data=chunk_data,
                    is_final=is_final,
                )

                if first_chunk:
                    chunk.metadata.CopyFrom(
                        pb.AudioMetadata(
                            format=format_type,
                            sample_rate=sample_rate,
                            channels=1,
                            bit_depth=16,
                        )
                    )
                    first_chunk = False

                yield chunk

        except Exception as e:
            logger.error(f"Reference synthesis failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))

    def PrepareVoiceProfile(self, request, context):
        """Prepare and cache a voice profile from reference audio."""
        try:
            profile_id = request.profile_id

            # Store the reference audio and text for later use
            self.voice_profiles[profile_id] = {
                "audio": request.reference_audio,
                "text": request.reference_text,
                "language": request.language if request.language else "en",
            }

            logger.info(f"Prepared voice profile: {profile_id}")

            return pb.PrepareVoiceProfileResponse(
                profile_id=profile_id,
                cached=True,
                embedding_size_bytes=len(request.reference_audio),
            )

        except Exception as e:
            logger.error(f"Failed to prepare profile: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return pb.PrepareVoiceProfileResponse(
                profile_id=request.profile_id,
                cached=False,
            )

    def Health(self, request, context):
        """Return health status."""
        return pb.HealthResponse(
            healthy=True,
            model_loaded=self.model_loaded,
            model_name=self.model_name,
            model_version=self.model_version,
            available_voices=list(self.voice_profiles.keys()),
        )

    def LoadModel(self, request, context):
        """Load the TTS model into memory."""
        if self.model_loaded:
            return pb.LoadModelResponse(
                success=True,
                load_time_ms=self._load_time_ms,
                memory_used_mb=self._memory_used_mb,
            )

        model_path = request.model_path if request.model_path else None
        success = self._load_model(model_path)

        if success:
            return pb.LoadModelResponse(
                success=True,
                load_time_ms=self._load_time_ms,
                memory_used_mb=self._memory_used_mb,
            )
        else:
            return pb.LoadModelResponse(
                success=False,
                error_message="Failed to load model. Check logs for details.",
            )

    def UnloadModel(self, request, context):
        """Unload the model from memory."""
        if not self.model_loaded:
            return pb.UnloadModelResponse(success=True, memory_freed_mb=0)

        try:
            memory_freed = self._memory_used_mb
            self.model = None
            self.model_loaded = False
            self._memory_used_mb = 0

            # Force garbage collection
            import gc
            gc.collect()

            logger.info(f"Model unloaded, freed ~{memory_freed}MB")

            return pb.UnloadModelResponse(
                success=True,
                memory_freed_mb=memory_freed,
            )

        except Exception as e:
            logger.error(f"Failed to unload model: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return pb.UnloadModelResponse(success=False)

    def RuntimeInfo(self, request, context):
        """Return runtime information."""
        import platform

        try:
            import mlx

            mlx_version = mlx.__version__
        except ImportError:
            mlx_version = "not installed"

        model_info = None
        if self.model_loaded:
            model_info = pb.ModelInfo(
                name=self.model_name,
                version=self.model_version,
                parameter_count=0,  # Unknown
                supported_languages=["en", "zh"],  # F5-TTS supports these
            )

        return pb.RuntimeInfoResponse(
            device_type="mlx",
            memory_used_mb=self._memory_used_mb,
            memory_available_mb=0,  # Would need psutil for accurate value
            framework_version=mlx_version,
            python_version=platform.python_version(),
            model_info=model_info,
        )


def serve(socket_path: str, auto_load: bool = False):
    """Start the gRPC server."""
    # Remove existing socket file
    if os.path.exists(socket_path):
        os.unlink(socket_path)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    pb_grpc.add_LocalVoiceServicer_to_server(
        F5TTSServicer(auto_load=auto_load), server
    )
    server.add_insecure_port(f"unix://{socket_path}")
    server.start()

    logger.info(f"F5-TTS gRPC server listening on unix://{socket_path}")
    if auto_load:
        logger.info("Model auto-loaded and ready for synthesis")
    else:
        logger.info("Model not loaded. Call LoadModel RPC to load.")

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        server.stop(grace=5)

        # Clean up socket file
        if os.path.exists(socket_path):
            os.unlink(socket_path)


def main():
    parser = argparse.ArgumentParser(
        description="F5-TTS MLX gRPC Server"
    )
    parser.add_argument(
        "--socket",
        default="/tmp/omnivoice-f5tts.sock",
        help="Unix socket path (default: /tmp/omnivoice-f5tts.sock)",
    )
    parser.add_argument(
        "--auto-load",
        action="store_true",
        help="Automatically load the model on startup",
    )
    args = parser.parse_args()

    serve(args.socket, auto_load=args.auto_load)


if __name__ == "__main__":
    main()
