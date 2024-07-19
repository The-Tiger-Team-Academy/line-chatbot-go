# Start from the official golang image
FROM golang:1.22-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN go build -o main .

# Expose the port the app runs on
EXPOSE 6789

# Start from the official golang image
FROM golang:1.22-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN go build -o main .

# Expose the port the app runs on
EXPOSE 6789

# Set environment variables (values need to be passed at runtime)
ENV OPENAI_API_KEY="sk-proj-rKqNiVTeXj9RhJYgCQLcT3BlbkFJ6JmYPcqm9bFIj9wijYvS"
ENV CHANNEL_SECRET="526de42e087ec0b992a99c5ecc0b0927"
ENV CHANNEL_TOKEN="iT0qWnEAhx1102TqTgKCjQkWxNeJosUhbpTWQAHm6GSd4K2PFYWd65uFJt9GGWeE+MbwRs3nXGb9EFnF4VSlVzW/VArBUsFo5kOfHzhT0eE+PHjkEZY4285AVJ5hKdRgVzn8ZV9VB5PjiBMM1c02ZwdB04t89/1O/w1cDnyilFU="

# Command to run the executable
CMD ["./main"]
