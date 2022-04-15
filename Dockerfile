FROM golang:alpine AS build-env

RUN apk add --no-cache curl make git libc-dev bash gcc linux-headers eudev-dev python3
RUN apk add --update ca-certificates

# Set working directory for the build
WORKDIR /opt/src/burnt

# Add source files
COPY . .
RUN go mod download

# Build
RUN go build -o burntd ./cmd/burntd/

# Build the runtime container
FROM golang:alpine AS runtime-env

WORKDIR /opt/burnt

COPY --from=build-env /opt/src/burnt/burntd /opt/burnt/burntd

ENTRYPOINT ["burntd"]
