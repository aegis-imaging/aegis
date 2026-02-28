"""LLM-based free-text PHI scrubbing backends.

Uses Gemini, OpenAI, or Anthropic to identify and replace PHI tokens in
free-text DICOM fields while preserving non-PHI content. Falls back to
regex-based scrubbing when no LLM backend is available.

Each backend receives free text + optional context (patient name, ID, etc.)
and returns the scrubbed text with PHI tokens replaced by [REMOVED].
"""

import json
import logging
import os
import re
from abc import ABC, abstractmethod

log = logging.getLogger(__name__)

_REPLACEMENT = "[REMOVED]"

_SYSTEM_PROMPT = """You are a HIPAA compliance expert. Your task is to identify Protected Health Information (PHI) in DICOM free-text fields and replace each PHI token with [REMOVED] while preserving all non-PHI text exactly as-is.

PHI includes: patient names, dates of birth, medical record numbers, accession numbers, Social Security numbers, phone numbers, email addresses, street addresses, ZIP codes, physician names, hospital/clinic names, IP addresses, URLs, and any other personally identifiable information.

NON-PHI that must be preserved includes: medical terminology, anatomical terms, imaging modality names (CT, MRI, PET), sequence types (T1, FLAIR, DWI), procedure descriptions, clinical findings, and standard radiology language.

Rules:
1. Replace ONLY PHI tokens with [REMOVED]
2. Preserve all non-PHI text exactly (including punctuation, spacing, and capitalization)
3. If consecutive PHI tokens are adjacent, use a single [REMOVED]
4. If the text contains NO PHI at all, return it unchanged

Respond with ONLY a JSON object: {"scrubbed": "<the scrubbed text>", "phi_found": true/false}
Do not include any explanation outside the JSON."""


class TextScrubBackend(ABC):
    """Abstract base for text PHI scrubbing backends."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Human-readable backend name."""

    @abstractmethod
    def available(self) -> bool:
        """Return True if this backend is ready to use."""

    @abstractmethod
    def scrub(self, text: str, context: dict | None = None) -> dict:
        """Scrub PHI from text.

        Args:
            text: Free-text value from a DICOM field.
            context: Optional dict with patient_name, patient_id,
                     referring_physician, institution_name.

        Returns:
            {"scrubbed": str, "phi_found": bool}
        """


def _build_user_prompt(text: str, context: dict | None = None) -> str:
    """Build the user prompt with text and optional context."""
    parts = [f'Scrub PHI from this DICOM free-text field:\n\n"{text}"']
    if context:
        hints = []
        if context.get("patient_name"):
            hints.append(f"Patient name: {context['patient_name']}")
        if context.get("patient_id"):
            hints.append(f"Patient ID: {context['patient_id']}")
        if context.get("referring_physician"):
            hints.append(f"Referring physician: {context['referring_physician']}")
        if context.get("institution_name"):
            hints.append(f"Institution: {context['institution_name']}")
        if hints:
            parts.append("\nKnown identifiers to watch for:\n" + "\n".join(f"- {h}" for h in hints))
    return "\n".join(parts)


def _parse_llm_response(response_text: str, original_text: str) -> dict:
    """Parse the JSON response from the LLM."""
    if not response_text:
        return {"scrubbed": original_text, "phi_found": False}

    # Strip code fences if present
    cleaned = re.sub(r"```(?:json)?\s*", "", response_text).strip().rstrip("`").strip()

    # Try to find a JSON object
    match = re.search(r"\{.*\}", cleaned, re.DOTALL)
    if not match:
        log.debug("LLM returned non-JSON: %s", cleaned[:200])
        return {"scrubbed": original_text, "phi_found": False}

    try:
        result = json.loads(match.group())
        scrubbed = str(result.get("scrubbed", original_text))
        phi_found = bool(result.get("phi_found", False))
        return {"scrubbed": scrubbed, "phi_found": phi_found}
    except (json.JSONDecodeError, ValueError):
        log.debug("LLM returned invalid JSON: %s", cleaned[:200])
        return {"scrubbed": original_text, "phi_found": False}


# ---------------------------------------------------------------------------
# Gemini backend
# ---------------------------------------------------------------------------

