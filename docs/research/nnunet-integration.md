# nnU-Net v2 Integration Research for AEGIS Analytics Pipeline

**Date:** 2026-02-28
**Purpose:** Evaluate nnU-Net v2 for integration into the AEGIS neuroimaging analytics service as a segmentation backend.

---

## 1. What nnU-Net Does

nnU-Net (no-new-Net) is a **self-configuring deep learning framework for biomedical image segmentation**. Developed by the Medical Image Computing division at the German Cancer Research Center (DKFZ), it automatically adapts its entire pipeline -- preprocessing, network architecture, training, and post-processing -- to any given dataset without manual tuning.

**Core capabilities:**
- 2D and 3D semantic segmentation of medical images
- Supports arbitrary input modalities/channels (CT, MRI, PET, RGB, microscopy)
- Handles anisotropic voxel spacings automatically
- Image sizes from 40x40x40 to 1500x1500x1500 (3D) and 40x40 to ~30000x30000 (2D)
- Self-configuring: analyzes dataset fingerprints and automatically selects architecture, preprocessing, augmentation, and post-processing

**Key segmentation tasks it excels at:**
- **Brain tumor segmentation (BraTS)** -- won BraTS 2020 challenge with Dice scores of 88.95/85.06/82.03 for whole tumor/tumor core/enhancing tumor
- Organ segmentation (liver, kidney, spleen, pancreas)
- Cardiac MRI segmentation
- Head and neck tumor segmentation
- Pediatric brain tumor segmentation
- Lung tumor nodule segmentation
- Any Medical Segmentation Decathlon task

**Reference:** Isensee, F., Jaeger, P.F., Kohl, S.A.A. et al. nnU-Net: a self-configuring method for deep learning-based biomedical image segmentation. *Nature Methods* 18, 203--211 (2021). DOI: 10.1038/s41592-020-01008-z

---

## 2. Installation

### Package Name
- **PyPI:** `nnunetv2` (latest: v2.6.4, released 2026-02-04)
- **License:** Apache-2.0 (permissive, allows commercial use)
- **Python:** >=3.10 required
- **Repository:** https://github.com/MIC-DKFZ/nnUNet

### Installation Commands

**Standard (inference/baseline use):**
```bash
# CRITICAL: Install PyTorch first with correct CUDA support
pip install torch torchvision --index-url https://download.pytorch.org/whl/cu121
pip install nnunetv2
```

**Development (modify source):**
```bash
git clone https://github.com/MIC-DKFZ/nnUNet.git
cd nnUNet
pip install -e .
```

### Dependencies (from pyproject.toml)
```
torch>=2.1.2
numpy>=1.24
scipy
scikit-learn
scikit-image>=0.19.3
SimpleITK>=2.2.1
nibabel
pandas
tqdm
matplotlib
seaborn
tifffile
requests
graphviz
imagecodecs
yacs
einops
dicom2nifti
batchgenerators>=0.25
batchgeneratorsv2>=0.2
acvl-utils>=0.2,<0.3
dynamic-network-architectures>=0.3.1,<0.4
```

**Known issue:** Severe performance regression with PyTorch 2.9.0 and 3D convolutions when using AMP. Use PyTorch <=2.8.0.

### Required Environment Variables
```bash
export nnUNet_raw="/path/to/nnUNet_raw"              # Raw dataset storage
export nnUNet_preprocessed="/path/to/nnUNet_preprocessed"  # Preprocessed data
export nnUNet_results="/path/to/nnUNet_results"       # Trained models & results
```

Optional:
```bash
export nnUNet_n_proc_DA=12  # Data augmentation worker count (10-12 for RTX 2080ti, 16-18 for RTX 4090)
```

---

## 3. CLI Interface

nnU-Net v2 provides the following command-line tools:

### nnUNetv2_plan_and_preprocess
Dataset fingerprint extraction, experiment planning, and preprocessing.
```bash
nnUNetv2_plan_and_preprocess -d DATASET_ID --verify_dataset_integrity
```
- `-d DATASET_ID`: Dataset identifier (supports multiple: `-d 1 2 3`)
- `-c CONFIGURATION`: Specify config (e.g., `3d_fullres`)
- `-np`: Number of processes
- Output: `dataset_fingerprint.json` and `nnUNetPlans.json`

