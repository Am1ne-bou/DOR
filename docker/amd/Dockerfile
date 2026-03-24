FROM alpine:3.14

RUN apk add --no-cache wget tar iproute2 iputils

# Installer Go manuellement
ENV GO_VERSION=1.22.0
ENV NETWORK_PROFILE=default

RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

COPY apply_network_profile.sh /usr/projet-dor/

RUN chmod +x /usr/projet-dor/apply_network_profile.sh

ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /usr/projet-dor

CMD ["sh", "-c", "./apply_network_profile.sh $NETWORK_PROFILE && exec sh"]