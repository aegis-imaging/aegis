"""Abstract base class for protocol compliance backends."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field


@dataclass
class Finding:
    """Result of checking a single parameter against a rule."""

    tag_keyword: str  # e.g. "RepetitionTime"
    severity: str  # "critical", "warning", "info"
    status: str  # "compliant", "deviated", "missing"
    message: str  # Human-readable explanation
    expected: str  # Expected value (as string for display)
    actual: str  # Actual value found (as string for display)
    deviation_pct: float | None = None  # Percentage deviation for numeric


@dataclass
class ProtocolResult:
    """Aggregated protocol compliance result for a study."""

    findings: list[Finding] = field(default_factory=list)
    overall: str = "compliant"  # "compliant", "minor_deviations", "non_compliant"


class ProtocolBackend(ABC):
    """A protocol backend compares DICOM parameters against template rules."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable name of this backend."""

    @abstractmethod
    def available(self) -> bool:
        """Return True if this backend's required tools are installed."""

    @abstractmethod
    def check(self, params: dict, rules: list[dict]) -> ProtocolResult:
        """Compare extracted parameters against template rules."""
