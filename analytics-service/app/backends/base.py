"""Abstract base classes for analytics backends."""

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
    """Base class for single-study neuroimaging analytics backends."""

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
        **kwargs,
    ) -> AnalyticsResult:
        """Run analysis on BIDS-converted NIfTI data.

        Args:
            bids_dir: Path to BIDS directory with NIfTI files.
            output_dir: Path to write analytics outputs.
            study_uid: Study identifier for naming outputs.
            **kwargs: Backend-specific options (e.g. atlas name).

        Returns:
            AnalyticsResult with outputs and metrics.
        """


class LongitudinalBackend(abc.ABC):
    """Base class for multi-timepoint analytics backends (e.g. TBM-SyN).

    These backends require paired input (baseline + follow-up) and are
    served via the ``POST /analyze-longitudinal`` endpoint.
    """

    @property
    @abc.abstractmethod
    def name(self) -> str:
        """Short name of this backend (e.g. 'tbm_syn')."""

    @abc.abstractmethod
    def available(self) -> bool:
        """Return True if the backend's tools are installed and usable."""

    @abc.abstractmethod
    def analyze_longitudinal(
        self,
        baseline_bids_dir: str,
        followup_bids_dir: str,
        output_dir: str,
        baseline_study_uid: str,
        followup_study_uid: str,
        scan_interval_days: float,
        **kwargs,
    ) -> AnalyticsResult:
        """Run longitudinal analysis on paired BIDS data.

        Args:
            baseline_bids_dir: BIDS directory for baseline scan.
            followup_bids_dir: BIDS directory for follow-up scan.
            output_dir: Path to write analytics outputs.
            baseline_study_uid: Baseline study identifier.
            followup_study_uid: Follow-up study identifier.
            scan_interval_days: Days between baseline and follow-up scans.
            **kwargs: Backend-specific options (e.g. atlas name).

        Returns:
            AnalyticsResult with outputs and metrics.
        """
