"""Optional Vertex AI Imagen backend for synthetic brain MRI generation.

Only importable when the Dockerfile is built with INCLUDE_GEMINI=true, which
installs google-genai >= 1.0.0.

Uses the unified google-genai SDK. Supports two authentication modes:
  - Vertex AI (default): Application Default Credentials / Workload Identity
  - AI Studio: API key via GEMINI_API_KEY env var

Import guard: callers must handle ImportError when this module is unavailable.
"""

from __future__ import annotations

import io
import os

from pydicom.uid import generate_uid


def _imagen_available() -> bool:
    """Return True if the google-genai SDK is importable."""
    try:
        from google import genai  # noqa: F401
        return True
    except ImportError:
        return False


def generate_gemini_slices(
    n_slices: int = 20,
    size: int = 256,
    seed: int = 42,
) -> list:
    """Generate synthetic brain MRI slices using Vertex AI Imagen.

    Calls Imagen with a T1-weighted brain MRI prompt for each slice position,
    converts the response to grayscale numpy arrays, and returns SliceTuples
    compatible with phantom.write_dicom_series().

    Requires google-genai >= 1.0.0 (INCLUDE_GEMINI=true build arg).
    """
    import numpy as np
    from google import genai
    from google.genai import types
    from PIL import Image

    project_id = os.environ.get("GEMINI_PROJECT_ID", "")
    location = os.environ.get("GEMINI_LOCATION", "us-central1")
    api_key = os.environ.get("GEMINI_API_KEY", "")
    model_name = os.environ.get("IMAGEN_MODEL", "imagen-3.0-generate-002")

    if api_key:
        client = genai.Client(api_key=api_key)
    else:
        init_kwargs: dict = {"vertexai": True, "location": location}
        if project_id:
            init_kwargs["project"] = project_id
        client = genai.Client(**init_kwargs)

    study_uid = generate_uid()
    series_uid = generate_uid()
    slices = []

    for i in range(n_slices):
        # Vary the slice position descriptor in the prompt to encourage
        # anatomical variation across the stack (superior -> inferior).
        z_fraction = i / max(n_slices - 1, 1)
        if z_fraction < 0.25:
            position_desc = "inferior, showing cerebellum and brainstem"
        elif z_fraction < 0.5:
            position_desc = "mid-brain, showing basal ganglia and temporal lobes"
        elif z_fraction < 0.75:
            position_desc = "mid-superior, showing corpus callosum and parietal lobes"
        else:
            position_desc = "superior, showing cortical surface and sulci"

        prompt = (
            f"Axial T1-weighted brain MRI slice, {position_desc}. "
            "Grayscale medical imaging, high resolution, clinical quality. "
            "White matter bright, gray matter mid-tone, CSF dark. "
            "No annotations, no text overlays, clean background."
        )

        response = client.models.generate_images(
            model=model_name,
            prompt=prompt,
            config=types.GenerateImagesConfig(
                number_of_images=1,
                seed=seed + i,
            ),
        )

        # Convert Imagen response bytes -> PIL -> grayscale numpy [0, 1]
        img_bytes = response.generated_images[0].image.image_bytes
        pil_img = Image.open(io.BytesIO(img_bytes)).convert("L")
        pil_img = pil_img.resize((size, size), Image.LANCZOS)
        arr = np.array(pil_img, dtype=np.float32) / 255.0

        slice_z = float(i) - n_slices / 2.0
        slices.append((arr, study_uid, series_uid, i + 1, slice_z))

    return slices
