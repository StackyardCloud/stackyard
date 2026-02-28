FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa-basic ./examples/mwaa/mwaa-basic
ENTRYPOINT ["/mwaa-basic"]
