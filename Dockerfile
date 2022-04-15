FROM golang:alpine AS build-env

RUN apk add --no-cache curl make git libc-dev bash gcc linux-headers eudev-dev python3
RUN apk add --update ca-certificates

# Set working directory for the build
WORKDIR /go/src/github.com/BurntFinance/burnt

RUN git clone https://github.com/ignite-hq/cli.git --depth=1
RUN cd cli && make install

# Get dependancies - will also be cached if we won't change mod/sum
COPY go.mod .
COPY go.sum .
RUN go mod download

# Add source files
COPY . .

## build Burnt daemon
#RUN ignite chain build
#
## Final image
#FROM alpine:edge
#
## Install ca-certificates
#RUN apk add --update ca-certificates bash
#
## Copy over binaries from the build-env
#COPY --from=build-env /go/bin/burntd /usr/bin/burntd
#
#CMD ["burntd", "start"]