class GeminiTextScrubBackend(TextScrubBackend):
    """Text PHI scrubbing using Gemini multimodal API."""

    def __init__(self):
        self._api_key = os.environ.get("GEMINI_API_KEY", "")
        self._project_id = os.environ.get("GEMINI_PROJECT_ID", "")
        self._location = os.environ.get("GEMINI_LOCATION", "us-central1")
        self._model_name = os.environ.get("GEMINI_MODEL", "gemini-2.5-flash")
        self._client = None

    @property
    def name(self) -> str:
        return "gemini"

    def available(self) -> bool:
        try:
            from google import genai  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            from google import genai
            if self._api_key:
                self._client = genai.Client(api_key=self._api_key)
            else:
                init_kwargs: dict = {"vertexai": True, "location": self._location}
                if self._project_id:
                    init_kwargs["project"] = self._project_id
                self._client = genai.Client(**init_kwargs)
        return self._client

    def scrub(self, text: str, context: dict | None = None) -> dict:
        from google.genai import types

        client = self._get_client()
        user_prompt = _build_user_prompt(text, context)

        response = client.models.generate_content(
            model=self._model_name,
            contents=[user_prompt],
            config=types.GenerateContentConfig(
                system_instruction=_SYSTEM_PROMPT,
                response_mime_type="application/json",
            ),
        )
        return _parse_llm_response(response.text, text)


# ---------------------------------------------------------------------------
# OpenAI backend
# ---------------------------------------------------------------------------

class OpenAITextScrubBackend(TextScrubBackend):
    """Text PHI scrubbing using OpenAI API (GPT-4o-mini)."""

    def __init__(self):
        self._api_key = os.environ.get("OPENAI_API_KEY", "")
        self._model = os.environ.get("OPENAI_MODEL", "gpt-4o-mini")
        self._client = None

    @property
    def name(self) -> str:
        return "openai"

    def available(self) -> bool:
        if not self._api_key:
            return False
        try:
            import openai  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            import openai
            self._client = openai.OpenAI(api_key=self._api_key)
        return self._client

    def scrub(self, text: str, context: dict | None = None) -> dict:
        client = self._get_client()
        user_prompt = _build_user_prompt(text, context)

        response = client.chat.completions.create(
            model=self._model,
            messages=[
                {"role": "system", "content": _SYSTEM_PROMPT},
                {"role": "user", "content": user_prompt},
            ],
            response_format={"type": "json_object"},
            temperature=0,
        )
        return _parse_llm_response(response.choices[0].message.content, text)


# ---------------------------------------------------------------------------
# Anthropic backend
# ---------------------------------------------------------------------------

class AnthropicTextScrubBackend(TextScrubBackend):
    """Text PHI scrubbing using Anthropic Claude API."""

    def __init__(self):
        self._api_key = os.environ.get("ANTHROPIC_API_KEY", "")
        self._model = os.environ.get("ANTHROPIC_MODEL", "claude-haiku-4-5-20251001")
        self._client = None

    @property
    def name(self) -> str:
        return "anthropic"

    def available(self) -> bool:
        if not self._api_key:
            return False
        try:
            import anthropic  # noqa: F401
            return True
        except ImportError:
            return False

    def _get_client(self):
        if self._client is None:
            import anthropic
            self._client = anthropic.Anthropic(api_key=self._api_key)
        return self._client

    def scrub(self, text: str, context: dict | None = None) -> dict:
        client = self._get_client()
        user_prompt = _build_user_prompt(text, context)

        response = client.messages.create(
            model=self._model,
            max_tokens=1024,
            system=_SYSTEM_PROMPT,
            messages=[{"role": "user", "content": user_prompt}],
        )
        response_text = response.content[0].text if response.content else ""
        return _parse_llm_response(response_text, text)


# ---------------------------------------------------------------------------
# Regex fallback (same patterns as client-side text_scrub.ts)
# ---------------------------------------------------------------------------

