FROM ubuntu:20.04

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get -y install -y \
    ca-certificates libssl1.1 vim htop iotop sysstat wget \
    dstat strace lsof curl jq tzdata && \
    rm -rf /var/cache/apt /var/lib/apt/lists/*

RUN rm /etc/localtime && ln -snf /usr/share/zoneinfo/America/Montreal /etc/localtime && dpkg-reconfigure -f noninteractive tzdata

COPY dummy-blockchain-x86_64-unknown-linux-gnu /app/dummy-blockchain

RUN chmod +x /app/dummy-blockchain

ENV PATH "$PATH:/app"
