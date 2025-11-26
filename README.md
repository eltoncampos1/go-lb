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
│       └── main.go
├── infra/
│   ├── Dockerfile
│   └── docker-compose.yml
├── go.mod
```

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
