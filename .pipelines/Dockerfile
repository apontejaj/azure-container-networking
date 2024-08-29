FROM --platform=$BUILDPLATFORM mcr.microsoft.com/oss/go/microsoft/golang:1.23 as gobin

ENV GO111MODULE=on
ENV CGO_ENABLED=0

RUN echo $HOME
RUN mkdir -p /root/.config/go

RUN ls -la

# Download Common UnitTest Packages
RUN go install github.com/AlekSi/gocov-xml@v1.1.0
RUN go install github.com/axw/gocov/gocov@v1.1.0
#RUN go install github.com/docker/libnetwork/driverapi@v0.5.6
RUN go install github.com/gorilla/mux@v1.8.1
RUN go install github.com/jstemmer/go-junit-report@v2.1.0
RUN go install gopkg.in/matm/v1/gocov-html@v1.4.0
RUN go install sigs.k8s.io/kind@v0.24.0

WORKDIR artifacts/azure-container-networking/
RUN make init
RUN go env > /root/.config/go/env

RUN go get ./...

FROM --platform=$BUILDPLATFORM onebranch.azurecr.io/linux/ubuntu-2204:latest as unittestbuild

ARG ACN_GO_PATH="$ACN_GO_PATH"

# Configure Golang Environment
RUN mkdir -p /usr/local/go
COPY --from=gobin /usr/local/go /usr/local/go
COPY --from=gobin /root/ /root/
RUN go env 

#RUN podman run --rm --privileged multiarch/qemu-user-static --reset -p yes
