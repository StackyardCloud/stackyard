FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-realtimesteraming ./examples/aws/ivs-realtimesteraming
ENTRYPOINT ["/ivs-realtimesteraming"]
