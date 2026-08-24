# VaultForge Disaster Recovery Runbook

## Overview

This runbook covers backup, restore, and failover procedures for VaultForge.

**Recovery Time Objective (RTO):** 15 minutes
**Recovery Point Objective (RPO):** 5 minutes (with WAL archiving)

---

## 1. Backup Procedures

### 1.1 Automated Daily Backup

```bash
# Run via cron (daily at 2 AM)
0 2 * * * /opt/vaultforge/scripts/backup-db.sh /var/backups/vaultforge full >> /var/log/vaultforge/backup.log 2>&1
```

### 1.2 Manual Backup

```bash
# Full database backup
./scripts/backup-db.sh /var/backups/vaultforge full

# Verify backup
gunzip -t /var/backups/vaultforge/vaultforge_*.sql.gz
```

### 1.3 Backup Verification

Weekly backup verification:

```bash
# Restore to a test database
createdb vaultforge_verify
gunzip -c /var/backups/vaultforge/vaultforge_LATEST.sql.gz | psql vaultforge_verify

# Run verification queries
psql vaultforge_verify -c "SELECT count(*) FROM intents;"
psql vaultforge_verify -c "SELECT count(*) FROM audit_events;"

# Drop test database
dropdb vaultforge_verify
```

---

## 2. Restore Procedures

### 2.1 Full Restore (Point-in-Time Recovery)

**When to use:** Database corruption, accidental data deletion, disaster recovery.

```bash
# 1. Stop the API server
systemctl stop vaultforge-api

# 2. Drop and recreate the database
dropdb vaultforge
createdb vaultforge

# 3. Restore from backup
gunzip -c /var/backups/vaultforge/vaultforge_YYYYMMDD_HHMMSS.sql.gz | psql vaultforge

# 4. (Optional) Apply WAL for point-in-time recovery
# If WAL archiving is enabled:
pg_waldump /var/backups/vaultforge/wal/... | psql vaultforge

# 5. Restart the API server
systemctl start vaultforge-api

# 6. Verify
curl http://localhost:8080/ready
```

### 2.2 Restore to a Different Server

```bash
# On new server:
# 1. Install PostgreSQL 16
# 2. Create database and user
createuser vaultforge
createdb -O vaultforge vaultforge

# 3. Copy backup to new server
scp /var/backups/vaultforge/vaultforge_*.sql.gz newserver:/tmp/

# 4. Restore
gunzip -c /tmp/vaultforge_*.sql.gz | psql vaultforge

# 5. Update DATABASE_URL in .env
# 6. Start API server
```

### 2.3 Selective Table Restore

```bash
# Restore only specific tables
gunzip -c /var/backups/vaultforge/vaultforge_*.sql.gz | \
    grep -A 10000 "COPY public.intents" | \
    head -n -1 | psql vaultforge
```

---

## 3. Failover Procedures

### 3.1 Database Failover (Primary → Replica)

**Prerequisites:** PostgreSQL streaming replication configured.

```bash
# 1. Promote replica to primary
pg_ctl promote -D /var/lib/postgresql/data

# 2. Update DATABASE_URL in API .env
# Change: host=primary-host → host=replica-host

# 3. Restart API server
systemctl restart vaultforge-api

# 4. Verify
curl http://localhost:8080/ready

# 5. (After primary recovers) Reconfigure as replica
# On former primary:
pg_ctl stop -D /var/lib/postgresql/data
# Add recovery.conf or standby.signal
pg_ctl start -D /var/lib/postgresql/data
```

### 3.2 API Server Failover

**For single server:**

```bash
# 1. Check if API is running
systemctl status vaultforge-api

# 2. Restart if failed
systemctl restart vaultforge-api

# 3. Check logs
journalctl -u vaultforge-api -n 50
```

**For Kubernetes (Helm):**

```bash
# Check pod status
kubectl get pods -l app=vaultforge

# Restart deployment
kubectl rollout restart deployment/vaultforge

# Scale up
kubectl scale deployment/vaultforge --replicas=3
```

### 3.3 Solana RPC Failover

If primary Solana RPC is down:

```bash
# 1. Switch to backup RPC
export SOLANA_RPC_URL="https://api.mainnet-beta.solana.com"

# 2. Or use a dedicated RPC provider (Helius, QuickNode, Triton)
export SOLANA_RPC_URL="https://mainnet.helius-rpc.com/?api-key=YOUR_KEY"

# 3. Restart API
systemctl restart vaultforge-api
```

---

## 4. Incident Response

### 4.1 Database Connection Exhaustion

**Symptoms:** `too many connections` errors, high latency.

```bash
# 1. Check active connections
psql -c "SELECT count(*) FROM pg_stat_activity WHERE datname='vaultforge';"

# 2. Kill idle connections
psql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
         WHERE datname='vaultforge' AND state='idle' AND query_start < now() - interval '10 minutes';"

# 3. Restart API to reset pool
systemctl restart vaultforge-api
```

### 4.2 High Memory Usage

**Symptoms:** OOM kills, swap usage.

```bash
# 1. Check memory
free -h
ps aux --sort=-%mem | head -5

# 2. Check goroutine count
curl http://localhost:8080/metrics | jq '.goroutines'

# 3. Restart if needed
systemctl restart vaultforge-api
```

### 4.3 Solana RPC Rate Limiting

**Symptoms:** `429 Too Many Requests` from Solana.

```bash
# 1. Check RPC health
curl -X POST https://api.devnet.solana.com \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# 2. Switch to paid RPC
export SOLANA_RPC_URL="https://mainnet.helius-rpc.com/?api-key=YOUR_KEY"

# 3. Reduce polling frequency in reconciler (if self-hosted)
```

---

## 5. Monitoring & Alerts

### Critical Alerts

| Alert | Threshold | Action |
|-------|-----------|--------|
| API down | 3 consecutive failures | Restart service |
| DB connection pool > 80% | 5 minutes | Kill idle connections, scale up |
| P99 latency > 500ms | 5 minutes | Profile, check DB, scale |
| Error rate > 1% | 5 minutes | Check logs, investigate |
| Disk usage > 85% | 1 hour | Clean old backups, expand |
| Solana RPC unhealthy | 3 minutes | Switch to backup RPC |

### Alert Configuration

See [deploy/monitoring/alerts.yml](../deploy/monitoring/alerts.yml) for Prometheus alert rules.

---

## 6. Communication Templates

### Customer-Facing Incident Notice

```
Subject: [VaultForge] Service Disruption - [DATE]

We are currently experiencing [ISSUE]. Our team has been notified
and is actively working on resolution.

Impact: [DESCRIPTION]
ETA: [TIME or "investigating"]

We will provide updates every 30 minutes until resolved.

Status page: https://status.vaultforge.io
```

### Internal Incident Update

```
[SEVERITY] [SERVICE] - [SUMMARY]

Status: [investigating/identified/monitoring/resolved]
Impact: [DESCRIPTION]
Root cause: [KNOWN or "investigating"]
Next update: [TIME]
```

---

## 7. Backup Schedule

| Type | Frequency | Retention | Storage |
|------|-----------|-----------|---------|
| Full pg_dump | Daily 2 AM | 30 days | Local + S3 |
| WAL archive | Continuous | 7 days | S3 |
| Config backup | On change | 90 days | Git |
| Docker image | On release | Latest 10 | Docker Hub |

## 8. Contacts

| Role | Contact | Escalation |
|------|---------|------------|
| Primary on-call | [NAME] | [PHONE] |
| Secondary on-call | [NAME] | [PHONE] |
| DBA | [NAME] | [PHONE] |
| Infrastructure | [NAME] | [PHONE] |
