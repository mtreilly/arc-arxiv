# arc-arxiv

A CLI tool to fetch, search, and manage arXiv papers with rich metadata, export capabilities, and local organization.

## Installation

```bash
go install github.com/mtreilly/arc-arxiv@latest
```

Or build from source:

```bash
git clone https://github.com/mtreilly/arc-arxiv.git
cd arc-arxiv
go build -o arc-arxiv .
```

## Usage

### Fetch Papers

Download papers with full metadata (title, authors, abstract, categories, DOI, etc.):

```bash
# By arXiv ID
arc-arxiv fetch 2304.00067

# By URL (abs or PDF)
arc-arxiv fetch https://arxiv.org/abs/2304.00067
arc-arxiv fetch https://arxiv.org/pdf/2304.00067.pdf

# Multiple papers at once
arc-arxiv fetch 2304.00067 2301.12345 2312.99999

# Re-fetch existing paper
arc-arxiv fetch 2304.00067 --force

# Extract text from PDF
arc-arxiv fetch 2304.00067 --extract-text
```

Papers are saved to `~/arc-engineering/docs/research-external/papers/<arxiv-id>/` with:
- `meta.yaml` - Full paper metadata
- `paper.pdf` - The PDF file
- `notes.md` - Template for your notes

### Search arXiv

```bash
# Free-text search
arc-arxiv search "transformer attention"

# Filter by field
arc-arxiv search --author "Hinton"
arc-arxiv search --title "dropout" --category cs.LG
arc-arxiv search --category physics.hep-th --max 20

# Sort results
arc-arxiv search "neural networks" --sort submitted
arc-arxiv search "quantum computing" --sort updated

# Auto-fetch search results
arc-arxiv search "attention is all you need" --fetch
```

### List Downloaded Papers

```bash
# List all papers
arc-arxiv list

# Filter by category
arc-arxiv list --category cs.LG

# Filter by author
arc-arxiv list --author "Vaswani"

# Filter by fetch date
arc-arxiv list --since 2024-01-01

# JSON output
arc-arxiv list --output json
```

### View Paper Details

```bash
arc-arxiv info 2304.00067
```

### Open Papers

```bash
# Open paper directory
arc-arxiv open 2304.00067

# Open PDF
arc-arxiv open 2304.00067 --pdf

# Open notes
arc-arxiv open 2304.00067 --notes

# Open in browser
arc-arxiv open 2304.00067 --web
```

### Export Papers

```bash
# Export to BibTeX
arc-arxiv export 2304.00067 --format bibtex
arc-arxiv export --all --format bibtex -o references.bib

# Export to CSV
arc-arxiv export --all --format csv -o papers.csv

# Export to JSON
arc-arxiv export --all --format json
```

### Update Metadata

```bash
# Update specific paper
arc-arxiv update 2304.00067

# Update all papers
arc-arxiv update --all

# Check for new versions without updating
arc-arxiv update --check
```

## Metadata Structure

Each paper's `meta.yaml` contains:

```yaml
id: paper-2304.00067
arxiv_id: "2304.00067"
title: "Paper Title"
source_type: arxiv
url: https://arxiv.org/abs/2304.00067
pdf_url: https://arxiv.org/pdf/2304.00067.pdf
published: "2023-04-01T00:00:00Z"
updated: "2023-04-15T00:00:00Z"
authors:
  - name: Author Name
    affiliation: University
abstract: "Paper abstract..."
categories:
  - cs.LG
  - cs.AI
primary_category: cs.LG
comment: "10 pages, 5 figures"
journal_ref: "Nature 2023"
doi: "10.1234/example"
version: 2
fetched_at: "2024-01-15T10:30:00Z"
```

## Dependencies

- [goarxiv](https://github.com/mtreilly/goarxiv) - arXiv API client library
- [cobra](https://github.com/spf13/cobra) - CLI framework
- [arc-sdk](https://github.com/mtreilly/arc-sdk) - Arc Engineering SDK (config, output utilities)

## License

MIT
