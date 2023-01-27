# docker build . -t cosmwasm/wasmd:latest
# docker run --rm -it cosmwasm/wasmd:latest /bin/sh
FROM golang:1.19.1-alpine3.16 AS go-builder

  ARG arch=x86_64

  # this comes from standard alpine nightly file
  #  https://github.com/rust-lang/docker-rust-nightly/blob/master/alpine3.12/Dockerfile
  # with some changes to support our toolchain, etc
  RUN set -eux; apk add --no-cache ca-certificates build-base;

  RUN apk add git
  # NOTE: add these to run with LEDGER_ENABLED=true
  # RUN apk add libusb-dev linux-headers

  # See https://github.com/CosmWasm/wasmvm/releases
  ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.1.1/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
  ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.1.1/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
  RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 9ecb037336bd56076573dc18c26631a9d2099a7f2b40dc04b6cae31ffb4c8f9a
  RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep 6e4de7ba9bad4ae9679c7f9ecf7e283dd0160e71567c6a7be6ae47c81ebe7f32
  # Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
  RUN cp /lib/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

  WORKDIR /code
  COPY ./.git /code/.git
  COPY ./app /code/app
  COPY ./cmd /code/cmd
  COPY ./contrib /code/contrib
  COPY ./proto /code/proto
  COPY ./testutil /code/testutil
  COPY ./x /code/x
  COPY go.mod /code/
  COPY go.sum /code/
  COPY Makefile /code/

  RUN ls /code

  # force it to use static lib (from above) not standard libgo_cosmwasm.so file
  RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build

  RUN echo "Ensuring binary is statically linked ..." \
    && (file /code/build/burntd | grep "statically linked")

# --------------------------------------------------------
FROM alpine:3.16 AS localdev

  COPY --from=go-builder /code/build/burntd /usr/bin/burntd

  COPY ./docker/local-config /burnt/config
  COPY ./docker/entrypoint.sh /root/entrypoint.sh
  RUN chmod +x /root/entrypoint.sh

  # rest server
  EXPOSE 1317
  # tendermint grpc
  EXPOSE 9090
  # tendermint p2p
  EXPOSE 26656
  # tendermint rpc
  EXPOSE 26657
  # tendermint prometheus
  EXPOSE 26660

  VOLUME [ "/burnt/data" ]

  CMD ["/root/entrypoint.sh"]


# --------------------------------------------------------
FROM alpine:3.16 AS burnt-release

  COPY --from=go-builder /code/build/burntd /usr/bin/burntd

  # rest server
  EXPOSE 1317
  # tendermint grpc
  EXPOSE 9090
  # tendermint p2p
  EXPOSE 26656
  # tendermint rpc
  EXPOSE 26657
  # tendermint prometheus
  EXPOSE 26660

  RUN set -euxo pipefail \
    && apk add --no-cache \
      bash \
      curl \
      htop \
      jq \
      tini

  RUN set -euxo pipefail \
    && addgroup -S burntd \
    && adduser \
       --disabled-password \
       --gecos burntd \
       --ingroup burntd \
       burntd

  RUN set -eux \
    && chown -R burntd:burntd /home/burntd

  USER burntd:burntd
  WORKDIR /home/burntd/.burnt

  CMD ["/usr/bin/burntd", "start"]