### nnUNetv2_train
Train U-Net models with 5-fold cross-validation.
```bash
nnUNetv2_train DATASET_NAME_OR_ID UNET_CONFIGURATION FOLD [options]
```
- `UNET_CONFIGURATION`: `2d`, `3d_fullres`, `3d_lowres`, `3d_cascade_fullres`
- `FOLD`: `0-4` or `all` for single model
- `--npz`: Save softmax outputs for ensembling
- `--val`: Validation only
- `-device DEVICE`: `cpu`, `cuda`, or `mps`
- `-num_gpus X`: Multi-GPU DDP training

### nnUNetv2_predict
Run inference on test cases.
```bash
nnUNetv2_predict -i INPUT_FOLDER -o OUTPUT_FOLDER -d DATASET_NAME_OR_ID -c CONFIGURATION
```
- `-i INPUT_FOLDER`: Input NIfTI files
- `-o OUTPUT_FOLDER`: Output segmentation masks
- `-f FOLD`: Fold selection (`all` for single, `0-4` for specific)
- `--save_probabilities`: Save probability maps (required for ensembling)
- Default: uses all 5 folds as ensemble

### nnUNetv2_find_best_configuration
Identifies optimal configuration and postprocessing.
```bash
nnUNetv2_find_best_configuration DATASET_NAME_OR_ID -c 2d 3d_fullres
```
- Output: `inference_instructions.txt` with ready-to-use prediction commands

### nnUNetv2_ensemble
Combine predictions from multiple configurations.
```bash
nnUNetv2_ensemble -i FOLDER1 FOLDER2 -o OUTPUT_FOLDER -np NUM_PROCESSES
```
- Requires `.npz` files from `nnUNetv2_predict --save_probabilities`

### nnUNetv2_apply_postprocessing
Apply determined postprocessing to predictions.
```bash
nnUNetv2_apply_postprocessing -i FOLDER_WITH_PREDICTIONS -o OUTPUT_FOLDER \
  --pp_pkl_file postprocessing.pkl -plans_json nnUNetPlans.json \
  -dataset_json dataset.json
```

### Model Portability
```bash
nnUNetv2_export_model_to_zip       # Package trained model for transfer
nnUNetv2_install_pretrained_model_from_zip path/to/model.zip  # Import model
```

### v1 Migration
```bash
nnUNetv2_convert_old_nnUNet_dataset INPUT_FOLDER OUTPUT_DATASET_NAME
```

---

## 4. Input Format

nnU-Net accepts **NIfTI files** (`.nii.gz`), **PNG**, and **TIFF** -- it does NOT accept raw DICOM directly.

### Dataset Directory Structure (Medical Segmentation Decathlon format)
```
nnUNet_raw/
  DatasetXXX_NAME/
    dataset.json          # Metadata: channels, labels, file list
    imagesTr/             # Training images
      CASE_001_0000.nii.gz   # Channel 0 (e.g., T1)
      CASE_001_0001.nii.gz   # Channel 1 (e.g., T2-FLAIR) -- if multi-modal
      CASE_002_0000.nii.gz
    labelsTr/             # Training labels (segmentation masks)
      CASE_001.nii.gz
      CASE_002.nii.gz
    imagesTs/             # Test images (same naming as imagesTr)
      CASE_100_0000.nii.gz
```

### File Naming Convention
- Images: `{CASE_ID}_{CHANNEL:04d}.nii.gz` (e.g., `brain_001_0000.nii.gz`)
- Labels: `{CASE_ID}.nii.gz` (e.g., `brain_001.nii.gz`)
- Channel suffix `_0000` = first modality, `_0001` = second, etc.

### dataset.json Format
```json
{
    "channel_names": {"0": "T1", "1": "T2-FLAIR"},
    "labels": {"background": 0, "whole_tumor": 1, "tumor_core": 2, "enhancing_tumor": 3},
    "numTraining": 100,
    "file_ending": ".nii.gz"
}
```

### AEGIS Integration Note
Since AEGIS already has a BIDS conversion service (dcm2niix), the pipeline would be:
**DICOM -> BIDS service (dcm2niix) -> NIfTI -> nnU-Net inference**

---

## 5. Output Format

nnU-Net produces:
- **Segmentation masks** as NIfTI files (`.nii.gz`) -- integer labels matching `dataset.json` label definitions
- **Probability maps** (optional, `.npz` format) -- per-class softmax probabilities
- Same spatial dimensions, orientation, and voxel spacing as input

