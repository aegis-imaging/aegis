"""Abstract base class for analytics backends."""

from __future__ import annotations

import abc
from dataclasses import dataclass, field


@dataclass
class AnalyticsResult:
    """Result from a single analytics backend."""

    tool: str
    success: bool
    duration_seconds: float = 0.0
    outputs: list[str] = field(default_factory=list)
    metrics: dict = field(default_factory=dict)
    error: str = ""


class AnalyticsBackend(abc.ABC):
    """Base class for neuroimaging analytics backends."""

    @property
    @abc.abstractmethod
    def name(self) -> str:
        """Short name of this backend (e.g. 'freesurfer', 'fsl')."""

    @abc.abstractmethod
    def available(self) -> bool:
        """Return True if the backend's tools are installed and usable."""

    @abc.abstractmethod
    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
    ) -> AnalyticsResult:
        """Run analysis on BIDS-converted NIfTI data.

        Args:
            bids_dir: Path to BIDS directory with NIfTI files.
            output_dir: Path to write analytics outputs.
            study_uid: Study identifier for naming outputs.

        Returns:
            AnalyticsResult with outputs and metrics.
        """
