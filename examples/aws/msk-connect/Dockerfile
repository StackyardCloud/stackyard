FROM golang:1.25 AS build
WORKDIR /src
COPY . .
RUN go build -o /msk-connect ./examples/aws/msk-connect
ENTRYPOINT ["/msk-connect"]
