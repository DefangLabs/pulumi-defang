# syntax=docker/dockerfile:1.4
#
# Build context must be the repo root (because cd/go.mod has replace directives to ../).
# Example: docker buildx build --platform linux/amd64 --build-arg CD_VERSION=0.1.0 --target aws .
#
ARG GOVERSION=1.25
ARG PULUMI_VERSION=latest
ARG BUILDBASE=golang:${GOVERSION}-alpine
ARG CDBASE=scratch
ARG PULUMIBASE=pulumi/pulumi-base:${PULUMI_VERSION}
ARG CD_VERSION=0.0.1
ARG PROVIDER_VERSION=0.0.1

# Shared build stage: downloads modules, copies source, builds the cd binary
FROM --platform=${BUILDPLATFORM} ${BUILDBASE} AS build-base

# These two are automatically set by docker buildx
ARG TARGETOS
ARG TARGETARCH

WORKDIR /repo
# ADD gocache.tgz* /root/.cache/
# ADD gomodcache.tgz* /go/pkg/

COPY --link go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY --link cd/go.mod cd/go.sum cd/
COPY --link sdk/v2 sdk/v2
RUN --mount=type=cache,target=/go/pkg/mod cd cd && go mod download

COPY --link provider/ provider/
COPY --link cd/ cd/

# Build the cd binary
ARG CD_VERSION
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    cd cd && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X main.version=${CD_VERSION}" \
    -o /out/cd .

# Per-cloud provider builds (BuildKit runs these in parallel when building 'all')
FROM build-base AS build-aws
ARG PROVIDER_VERSION
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defangaws.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-aws ./provider/cmd/pulumi-resource-defang-aws

FROM build-base AS build-gcp
ARG PROVIDER_VERSION
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defanggcp.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-gcp ./provider/cmd/pulumi-resource-defang-gcp

FROM build-base AS build-azure
ARG PROVIDER_VERSION
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defangazure.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-azure ./provider/cmd/pulumi-resource-defang-azure

# Install Pulumi cloud-provider plugins (versions extracted from go.mod)
# This stage runs on the target platform so pulumi downloads the correct plugin binaries.
#
# Both root go.mod and cd/go.mod are copied because plugin SDKs appear in
# different places: pulumi-awsx and pulumi-random are only imported by the
# provider packages (root go.mod); pulumi-gcp and pulumi-azure-native-sdk
# appear in both and cd/go.mod typically carries the newer version (the CD
# binary's imports determine the wire-level SDK version). Each RUN below
# greps the file where the plugin is actually declared so the installed
# plugin version matches whatever binary talks to it at runtime.
#
# Each cloud's plugins are split into two stages so the heavy upstream layer
# is content-addressable across builds while the defang plugin lives in its
# own small layer:
#   plugins-<cloud>-upstream — pulumi-owned providers only. Output of
#     /root/.pulumi/plugins is byte-identical when go.mod is unchanged, so
#     the COPY into the final image dedups across builds (a ~1 GB AWS layer
#     is created once per upstream-version bump, not per commit).
#   plugins-<cloud>-defang   — defang plugin only, installed from the local
#     binary produced by build-<cloud>. Small (~10 MB) layer that legitimately
#     changes every build. Pulumi plugin paths are version-namespaced
#     (resource-aws-vX.Y.Z/, resource-defang-aws-vA.B.C/), so the final
#     image's two COPYs don't collide.
FROM --platform=${TARGETPLATFORM} ${PULUMIBASE} AS plugins-base
COPY --link go.mod go.mod
COPY --link cd/go.mod cd/go.mod

FROM plugins-base AS plugins-aws-upstream
RUN pulumi plugin install resource aws $(grep 'pulumi-aws/sdk/v7' cd/go.mod | awk '{print $2}') && \
    pulumi plugin install resource awsx $(grep 'pulumi-awsx/sdk/v3' go.mod | awk '{print $2}')

FROM plugins-base AS plugins-aws-defang
ARG PROVIDER_VERSION
COPY --link --from=build-aws /out/pulumi-resource-defang-aws /tmp/
RUN pulumi plugin install resource defang-aws ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-aws

FROM plugins-base AS plugins-gcp-upstream
RUN pulumi plugin install resource gcp $(grep 'pulumi-gcp/sdk/v9' cd/go.mod | awk '{print $2}')

FROM plugins-base AS plugins-gcp-defang
ARG PROVIDER_VERSION
COPY --link --from=build-gcp /out/pulumi-resource-defang-gcp /tmp/
RUN pulumi plugin install resource defang-gcp ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-gcp

FROM plugins-base AS plugins-azure-upstream
RUN pulumi plugin install resource azure-native $(grep 'pulumi-azure-native-sdk/v3' cd/go.mod | awk '{print $2}') && \
    pulumi plugin install resource random $(grep 'pulumi-random/sdk/v4' go.mod | awk '{print $2}')

FROM plugins-base AS plugins-azure-defang
ARG PROVIDER_VERSION
COPY --link --from=build-azure /out/pulumi-resource-defang-azure /tmp/
RUN pulumi plugin install resource defang-azure ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-azure

# Final minimal image — no OS, no language runtimes
FROM ${CDBASE} AS cd-base
# CA certs for HTTPS
COPY --link --from=build-base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Pulumi CLI only (no language runtimes)
COPY --link --from=plugins-base /pulumi/bin/pulumi /pulumi/bin/pulumi
ENV PATH="/pulumi/bin:${PATH}" HOME="/root" USER=root
# /tmp is required by Pulumi for workspace temp files; scratch has no filesystem.
WORKDIR /tmp
# App
WORKDIR /app
COPY --link --from=build-base /out/cd ./
ENTRYPOINT [ "/app/cd" ]

FROM cd-base AS aws
COPY --link --from=plugins-aws-upstream /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-aws-defang /root/.pulumi/plugins /root/.pulumi/plugins

FROM cd-base AS gcp
COPY --link --from=plugins-gcp-upstream /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-gcp-defang /root/.pulumi/plugins /root/.pulumi/plugins

FROM cd-base AS azure
COPY --link --from=plugins-azure-upstream /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-azure-defang /root/.pulumi/plugins /root/.pulumi/plugins

FROM cd-base AS all
COPY --link --from=plugins-aws-upstream /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-aws-defang /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-azure-upstream /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-azure-defang /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-gcp-upstream /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-gcp-defang /root/.pulumi/plugins /root/.pulumi/plugins