_PHI_PATTERNS = [
    (r"\b\d{3}-\d{2}-\d{4}\b", "SSN"),
    (r"\b\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b", "phone"),
    (r"\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b", "email"),
    (r"\b(?:MRN|MR#|MED\s*REC)\s*[:#]?\s*\d+\b", "MRN"),
    (r"\b(?:ACC|ACCESSION)\s*[:#]?\s*[A-Z0-9\-]+\b", "accession"),
    (r"\b(?:0?[1-9]|1[0-2])[/\-](?:0?[1-9]|[12]\d|3[01])[/\-](?:19|20)?\d{2}\b", "date_us"),
    (r"\b(?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|Jul(?:y)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\s+\d{1,2},?\s*\d{4}\b", "date_written"),
    (r"\b(?:19|20)\d{2}[/\-](?:0?[1-9]|1[0-2])[/\-](?:0?[1-9]|[12]\d|3[01])\b", "date_iso"),
    (r"\b\d+\s+\w+(?:\s+\w+)?\s+(?:street|avenue|road|boulevard|drive|lane|court|way|place|circle|parkway)\b", "address"),
    (r"\b(?:Dr|Mr|Mrs|Ms|Prof|Rev)\.?\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*", "title_name"),
    (r"\b\w+(?:\s+\w+)*\s+(?:Hospital|Clinic|Medical\s+Center|Health\s+System|Health\s+Center|Healthcare|University\s+Hospital)\b", "hospital"),
    (r"\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b", "ip_address"),
    (r"\bhttps?://[^\s]+", "url"),
    (r"\b(?:Patient|Pt)\s*[:#]?\s*[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*", "patient_prefix"),
]


class RegexTextScrubBackend(TextScrubBackend):
    """Regex-based fallback text PHI scrubber (no LLM needed)."""

    @property
    def name(self) -> str:
        return "regex"

    def available(self) -> bool:
        return True  # Always available

    def scrub(self, text: str, context: dict | None = None) -> dict:
        if not text:
            return {"scrubbed": "", "phi_found": False}

        spans: list[tuple[int, int]] = []

        # Context-based matching
        if context:
            for key in ("patient_name", "patient_id", "referring_physician"):
                val = context.get(key, "")
                if not val:
                    continue
                # Split DICOM name on ^ , whitespace
                tokens = [t.strip().lower() for t in re.split(r"[\^,\s]+", val) if len(t.strip()) > 1]
                for token in tokens:
                    for m in re.finditer(rf"\b{re.escape(token)}\b", text, re.IGNORECASE):
                        spans.append((m.start(), m.end()))

            inst = context.get("institution_name", "")
            if inst:
                for m in re.finditer(rf"\b{re.escape(inst)}\b", text, re.IGNORECASE):
                    spans.append((m.start(), m.end()))

        # Pattern-based matching
        for pattern, _label in _PHI_PATTERNS:
            for m in re.finditer(pattern, text, re.IGNORECASE):
                spans.append((m.start(), m.end()))

        if not spans:
            return {"scrubbed": text, "phi_found": False}

        # Merge overlapping spans
        spans.sort()
        merged: list[tuple[int, int]] = [spans[0]]
        for s, e in spans[1:]:
            if s <= merged[-1][1]:
                merged[-1] = (merged[-1][0], max(merged[-1][1], e))
            else:
                merged.append((s, e))

        # Build scrubbed string
        result = []
        cursor = 0
        for s, e in merged:
            result.append(text[cursor:s])
            result.append(_REPLACEMENT)
            cursor = e
        result.append(text[cursor:])
        scrubbed = "".join(result)

        # Collapse consecutive [REMOVED] tokens
        scrubbed = re.sub(r"(\[REMOVED\](?:\s+\[REMOVED\])+)", _REPLACEMENT, scrubbed)

        return {"scrubbed": scrubbed, "phi_found": True}


# ---------------------------------------------------------------------------
# Backend selection
# ---------------------------------------------------------------------------

def select_text_scrub_backend(tool: str = "auto") -> TextScrubBackend:
    """Select the best available text scrubbing backend.

    Priority: gemini > openai > anthropic > regex (always available).
    """
    gemini = GeminiTextScrubBackend()
    openai_be = OpenAITextScrubBackend()
    anthropic_be = AnthropicTextScrubBackend()
    regex = RegexTextScrubBackend()

    if tool == "gemini":
        candidates = [gemini]
    elif tool == "openai":
        candidates = [openai_be]
    elif tool == "anthropic":
        candidates = [anthropic_be]
    elif tool == "regex":
        candidates = [regex]
    else:  # "auto"
        candidates = [gemini, openai_be, anthropic_be, regex]

    for backend in candidates:
        if backend.available():
            log.info("Text scrub backend selected: %s", backend.name)
            return backend

    # Should never reach here since regex is always available
    return regex
