"""Abstract base class for QC check backends."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field


@dataclass
class CheckResult:
    """Result of a single QC check."""

    name: str  # e.g. "slice_consistency", "snr", "coverage", "missing_slices", "file_integrity"
    status: str  # "pass", "warn", "fail"
    message: str  # Human-readable explanation
    details: dict = field(default_factory=dict)


@dataclass
class QCResult:
    """Aggregated QC results for a study."""

    checks: list[CheckResult] = field(default_factory=list)
    overall: str = "pass"  # "pass", "warn", "fail" — worst of all checks


class QCBackend(ABC):
    """A QC backend runs image quality checks on DICOM files."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable name of this backend."""

    @abstractmethod
    def available(self) -> bool:
        """Return True if this backend's required tools are installed."""

    @abstractmethod
    def check(self, dicom_paths: list[str]) -> QCResult:
        """Run all QC checks on the given DICOM files."""
