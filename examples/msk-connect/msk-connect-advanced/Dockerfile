FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /msk-connect-advanced ./examples/msk-connect/msk-connect-advanced
ENTRYPOINT ["/msk-connect-advanced"]
