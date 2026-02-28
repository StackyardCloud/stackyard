FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /msk-connect-basic ./examples/msk-connect/msk-connect-basic
ENTRYPOINT ["/msk-connect-basic"]
