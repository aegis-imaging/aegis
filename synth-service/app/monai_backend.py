"""Optional MONAI BraTS LDM GPU backend.

Only importable when the Dockerfile is built with INCLUDE_MONAI=true, which installs:
  monai-generative, huggingface-hub, torch, scikit-image

Model: monai-test/brats_mri_axial_slices_generative_diffusion (HuggingFace)
Architecture: AutoencoderKL + DDIM 50-step DiffusionModelUNet, scale_factor=0.3

Import guard: callers must handle ImportError when this module is unavailable.
"""

from __future__ import annotations


def generate_monai_slices(
    n_slices: int = 20,
    size: int = 256,
) -> list:
    """Generate synthetic brain MRI slices with the MONAI BraTS LDM.

    Requires CUDA GPU and monai-generative + torch to be installed.
    Returns a list of SliceTuples compatible with phantom.write_dicom_series().
    """
    import torch
    from generative.inferers import LatentDiffusionInferer
    from generative.networks.nets import AutoencoderKL, DiffusionModelUNet
    from generative.schedulers import DDIMScheduler
    from huggingface_hub import hf_hub_download
    from monai.utils import set_determinism
    from pydicom.uid import generate_uid

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    set_determinism(42)

    # Download pretrained weights from HuggingFace Hub (cached after first run).
    repo = "monai-test/brats_mri_axial_slices_generative_diffusion"
    ae_weights = hf_hub_download(repo_id=repo, filename="autoencoder.pth")
    unet_weights = hf_hub_download(repo_id=repo, filename="diffusion_model.pth")

    autoencoder = AutoencoderKL(
        spatial_dims=2,
        in_channels=1,
        out_channels=1,
        latent_channels=3,
        num_channels=(256, 512, 512),
        attention_levels=(False, True, True),
        with_encoder_nonlocal_attn=False,
        with_decoder_nonlocal_attn=False,
        num_res_blocks=2,
        norm_num_groups=16,
    ).to(device)
    autoencoder.load_state_dict(torch.load(ae_weights, map_location=device))
    autoencoder.eval()

    diffusion_model = DiffusionModelUNet(
        spatial_dims=2,
        in_channels=3,
        out_channels=3,
        num_channels=(256, 256, 512),
        attention_levels=(False, True, True),
        num_head_channels=(0, 256, 512),
        num_res_blocks=2,
        with_conditioning=False,
    ).to(device)
    diffusion_model.load_state_dict(torch.load(unet_weights, map_location=device))
    diffusion_model.eval()

    scheduler = DDIMScheduler(num_train_timesteps=1000)
    scheduler.set_timesteps(num_inference_steps=50)
    scale_factor = 0.3
    inferer = LatentDiffusionInferer(scheduler=scheduler, scale_factor=scale_factor)

    # Latent spatial size is size // 4 (VAE encoder downsamples 4x).
    latent_size = size // 4
    study_uid = generate_uid()
    series_uid = generate_uid()

    slices = []
    with torch.no_grad():
        for i in range(n_slices):
            noise = torch.randn((1, 3, latent_size, latent_size), device=device)
            synth = inferer.sample(
                input_noise=noise,
                autoencoder_model=autoencoder,
                diffusion_model=diffusion_model,
                scheduler=scheduler,
            )
            img = synth[0, 0].cpu().numpy()
            img_min, img_max = img.min(), img.max()
            img = (img - img_min) / (img_max - img_min + 1e-8)

            if img.shape != (size, size):
                from skimage.transform import resize as sk_resize
                img = sk_resize(img, (size, size), anti_aliasing=True)

            slice_z = float(i) - n_slices / 2.0
            slices.append((img, study_uid, series_uid, i + 1, slice_z))

    return slices
