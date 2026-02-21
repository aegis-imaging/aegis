"""Abstract base class for burned-in PHI detection backends."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field


@dataclass
class Region:
    """A single detected text region in a DICOM image."""
    text: str
    confidence: float
    bbox: list[int] = field(default_factory=lambda: [0, 0, 0, 0])  # [x, y, w, h]


@dataclass
class FileFinding:
    """Detection results for a single DICOM file."""
    file: str
    regions: list[Region] = field(default_factory=list)


class PHIDetectionBackend(ABC):
    """
    A PHI detection backend scans DICOM pixel data for burned-in text.

    It reads pixel arrays from DICOM files and runs OCR to detect any
    text content (patient names, dates of birth, accession numbers, etc.)
    that may be overlaid on the image pixels.
    """

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable name of this backend."""

    @abstractmethod
    def available(self) -> bool:
        """Return True if this backend's required tools are installed."""

    @abstractmethod
    def detect(self, dicom_paths: list[str]) -> list[FileFinding]:
        """
        Scan DICOM files for burned-in text.

        Args:
            dicom_paths: List of absolute paths to DICOM files.

        Returns:
            List of FileFinding objects — one per file that had detections.
            Files with no detected text are omitted.

        Raises:
            RuntimeError on failure.
        """
