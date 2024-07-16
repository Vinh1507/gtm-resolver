# Stage 1: Build the Go binary
FROM golang:1.22.4 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main main.go

# Stage 2: Run the Go binary
FROM golang:1.22.4

# RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk*

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# COPY --from=builder /app/main . isn't copying the .env file, it copies only the application binary
COPY .env /root


# Command to run the executable
ENTRYPOINT ["./main"]
