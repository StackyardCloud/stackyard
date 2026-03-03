FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /msk-connect ./examples/aws/msk-connect/msk-connect
ENTRYPOINT ["/msk-connect"]
