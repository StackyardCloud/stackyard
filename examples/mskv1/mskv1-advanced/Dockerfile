FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mskv1-advanced ./examples/mskv1/mskv1-advanced
ENTRYPOINT ["/mskv1-advanced"]
