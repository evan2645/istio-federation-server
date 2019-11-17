FROM ubuntu:bionic

RUN apt-get update && \
    apt-get install --no-install-recommends -y \
      ca-certificates \
   && update-ca-certificates \
   && apt-get upgrade -y \
   && apt-get clean \
   && rm -rf  /var/log/*log /var/lib/apt/lists/* /var/log/apt/* /var/lib/dpkg/*-old /var/cache/debconf/*-old

COPY istio-federation-server /usr/local/bin/istio-federation-server
ENTRYPOINT ["/usr/local/bin/istio-federation-server"]
