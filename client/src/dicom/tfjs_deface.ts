/**
 * TF.js-backed face de-identification.
 *
 * Architecture-only in this iteration — the runtime imports TensorFlow.js
 * lazily so the standard upload (and even the anterior-heuristic strategy)
 * never pays the bundle cost. The actual model weights ship out-of-band:
 * point `modelUrl` at a hosted `model.json` and the binary is fetched on
 * first use, cached forever after.
 *
 * Pipeline:
 *   1. Compose the per-series 3D volume (already done by face_deid.ts).
 *   2. Pick the longest axial-orientation series; bail if shape isn't ≥3D.
 *   3. Resample to the model's input shape (typically 128³ for DeepDefacer)
 *      via trilinear interpolation in TF.js — no native ONNX needed.
 *   4. Normalize intensities (z-score against the volume's mean/std).
 *   5. Run inference → 3D foreground-vs-face probability map.
 *   6. Threshold + light morphological close (single dilation) to fill holes.
 *   7. Upsample mask back to original resolution.
 *   8. Hand off to writebackVolume to zero face voxels in the slices.
 *
 * Backend selection: prefer WebGPU (fast on modern Chrome/Edge/Safari TP),
 * fall back to WebGL, then CPU/WASM. Caller can override.
 *
 * Model conversion (one-off, by the cloud admin or model author):
 *   tensorflowjs_converter --input_format=tf_saved_model \
 *       --output_format=tfjs_graph_model deepdefacer/saved_model/ \
 *       static/models/deepdefacer-tfjs/
 *   # Host static/models/deepdefacer-tfjs/ behind CORS, point modelUrl at
 *   # the resulting model.json.
 */

import type { Volume } from './volume'

export type TFBackend = 'webgpu' | 'webgl' | 'wasm' | 'cpu'

export interface TFJSDefaceOptions {
  /** Required: URL to a TF.js model.json (graph-model format from tensorflowjs_converter). */
  modelUrl: string
  /**
   * Backend preference order. Default ['webgpu', 'webgl', 'wasm']. The first
   * one that initialises successfully is used.
   */
  backendPreference?: TFBackend[]
  /** Model input edge length in voxels. Default 128 (matches DeepDefacer). */
  modelInputSize?: number
  /** Foreground threshold for the model output (0-1). Default 0.5. */
  maskThreshold?: number
  /** Padding applied around the predicted face mask before writeback (voxels). Default 2. */
  maskPaddingVoxels?: number
  /** Progress callback for "loading model" / "running inference" / "writing back". */
  onProgress?: (phase: TFJSPhase, fraction: number) => void
}

export type TFJSPhase = 'init' | 'loading_model' | 'preparing' | 'inference' | 'postprocess' | 'writeback'

export interface TFJSDefaceResult {
  voxelsModified: number
  backendUsed: TFBackend
  modelInferenceMs: number
  totalMs: number
}

export class TFJSUnavailableError extends Error {
  constructor(detail: string) {
    super(`TF.js model strategy unavailable: ${detail}`)
    this.name = 'TFJSUnavailableError'
  }
}

/**
 * Run TF.js defacing on a composed volume. Returns the number of voxels that
 * were zeroed (and the volume is mutated in place via writebackVolume).
 *
 * Throws TFJSUnavailableError if TF.js isn't installed or no backend
 * initialised — callers should catch and fall back to the anterior-heuristic
 * strategy (see face_deid.ts).
 */
