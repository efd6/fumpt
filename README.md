# `fumpt`

The `fumpt` program formats Fleet integration package YAML source files with a consistent and minimal style.

## Known issues

- Comments following fold or literal string values will be lost (first line only) — [issue](https://github.com/goccy/go-yaml/issues/326).
- Vertical white-space is lost.