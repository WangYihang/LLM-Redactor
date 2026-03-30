# LLM-Redactor Development Notes

## Building

Go 1.25.4 is required. The local toolchain cannot auto-download it, so always build via Docker:

```bash
docker run --rm \
  -v $(pwd):/src \
  -v $HOME/go/bin:/gobin \
  -w /src golang:1.25 \
  sh -c "go build -buildvcs=false -o /gobin/llm-redactor-proxy ./cmd/llm-redactor-proxy/ && \
         go build -buildvcs=false -o /gobin/llm-redactor-exec ./cmd/llm-redactor-exec/"
```

This builds both binaries and installs them directly to `~/go/bin/`.