export async function defaceVolumeWithTFJS(
  volume: Volume,
  options: TFJSDefaceOptions,
): Promise<TFJSDefaceResult> {
  const started = nowMs()
  options.onProgress?.('init', 0)

  const tf = await loadTFJS()
  options.onProgress?.('init', 0.5)
  const backend = await selectBackend(tf, options.backendPreference)

  // Load the model. The first time this runs it fetches modelUrl + the binary
  // shards; downloads are cached by the browser, subsequent runs are instant.
  options.onProgress?.('loading_model', 0)
  const model = await tf.loadGraphModel(options.modelUrl)
  options.onProgress?.('loading_model', 1)

  // ── Preprocess ─────────────────────────────────────────────────────────
  options.onProgress?.('preparing', 0)
  const inputSize = Math.max(32, options.modelInputSize ?? 128)
  const inputTensor = tf.tidy(() => {
    // [Z, Y, X] → trilinear-interpolated [inputSize, inputSize, inputSize]
    const flat = tf.tensor3d(
      Array.from(volume.voxels), [volume.shape.z, volume.shape.y, volume.shape.x],
    )
    const resampled = resample3D(tf, flat, inputSize)
    // Z-score normalise.
    const mean = resampled.mean()
    const std = resampled.sub(mean).square().mean().sqrt()
    const z = resampled.sub(mean).div(std.add(1e-6))
    // Add batch and channel dims: [1, D, H, W, 1]
    return z.expandDims(0).expandDims(-1)
  })
  options.onProgress?.('preparing', 1)

  // ── Inference ──────────────────────────────────────────────────────────
  const infStart = nowMs()
  options.onProgress?.('inference', 0)
  const output = model.execute(inputTensor) as TFTensor
  const inferenceMs = nowMs() - infStart
  options.onProgress?.('inference', 1)

  // ── Postprocess ────────────────────────────────────────────────────────
  options.onProgress?.('postprocess', 0)
  const threshold = options.maskThreshold ?? 0.5
  const lowResMask = tf.tidy(() => output.squeeze().greater(threshold))
  // Resample the low-res mask back to the original volume shape using
  // nearest-neighbour (face voxels are a binary mask; smooth interpolation
  // would soften the edges into ambiguity).
  const fullMask = tf.tidy(() =>
    resample3D(tf, lowResMask.toFloat() as TFTensor, undefined, [volume.shape.z, volume.shape.y, volume.shape.x])
      .greater(0.5),
  )
  const padding = Math.max(0, options.maskPaddingVoxels ?? 2)
  const dilatedMask = padding > 0 ? dilate3D(tf, fullMask, padding) : fullMask
  const maskData = await dilatedMask.data() as Uint8Array | Int32Array | Float32Array

  // Clean up tensors before the (potentially slow) JS writeback loop.
  tf.dispose([inputTensor, output, lowResMask, fullMask, dilatedMask])
  options.onProgress?.('postprocess', 1)

  // ── Writeback ──────────────────────────────────────────────────────────
  options.onProgress?.('writeback', 0)
  const { writebackVolume } = await import('./volume')
  const sliceVoxels = volume.shape.y * volume.shape.x
  const background = volume.stats.min
  const { modifiedVoxels } = writebackVolume(volume, (x, y, z, current) => {
    const i = z * sliceVoxels + y * volume.shape.x + x
    return maskData[i] ? background : current
  })
  options.onProgress?.('writeback', 1)

  return {
    voxelsModified: modifiedVoxels,
    backendUsed: backend,
    modelInferenceMs: inferenceMs,
    totalMs: nowMs() - started,
  }
}

// ── helpers ────────────────────────────────────────────────────────────────

/** Lazily load TensorFlow.js via the string-import trick so this file
 *  type-checks even when `@tensorflow/tfjs` isn't in node_modules. */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function loadTFJS(): Promise<any> {
  try {
    const pkg = '@tensorflow/tfjs'
    return await import(/* @vite-ignore */ pkg)
  } catch (e) {
    throw new TFJSUnavailableError(
      '@tensorflow/tfjs is not installed. Add it as a peer dep and install in the host that wants TF.js defacing.'
    )
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function selectBackend(tf: any, preference?: TFBackend[]): Promise<TFBackend> {
  const order: TFBackend[] = preference?.length
    ? preference
    : ['webgpu', 'webgl', 'wasm', 'cpu']
  for (const candidate of order) {
    try {
      await tf.setBackend(candidate)
      await tf.ready()
      const actual = tf.getBackend()
      if (actual === candidate) return candidate as TFBackend
    } catch {
      continue
    }
  }
  throw new TFJSUnavailableError(`no usable backend (tried ${order.join(', ')})`)
}

/**
 * Trilinear resample a 3-D tensor to either a uniform cube edge `size`, or
 * an explicit target shape `[Z, Y, X]`. TF.js has 2-D image.resize but no
 * 3-D primitive — we slice along Z, resize each XY plane, then slice along
 * the new XY plane stack, resize Z. Two passes, both on the GPU.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function resample3D(tf: any, vol: any, size?: number, target?: [number, number, number]): any {
  const [origZ, origY, origX] = vol.shape as [number, number, number]
  const [outZ, outY, outX] = target ?? [size!, size!, size!]
  // Pass 1: resize each Z-slice in the XY plane → [origZ, outY, outX]
  const reshapedXY = vol.reshape([origZ, origY, origX, 1])
  const xyResized = tf.image.resizeBilinear(reshapedXY, [outY, outX]).reshape([origZ, outY, outX])
  // Pass 2: resize along Z. Treat the volume as [outY, origZ, outX, 1].
  const transposed = xyResized.transpose([1, 0, 2]).reshape([outY, origZ, outX, 1])
  const zResized = tf.image.resizeBilinear(transposed, [outZ, outX]).reshape([outY, outZ, outX])
  return zResized.transpose([1, 0, 2])
}

/** Crude 3D dilation by `radius` voxels via three orthogonal 1-D
 *  max-pools. Doesn't grow strictly equal to a sphere, but is plenty for
 *  closing pinprick holes in the predicted face mask. */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function dilate3D(tf: any, mask: any, radius: number): any {
  // Mask shape: [Z, Y, X]. Dilation by max-pool with kernel 2r+1 and stride 1.
  const k = 2 * radius + 1
  const padded = mask.reshape([1, ...mask.shape, 1]).toFloat()
  // axis-X
  const x = tf.maxPool3d(padded, [1, 1, k], 1, 'same')
  // axis-Y
  const y = tf.maxPool3d(x, [1, k, 1], 1, 'same')
  // axis-Z
  const z = tf.maxPool3d(y, [k, 1, 1], 1, 'same')
  return z.greater(0.5).squeeze()
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type TFTensor = any

function nowMs(): number {
  return typeof performance !== 'undefined' && typeof performance.now === 'function'
    ? performance.now()
    : Date.now()
}
