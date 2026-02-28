FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthimaging-basic ./examples/healthimaging/healthimaging-basic
ENTRYPOINT ["/healthimaging-basic"]
