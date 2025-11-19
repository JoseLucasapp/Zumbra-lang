FROM golang:1.24-bookworm AS build
ENV CGO_ENABLED=0
WORKDIR /src
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go build -trimpath -ldflags="-s -w" -o /out/zumbra .

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=build /out/zumbra /usr/local/bin/zumbra
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/zumbra"]
