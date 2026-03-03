FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthimaging-advanced ./examples/aws/healthimaging/healthimaging-advanced
ENTRYPOINT ["/healthimaging-advanced"]
