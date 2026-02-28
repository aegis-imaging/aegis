# SAM 2 (Segment Anything Model 2) — Medical Imaging Integration Research

**Date:** 2026-02-28
**Purpose:** Evaluate SAM 2 and its medical derivatives as a general-purpose segmentation backend for the AEGIS analytics pipeline.

---

## 1. SAM 2 Fundamentals

### Repository & License

- **GitHub:** https://github.com/facebookresearch/sam2
- **License:** Apache 2.0 (model checkpoints, training code, demo code)
- **Paper:** Ravi et al., "SAM 2: Segment Anything in Images and Videos" (Meta FAIR, 2024)
- **Released:** July 2024; SAM 2.1 update December 2024

### Installation

```bash
# Requires Python >= 3.10, torch >= 2.5.1, torchvision >= 0.20.1
git clone https://github.com/facebookresearch/sam2.git && cd sam2
pip install -e .
# Or from PyPI:
pip install sam2
```

CUDA-enabled GPU recommended. Works on CPU but slowly. Windows users should use WSL.

### Model Checkpoints (SAM 2.1)

| Model | Parameters | Checkpoint Size | FPS (A100) |
|-------|-----------|----------------|------------|
| sam2.1_hiera_tiny | 38.9M | ~156 MB | 91.2 |
| sam2.1_hiera_small | 46M | ~184 MB | 84.8 |
| sam2.1_hiera_base_plus | 80.8M | ~323 MB | 64.1 |
| sam2.1_hiera_large | 224.4M | ~898 MB | 39.5 |

All models use the Hiera (Hierarchical Vision Transformer) backbone.

### Python API

**Image prediction (single image):**
```python
from sam2.sam2_image_predictor import SAM2ImagePredictor

predictor = SAM2ImagePredictor.from_pretrained("facebook/sam2-hiera-large")
predictor.set_image(image)  # numpy array HxWx3

# Prompted segmentation (points, boxes, or masks)
masks, scores, logits = predictor.predict(
    point_coords=np.array([[x, y]]),      # optional
    point_labels=np.array([1]),            # 1=foreground, 0=background
    box=np.array([x1, y1, x2, y2]),       # optional bounding box
    multimask_output=True                  # returns 3 candidate masks
)
# masks: (N, H, W) boolean arrays
# scores: confidence per mask
```

**Video prediction (tracking across frames):**
```python
from sam2.sam2_video_predictor import SAM2VideoPredictor

predictor = SAM2VideoPredictor.from_pretrained("facebook/sam2-hiera-large")
state = predictor.init_state(video_path)

# Add prompt on one frame, propagates to all
frame_idx, obj_ids, masks = predictor.add_new_points_or_box(
    state, frame_idx=0, obj_id=1,
    points=np.array([[x, y]]), labels=np.array([1])
)

# Propagate through video
for frame_idx, obj_ids, masks in predictor.propagate_in_video(state):
    # masks per frame per object
    pass
```

**Automatic mask generation (everything mode):**
```python
from sam2.automatic_mask_generator import SAM2AutomaticMaskGenerator

generator = SAM2AutomaticMaskGenerator.from_pretrained("facebook/sam2-hiera-large")
masks = generator.generate(image)
# Returns list of dicts: {segmentation, area, bbox, predicted_iou, ...}
```

### Input/Output Formats

- **Input:** NumPy arrays (H, W, 3) for images; JPEG/PNG directory or MP4 for video
- **Output:** Binary masks as boolean NumPy arrays (N, H, W); confidence scores; bounding boxes
- **No native NIfTI support** — requires preprocessing to extract 2D slices

### GPU Memory (VRAM) — Measured Benchmarks

Single image inference (float16, no attention optimizations):

| Model | 576x1024 | 1024x1024 | 2048x2048 |
|-------|----------|-----------|-----------|
| Tiny | 240 MB | 458 MB | 1,158 MB |
| Base+ | 328 MB | 551 MB | 1,275 MB |
| Large | 624 MB | 855 MB | 1,626 MB |

