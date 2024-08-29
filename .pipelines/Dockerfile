FROM --platform=$BUILDPLATFORM mcr.microsoft.com/oss/go/microsoft/golang:1.23 as gobin

ENV GO111MODULE=on
RUN go env > go.env
RUN go env

# Mount repo directory

FROM --platform=$BUILDPLATFORM onebranch.azurecr.io/linux/ubuntu-2204:latest as unittestbuild

COPY --from=gobin /usr/local/go/bin/go /usr/local/go/bin/go
COPY --from=gobin go.env .
RUN go env 
CMD while IFS="" read -r p || [ -n "$p" ]; do \n\
  printf '%s\n' "$p" \n\
done < go.env \n\

RUN go install github.com/AlekSi/gocov-xml@v1.1.0
RUN go install github.com/axw/gocov/gocov@v1.1.0
RUN go install github.com/docker/libnetwork/driverapi@v0.5.6
RUN go install github.com/gorilla/mux@v1.8.1
RUN go install github.com/jstemmer/go-junit-report@v2.1.0
RUN go install gopkg.in/matm/v1/gocov-html@v1.4.0
RUN go install sigs.k8s.io/kind@v0.24.0
#RUN podman run --rm --privileged multiarch/qemu-user-static --reset -p yes
