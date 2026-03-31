# syntax=docker/dockerfile:1.4
#
# Build context must be the repo root (because cd/go.mod has replace directives to ../).
# Example: docker buildx build --platform linux/amd64 --build-arg CD_VERSION=0.1.0 .
#
ARG GOVERSION=1.25
ARG PULUMI_VERSION=latest
ARG BUILDBASE=golang:${GOVERSION}-alpine
ARG CDBASE=pulumi/pulumi-go:${PULUMI_VERSION}
ARG CD_VERSION=0.0.1
ARG PROVIDER_VERSION=0.0.1

# Build stage runs on the host (e.g. ARM Mac) and cross-compiles for the target platform
FROM --platform=${BUILDPLATFORM} ${BUILDBASE} AS build

# These two are automatically set by docker buildx
ARG TARGETOS
ARG TARGETARCH

WORKDIR /repo
ADD gocache.tgz* /root/.cache/
ADD gomodcache.tgz* /go/pkg/

COPY --link go.mod go.sum ./
RUN go mod download

COPY --link cd/go.mod cd/go.sum cd/
COPY --link sdk/v2 sdk/v2
RUN cd cd && go mod download

COPY --link provider/ provider/
COPY --link cd/ cd/

# Build the cd binary
ARG CD_VERSION
RUN cd cd && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X main.version=${CD_VERSION}" \
    -o /out/cd .

# Build the three defang provider binaries
ARG PROVIDER_VERSION
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defangaws.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-aws ./provider/cmd/pulumi-resource-defang-aws
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defanggcp.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-gcp ./provider/cmd/pulumi-resource-defang-gcp
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false \
    -ldflags="-w -s -X github.com/DefangLabs/pulumi-defang/provider/defangazure.Version=${PROVIDER_VERSION}" \
    -o /out/pulumi-resource-defang-azure ./provider/cmd/pulumi-resource-defang-azure

# Install Pulumi cloud-provider plugins (versions extracted from go.mod)
# This stage runs on the target platform so pulumi downloads the correct plugin binaries
FROM --platform=${TARGETPLATFORM} ${CDBASE} AS plugins
COPY --link go.mod go.mod
# GITHUB_TOKEN will be used by pulumi to access github releases when downloading plugins
ARG GITHUB_TOKEN
RUN pulumi plugin install resource aws $(grep 'pulumi-aws/sdk/v7' go.mod | awk '{print $2}') && \
    pulumi plugin install resource awsx $(grep 'pulumi-awsx/sdk/v3' go.mod | awk '{print $2}') && \
    pulumi plugin install resource gcp $(grep 'pulumi-gcp/sdk/v9' go.mod | awk '{print $2}') && \
    pulumi plugin install resource azure-native $(grep 'pulumi-azure-native-sdk/v2 ' go.mod | awk '{print $2}')

ARG PROVIDER_VERSION
COPY --link --from=build /out/pulumi-resource-defang-aws /tmp/
COPY --link --from=build /out/pulumi-resource-defang-gcp /tmp/
COPY --link --from=build /out/pulumi-resource-defang-azure /tmp/
RUN pulumi plugin install resource defang-aws ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-aws && \
    pulumi plugin install resource defang-gcp ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-gcp && \
    pulumi plugin install resource defang-azure ${PROVIDER_VERSION} -f /tmp/pulumi-resource-defang-azure

# Final image runs on the target platform (linux/amd64 in the cloud)
FROM --platform=${TARGETPLATFORM} ${CDBASE}
COPY --link --from=plugins /root/.pulumi/plugins /root/.pulumi/plugins
WORKDIR /app
COPY --link --from=build /out/cd ./
ENTRYPOINT [ "/app/cd" ]
