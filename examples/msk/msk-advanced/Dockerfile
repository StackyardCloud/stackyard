FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /msk-advanced ./examples/msk/msk-advanced
ENTRYPOINT ["/msk-advanced"]
