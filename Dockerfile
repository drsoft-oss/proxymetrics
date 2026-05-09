# syntax=docker/dockerfile:1.7

# ─── Downloader ───────────────────────────────────────────────────────────────
# Pulls a published release tarball from GitHub Releases. Override VERSION to
# pin a specific tag (e.g. v0.1.2); the default resolves "latest" at build time.
FROM alpine:3.19 AS download

ARG VERSION=latest
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache ca-certificates curl tar

WORKDIR /work
RUN set -eux; \
    if [ "${VERSION}" = "latest" ]; then \
      tag=$(curl -fsSL https://api.github.com/repos/drsoft-oss/proxymetrics/releases/latest \
              | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p'); \
    else \
      tag="${VERSION}"; \
    fi; \
    test -n "${tag}"; \
    semver="${tag#v}"; \
    base="https://github.com/drsoft-oss/proxymetrics/releases/download/${tag}"; \
    archive="proxymetrics_${semver}_${TARGETOS}_${TARGETARCH}.tar.gz"; \
    checksums="proxymetrics_${semver}_checksums.txt"; \
    curl -fsSL -o "${archive}"   "${base}/${archive}"; \
    curl -fsSL -o "${checksums}" "${base}/${checksums}"; \
    grep "  ${archive}$" "${checksums}" | sha256sum -c -; \
    tar -xzf "${archive}" proxymetrics; \
    chmod +x ./proxymetrics

# ─── Runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S -g 65532 proxymetrics \
 && adduser  -S -u 65532 -G proxymetrics proxymetrics \
 && mkdir -p /data \
 && chown -R proxymetrics:proxymetrics /data

COPY --from=download /work/proxymetrics /usr/local/bin/proxymetrics

USER 65532:65532
WORKDIR /data
VOLUME ["/data"]

EXPOSE 8080 8081

ENTRYPOINT ["/usr/local/bin/proxymetrics"]
CMD ["serve"]
