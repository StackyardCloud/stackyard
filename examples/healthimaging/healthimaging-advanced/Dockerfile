FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthimaging-advanced ./examples/healthimaging/healthimaging-advanced
ENTRYPOINT ["/healthimaging-advanced"]
