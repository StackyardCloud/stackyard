FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mskv1 ./examples/aws/mskv1
ENTRYPOINT ["/mskv1"]
