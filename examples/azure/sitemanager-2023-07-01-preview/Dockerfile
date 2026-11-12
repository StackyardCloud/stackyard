FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sitemanager-2023-07-01-preview ./examples/azure/sitemanager-2023-07-01-preview

FROM gcr.io/distroless/base-debian12

COPY --from=builder /out/sitemanager-2023-07-01-preview /sitemanager-2023-07-01-preview

ENTRYPOINT ["/sitemanager-2023-07-01-preview"]
