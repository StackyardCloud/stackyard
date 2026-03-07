FROM golang:1.24-alpine

WORKDIR /app
COPY . .
RUN go build -o /identitytoolkit-apiv2 ./examples/gcp/identitytoolkit-apiv2

ENTRYPOINT ["/identitytoolkit-apiv2"]
