# syntax=docker/dockerfile:1.4
#
# Build context must be the repo root (because cd/go.mod has replace directives to ../).
# Example: docker buildx build --platform linux/amd64 --build-arg CD_VERSION=0.1.0 .
#
ARG GOVERSION=1.26
ARG PULUMI_VERSION=latest
ARG BUILDBASE=golang:${GOVERSION}-alpine
ARG CDBASE=scratch
ARG PULUMIBASE=pulumi/pulumi-base:${PULUMI_VERSION}
ARG CD_VERSION=0.0.1
ARG PROVIDER_VERSION=0.0.1

# CLOUDS controls which cloud providers to include (default: all).
# Use any combination: "aws", "gcp,azure", "aws,gcp,azure", or "all".
ARG CLOUDS=all

# Build stage runs on the host (e.g. ARM Mac) and cross-compiles for the target platform
FROM --platform=${BUILDPLATFORM} ${BUILDBASE} AS build

# These two are automatically set by docker buildx
ARG TARGETOS
ARG TARGETARCH
ARG CLOUDS

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

# Build the defang provider binaries
ARG PROVIDER_VERSION
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    case ",${CLOUDS}," in *,all,*|*,aws,*) ;; *) exit 0;; esac && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defangaws.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-aws ./provider/cmd/pulumi-resource-defang-aws
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    case ",${CLOUDS}," in *,all,*|*,gcp,*) ;; *) exit 0;; esac && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defanggcp.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-gcp ./provider/cmd/pulumi-resource-defang-gcp
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    case ",${CLOUDS}," in *,all,*|*,azure,*) ;; *) exit 0;; esac && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defangazure.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-azure ./provider/cmd/pulumi-resource-defang-azure

# Install Pulumi cloud-provider plugins (versions extracted from go.mod)
# This stage runs on the target platform so pulumi downloads the correct plugin binaries
FROM --platform=${TARGETPLATFORM} ${PULUMIBASE} AS plugins
ARG CLOUDS
COPY --link go.mod go.mod
# GITHUB_TOKEN will be used by pulumi to access github releases when downloading plugins
ARG GITHUB_TOKEN
RUN case ",${CLOUDS}," in *,all,*|*,aws,*) ;; *) exit 0;; esac && \
    pulumi plugin install resource aws $(grep 'pulumi-aws/sdk/v7' go.mod | awk '{print $2}') && \
    pulumi plugin install resource awsx $(grep 'pulumi-awsx/sdk/v3' go.mod | awk '{print $2}')
RUN case ",${CLOUDS}," in *,all,*|*,gcp,*) ;; *) exit 0;; esac && \
    pulumi plugin install resource gcp $(grep 'pulumi-gcp/sdk/v9' go.mod | awk '{print $2}')
RUN case ",${CLOUDS}," in *,all,*|*,azure,*) ;; *) exit 0;; esac && \
    pulumi plugin install resource azure-native $(grep 'pulumi-azure-native-sdk/v3' go.mod | awk '{print $2}')

ARG PROVIDER_VERSION
COPY --link --from=build /out/pulumi-resource-defang-* /tmp/
RUN case ",${CLOUDS}," in *,all,*|*,aws,*) ;; *) exit 0;; esac && \
    pulumi plugin install resource defang-aws ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-aws
RUN case ",${CLOUDS}," in *,all,*|*,gcp,*) ;; *) exit 0;; esac && \
    pulumi plugin install resource defang-gcp ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-gcp
RUN case ",${CLOUDS}," in *,all,*|*,azure,*) ;; *) exit 0;; esac && \
    pulumi plugin install resource defang-azure ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-azure

# Final minimal image — no OS, no language runtimes
FROM ${CDBASE} AS cd
# CA certs for HTTPS
COPY --link --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Pulumi CLI only (no language runtimes)
COPY --link --from=plugins /pulumi/bin/pulumi /pulumi/bin/pulumi
ENV PATH="/pulumi/bin:${PATH}"
# Plugins
COPY --link --from=plugins /root/.pulumi/plugins /root/.pulumi/plugins
# App
WORKDIR /app
COPY --link --from=build /out/cd ./
ENTRYPOINT [ "/app/cd" ]
