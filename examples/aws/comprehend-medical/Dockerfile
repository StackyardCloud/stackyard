FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /comprehend-medical ./examples/aws/comprehend-medical
ENTRYPOINT ["/comprehend-medical"]
