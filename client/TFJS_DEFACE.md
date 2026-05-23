# TF.js face de-identification

How to upgrade AEGIS's on-device face de-id from the anterior-heuristic
baseline (zero out the front of every axial slice) to a real
DeepDefacer-class model running in the browser via TensorFlow.js +
WebGPU.

The wire-up lives in `client/src/dicom/tfjs_deface.ts`; this doc explains
how to actually ship a model alongside it.

## What ships in this PR (the wire-up)

- `defaceVolumeWithTFJS(volume, options)` — given a composed 3D `Volume`
  and a `modelUrl`, lazy-loads TF.js, selects WebGPU/WebGL/WASM in that
  preference order, runs inference, dilates the predicted mask, and
  hands off to `writebackVolume()` to zero face voxels in the slices.
- `defaceStudy(..., { strategy: 'tfjs-model', modelUrl })` now dispatches
  to the TF.js path instead of returning `not_implemented`.

What's deliberately **not** in this PR: an actual trained model file.
That's a separate decision (which model, what license, where to host)
that should be made by whoever owns the deployment.

## Choosing a model

Three reasonable options, in increasing order of effort:

| Source | License | Approx. size | Notes |
|---|---|---|---|
| **DeepDefacer** | MIT | ~30 MB | 3-D U-Net trained on IXI + ABIDE. Designed for T1 MRI; degrades on T2 / CT. Has TF SavedModel format upstream. |
| **PyDeface + DL head detector** | Apache 2.0 | varies | Hybrid pipeline; would need ports for both components. |
| **In-house** | yours | varies | Train on whatever distribution your sites actually see. Most accurate but most work. |

For first deployment, DeepDefacer is the right starting point.

## Converting an upstream TF model to TF.js (one time, by a model author)

```bash
pip install tensorflowjs

tensorflowjs_converter \
    --input_format=tf_saved_model \
    --output_format=tfjs_graph_model \
    --signature_name=serving_default \
    deepdefacer/saved_model/ \
    static/models/deepdefacer-tfjs/

# Output:
#   static/models/deepdefacer-tfjs/model.json
#   static/models/deepdefacer-tfjs/group1-shard1of4.bin   ~8 MB
#   static/models/deepdefacer-tfjs/group1-shard2of4.bin   ~8 MB
#   static/models/deepdefacer-tfjs/group1-shard3of4.bin   ~8 MB
#   static/models/deepdefacer-tfjs/group1-shard4of4.bin   ~6 MB
```

The shards are downloaded on first use and cached by the browser
forever after, so this is a one-time ~30 MB hit per device per major
model version.

## Hosting the model

Three options:

1. **Bake into the upload-portal build** — drop the converted directory
   into `frontend/upload-portal/public/models/deepdefacer-tfjs/` and
   reference it as `/models/deepdefacer-tfjs/model.json`. Simplest, but
   ships the weights inside the SPA bundle on every static deploy.
2. **Host on a CDN** (recommended) — point a `Cache-Control: immutable`
   header at the directory; reference with a CDN URL. Bundle stays
   small, model updates are independent of code releases.
3. **Same-origin S3/GCS** — for sites that won't let the browser load
   model weights from an external CDN, host on the same origin as the
   API behind a `Cache-Control: max-age=31536000` header.

In all three, set:
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Headers: Content-Type, Range
```
TF.js does Range-request fetches for the shards.

## Wiring it into the upload portal

Once a model is hosted, the host passes `modelUrl` through `UploadFlow`'s
heavy mode:

```ts
<UploadFlow
  projectSlug="default"
  heavyModeDefaults={{ pixelScrub: true, faceDeid: true }}
  // ... and on the @aegis/client side via uploadStudy options:
  // faceDeid: {
  //   enabled: true,
  //   strategy: 'tfjs-model',
  //   modelUrl: 'https://models.aegisimaging.ai/deepdefacer-v1/model.json',
  // }
/>
```

Today the host doesn't pass `modelUrl` so the `'auto'` strategy in
`face_deid.ts` falls back to `'anterior-heuristic'`. When you're ready
to enable the real model, add the `modelUrl` to the project config and
flip the default strategy.

## Performance expectations

| Backend | 256³ volume inference | Notes |
|---|---|---|
| WebGPU | ~30 – 90 s (first run) | First run includes shader compilation; subsequent runs are ~5–15 s. Requires Chrome 113+, Safari 18+, Firefox behind a flag. |
| WebGL | ~90 – 240 s | Fallback for older browsers. |
| WASM (SIMD) | ~3 – 6 minutes | CPU fallback; only viable as a last resort. |
| CPU (vanilla JS) | "very long" | TF.js bails on volumes this size; don't ship this path to users. |

A 200-slice T1 study at 256³ resolution would take ~30 s end-to-end on a
modern M-series Mac (WebGPU) — including the volume composition, the
inference, the mask dilation, and the per-slice writeback.

## Validation against the server-side pipeline

Before flipping the default, validate parity:

```python
# 1. Run a corpus through the existing server-side defacing service.
# 2. Run the same corpus through the browser flow with tfjs-model.
# 3. Compute Dice between the resulting masks.
# 4. Require Dice ≥ 0.95 across the corpus.
```

The `defacing-service` already exposes `/deface` returning the modified
DICOM bytes; diff those against the TF.js-defaced bytes voxel-by-voxel.

## Dev tip

To experiment without hosting weights, point `modelUrl` at any TF.js
graph-model fixture (Keras MNIST works as a smoke test for the
backend-selection + inference plumbing). The mask output will be
garbage but the pipeline will execute end-to-end.
