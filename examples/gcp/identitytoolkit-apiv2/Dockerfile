FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /identitytoolkit-apiv2 ./examples/gcp/identitytoolkit-apiv2

ENTRYPOINT ["/identitytoolkit-apiv2"]
