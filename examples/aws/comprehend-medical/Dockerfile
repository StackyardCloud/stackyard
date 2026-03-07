FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-medical ./examples/aws/comprehend-medical
ENTRYPOINT ["/comprehend-medical"]
