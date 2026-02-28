FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-realtimesteraming-advanced ./examples/ivs-realtimesteraming/ivs-realtimesteraming-advanced
ENTRYPOINT ["/ivs-realtimesteraming-advanced"]
