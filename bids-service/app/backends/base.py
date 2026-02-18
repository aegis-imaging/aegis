"""Abstract base for BIDS conversion backends."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field


@dataclass
class ConversionResult:
    """Result of a BIDS conversion."""
    output_files: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


class BidsBackend(ABC):
    """Interface that every BIDS conversion backend must implement."""

    @property
    @abstractmethod
    def name(self) -> str:
        ...

    @abstractmethod
    def available(self) -> bool:
        """Return True if all required external tools are present."""
        ...

    @abstractmethod
    def convert(self, input_dir: str, output_dir: str, study_uid: str) -> ConversionResult:
        """Convert DICOM files to BIDS-compliant NIfTI output."""
        ...