For brain tumor segmentation (BraTS), output labels are typically:
- 0 = background
- 1 = necrotic/non-enhancing tumor core
- 2 = peritumoral edema
- 3 = GD-enhancing tumor

---

## 6. Pre-trained Models

### Distribution Mechanism
- **Export:** `nnUNetv2_export_model_to_zip` packages trained model folders into portable zip files
- **Import:** `nnUNetv2_install_pretrained_model_from_zip path/to/model.zip` installs into `nnUNet_results`
- Both machines must have nnU-Net installed with matching dependencies

### Available Pre-trained Models

**Zenodo (official reference evaluation):**
- All datasets from the original nnU-Net paper: https://zenodo.org/record/3734294
- Specific models hosted at various Zenodo DOIs by challenge participants

**BraTS Brain Tumor Models:**
- BraTS 2020 winning model (nnU-Net variant): Dice 88.95/85.06/82.03
- BraTS 2024/2025 challenge submissions available through respective challenge repositories
- Many researchers share trained models via Zenodo or HuggingFace

**MONAI Model Zoo:**
- Pre-trained checkpoints for BraTS 2018/2021 datasets
- Swin UNETR BraTS 2021 weights: https://github.com/Project-MONAI/research-contributions/tree/main/SwinUNETR/BRATS21
- MONAI has an nnUNet runner wrapper: `monai.apps.nnunet.nnunetv2_runner`

**NVIDIA NGC:**
- NVIDIA provides an optimized nnU-Net implementation in their Deep Learning Examples repository
- Includes training and inference scripts with mixed precision support

### For AEGIS: Recommended Approach
1. **Short-term:** Use MONAI's pre-trained BraTS models (easiest to download and deploy)
2. **Medium-term:** Train custom nnU-Net models on AEGIS datasets for specific segmentation tasks
3. **Long-term:** Maintain a model registry with versioned nnU-Net model zips

---

## 7. Resource Requirements

### Training
| Resource | Minimum | Recommended |
|----------|---------|-------------|
| GPU VRAM | 10 GB | 24+ GB (RTX 3090/4090) |
| CPU cores | 6 (12 threads) | 16+ |
| RAM | 32 GB | 64 GB |
| Storage | SSD required | M.2 PCIe Gen 3+ |

### Inference
| Resource | Minimum | Recommended |
|----------|---------|-------------|
| GPU VRAM | 4 GB | 8+ GB |
| CPU cores | 4 | 8+ |
| RAM | 16 GB | 32 GB |
| CPU-only | Supported | Much slower |
| Apple MPS | Supported (2D only) | No 3D conv support |

### Processing Time Benchmarks (Brain MRI)
| Configuration | Hardware | Time |
|---------------|----------|------|
| Standard nnU-Net (3d_fullres, 5-fold ensemble) | GPU (T4/V100) | ~84 seconds per volume |
| Advanced nnU-Net variant | GPU | ~119 seconds per volume |
| Single fold, no TTA | GPU | ~10-30 seconds per volume |
| With full test-time augmentation | GPU | Up to 2-5 minutes |
| CPU inference | CPU | 5-20 minutes per volume |

**Note:** Disabling test-time augmentation (`use_mirroring=False`) significantly reduces inference time with moderate accuracy loss. For production pipelines, single-fold inference without TTA is often sufficient.

### Data Augmentation Worker Recommendations
| GPU | `nnUNet_n_proc_DA` |
|-----|---------------------|
| RTX 2080ti | 10-12 |
| RTX 3090 | 12 |
| RTX 4090 | 16-18 |
| A100 | 28-32 |

---

## 8. Docker

### NVIDIA NGC Container (Official)
NVIDIA provides an optimized nnU-Net Docker container through the NGC catalog:

**Repository:** https://github.com/NVIDIA/DeepLearningExamples/blob/master/PyTorch/Segmentation/nnUNet/

**Build:**
```bash
git clone https://github.com/NVIDIA/DeepLearningExamples.git
cd DeepLearningExamples/PyTorch/Segmentation/nnUNet
docker build -t nnunet .
```

**Run:**
```bash
docker run -it --gpus all --shm-size=8g \
  --ulimit memlock=-1 --ulimit stack=67108864 --rm \
  -v ${PWD}/data:/data -v ${PWD}/results:/results \
  nnunet:latest /bin/bash
```

- Based on PyTorch NGC container
- Includes NVIDIA DALI, PyTorch Lightning
- Mixed precision (AMP) with Tensor Cores
- Supports Volta, Turing, Ampere architectures

