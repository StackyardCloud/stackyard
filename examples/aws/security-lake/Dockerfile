FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-lake ./examples/aws/security-lake
ENTRYPOINT ["/security-lake"]
