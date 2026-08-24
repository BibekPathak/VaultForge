#!/usr/bin/env bash
set -euo pipefail

# VaultForge Database Backup Script
# Performs pg_dump with compression and optional WAL archiving
#
# Usage:
#   ./scripts/backup-db.sh                    # Full backup to default path
#   ./scripts/backup-db.sh /mnt/backups       # Full backup to custom path
#   ./scripts/backup-db.sh /mnt/backups full   # Explicit full backup
#   ./scripts/backup-db.sh /mnt/backups wal    # WAL archiving only
#
# Environment:
#   DATABASE_URL   - PostgreSQL connection string (required)
#   BACKUP_RETAIN  - Days to keep backups (default: 30)
#   BACKUP_PREFIX  - Filename prefix (default: vaultforge)

BACKUP_DIR="${1:-/var/backups/vaultforge}"
BACKUP_TYPE="${2:-full}"
BACKUP_RETAIN="${BACKUP_RETAIN:-30}"
BACKUP_PREFIX="${BACKUP_PREFIX:-vaultforge}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATABASE_URL="${DATABASE_URL:-host=localhost user=vaultforge password=vaultforge dbname=vaultforge port=5432 sslmode=disable}"

echo "=== VaultForge Database Backup ==="
echo "Type:      $BACKUP_TYPE"
echo "Directory: $BACKUP_DIR"
echo "Timestamp: $TIMESTAMP"
echo ""

mkdir -p "$BACKUP_DIR"

case "$BACKUP_TYPE" in
    full)
        BACKUP_FILE="$BACKUP_DIR/${BACKUP_PREFIX}_${TIMESTAMP}.sql.gz"
        echo "Creating full backup: $BACKUP_FILE"

        pg_dump "$DATABASE_URL" \
            --format=plain \
            --no-owner \
            --no-privileges \
            --verbose 2>&1 | gzip > "$BACKUP_FILE"

        BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
        echo "Backup complete: $BACKUP_SIZE"

        # Verify backup integrity
        echo "Verifying backup..."
        gunzip -t "$BACKUP_FILE" && echo "Integrity check: OK" || {
            echo "ERROR: Backup integrity check failed!"
            exit 1
        }

        # Generate checksum
        sha256sum "$BACKUP_FILE" > "${BACKUP_FILE}.sha256"
        echo "Checksum: ${BACKUP_FILE}.sha256"
        ;;

    wal)
        echo "Enabling WAL archiving (for point-in-time recovery)..."
        echo ""
        echo "Add these to postgresql.conf:"
        echo "  wal_level = replica"
        echo "  archive_mode = on"
        echo "  archive_command = 'test ! -f /var/backups/vaultforge/wal/%f && cp %p /var/backups/vaultforge/wal/%f'"
        echo ""
        echo "Or use pgBackRest / Barman for production WAL archiving."
        ;;

    *)
        echo "ERROR: Unknown backup type: $BACKUP_TYPE"
        echo "Usage: $0 [backup_dir] [full|wal]"
        exit 1
        ;;
esac

# ── Cleanup Old Backups ───────────────────────────────
echo ""
echo "Cleaning up backups older than $BACKUP_RETAIN days..."
DELETED=$(find "$BACKUP_DIR" -name "${BACKUP_PREFIX}_*.sql.gz" -mtime +$BACKUP_RETAIN -delete -print | wc -l)
find "$BACKUP_DIR" -name "${BACKUP_PREFIX}_*.sql.gz.sha256" -mtime +$BACKUP_RETAIN -delete 2>/dev/null || true
echo "Removed $DELETED old backup(s)"

# ── Summary ───────────────────────────────────────────
echo ""
echo "=== Backup Summary ==="
echo "Backups on disk:"
ls -lh "$BACKUP_DIR"/${BACKUP_PREFIX}_*.sql.gz 2>/dev/null || echo "  (none)"
echo ""
echo "Total size: $(du -sh "$BACKUP_DIR" | cut -f1)"
