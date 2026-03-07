FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-realtimesteraming ./examples/aws/ivs-realtimesteraming
ENTRYPOINT ["/ivs-realtimesteraming"]
