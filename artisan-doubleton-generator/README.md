# Artisan Doubleton Set Cube Generator

A small Go utility that fetches all common and uncommon, non-basic cards from a Magic set using the [Scryfall](https://scryfall.com) API and writes a CSV for an artisan doubleton cube list.

## How It Works

1. Queries Scryfall for cards in the requested set.
2. Filters to commons and uncommons and excludes basic lands.
3. Writes a CSV with two copies of each matching card.

## Prerequisites

- [Go](https://go.dev) 1.25+ for local builds and runs.
- [Task](https://taskfile.dev) if you want to use the included `Taskfile.yml`.
- [Docker](https://docs.docker.com/get-started/) if you want to run the generator in a container.

## Usage

The generator requires a set code, such as `dsk` or `bro`.

### Run with Go

```bash
go run . dsk
```

### Run with Task

```bash
task run SET_CODE=dsk
```

### Run with Docker via Task

```bash
task run-docker SET_CODE=dsk
```

### Build the Binary

```bash
task build
```

## Output

The generator writes a CSV file named:

```text
<set-code>-artisan-doubleton-set-cube.csv
```

For example, `dsk-artisan-doubleton-set-cube.csv`.

The CSV contains these headers:

- `name`
- `Set`
- `Collector Number`

## Debugging

Set `DEBUG=1` to log API requests while running the generator.
