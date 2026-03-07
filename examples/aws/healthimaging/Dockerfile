FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthimaging ./examples/aws/healthimaging
ENTRYPOINT ["/healthimaging"]
