# Unified Docker Container

A single Docker image containing all inference servers and llama-swap:

- **llama-server** (llama.cpp) for LLMs, reranking, and embeddings
- **whisper-server** (whisper.cpp) for speech recognition
- **sd-server** (stable-diffusion.cpp) for image generation
- **llama-swap** for automatic model management and swapping

Built against CUDA 12.4 (NVIDIA) or Vulkan (AMD/cross-platform).

## Quick Start

```bash
./build-image.sh --cuda
```

This builds `llama-swap:unified-cuda` using the latest llama.cpp release, latest whisper.cpp and stable-diffusion.cpp HEAD, and the latest llama-swap release.

## Build Options

```
./build-image.sh --cuda|--vulkan [--no-cache]
```

| Flag | Description |
|------|-------------|
| `--cuda` | Build for NVIDIA GPUs (CUDA 12.4) |
| `--vulkan` | Build for AMD GPUs / cross-platform |
| `--no-cache` | Force full rebuild without Docker cache |

## Version Pinning

Control which version of each component gets built via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `LLAMA_REF` | Latest release | llama.cpp version (tag, branch, or commit hash) |
| `WHISPER_REF` | Latest HEAD | whisper.cpp version |
| `SD_REF` | Latest HEAD | stable-diffusion.cpp version |
| `LS_VERSION` | Latest HEAD | llama-swap version (tag or commit hash) |
| `LLAMA_SWAP_REPO` | `mostlygeek/llama-swap` | llama-swap GitHub repo (for forks) |
| `DOCKER_IMAGE_TAG` | `llama-swap:unified-cuda` | Custom image tag |

### Examples

```bash
# Pin llama.cpp to a specific release
LLAMA_REF=b8660 ./build-image.sh --cuda

# Pin everything
LLAMA_REF=b8660 WHISPER_REF=v1.7.5 SD_REF=master LS_VERSION=v3 ./build-image.sh --cuda

# Build from a fork
LLAMA_SWAP_REPO=myuser/llama-swap-fork LS_VERSION=v3 ./build-image.sh --cuda

# Custom image name
DOCKER_IMAGE_TAG=my-registry/llama-swap:latest ./build-image.sh --cuda

# Rebuild after a new llama.cpp release (picks up latest automatically)
./build-image.sh --cuda
```

## Running

### NVIDIA (CUDA)

```bash
docker run --gpus all \
  -v ./config.yaml:/etc/llama-swap/config/config.yaml \
  -v ./models:/models \
  -p 8080:8080 \
  llama-swap:unified-cuda
```

### AMD (Vulkan)

```bash
docker run --device /dev/dri:/dev/dri --group-add video \
  -v ./config.yaml:/etc/llama-swap/config/config.yaml \
  -v ./models:/models \
  -p 8080:8080 \
  llama-swap:unified-vulkan
```

### Docker Compose

```yaml
services:
  llama-swap:
    image: llama-swap:unified-cuda
    volumes:
      - ./models:/models
      - ./config:/config
    environment:
      NVIDIA_VISIBLE_DEVICES: all
      NVIDIA_DRIVER_CAPABILITIES: compute,utility
      LLAMA_CACHE: /models
    entrypoint:
      - llama-swap
      - -config
      - /config/config.yaml
      - -models-dir
      - /models
      - -listen
      - 0.0.0.0:8080
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
```

### Example config.yaml

All binaries (`llama-server`, `whisper-server`, `sd-server`) are on `PATH` inside the container:

```yaml
macros:
  default-call: >
    llama-server --host 0.0.0.0 --port ${PORT} --flash-attn on --jinja

models:
  my-llm:
    cmd: ${default-call} -hf unsloth/Qwen3.5-27B-GGUF:UD-Q4_K_XL
    proxy: http://localhost:${PORT}

  whisper:
    cmd: >
      whisper-server --host 0.0.0.0 --port ${PORT}
      -m /models/whisper/ggml-large-v3-turbo.bin
      --flash-attn -l auto
    proxy: http://localhost:${PORT}
```

## Included Binaries

| Binary | Source |
|--------|--------|
| `llama-server` | [llama.cpp](https://github.com/ggml-org/llama.cpp) |
| `llama-cli` | llama.cpp |
| `whisper-server` | [whisper.cpp](https://github.com/ggml-org/whisper.cpp) |
| `whisper-cli` | whisper.cpp |
| `sd-server` | [stable-diffusion.cpp](https://github.com/leejet/stable-diffusion.cpp) |
| `sd-cli` | stable-diffusion.cpp |
| `llama-swap` | [llama-swap](https://github.com/mostlygeek/llama-swap) |
