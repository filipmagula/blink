# Stage 1: Build Vue Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.22-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app

# Enable CGO for SQLite
ENV CGO_ENABLED=1

# We do not have a locally generated go.sum, so we depend on generating it here.
COPY go.mod ./
RUN go get github.com/mattn/go-sqlite3 github.com/matoous/go-nanoid/v2 github.com/golang-jwt/jwt/v5

# Copy source code
COPY main.go ./
COPY store/ store/
COPY api/ api/
COPY worker/ worker/

# Generate go.sum internally before building
RUN go mod tidy

# Copy the built frontend into a directory the Go compiler can read via embed.FS
COPY --from=frontend-builder /app/dist /app/frontend/dist

# Build the final binary
RUN GOOS=linux go build -ldflags "-extldflags '-static'" -o blink .

# Stage 3: Minimal Runtime
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder /app/blink /app/blink

# Ensure data and upload directories exist
RUN mkdir -p /app/data/uploads && chmod 777 -R /app/data

EXPOSE 8080
ENV PORT=8080

CMD ["/app/blink"]
