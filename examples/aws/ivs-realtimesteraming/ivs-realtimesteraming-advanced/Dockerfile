FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-realtimesteraming-advanced ./examples/aws/ivs-realtimesteraming/ivs-realtimesteraming-advanced
ENTRYPOINT ["/ivs-realtimesteraming-advanced"]
