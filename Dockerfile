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

CMD ["ignite", "chain", "serve"]
