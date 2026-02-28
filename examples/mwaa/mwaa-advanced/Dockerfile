FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa-advanced ./examples/mwaa/mwaa-advanced
ENTRYPOINT ["/mwaa-advanced"]
