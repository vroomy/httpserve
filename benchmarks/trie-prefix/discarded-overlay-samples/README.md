# Discarded setup run

The Go overlay used a `/tmp` source key while the build used its canonical `/private/tmp` path. The replacement was not applied: both binaries had SHA256 `2701a30d887075537d3d65066fd68301159d3b08777427cbe89729e2869ea448`. The run was interrupted during focused round 3 and excluded from every reported statistic. Raw output is retained for the audit trail. The valid run uses a separate physical source copy and rejects identical binaries before timing.
