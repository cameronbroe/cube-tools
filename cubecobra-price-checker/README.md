# CubeCobra Cube Price Calculator

A GitHub Action that fetches a [CubeCobra](https://cubecobra.com) cube's mainboard and calculates the total minimum paper cost using [Scryfall](https://scryfall.com) price data.

## How It Works

1. Fetches the cube's mainboard card list from the CubeCobra API.
2. Downloads Scryfall's default-cards bulk data (cached for 12 hours).
3. For each card in the mainboard, finds all valid physical printings (excludes digital-only and oversized cards).
4. Selects the cheapest printing/treatment (normal or foil) for each card.
5. Sums the minimum costs and writes a Markdown summary to the GitHub Actions job summary.

## Usage

```yaml
- name: Calculate cube price
  uses: cameronbroe/misc-actions/cubecobra-price-checker@main
  with:
    cube_id: your-cube-id
```

## Inputs

| Input     | Required | Description                          |
|-----------|----------|--------------------------------------|
| `cube_id` | ✅ Yes   | The CubeCobra cube ID to price-check |

## Outputs

This action produces no step outputs. Instead, it appends a Markdown summary to the [GitHub Actions job summary](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/workflow-commands-for-github-actions#adding-a-job-summary) (`$GITHUB_STEP_SUMMARY`) with:

- **Total Minimum Cost** — the sum of the cheapest available printing for every card in the mainboard.
- **Cards Above $5** — a list of individual cards whose cheapest printing exceeds $5, with links to their Scryfall pages.

## Example Summary Output

```markdown
## Cube Cost Analysis

### Total Minimum Cost of Cube: $843.21

### Cards Above $5

* Tarmogoyf - https://scryfall.com/card/...
* Force of Will - https://scryfall.com/card/...
```

## Prerequisites

This action requires [Docker](https://docs.docker.com/get-started/) to be available on the runner. It uses a Docker container (built from the included `Dockerfile`) to run the Go binary, so no Go toolchain needs to be pre-installed on the runner.

## Dependencies

- [go-task/setup-task](https://github.com/go-task/setup-task) — used internally to orchestrate the build and run steps.
- [CubeCobra API](https://cubecobra.com/cube/api/cubeJSON) — fetches cube card lists.
- [Scryfall Bulk Data API](https://scryfall.com/docs/api/bulk-data) — provides card price data.
