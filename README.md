# Go Load Balancer — Study Project

This project is a practical study on how to build a **simple Load Balancer in Go**, featuring:

- Round-robin balancing
- Automatic health checks
- Retry and attempt logic using request context
- Reverse proxy with `net/http/httputil`
- Environment for realistic testing using Docker and Docker Compose

The goal is to understand the internal mechanics of a load balancer and how to distribute traffic between multiple upstream services.

---

## 🚀 How It Works

The Load Balancer:

- Listens for HTTP requests on port `3030`
- Forwards requests to backends using **round robin**
- Marks backends as **up/down** using a TCP health check
- When a backend goes down:
  - It is marked as `down`
  - The balancer automatically routes traffic only to healthy backends

---

## 📂 Project Structure

```
.
├── cmd/
│   └── lb/
│       └── main.go                    # Application entry point
├── internal/
│   ├── backend/
│   │   └── backend.go                 # Backend struct and methods
│   ├── config/
│   │   └── config.go                  # Configuration and flag parsing
│   ├── handler/
│   │   └── handler.go                 # HTTP handlers and context helpers
│   ├── healthcheck/
│   │   └── healthcheck.go             # Backend health checking
│   └── pool/
│       └── pool.go                    # ServerPool and round-robin logic
├── infra/
│   ├── Dockerfile
│   └── docker-compose.yml
├── go.mod
└── README.md
```

The project follows Go's standard layout:
- `cmd/` - Main applications for this project
- `internal/` - Private application and library code that can't be imported by other projects

---

## 🖼️ Step 1 — Starting the Containers

Inside the `/infra` folder:

```bash
cd infra
docker compose up --build
```

This starts:

- `backend1` → responds “backend 1”
- `backend2` → responds “backend 2”
- `lb` → your Go load balancer

---

## 🧪 Step 2 — Testing the Load Balancer

```bash
curl http://localhost:3030
curl http://localhost:3030
curl http://localhost:3030
curl http://localhost:3030
```

Expected output alternating:

```
backend 1
backend 2
backend 1
backend 2
```
---

## 🔥 Step 3 — Killing One Backend

```bash
docker compose stop backend1
```

The health checker will mark backend1 as down.

---

## 🧪 Step 4 — Testing After the Failure

Now all requests should be served only by backend2:

```bash
curl http://localhost:3030
curl http://localhost:3030
curl http://localhost:3030
```

Output:

```
backend 2
backend 2
backend 2
```

---

## 🤝 Contributing

Pull requests are welcome!