Video tracking at 1024x1024 (steady-state, caching disabled):
- Tiny: ~1.2 GB
- Base+: ~1.3 GB
- Large: ~1.7 GB
- Each additional tracked object adds ~3-4 MB

Source: https://github.com/facebookresearch/sam2/issues/118

**Key finding:** SAM 2 is extremely memory-efficient — even the largest model runs on a 4 GB GPU for single images. Video/3D volume tracking needs ~2 GB. This is far lighter than most DL segmentation models.

---

## 2. Medical Imaging Adaptations

### 2.1 MedSAM2 (bowang-lab) — Primary Medical Derivative

- **GitHub:** https://github.com/bowang-lab/MedSAM2
- **Paper:** Cheng et al., "MedSAM2: Segment Anything in 3D Medical Images and Videos" (arXiv:2504.03600, April 2025)
- **Project page:** https://medsam2.github.io/
- **HuggingFace:** https://huggingface.co/wanglab/MedSAM2

**Architecture:** Fine-tuned SAM 2.1 Tiny variant with:
- Input resolution reduced to 512x512 (from 1024x1024)
- Memory-attention block allowing each slice to examine 8 neighbors
- Retains SAM 2's Hiera backbone

**Training:**
- 455,000+ 3D image-mask pairs + 76,000 annotated video frames
- 10 imaging modalities: CT, MRI, PET, ultrasound, endoscopy, etc.
- Trained 70 epochs on 3 nodes x 4 H100 GPUs (4 days)
- Human-in-the-loop pipeline for annotation

**Accuracy (Dice scores):**

| Task | MedSAM2 DSC | Notes |
|------|-------------|-------|
| CT organs | 88.84% | IQR: 80.03-94.03% |
| CT lesions | 86.68% | IQR: 74.32-91.14% |
| MRI organs | 87.06% | IQR: 82.96-90.04% |
| MRI lesions | 88.37% | IQR: 79.91-93.26% |
| PET lesions | 87.22% | IQR: 79.07-90.45% |
| LV (echo video) | 96.13% | Left ventricle |
| LV epicardium | 93.10% | Echo video |
| Left atrium | 95.79% | Echo video |

**3D Volume Processing:** Treats sequential 2D slices as video frames. Applies bounding box prompt on the middle slice and propagates bidirectionally toward both ends. Does NOT natively process 3D volumes — uses SAM 2's video tracking paradigm.

**Installation:**
```bash
conda create -n medsam2 python=3.12 -y
pip install torch==2.5.1 torchvision==0.20.1
git clone https://github.com/bowang-lab/MedSAM2.git && cd MedSAM2
pip install -e ".[dev]"
bash download.sh  # download checkpoints
```

**Prompting:** Supports bounding box, point prompts, and automatic prompt generation (RECIST markers on middle slice).

**Clinical impact:** Reduces manual annotation costs by >85%.

### 2.2 SAM-Med3D — Native 3D Architecture

- **GitHub:** https://github.com/uni-medical/SAM-Med3D (also https://github.com/openmedlab/SAM-Med3D)
- **Paper:** Wang et al., "SAM-Med3D: Towards General-Purpose Segmentation Models for Volumetric Medical Images" (ECCV 2024 Workshops)
- **License:** Apache 2.0

**Architecture:** Converts all SAM components to 3D: 3D image encoder, 3D prompt encoder, 3D mask decoder. Natively processes volumetric data (not slice-by-slice).

**Key advantage:** Requires 10-100x fewer prompt points than 2D slice-by-slice approaches.

**Training data:** SA-Med3D-140K dataset (70 public datasets + 8K private cases = 22K 3D images, 143K masks).

**Input:** NIfTI format (.nii.gz) natively supported.

**Installation:**
```bash
conda create --name sammed3d python=3.10
pip install torch==2.6.0 torchvision==0.21.0
pip install torchio opencv-python-headless monai medim
```

