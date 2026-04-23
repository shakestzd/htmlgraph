# Fixture — only real flags

Each of the invocations below uses a flag that is registered on its target
command. The skill-flag validator must not report any violation.

```
htmlgraph track show trk-abc123 --format json
htmlgraph track show trk-abc123 --deep
htmlgraph feature show feat-abc123 --format json
```
