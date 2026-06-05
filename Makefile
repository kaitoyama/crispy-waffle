.PHONY: tidy run-backend run-frontend dev test build reset-db install-frontend

# Go module housekeeping
tidy:
	go mod tidy

# Run the backend (creates ./data/app.db, migrates, seeds, serves :8080)
run-backend:
	go run ./cmd/server

# Run the Vite dev server (:5173, proxies /api -> :8080)
run-frontend:
	cd frontend && npm run dev

# Install frontend deps
install-frontend:
	cd frontend && npm install

# Run backend and frontend together
dev:
	$(MAKE) -j2 run-backend run-frontend

# Go tests
test:
	go test ./...

# Build backend binary + frontend bundle
build:
	go build -o bin/server ./cmd/server
	cd frontend && npm install && npm run build

# Drop the local database (re-seeds on next start) — dev only
reset-db:
	rm -f data/app.db data/app.db-shm data/app.db-wal
