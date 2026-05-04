FROM alpine:3.14

RUN apk add --no-cache wget tar iproute2 iputils git

ENV GO_VERSION=1.25.6
ENV NETWORK_PROFILE=default
ENV SERVER_ADDR=host.docker.internal:8080

RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /usr/projet-dor/

# ✅ 1. Clone d'abord
RUN git clone https://github.com/ThomasRegnault/Projet-DOR.git

# ✅ 2. Ensuite copie dans le dossier existant
COPY apply_network_profile.sh /usr/projet-dor/Projet-DOR/node_server/node/

RUN chmod +x /usr/projet-dor/Projet-DOR/node_server/node/apply_network_profile.sh

WORKDIR /usr/projet-dor/Projet-DOR/node_server/node/

CMD ["sh", "-c", "./apply_network_profile.sh $NETWORK_PROFILE && until nc -z directory 8080; do echo 'waiting for directory...'; sleep 1; done && go run main.go"]