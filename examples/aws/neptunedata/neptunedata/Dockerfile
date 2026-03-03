FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptunedata ./examples/aws/neptunedata/neptunedata
ENTRYPOINT ["/neptunedata"]
