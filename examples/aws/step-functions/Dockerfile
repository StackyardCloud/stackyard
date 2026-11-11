FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /step-functions ./examples/aws/step-functions
ENTRYPOINT ["/step-functions"]
