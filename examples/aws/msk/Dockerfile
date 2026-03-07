FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /msk ./examples/aws/msk
ENTRYPOINT ["/msk"]
