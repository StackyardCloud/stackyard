FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-realtimesteraming-basic ./examples/aws/ivs-realtimesteraming/ivs-realtimesteraming-basic
ENTRYPOINT ["/ivs-realtimesteraming-basic"]
