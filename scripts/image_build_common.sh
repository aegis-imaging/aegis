# Shared by aws_build_push_images.sh and azure_build_push_images.sh.
# Source this file; it defines the service → build-context manifest and a
# build helper. Plain bash 3 (macOS) — no associative arrays.

# Services built when no --services / SERVICES filter is given.
DEFAULT_IMAGE_SERVICES="api admin-dashboard defacing phi-detection qc-service bids-service classification-service protocol-service dimse-receiver"

# Prints "<context>:<dockerfile>" for a service name, or fails for unknown names.
# The six light Python sidecars ship in the shared dicom-tools image, so they
# are built from that context and pushed under each service's own name.
image_build_spec() {
  case "$1" in
    api)                    echo "api:api/Dockerfile" ;;
    admin-dashboard)        echo ".:frontend/admin-dashboard/Dockerfile" ;;
    upload-portal)          echo ".:frontend/upload-portal/Dockerfile" ;;
    dwv)                    echo "dwv:dwv/Dockerfile" ;;
    mcp-server)             echo "mcp-server:mcp-server/Dockerfile" ;;
    defacing)               echo "defacing:defacing/Dockerfile" ;;
    phi-detection|qc-service|bids-service|classification-service|protocol-service|synth-service)
                            echo "dicom-tools:dicom-tools/Dockerfile" ;;
    analytics-service)      echo "analytics-service:analytics-service/Dockerfile" ;;
    sct-service)            echo "sct-service:sct-service/Dockerfile" ;;
    dimse-receiver)         echo "dimse-receiver:dimse-receiver/Dockerfile" ;;
    *) return 1 ;;
  esac
}

# Normalises "a,b c" into "a b c" and rejects unknown names.
resolve_image_services() {
  local requested="${1:-}" name resolved=""
  [ -n "$requested" ] || requested="$DEFAULT_IMAGE_SERVICES"
  for name in $(echo "$requested" | tr ',' ' '); do
    if ! image_build_spec "$name" >/dev/null; then
      echo "error: unknown service '$name'" >&2
      return 1
    fi
    resolved="$resolved $name"
  done
  echo "$resolved"
}

# build_image <image-ref> <service-name> <platform> <push:0|1>
# Build args for the admin dashboard come from DWV_BASE_URL / OHIF_BASE_URL
# (optional; empty means the viewer links are disabled in the bundle).
build_image() {
  local image="$1" name="$2" platform="$3" push="$4" spec context dockerfile
  spec="$(image_build_spec "$name")"
  context="${spec%%:*}"
  dockerfile="${spec#*:}"

  local -a extra=()
  if [ "$name" = "admin-dashboard" ]; then
    extra+=(--build-arg "VITE_DWV_BASE_URL=${DWV_BASE_URL:-}" --build-arg "VITE_OHIF_BASE_URL=${OHIF_BASE_URL:-}")
  fi

  echo "==> Building ${image} (context: ${context}, dockerfile: ${dockerfile})"
  if [ "$push" -eq 1 ]; then
    docker buildx build --platform "$platform" -f "$dockerfile" -t "$image" "${extra[@]}" --push "$context"
  else
    docker buildx build --platform "$platform" -f "$dockerfile" -t "$image" "${extra[@]}" --load "$context"
  fi
}
