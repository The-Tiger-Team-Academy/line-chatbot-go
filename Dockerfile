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

EXPOSE 6789

# Set environment variables (values need to be passed at runtime)
ARG OPENAI_API_KEY
ARG CHANNEL_SECRET
ARG CHANNEL_TOKEN

ENV OPENAI_API_KEY=$OPENAI_API_KEY
ENV CHANNEL_SECRET=$CHANNEL_SECRET
ENV CHANNEL_TOKEN=$CHANNEL_TOKEN

# Command to run the executable

CMD ["./main"]
