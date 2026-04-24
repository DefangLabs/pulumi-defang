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
# This stage runs on the target platform so pulumi downloads the correct plugin binaries
FROM --platform=${TARGETPLATFORM} ${PULUMIBASE} AS plugins-base
COPY --link go.mod go.mod
# GITHUB_TOKEN will be used by pulumi to access github releases when downloading plugins
ARG GITHUB_TOKEN
ARG PROVIDER_VERSION

FROM plugins-base AS plugins-aws
COPY --link --from=build-aws /out/pulumi-resource-defang-aws /tmp/
RUN pulumi plugin install resource aws $(grep 'pulumi-aws/sdk/v7' go.mod | awk '{print $2}') && \
    pulumi plugin install resource awsx $(grep 'pulumi-awsx/sdk/v3' go.mod | awk '{print $2}')
RUN pulumi plugin install resource defang-aws ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-aws

FROM plugins-base AS plugins-gcp
COPY --link --from=build-gcp /out/pulumi-resource-defang-gcp /tmp/
RUN pulumi plugin install resource gcp $(grep 'pulumi-gcp/sdk/v9' go.mod | awk '{print $2}')
RUN pulumi plugin install resource defang-gcp ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-gcp

FROM plugins-base AS plugins-azure
COPY --link --from=build-azure /out/pulumi-resource-defang-azure /tmp/
RUN pulumi plugin install resource azure-native $(grep 'pulumi-azure-native-sdk/v3' go.mod | awk '{print $2}') && \
    pulumi plugin install resource random $(grep 'pulumi-random/sdk/v4' go.mod | awk '{print $2}')
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
COPY --link --from=plugins-aws /root/.pulumi/plugins /root/.pulumi/plugins

FROM cd-base AS gcp
COPY --link --from=plugins-gcp /root/.pulumi/plugins /root/.pulumi/plugins

FROM cd-base AS azure
COPY --link --from=plugins-azure /root/.pulumi/plugins /root/.pulumi/plugins

FROM cd-base AS all
COPY --link --from=plugins-aws /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-gcp /root/.pulumi/plugins /root/.pulumi/plugins
COPY --link --from=plugins-azure /root/.pulumi/plugins /root/.pulumi/plugins
