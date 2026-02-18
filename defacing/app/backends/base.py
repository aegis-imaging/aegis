"""Abstract base class for defacing backends."""

from abc import ABC, abstractmethod


class DefacingBackend(ABC):
    """
    A defacing backend processes a DICOM series directory and writes
    defaced DICOM files to an output directory.

    Inputs:
      - series_dir: directory containing DICOM files for one series
      - output_dir: directory to write defaced DICOM files

    Returns:
      - list of output file paths
    """

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable name of this backend."""

    @abstractmethod
    def available(self) -> bool:
        """Return True if this backend's required tools are installed."""

    @abstractmethod
    def deface(self, series_dir: str, output_dir: str) -> list[str]:
        """
        Deface a DICOM series.

        Args:
            series_dir: Directory containing input DICOM files.
            output_dir: Directory to write defaced DICOM files.

        Returns:
            List of paths to the defaced DICOM files.

        Raises:
            RuntimeError on failure.
        """
