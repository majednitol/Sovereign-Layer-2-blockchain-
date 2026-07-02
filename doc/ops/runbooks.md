# Operations Runbooks

This document outlines key operational playbooks and disaster recovery procedures for the Sovereign L1 Blockchain.

---

## 1. Key Rotation Playbook

Sovereign L1 uses specialized keys for various critical functions. In the event of compromise, routine expiration, or policy updates, follow these procedures to rotate them safely.

### 1.1 Oracle Operator Key Rotation
Oracle operators commit and reveal price feed reports. If an operator key is compromised:
1. **Generate New Operator Key**:
   ```bash
   chaind keys add <new-oracle-operator> --keyring-backend file
   ```
2. **Submit Replacement Proposal**:
   Submit a governance proposal (`MsgUpdateOracleOperator`) to update the registered validator operator mapping to the new address.
   ```bash
   chaind tx gov submit-proposal update-oracle-operator <operator-address> <new-oracle-address> --from <proposer> --fees 2000usov
   ```
3. **Transition Operator Infrastructure**:
   Configure the price feed watcher daemon to sign commitments using the new key once the governance proposal passes.

### 1.2 Witness Key Rotation
Witnesses verify and sign settlement payloads. If a witness key is compromised:
1. **Generate New Witness Keypair**:
   Using `ed25519` key generation tools, generate a new private/public keypair.
2. **Register New Witness PubKey**:
   Submit a governance transaction (`MsgUpdateWitnessRegistry`) to replace the active public key of the witness ID in `x/settlement`.
   ```bash
   chaind tx gov submit-proposal update-witness-registry <witness-id> <new-public-key-hex> --from <proposer> --fees 2000usov
   ```
3. **Deploy New Private Key**:
   Update the witness daemon configuration with the new private key and restart the service.

### 1.3 Relayer Key Rotation
Bridge relayers submit cross-chain transfers to the bridge module. To rotate relayer keys:
1. **Update Relayer Local Keys**:
   Generate a new relayer key on the target chain.
2. **Propose Relayer Set Update**:
   Submit a governance proposal (`MsgUpdateBridgeRelayerSet`) to register the new relayer address and remove the old one.
   ```bash
   chaind tx gov submit-proposal update-bridge-relayer-set <new-relayer-address> <old-relayer-address> --from <proposer> --fees 2000usov
   ```
3. **Activate New Relayer**:
   Restart the relayer daemon with the new key once the proposal is executed on-chain.

### 1.4 Circuit-Breaker Key Rotation
The circuit-breaker address can pause the bridge module in emergencies.
1. **Define New Circuit-Breaker Address**:
   Acquire the address of the backup multisig or security administrator account.
2. **Propose Parameter Update**:
   Update the bridge module parameters via governance proposal, setting `CircuitBreakerAddress` to the new address.

---

## 2. PostgreSQL PITR (Point-in-Time Recovery) Restore Drill

Sovereign indexing and database layers rely on PostgreSQL. Follow this drill to verify and execute a point-in-time recovery.

### 2.1 WAL Archiving Verification
Ensure continuous WAL archiving is configured and healthy:
```sql
-- Check archive status
SELECT name, last_archived_wal, last_archived_time, failed_count, last_failed_time 
FROM pg_stat_archiver;
```

### 2.2 Restore Procedure
In the event of database corruption or accidental write:
1. **Stop the PostgreSQL Server**:
   ```bash
   pg_ctl -D /var/lib/postgresql/data stop
   ```
2. **Restore Base Backup**:
   Move the current data directory to a backup location and restore the latest clean base backup:
   ```bash
   mv /var/lib/postgresql/data /var/lib/postgresql/data.corrupted
   tar -xf /backups/base_backup.tar.gz -C /var/lib/postgresql/
   ```
3. **Configure Recovery**:
   Create a `recovery.signal` file in the data directory:
   ```bash
   touch /var/lib/postgresql/data/recovery.signal
   ```
   Add the following lines to `postgresql.conf` (or `postgresql.auto.conf`):
   ```ini
   restore_command = 'cp /backups/archived_wals/%f "%p"'
   recovery_target_time = '2026-06-24 02:00:00+06'
   recovery_target_action = 'promote'
   ```
4. **Start PostgreSQL and Monitor Logs**:
   ```bash
   pg_ctl -D /var/lib/postgresql/data start
   tail -f /var/lib/postgresql/data/log/postgresql.log
   ```
5. **Verify Database Integrity**:
   Check if the database state matches the desired recovery timestamp and ensure no errors are reported.
