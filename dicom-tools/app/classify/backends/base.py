"""Abstract base for classification backends."""

from abc import ABC, abstractmethod
from dataclasses import dataclass, field


@dataclass
class ClassificationResult:
    """Result of a DICOM metadata classification."""
    modality: str = ""
    body_part: str = ""
    confidence: float = 0.0
    method: str = ""  # e.g. "dicom_tags", "sop_class", "series_description"


class ClassificationBackend(ABC):
    """Interface that every classification backend must implement."""

    @property
    @abstractmethod
    def name(self) -> str: ...

    @abstractmethod
    def available(self) -> bool: ...

    @abstractmethod
    def classify(self, input_paths: list[str]) -> ClassificationResult: ...