### Custom Dockerfile for AEGIS (Inference-only)
```dockerfile
FROM pytorch/pytorch:2.8.0-cuda12.1-cudnn9-runtime

ENV nnUNet_raw="/app/data/nnunet_raw" \
    nnUNet_preprocessed="/app/data/nnunet_preprocessed" \
    nnUNet_results="/app/data/nnunet_results"

RUN pip install --no-cache-dir nnunetv2==2.6.4

# Copy pre-trained models
COPY models/ /app/data/nnunet_results/

WORKDIR /app
COPY app/ /app/

EXPOSE 8080
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8080"]
```

**Estimated Docker image size:** ~4-6 GB (PyTorch + CUDA runtime + nnU-Net + model weights)

---

## 9. Python API

The `nnUNetPredictor` class is the primary programmatic interface for inference.

### Basic Inference Example
```python
import torch
from nnunetv2.inference.predict_from_raw_data import nnUNetPredictor

# Initialize predictor
predictor = nnUNetPredictor(
    tile_step_size=0.5,           # Overlap for sliding window
    use_gaussian=True,            # Gaussian weighting for tile borders
    use_mirroring=True,           # Test-time augmentation (set False for speed)
    perform_everything_on_device=True,
    device=torch.device('cuda', 0),
    verbose=False,
    verbose_preprocessing=False,
    allow_tqdm=True
)

# Load trained model
predictor.initialize_from_trained_model_folder(
    '/path/to/nnUNet_results/DatasetXXX_NAME/nnUNetTrainer__nnUNetPlans__3d_fullres',
    use_folds=(0,),                    # Single fold for speed, or (0,1,2,3,4) for ensemble
    checkpoint_name='checkpoint_final.pth',
)
```

### Method 1: predict_from_files (Recommended -- folder or file list)
```python
# From folders
predictor.predict_from_files(
    '/path/to/input_niftis/',
    '/path/to/output_segmentations/',
    save_probabilities=False,
    overwrite=False,
    num_processes_preprocessing=2,
    num_processes_segmentation_export=2,
)

# From file lists (multi-modal: nested lists)
predictor.predict_from_files(
    [['/path/to/case1_0000.nii.gz'], ['/path/to/case2_0000.nii.gz']],
    ['/path/to/output/case1', '/path/to/output/case2'],
    save_probabilities=False,
    overwrite=False,
)

# Return arrays instead of writing files
segmentations = predictor.predict_from_files(
    [['/path/to/case1_0000.nii.gz']],
    None,  # None = return arrays, don't write files
    save_probabilities=False,
)
```

### Method 2: predict_from_list_of_npy_arrays (In-memory numpy arrays)
```python
from nnunetv2.imageio.simpleitk_reader_writer import SimpleITKIO

img, props = SimpleITKIO().read_images(['/path/to/input_0000.nii.gz'])

results = predictor.predict_from_list_of_npy_arrays(
    [img],
    None,            # Output files (None = return results)
    [props],
    None,            # Output properties
    num_processes_preprocessing=2,
    save_probabilities=False,
)
```

### Method 3: predict_single_npy_array (Single volume, main thread)
```python
from nnunetv2.imageio.simpleitk_reader_writer import SimpleITKIO

img, props = SimpleITKIO().read_images(['/path/to/input_0000.nii.gz'])
segmentation = predictor.predict_single_npy_array(img, props, None, None, False)
```

### Method 4: predict_from_data_iterator (Custom pipeline)
```python
def my_iterator(input_arrays, input_props):
    preprocessor = predictor.configuration_manager.preprocessor_class(verbose=False)
    for arr, prop in zip(input_arrays, input_props):
        data, seg = preprocessor.run_case_npy(
            arr, None, prop,
            predictor.plans_manager,
            predictor.configuration_manager,
            predictor.dataset_json
        )
        yield {
            'data': torch.from_numpy(data).contiguous().pin_memory(),
            'data_properties': prop,
            'ofile': None
        }

results = predictor.predict_from_data_iterator(
    my_iterator([img1, img2], [props1, props2]),
    save_probabilities=False,
    num_processes_segmentation_export=3
)
```

### I/O Backends
```python
from nnunetv2.imageio.simpleitk_reader_writer import SimpleITKIO  # Default
from nnunetv2.imageio.nibabel_reader_writer import NibabelIO       # Alternative
```

---

