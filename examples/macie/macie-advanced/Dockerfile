FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /macie-advanced ./examples/macie/macie-advanced
ENTRYPOINT ["/macie-advanced"]
