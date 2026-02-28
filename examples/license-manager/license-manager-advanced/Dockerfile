FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager-advanced ./examples/license-manager/license-manager-advanced
ENTRYPOINT ["/license-manager-advanced"]
