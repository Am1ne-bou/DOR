FROM alpine:3.14

RUN apk add --no-cache wget tar git

ENV GO_VERSION=1.25.6

RUN wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /usr/projet-dor/

RUN git clone https://github.com/ThomasRegnault/Projet-DOR.git

WORKDIR /usr/projet-dor/Projet-DOR/node_server/List_Serveur/

EXPOSE 8080

CMD ["go", "run", "serveur.go"]
