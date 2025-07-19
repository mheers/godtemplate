ARG go="golang:1.24.5-bookworm"
ARG base="debian:bookworm-slim"

FROM --platform=$BUILDPLATFORM ${go} AS builder

RUN apt-get update && \
    apt-get install -y bash git && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ARG TARGETPLATFORM
ARG BUILDPLATFORM

# Copy the code from the host and compile it
WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download

ADD . ./

RUN [ "$(uname)" = Darwin ] && system=darwin || system=linux; \
    ./ci/go-build.sh --os ${system} --arch $(echo $TARGETPLATFORM  | cut -d/ -f2)

# final stage
FROM ${base}
WORKDIR /app

RUN apt-get update && \
    apt-get install -y libreoffice libreoffice-java-common && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/src/app/goapp /usr/local/bin/godtemplate
RUN chmod +x /usr/local/bin/godtemplate

ENTRYPOINT ["/usr/local/bin/godtemplate"]
CMD ["help"]
