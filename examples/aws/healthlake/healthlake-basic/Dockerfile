FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthlake-basic ./examples/aws/healthlake/healthlake-basic
ENTRYPOINT ["/healthlake-basic"]
