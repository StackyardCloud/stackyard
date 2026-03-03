FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mskv1-basic ./examples/aws/mskv1/mskv1-basic
ENTRYPOINT ["/mskv1-basic"]
