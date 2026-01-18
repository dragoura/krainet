## Overview

This repository contains a minimal Go + Echo microservice with Prometheus metrics, PostgreSQL integration, Docker (multi-stage, non-root), Nginx reverse proxy, GitLab CI/CD.
Ansible and Terraform for provisioning a DigitalOcean droplet, firewall rules, and DNS for the app are in this repo https://github.com/dragoura/krainet-terraform

### Components
- App: Go 1.23 + Echo
- Reverse proxy: Nginx
- Database: PostgreSQL 
- Metrics: Prometheus-format at `/metrics` (scrape externally)
- CI/CD: GitLab CI builds/pushes image to GitLab Registry and deploys via SSH to the app droplet

### Local quick start
1) Create `.env` in repo root with variables (see `env/app.env.example` for keys), or export them in your shell.
2) Start stack: `docker compose -f docker-compose.local.yml up -d --build`
3) Endpoints via Nginx on http://localhost:8080
   - `/health`
   - `/metrics`
   - `/api/users` (GET/POST)

### CI/CD
- Push to GitLab → pipeline builds and pushes to built-in GitLab Container Registry → deploy stage SSHes to droplet and runs `docker compose pull && up -d`.
- Required CI variables (masked):
  - `CI_REGISTRY_USER`, `CI_REGISTRY_PASSWORD` (usually provided automatically)
  - `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_PATH` (e.g. `/opt/app`), `DEPLOY_SSH_PRIVATE_KEY`
  - `APP_DATABASE_URL` (production DB connection string)

### Notes
- Grafana is hosted on a separate droplet. Scraping of `/metrics` should be done by a Prometheus/Agent you control (on Grafana droplet or elsewhere).
- GitLab and its Registry are already hosted at `gitlab.julia-b.work` (currently unavailable).