**Variants:** SAM-Med3D-turbo (fine-tuned on 44 datasets for improved performance).

### 2.3 Brain-SAM — Brain MRI Lesion Specialist

- **Paper:** "Brain-SAM: A SAM-based Model Tailored for Brain MRI Lesion Segmentation" (medRxiv, February 2026)
- **Preprint:** https://www.medrxiv.org/content/10.64898/2026.01.30.26345164v2.full

**Architecture:**
- Extends SAM 2's Hiera encoder to process 3D volumetric data directly (Conv3D patch embedding, 3D position embedding)
- UNETR-inspired decoder for hierarchical feature decoding
- Supports both prompted and fully automatic segmentation

**Accuracy:**
- 3D-Hiera achieves Dice 0.7286 (vs SAM-Med3D's 3D-ViT at 0.6880)
- UNETR decoder adds +8% Dice improvement
- Tested on brain tumors, stroke, epilepsy datasets

### 2.4 SAM2-UNet — Encoder Transfer Architecture

- **GitHub:** https://github.com/WZH0120/SAM2-UNet
- **Paper:** Published at VINT 2026

Uses SAM 2's Hiera as encoder backbone paired with a UNet-style decoder. Applicable to both natural and medical image segmentation.

### 2.5 SAM2-3dMed

- **Paper:** "SAM2-3dMed: Empowering SAM2 for 3D Medical Image Segmentation" (arXiv:2510.08967, October 2025)

Introduces Slice Relative Position Prediction (SRPP) module that explicitly models bidirectional inter-slice dependencies, addressing the gap between video sequences and medical volume slices.

### 2.6 SAMed-2 (Selective Memory Enhanced)

- **Paper:** MICCAI 2025

Adds temporal adapter to the image encoder + confidence-driven memory mechanism to store high-certainty features. Captures inter-slice correlations more effectively.

### 2.7 SpinalSAM-R1 — Spine CT Specialist

- **Paper:** "SpinalSAM-R1: A Vision-Language Multimodal Interactive System for Spine CT Segmentation" (arXiv:2511.00095, October 2025)

Integrates fine-tuned SAM with DeepSeek-R1 LLM for natural language-guided spine CT segmentation. Anatomy-guided attention mechanism.

### 2.8 Other Notable Adaptations

| Model | Focus | Year | Notes |
|-------|-------|------|-------|
| SAM2-SGP | Few-shot medical segmentation | 2025 | Support-set guided prompting; outperforms nnU-Net and SwinUNet |
| TGSAM-2 | Text-guided medical segmentation | MICCAI 2025 | Text prompts instead of points/boxes |
| Sam2Rad | Radiology with learnable prompts | 2025 | Eliminates manual prompts entirely |
| Onco-Seg | Oncology concept segmentation | 2026 | Cancer-specific adaptation |
| FS-MedSAM2 | Few-shot without fine-tuning | 2025 | Zero-shot transfer |

---

## 3. SAM 3 — Latest Generation (November 2025)

- **GitHub:** https://github.com/facebookresearch/sam3
- **Paper:** ICLR 2026
- **License:** Open-source

**Key difference from SAM 2:** Introduces Promptable Concept Segmentation (PCS) — can detect, segment, and track all instances of a concept specified by text prompts or image exemplars. Uses dual encoder-decoder Transformer architecture.

**Medical relevance:** Still struggles with ultra-fine-grained "out-of-domain" concepts (specific cells) but adapts fast with small fine-tuning datasets. Uses less VRAM than SAM 2 (fits on 16 GB GPUs).

**Relevance for AEGIS:** SAM 3 could enable text-prompted segmentation ("segment all vertebrae", "segment hippocampus") without point/box prompts. However, medical-specific fine-tuned versions do not yet exist.

---

## 4. Key Integration Questions — Answered

### Does SAM 2 work on 3D volumes (NIfTI) or only 2D images?

**SAM 2 base:** 2D only. Processes images or video frames. No native NIfTI support.

**For 3D volumes, three approaches exist:**
1. **MedSAM2 approach:** Treat volume slices as video frames, prompt on middle slice, propagate bidirectionally. Works well but is fundamentally 2D-per-slice.
2. **SAM-Med3D approach:** Native 3D architecture with 3D convolutions. True volumetric understanding. Requires NIfTI input.
3. **Brain-SAM approach:** 3D-adapted Hiera encoder with UNETR decoder. True 3D but brain-specific.

### Does it need prompts or can it segment automatically?

**SAM 2 base:** Has "everything mode" (`SAM2AutomaticMaskGenerator`) but it aggressively prunes low-confidence proposals — poor recall for medical structures.

**For automatic medical segmentation:**
- MedSAM2 can auto-generate box prompts from RECIST markers
- Brain-SAM supports fully automatic mode
- Sam2Rad uses learnable prompts (no manual input needed)
- SAM2-SGP uses support-set guided prompting (few-shot, no manual prompts)
- Best results still come from minimal manual prompting (1 point or 1 box per target)

**Bottom line:** For a server-side pipeline (no human in the loop), you need either (a) a fine-tuned automatic model, or (b) a detection model (YOLO, etc.) to generate box prompts automatically.

### GPU requirements (VRAM, inference time)?

**SAM 2 base:** Extremely lightweight — 240 MB to 1.6 GB VRAM depending on model size and resolution. Can run on 4 GB consumer GPUs for single images.

**MedSAM2:** Uses Tiny variant at 512x512 — inference is <1 second per slice. Training needed 12 H100 GPUs.

**For AEGIS integration:** A T4 GPU (16 GB VRAM) or even a consumer RTX 3060 (12 GB) would be sufficient for inference.

### Can it do instance segmentation (label individual structures)?

**SAM 2 base:** Produces binary masks. Each prompt generates one mask. To label multiple structures, you need multiple prompts (one per structure) or multi-pass inference.

**SAM 2 does NOT produce semantic labels.** It segments "something" but does not know what it segmented. You need to provide the label externally.

**For anatomical labeling (e.g., individual vertebrae, brain ROIs):**
- Requires either atlas registration or a specialized model
- TotalSpineSeg, SPINEPS, SynthSeg, TotalSegmentator all provide labeled segmentation natively
- SAM-family models would need post-processing to assign anatomical labels

### Pre-trained models for spine/brain?

**Spine:**
- SpinalSAM-R1 (SAM + DeepSeek-R1 for spine CT)
- General SAM/MedSAM evaluated on lumbar spine MRI — underperforms nnU-Net (Sensors, June 2025)
- No SAM 2 model matches TotalSpineSeg or SPINEPS accuracy on vertebral instance segmentation

**Brain:**
- Brain-SAM (3D-Hiera + UNETR, tumors/stroke/epilepsy, Dice ~0.73)
- SAM2-UNet for brain tumors (Dice 0.92 with bounding box prompts on LGG dataset)
- YOLOv12-SAM 2 hybrid for brain tumors (99.71% segmentation accuracy, Dice 0.9185)
- No SAM 2 model matches SynthSeg for whole-brain parcellation (32/97 ROIs)

---

## 5. Comparison with Specialized Tools

### Brain Segmentation

| Tool | Dice (brain) | Prompt needed? | Labels | Speed | 3D native? |
|------|-------------|----------------|--------|-------|------------|
| SynthSeg | ~0.85-0.90 | No (automatic) | 32 or 97 ROIs | ~6s GPU / ~2min CPU | Yes |
| Brain-SAM | ~0.73 | Optional | Binary per prompt | Seconds per volume | Yes (3D Hiera) |
| MedSAM2 | ~0.87 (organs) | 1 box/point | Binary per prompt | <1s per slice | No (2D+propagation) |
| FreeSurfer recon-all | ~0.85-0.90 | No (automatic) | 100+ ROIs | 6-8 hours | Yes |

**Verdict:** SynthSeg and FreeSurfer remain far superior for brain parcellation. SAM variants are better for lesion/tumor segmentation where the target is a single focal structure.

### Spine Segmentation

| Tool | Dice (vertebrae) | Instance labeling? | Modality | Notes |
|------|------------------|-------------------|----------|-------|
| TotalSpineSeg | ~0.90+ | Yes (C1-S5 + discs) | Multi-contrast MRI + CT | nnU-Net backbone |
| SPINEPS | 0.933 (T2w) | Yes (14 structures) | T2w MRI only | Two-phase semantic + instance |
| MedSAM/SAM | < nnU-Net | No (binary only) | Any (with prompts) | Cannot label individual vertebrae |

**Verdict:** TotalSpineSeg and SPINEPS dramatically outperform SAM-family models for vertebral instance segmentation. SAM lacks the ability to assign anatomical labels.

### Whole-Body / Multi-Organ

| Tool | Structures | Automatic? | Dice range | Notes |
|------|-----------|------------|------------|-------|
| TotalSegmentator | 117 structures | Yes | 0.85-0.95 | nnU-Net, CT primary |
| MedSAM2 | Any (prompted) | Semi-auto | 0.87-0.89 | Needs 1 prompt per target |
| nnU-Net | Task-specific | Yes | 0.85-0.95+ | Gold standard, task-specific training |

---

## 6. Known Limitations of SAM 2 for Medical Imaging

1. **Low contrast structures:** Struggles with structures lacking sharp boundaries (caudate, thalamus, soft tissue interfaces). Most errors within 1-2 pixels of boundary. (arXiv:2511.19471)
2. **Small structures:** Poor performance on small targets (pancreas Dice as low as 0.361 without fine-tuning).
3. **No semantic labels:** Produces binary masks — cannot differentiate "liver" from "kidney" without external label assignment.
4. **3D understanding gap:** Video-frame propagation is not the same as true 3D spatial reasoning. Inter-slice consistency can break down.
5. **Prompt dependency:** Zero-shot "everything mode" has low recall for medical structures. Fine-tuning or external detector needed for automatic operation.
6. **Cross-modality inconsistency:** Without fine-tuning, performance varies heavily by modality. Ultrasound and X-ray are particularly challenging.
7. **Memory growth in video mode:** VRAM increases linearly with frame count unless caching is disabled, which can cause OOM on long volumes (200+ slices).

---

## 7. Integration Recommendation for AEGIS

### Role: Complementary/Interactive Tool, NOT Primary Backend

SAM 2 (and its medical derivatives) is best suited as a **complementary interactive segmentation tool** rather than a replacement for specialized backends. Here is the recommended integration strategy:

#### Keep specialized backends for automated pipelines:
- **SynthSeg** — brain parcellation (32/97 ROIs, fully automatic, contrast-agnostic)
- **TotalSegmentator** — whole-body CT/MRI (117 structures, fully automatic)
- **nnU-Net** — task-specific segmentation (tumor, lesion detection)
- **FreeSurfer** — cortical reconstruction and volumetric analysis
- **FSL/ANTs** — brain extraction, registration, cortical thickness

These tools produce labeled, multi-class segmentations automatically without prompts — exactly what an automated pipeline needs.

#### Use SAM 2/MedSAM2 for interactive annotation and edge cases:
1. **Interactive segmentation UI** — admin clicks on a structure in a NIfTI viewer, SAM 2 segments it instantly. Useful for manual QC corrections or annotating novel structures.
2. **Lesion annotation** — MedSAM2 excels at focal lesion segmentation with a single box prompt. Faster than manual contouring.
3. **Custom ROI extraction** — when researchers need to segment a structure not covered by the standard atlas (e.g., specific tumor subregions).
4. **Segmentation refinement** — use SAM 2 to refine or correct masks produced by automated pipelines.

#### If adding SAM 2 as an AEGIS backend:

**Recommended model:** MedSAM2 (bowang-lab) — best accuracy on medical data, lightweight (Tiny variant), <1s per slice.

**Implementation approach:**
```python
# In analytics-service/app/backends/medsam2.py
class MedSAM2Backend:
    """Interactive/prompted segmentation using MedSAM2."""

    def is_available(self) -> bool:
        # Check for torch + medsam2 + checkpoint
        pass

    def segment_volume(self, nifti_path: str, prompts: dict) -> str:
        """
        prompts: {
            "slice_idx": int,  # which slice to prompt on
            "box": [x1, y1, x2, y2],  # or
            "points": [[x, y, label], ...],
            "propagate": "bidirectional"  # or "forward"/"backward"
        }
        Returns path to output segmentation NIfTI.
        """
        # 1. Load NIfTI, extract slices
        # 2. Initialize MedSAM2 video predictor
        # 3. Apply prompt on target slice
        # 4. Propagate through volume
        # 5. Save masks as NIfTI
        pass
```

**Priority:** Low — add after BrainSuite, ITK-SNAP, PETSurfer, and MONAI Label backends are stable. SAM 2 is most valuable when paired with a viewer UI for interactive use, which AEGIS does not currently have for NIfTI data.

---

## 8. Citations

1. Ravi, N. et al. "SAM 2: Segment Anything in Images and Videos." Meta FAIR, 2024. GitHub: https://github.com/facebookresearch/sam2. Apache 2.0.

2. Cheng, J. et al. "MedSAM2: Segment Anything in 3D Medical Images and Videos." arXiv:2504.03600, April 2025. GitHub: https://github.com/bowang-lab/MedSAM2.

3. Wang, H. et al. "SAM-Med3D: Towards General-Purpose Segmentation Models for Volumetric Medical Images." ECCV 2024 Workshops. GitHub: https://github.com/uni-medical/SAM-Med3D. Apache 2.0.

4. "Brain-SAM: A SAM-based Model Tailored for Brain MRI Lesion Segmentation." medRxiv, February 2026. https://www.medrxiv.org/content/10.64898/2026.01.30.26345164v2.full. Preprint.

5. "Advanced Brain Tumor Segmentation Using SAM2-UNet." Applied Sciences 15(6):3267, 2025. https://www.mdpi.com/2076-3417/15/6/3267. Peer-reviewed.

6. "SpinalSAM-R1: A Vision-Language Multimodal Interactive System for Spine CT Segmentation." arXiv:2511.00095, October 2025.

7. "Improving medical image segmentation with SAM2." European Journal of Radiology AI, 2025. https://www.ejrai.com/article/S3050-5771(25)00032-5/fulltext. Peer-reviewed.

8. "Segment Anything Model (SAM) and Medical SAM (MedSAM) for Lumbar Spine MRI." Sensors 25(12):3596, June 2025. https://pubmed.ncbi.nlm.nih.gov/40573483/. Peer-reviewed.

9. "Not Quite Anything: Overcoming SAM's Limitations for 3D Medical Imaging." arXiv:2511.19471, November 2025. Preprint.

10. "Using Segment Anything Model 2 for Zero-Shot 3D Segmentation of Abdominal Organs in CT." JMIR AI, 2025. https://ai.jmir.org/2025/1/e72109/. Peer-reviewed.

11. "Research on Medical Image Segmentation Based on SAM and Its Future Prospects." PMC/Bioengineering 12(6):608, 2025. https://pmc.ncbi.nlm.nih.gov/articles/PMC12189367/. Peer-reviewed.

12. Meta. "SAM 3: Segment Anything with Concepts." ICLR 2026. GitHub: https://github.com/facebookresearch/sam3.

13. "SAM4MIS: Segment Anything Model for Medical Image Segmentation — Open-Source Project Summary." GitHub: https://github.com/YichiZhang98/SAM4MIS (comprehensive list of SAM-for-medical repos).
