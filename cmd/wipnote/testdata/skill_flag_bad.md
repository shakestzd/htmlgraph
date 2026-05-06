# Fixture — bad flag that should be flagged

The skill-flag validator must fail on the invocation below, because
`htmlgraph feature show` does not register `--this-flag-doesnt-exist`.

```
htmlgraph feature show feat-abc123 --this-flag-doesnt-exist
```
