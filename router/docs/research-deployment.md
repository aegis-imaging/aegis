# Research deployment guide

For PhD students, lab admins, and anyone who wants to use the AEGIS Router as
a research instrument — e.g. measuring anonymization accuracy, throughput,
or modality-specific de-id behavior, then writing about it.

## Why a research install differs from a production spoke

| Concern | Production spoke | Research install |
|---|---|---|
| Cloud forwarding | Required (always-on link to the central AEGIS) | Optional (you might just hold studies locally for analysis) |
| Persistence | Named volume is fine | Bind-mount to research storage; never lose a study |
| PHI input | Real patient studies (under DUA) | Mostly synthetic + IRB-approved real studies |
| Operator policy | Auto-clear quarantines on schedule | Inspect every quarantine; they're the interesting cases |
| Metrics | Health-only | Throughput + per-stage timing for the paper |
| Reproducibility | Pin to the cloud's expectation | Pin specific image tags; record `git rev-parse HEAD` |

## A reasonable thesis-grade setup

Hardware: anything from a workstation upward. The router itself is light;
defacing is the heaviest stage and benefits from 8+ cores. GPU optional.

```bash
# Pin the version you ran your experiments on
git checkout v0.1.0  # or whatever tag exists

cd router
cat > .env <<'EOF'
ROUTER_SITE_ID=umn-lab
ROUTER_SITE_NAME=University of Minnesota imaging lab
MIDI_B_SALT=replace-with-a-real-random-string

DIMSE_AE_TITLE=UMN_LAB
DIMSE_PORT=11112
HTTP_PORT=8080

# All processing happens locally; cloud forwarding off.
CLOUD_RECEIVER_URL=

DEFACING_SERVICE_URL=http://defacing:8080
PHI_DETECTION_SERVICE_URL=http://phi-detection:8080

# An operator key so /metrics and /audit aren't fully open even on a LAN
OPERATOR_API_KEY=$(openssl rand -hex 32)
EOF

# Bind-mount the data dir to durable research storage
cat > docker-compose.override.yml <<'EOF'
services:
  router:
    volumes:
      - /srv/aegis-router:/var/lib/aegis-router
EOF

sudo mkdir -p /srv/aegis-router
docker compose --profile defacing --profile phi up -d
```

## Sending studies

Synthetic (fastest for benchmarking — no IRB friction):

```bash
pip install pynetdicom pydicom httpx
./bin/aegis-router-send-test --slices 60 --runs 50 --no-verify --quiet
```

Real studies you have rights to:

```bash
storescu -aec UMN_LAB localhost 11112 /path/to/study/
# or
./bin/aegis-router-send-test --from-dir /path/to/study
```

Or use the browser uploader at `http://localhost:8080` (same UI as the
cloud's upload-portal; just pointed at the router).

## Measuring de-identification accuracy

After a study completes, you can compare raw vs de-identified tag-by-tag:

```bash
./bin/aegis-router-deid-diff \
    --raw  /srv/aegis-router/studies/<study_uid>/raw \
    --deid /srv/aegis-router/studies/<study_uid>/deid \
    --csv > experiment_001.csv
```

This is exactly the format you want for an evaluation table: tag, keyword,
VR, change kind (removed / blanked / modified / retained), old value, new
value. Re-run across a corpus of studies and aggregate by tag for a "what
percentage of PHI-bearing tags did we touch" number.

For pixel PHI (burned-in text), the `phi-detection` sidecar already returns
bounding boxes and confidence scores per finding — those land in the audit
log entries for the study. Pull them via `GET /audit?event=pipeline.shipped`
or query the SQLite quarantine for studies that failed the OCR threshold.

## Measuring throughput

The router exposes Prometheus metrics at `:8080/metrics`. Key series for a
throughput paper:

| Metric | Meaning |
|---|---|
| `aegis_router_studies_received_total{source=...}` | Counter of studies received, labelled by ingress path |
| `aegis_router_studies_shipped_total` | Counter of studies that completed the full pipeline (incl. ship if cloud configured) |
| `aegis_router_studies_local_only_total` | Counter of studies that completed but were not shipped (cloud off) |
| `aegis_router_studies_quarantined_total{stage=...}` | Counter of studies quarantined, labelled by failed stage |
| `aegis_router_pipeline_duration_seconds` | Histogram of end-to-end pipeline durations |
| `aegis_router_deid_duration_seconds` | Histogram of just the de-id portion |

A minimal Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: aegis-router
    static_configs:
      - targets: ['localhost:8080']
```

For ad-hoc measurements without Prometheus:

```bash
# Snapshot
curl http://localhost:8080/metrics

# Run a benchmark and capture per-study timings inline
./bin/aegis-router-send-test --slices 40 --runs 100 --no-verify
# → prints avg/median/max per-study at the end
```

## Reproducibility notes for the paper

When you publish results, include:
- The router image tag (`docker image inspect aegis-router:local | jq '.[0].Id'`)
- The compose profiles that were enabled
- The `MIDI_B_SALT` *was not the default* (so your pseudonymization is
  reproducible only by you — important for re-identification risk analysis)
- The `.env` values for `MIDI_B_DATE_SHIFT_MAX_DAYS` and `MIDI_B_RETAINED_TAGS`
- Sidecar backend used for defacing (`mri_deface` vs `deepdefacer` vs
  `mri_reface` — they have very different speed/quality tradeoffs; see
  `docs/research/mri-defacing-tools-comparison.md` in the main repo).

## What the router does NOT include (yet)

- Per-cohort batch analysis (would naturally hang off the `bids` and
  `analytics` profiles — those work, but the orchestration is per-study, not
  per-cohort).
- A built-in dashboard. Use the cloud admin-dashboard if you're forwarding,
  or hit the JSON APIs directly. (`/quarantine`, `/audit`, `/info`.)
- IRB/DUA enforcement. The router will happily ship anything pointed at it;
  policy enforcement is your responsibility.

## Citing this

If you publish work using the AEGIS Router, cite the repository:

> *Anonymization & Exchange Gateway for Imaging Studies (AEGIS).*
> https://github.com/aegis-imaging/aegis

And ideally note the specific git commit you ran. Pull requests welcome.
