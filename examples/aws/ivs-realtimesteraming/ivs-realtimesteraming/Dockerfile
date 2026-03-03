FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-realtimesteraming ./examples/aws/ivs-realtimesteraming/ivs-realtimesteraming
ENTRYPOINT ["/ivs-realtimesteraming"]
