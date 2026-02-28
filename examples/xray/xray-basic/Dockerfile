FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /xray-basic ./examples/xray/xray-basic
ENTRYPOINT ["/xray-basic"]
