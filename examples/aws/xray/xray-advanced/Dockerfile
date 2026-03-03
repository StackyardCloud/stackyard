FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /xray-advanced ./examples/aws/xray/xray-advanced
ENTRYPOINT ["/xray-advanced"]
