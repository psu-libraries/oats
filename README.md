# OATS

This repository provides a collection of programs for managing the OA workflow. 

Organization:

- `cmd` - The primary `oats` command is `cmd/oats`.
- `crossref`: library for querying Crossref
- `oabutton`: library for querying OA Button
- `rmdb`: library for querying RMD.
- `scholargo`: library for ScholarSphere query/deposit
- `unpaywall`: library for querying unpaywall

## Development

Requires [Go](https://go.dev/dl/) v1.17 or greater.

```sh
# build primary oats command from source:
git@git.psu.edu:sre53/oats.git
cd oats/cmd/oats
go build
```

This project uses [GoReleaser](https://goreleaser.com/intro/)