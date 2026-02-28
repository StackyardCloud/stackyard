FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /license-manager-basic ./examples/license-manager/license-manager-basic
ENTRYPOINT ["/license-manager-basic"]
