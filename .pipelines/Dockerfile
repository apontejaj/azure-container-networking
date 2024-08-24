FROM --platform=$BUILDPLATFORM mcr.microsoft.com/oss/go/microsoft/golang:1.23 as gobin

# Mount repo directory

FROM --platform=$BUILDPLATFORM onebranch.azurecr.io/linux/ubuntu-2204:latest as unittestbuild

COPY --from=gobin /usr/local/go/bin/go /usr/local/go/bin/go

RUN go install github.com/AlekSi/gocov-xml
RUN go install github.com/axw/gocov/gocov
RUN go install github.com/docker/libnetwork/driverapi
RUN go install github.com/gorilla/mux
RUN go install github.com/jstemmer/go-junit-report
RUN go install gopkg.in/matm/v1/gocov-html
RUN go install sigs.k8s.io/kind@v0.24.0
#RUN podman run --rm --privileged multiarch/qemu-user-static --reset -p yes
