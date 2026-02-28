FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /lexv2-basic ./examples/lexv2/lexv2-basic
ENTRYPOINT ["/lexv2-basic"]
