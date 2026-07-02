#!/bin/bash
# Enable replication connections from the Docker network
# This script runs as part of the PostgreSQL init process
echo "host replication all 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
echo "host replication all ::/0 md5" >> "$PGDATA/pg_hba.conf"
