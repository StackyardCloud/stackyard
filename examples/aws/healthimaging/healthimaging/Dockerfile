FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthimaging ./examples/aws/healthimaging/healthimaging
ENTRYPOINT ["/healthimaging"]