## 10. License

**Apache License 2.0** -- permissive open-source license.

- Commercial use: Allowed
- Modification: Allowed
- Distribution: Allowed
- Patent use: Granted
- Private use: Allowed
- Condition: Must include license notice and attribution
- No warranty

Pre-trained models may have separate licenses (check individual Zenodo/HuggingFace entries). BraTS challenge models often use CC-BY or similar academic licenses.

---

## AEGIS Integration Architecture

### Proposed Backend: `nnunet` in analytics-service

```
analytics-service/
  app/
    backends/
      nnunet_backend.py      # nnUNetPredictor wrapper
    main.py                   # FastAPI endpoints
  models/                     # Pre-trained model zips
  Dockerfile
```

### Integration Flow
```
DICOM (raw) -> BIDS service (dcm2niix) -> NIfTI (.nii.gz)
    -> analytics-service (nnU-Net backend)
        -> Load NIfTI
        -> Run nnUNetPredictor.predict_single_npy_array()
        -> Save segmentation mask as NIfTI
        -> Return volumetric statistics (label counts, volumes in mm3)
    -> Store results in audit trail / analytics output directory
```

### Environment Variables (proposed)
```
NNUNET_MODEL_DIR=/app/data/nnunet_results    # Pre-trained models
NNUNET_DATASET_ID=001                         # Dataset identifier for the model
NNUNET_CONFIGURATION=3d_fullres               # U-Net configuration
NNUNET_FOLDS=0                                # Fold(s) to use (0 for speed, 0,1,2,3,4 for accuracy)
NNUNET_USE_MIRRORING=false                    # Test-time augmentation (false for speed)
NNUNET_DEVICE=cuda                            # cuda, cpu, or mps
```

### Dockerfile Build Args
```
INCLUDE_NNUNET=false    # Install nnunetv2 + PyTorch (~4-6 GB)
NNUNET_GPU=false        # Include CUDA runtime
```

### Key Considerations
1. **Large image size:** PyTorch + CUDA + nnU-Net + model weights = 4-6 GB Docker image
2. **GPU requirement:** Inference is feasible on CPU but slow (~5-20 min vs ~30s-2min on GPU)
3. **Input format:** Requires NIfTI -- AEGIS BIDS service already produces this
4. **Model management:** Pre-trained models need to be bundled or downloaded at container startup
5. **Memory:** 4+ GB GPU VRAM for inference; 8+ GB recommended for brain volumes

---

## Sources

- [GitHub - MIC-DKFZ/nnUNet](https://github.com/MIC-DKFZ/nnUNet) -- Official repository
- [nnunetv2 on PyPI](https://pypi.org/project/nnunetv2/) -- Package (v2.6.4, Apache-2.0)
- [nnU-Net Inference API](https://github.com/MIC-DKFZ/nnUNet/blob/master/nnunetv2/inference/readme.md) -- Python API reference
- [Installation Instructions](https://github.com/MIC-DKFZ/nnUNet/blob/master/documentation/installation_instructions.md) -- System requirements
- [How to Use nnU-Net](https://github.com/MIC-DKFZ/nnUNet/blob/master/documentation/how_to_use_nnunet.md) -- CLI reference
- [Environment Variables](https://github.com/MIC-DKFZ/nnUNet/blob/master/documentation/set_environment_variables.md) -- Path setup
- [NVIDIA NGC nnU-Net for PyTorch](https://catalog.ngc.nvidia.com/orgs/nvidia/teams/dle/resources/nnunet_pyt) -- NGC container
- [NVIDIA DeepLearningExamples nnU-Net](https://github.com/NVIDIA/DeepLearningExamples/blob/master/PyTorch/Segmentation/nnUNet/README.md) -- Docker reference
- [nnU-Net for Brain Tumor Segmentation (arXiv)](https://arxiv.org/abs/2011.00848) -- BraTS 2020 winning submission
- [MONAI Model Zoo - SwinUNETR BraTS21](https://github.com/Project-MONAI/research-contributions/tree/main/SwinUNETR/BRATS21) -- Alternative pre-trained models
- [Zenodo Pre-trained Models](https://zenodo.org/record/3734294) -- Official reference evaluation models
- Isensee, F. et al. "nnU-Net: a self-configuring method for deep learning-based biomedical image segmentation." *Nature Methods* 18, 203-211 (2021). DOI: 10.1038/s41592-020-01008-z (peer-reviewed)
