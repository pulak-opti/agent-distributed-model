# Start from latest golang base image
FROM golang:latest as builder

# Set the current directory inside the container
WORKDIR /app

# Copy sources inside the docker
COPY . .

# install the dependencies
RUN go mod tidy

# Build the binaries from the source
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

###### Start a new stage from scratch #######
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

# Declare entry point of the docker command
ENTRYPOINT ["./main"]
