FROM golang:1.20

RUN apt-get update && \
    apt-get install -y wget vim

ARG ARCH=amd64
ARG USEARCH_VERSION=2.12.0
ARG RELEASE_TAG=v${USEARCH_VERSION}

RUN wget https://github.com/unum-cloud/usearch/releases/download/${RELEASE_TAG}/usearch_linux_${ARCH}_${USEARCH_VERSION}.deb && \
    dpkg -i usearch_linux_${ARCH}_${USEARCH_VERSION}.deb && \
    rm usearch_linux_${ARCH}_${USEARCH_VERSION}.deb

ENV LD_LIBRARY_PATH=/usr/local/lib

WORKDIR /go/src/app

COPY go.mod .

RUN go mod tidy
