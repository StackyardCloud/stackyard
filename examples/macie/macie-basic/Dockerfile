FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /macie-basic ./examples/macie/macie-basic
ENTRYPOINT ["/macie-basic"]
