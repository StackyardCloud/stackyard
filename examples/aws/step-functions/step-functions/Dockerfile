FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /step-functions ./examples/aws/step-functions/step-functions
ENTRYPOINT ["/step-functions"]
