# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build backend (with embedded frontend)
FROM golang:1.22-alpine AS backend-builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
# Copy the frontend dist into backend source so go:embed can pick it up
COPY --from=frontend-builder /frontend/dist ./dist/
RUN CGO_ENABLED=0 go build -o /app/server .

# Stage 3: Production
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/server .
RUN mkdir -p /app/uploads
EXPOSE 8080
CMD ["./server"]
