# Heavy mode: in-browser pixel PHI + face de-identification

The default `@aegis/client` upload flow does **tag-level** de-identification
in the browser (PS3.15 Annex E Basic Profile), then ships the bytes to the
cloud for everything else. "Heavy mode" extends that with **pixel-level**
de-identification — burned-in text scrubbing today, MRI face removal in a
follow-up — so PHI-bearing pixel data never leaves the user's device.

## Use cases

- A researcher contributing handfuls of studies who wants the strongest
  on-device privacy posture and doesn't care that the upload is slower.
- A regulatory environment where pixel PHI cannot be sent to a third party
  for processing, even if the third party is HIPAA-compliant.
- IRB protocols that mandate on-device de-id as a control.

## What's shipping today

| Capability | Status | Notes |
|---|---|---|
| Tag de-id (PS3.15 Basic Profile) | ✅ default | Always on |
| Date shift | ✅ optional | Pass `deid: { dateShift }` to `uploadStudy` |
| **Pixel PHI scrub (text OCR + redact)** | ✅ heavy mode | This release — uncompressed + JPEG Baseline transfer syntaxes |
| **Face removal (MRI defacing)** | ⏳ stub | API exists; model inference is the next PR |
| Compressed pixel scrub (JPEG 2000 / JPEG-LS / RLE) | ❌ | Falls back to server-side scrub |

## API

```ts
import { uploadStudy } from '@aegis/client'

await uploadStudy(files, projectSlug, summary, {
  // ... existing options
  pixelScrub: {
    enabled: true,
    minConfidence: 0.4,         // OCR confidence floor (0-1)
    paddingPixels: 4,           // pad redaction boxes a few pixels for safety
    phiPatternsOnly: true,      // only redact PHI-shaped text (dates, names, MRNs, …)
    onResult: (filename, _i, result) => {
      console.log(filename, result.status, result.totalFindings, 'PHI regions')
    },
  },
})
```

The pixel scrub runs *between* tag de-id and the wire encode, so:

1. Parse DICOM → mutable dcmjs dataset.
2. Apply tag de-id (mutates in place).
3. **Heavy mode**: decode pixel data → OCR each frame → burn black rectangles
   over PHI findings into the dataset's `PixelData` bytes.
4. Serialize dataset → bytes are uploaded.

Compressed transfer syntaxes we can't decode in-browser today
(JPEG 2000 / JPEG-LS / RLE) are flagged in the returned `PixelScrubResult`
with `status: 'unsupported_transfer_syntax'`. The UI can decide whether to
upload anyway and let the server-side `phi-detection` sidecar handle it,
skip the file, or transcode upstream.

## Bundle size

- The default chunk is **unchanged** — `tesseract.js` is a peer dep, dynamic-imported.
- First time a user enables heavy mode in a session, the browser downloads
  `tesseract.js` (≈ 1.5 MB minified gzipped) plus its language data and WASM
  core (≈ 8-10 MB combined, cacheable forever once loaded).
- The OCR worker is reused across files within a session.

For self-hosted / offline deployment, pass `workerPath`, `corePath`, and
`langPath` in `PixelScrubOptions` to point at a copy of those assets you
host yourself rather than the default CDN.

## Performance

On a 2024-era laptop (M-series MacBook / modern x86):

| Image type | OCR per frame | Comments |
|---|---|---|
| 256×256 MR slice | 0.3–0.8 s | Typical, no PHI most of the time |
| 512×512 CT slice | 0.5–1.2 s | Slightly slower due to size |
| 1024×1024 secondary capture | 1–2 s | These are where burned-in PHI is most likely |
| 200-slice T1 study | 60–180 s | Largely sequential; one-off per study |

OCR is the dominant cost. The worker uses Web Workers internally so it
doesn't block the main thread, but only one frame at a time.

## Why not face de-id yet?

The model inference path is feasible (TF.js port of DeepDefacer, ~30 MB
weights) but two pieces aren't ready:

1. **Volume reconstruction**: composing an N-slice MR series into an [Z, Y, X]
   Float32Array, handling ImagePositionPatient / pixel spacing / rescale
   slope-intercept. Manageable but precise; deserves its own PR.
2. **Model conversion**: porting DeepDefacer's TensorFlow checkpoint to
   TF.js / ONNX Runtime Web with WebGPU acceleration. Validating the
   defaced output against the server-side `defacing` sidecar to confirm
   parity is the real work.

The face-deid module exists today as a stub so the UI toggle has a stable
surface to wire against. Calling `defaceStudy()` returns
`status: 'not_implemented'` with a `reason` string the UI surfaces.

## Tests

```bash
cd client
npm test            # vitest — pixel_decode, pixel_ocr (heuristic), pixel_scrub (with mock OCR)
```

`pixel_scrub.test.ts` mocks the OCR engine via vitest spies so the tests
don't need tesseract.js or a real WASM runtime — they verify the redaction
logic, frame iteration, padding, and error handling end-to-end against the
generic DICOM datasets defined in the test fixtures.

## Open questions / roadmap

- **JPEG 2000 / JPEG-LS / RLE pixel scrub** — needs WASM codecs (probably
  via `@cornerstonejs/dicom-image-loader`). Adds ~8 MB to the bundle.
- **Confidence threshold tuning** — 0.4 is a defensible default but a higher
  threshold reduces false-positive redactions of small protocol labels. The
  upload portal will eventually expose this in advanced settings.
- **Audit log of redactions** — embed the redaction set into a Private
  Creator block so downstream auditors can verify what was changed.
- **WebGPU + WebGL fallback for OCR** — Tesseract's WASM path is fast enough
  for the volumes we see today, but WebGPU-accelerated text detection (e.g.
  PaddleOCR) would be ~10× faster on supported hardware.
- **Desktop app variant** — same TS de-id pipeline, wrapped in Tauri,
  with native ONNX/Tesseract for hardware acceleration. Discussion at #TBD.
