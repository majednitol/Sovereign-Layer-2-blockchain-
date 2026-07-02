# Testnet Stability Window & Freeze Checklist (Milestones 6.8 & 6.9)

## Liveness Target Metrics
- **Consensus Liveness**: 100% blocks finalized with block times ≤ 5.0 seconds.
- **Missed Block Thresholds**: Alert triggered if a validator misses >10 blocks in 5 minutes.
- **Network Uptime**: Zero system outages or consensus partition incidents over a 4-week window.

## Code Freeze Policy (Week 17)
- Zero protocol modifications allowed post-Week 17.
- Bugfixes restricted to high/critical CVEs or liveness issues.
- Code audits scheduled immediately following the 4-week validation phase.
