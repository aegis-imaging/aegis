"""Abstract base class for SCT backends."""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field


@dataclass
class SctResult:
    """Result from an SCT analysis run."""

    tool: str
    success: bool
    duration_seconds: float = 0.0
    outputs: list[str] = field(default_factory=list)
    metrics: dict = field(default_factory=dict)
    error: str = ""


class SctBackend(ABC):
    """Base class for SCT backends."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable backend name."""

    @abstractmethod
    def available(self) -> bool:
        """Return True if required tools are installed."""

    @abstractmethod
    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> SctResult:
        """Run SCT analysis on BIDS-converted NIfTI data."""
