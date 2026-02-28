FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /proton-advanced ./examples/proton/proton-advanced
ENTRYPOINT ["/proton-advanced"]
