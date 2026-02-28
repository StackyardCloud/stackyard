FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /inspectorv2-advanced ./examples/inspectorv2/inspectorv2-advanced
ENTRYPOINT ["/inspectorv2-advanced"]
