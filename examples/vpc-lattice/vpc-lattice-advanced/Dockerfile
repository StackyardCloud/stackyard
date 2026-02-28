FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /vpc-lattice-advanced ./examples/vpc-lattice/vpc-lattice-advanced
ENTRYPOINT ["/vpc-lattice-advanced"]
