FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /smsvoicev2-advanced ./examples/smsvoicev2/smsvoicev2-advanced
ENTRYPOINT ["/smsvoicev2-advanced"]
