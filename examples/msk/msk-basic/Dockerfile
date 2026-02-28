FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /msk-basic ./examples/msk/msk-basic
ENTRYPOINT ["/msk-basic"]
