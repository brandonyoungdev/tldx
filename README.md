![tldx logo](https://github.com/brandonyoungdev/tldx/raw/main/assets/logo.png)

# tldx

![GitHub release (latest by date)](https://img.shields.io/github/v/release/brandonyoungdev/tldx)
![Tests](https://img.shields.io/github/actions/workflow/status/brandonyoungdev/tldx/test.yml?branch=main)
![GitHub](https://img.shields.io/github/license/brandonyoungdev/tldx)
[![Go Report Card](https://goreportcard.com/badge/github.com/brandonyoungdev/tldx)](https://goreportcard.com/report/github.com/brandonyoungdev/tldx)


`tldx` helps you brainstorm available domain names fast.

```sh
tldx openai -p get,use -s ly,hub -t com,io,ai --only-available
✔️ getopenaily.com is available
✔️ useopenaihub.io is available
  ...
```


![tldx demo](https://github.com/brandonyoungdev/tldx/raw/main/tapes/demo.gif)

## 📚 Table of Contents

- [Features](#-features)
- [Usage](#-usage)
- [Examples](#-examples)
  - [Domain Availability](#domain-availability)
  - [Regex Domain Selection](#regex-domain-selection)
  - [Presets](#presets)
  - [Permutations](#permutations)
  - [Brace Expansion](#brace-expansion-macos-linux)
  - [Show Only Available Domains](#show-only-available-domains)
  - [Limit Results](#limit-results)
  - [Dry Run](#dry-run)
  - [Input from File or Stdin](#input-from-file-or-stdin)
  - [Output Formats](#output-formats)
- [MCP](#mcp)
- [Installation](#-installation)

## ⚡ Features

- 🔍 Smart keyword-based domain permutations (prefixes, suffixes, TLDs)
- 🎯 Regex pattern support for generating domain combinations (e.g., all 3-letter domains)
- 🚀 Fast and concurrent availability checks with RDAP
- 📤 Streams results as they're found
- 📦 Multiple output formats: `text`, `json`, `json-stream`, `json-array`, `csv`, `grouped`, `grouped-tld`
- 🔧 TLD presets to quickly select curated TLD sets
- 📏 Optional filtering by domain length
- 🤖 MCP server (`tldx mcp`) for AI agent integration


## 🛠️ Usage

```sh
Usage:
  tldx [keywords] [flags]
  tldx [command]

Available Commands:
  completion       Generate the autocompletion script for the specified shell
  help             Help about any command
  mcp              Start an MCP (Model Context Protocol) server over stdio
  show-tld-presets Show available TLD presets

Flags:
      --dry-run                 Print domains that would be checked without making network calls
  -f, --format string           Format of output (text, json, json-stream, json-array, csv, grouped, grouped-tld) (default "text")
  -h, --help                    help for tldx
  -i, --input string            File to read keywords from. Use "-" to read from stdin.
  -l, --limit int               Stop after finding this many available domains (0 = no limit)
  -m, --max-domain-length int   Maximum length of domain name (default 64)
      --no-color                Disable colored output
  -a, --only-available          Show only available domains
  -p, --prefixes strings        Prefixes to add (e.g. get,my,use)
  -r, --regex                   Enable regex pattern matching for domain keywords
      --show-stats              Show statistics at the end of execution
  -s, --suffixes strings        Suffixes to add (e.g. ify,ly)
      --tld-preset string       Use a tld preset (e.g. popular, tech)
  -t, --tlds strings            TLDs to check (e.g. com,io,ai)
  -v, --verbose                 Show verbose output
      --version                 version for tldx
```

Exit code `2` is returned when `--only-available` is set but no available domains are found.

## 🔗 Examples

### Domain Availability

```sh
$ tldx google
❌ google.com is not available
```

```sh
$ tldx google youtube reddit
  ❌ reddit.com is not available
  ❌ google.com is not available
  ❌ youtube.com is not available
```

### Regex Domain Selection

Use regex patterns with the `--regex` flag to generate domain combinations based on patterns:

```sh
# Check all 3-letter .com domains
$ tldx '[a-z]{3}' --regex --tlds com --only-available
  ✔️  aaa.com is available
  ✔️  aab.com is available
  ...
```

```sh
# Check all 2-letter domains with specific TLDs
$ tldx '[a-z]{2}' --regex --tlds io,ai --only-available
  ✔️  qa.io is available
  ✔️  zx.ai is available
  ...
```

```sh
# Combine patterns with prefixes
$ tldx '[a-z]{2}' --regex --prefixes my,get --tlds app --only-available
  ✔️  myaa.app is available
  ✔️  getab.app is available
  ...
```

**Note:** Patterns generating more than 500,000 combinations will be skipped.

### Presets

```sh
$ tldx google --tld-preset popular
  ❌ google.com is not available
  ❌ google.io is not available
  ...
```

```sh
$ tldx google --tld-preset geo
  ❌ google.au is not available
  ❌ google.de is not available
  ❌ google.us is not available
  ...
```

You can see all available presets:
```sh
$ tldx show-tld-presets

TLD Presets:

all                     (use all available TLDs)

cheap                   pw fun icu top xyz blog info shop site click
                        space store online website

popular                 ai me app com dev net org

tech                    io ai gg app dev tech codes tools cloud games
                        software digital network security systems
                        data technology
...

```

### Permutations

```sh
$ tldx google --prefixes get,my --suffixes ly,hub --tlds com,io,ai
  ✔️  mygooglely.com is available
  ✔️  getgooglely.ai is available
  ❌  mygoogle.ai is not available
  ...
```


### Brace Expansion (macOS, Linux)

[Brace expansion](https://www.gnu.org/software/bash/manual/html_node/Brace-Expansion.html) works out of the box in bash/zsh:

```sh
tldx {get,use}{tldx,domains} {star,fork}ongithub
  ✔️ gettldx.com is available
  ✔️ usetldx.com is available
  ❌ getdomains.com is not available
  ...
```


### Show Only Available Domains

```sh
$ tldx google reddit facebook -p get,my -s ly,hub -t com,io,ai --only-available
  ✔️  getgooglely.ai is available
  ✔️  getreddithub.com is available
  ...
```

### Limit Results

```sh
$ tldx stripe -p get,use -t com,io,ai --only-available --limit 3
  ✔️  getstripe.io is available
  ✔️  usestripe.ai is available
  ✔️  stripe.ai is available
```

### Dry Run

```sh
$ tldx stripe -p get,use -t com,io --dry-run
Would check 6 domain(s):
  stripe.com
  stripe.io
  getstripe.com
  ...
```

### Input from File or Stdin

```sh
$ tldx --input keywords.txt --tlds com,io --only-available
$ echo -e "stripe\natlas\nlinear" | tldx --input - --tlds com,io --only-available
```

### Output Formats

By default output is human-readable (`text`). Change it with `--format` / `-f`.

Color is automatically disabled when stdout is not a terminal.

#### JSON Array
```sh
$ tldx openai -p use -s ly -t io --format json-array
[
  { "domain": "useopenaily.io", "available": true, "keyword": "openai", "prefix": "use", "suffix": "ly", "tld": "io" },
  { "domain": "openai.io", "available": false, "keyword": "openai", "tld": "io" },
  ...
]
```

With `--show-stats` the output is wrapped in an object:
```sh
$ tldx openai -p use -s ly -t io --format json-array --show-stats
{
  "results": [ ... ],
  "stats": { "total": 4, "available": 1, "not_available": 2, "errored": 1 }
}
```

Results include `keyword`, `prefix`, `suffix`, and `tld` metadata (empty fields are omitted).

#### JSON Stream
```sh
$ tldx openai -p use -s ly -t io --format json-stream
{"domain":"useopenaily.io","available":true,"keyword":"openai","prefix":"use","suffix":"ly","tld":"io"}
{"domain":"openai.io","available":false,"keyword":"openai","tld":"io"}
```

#### CSV
```sh
$ tldx openai -p use -s ly -t io --format csv
domain,available,keyword,prefix,suffix,tld,details,error
useopenaily.io,true,openai,use,ly,io,
openai.io,false,openai,,,io,
```

#### Grouped by Keyword
```sh
$ tldx openai google -p get,use -t com,io --format grouped

  google
  getgoogle.com
  getgoogle.io
  google.com
  google.io
  usegoogle.com
  usegoogle.io

  openai
  getopenai.com
  getopenai.io
  openai.com
  openai.io
  useopenai.com
  useopenai.io
```

#### Grouped by TLD
```sh
$ tldx openai google -p get,use -t com,io --format grouped-tld

  .com
  getgoogle.com
  getopenai.com
  google.com
  openai.com
  usegoogle.com
  useopenai.com

  .io
  getgoogle.io
  getopenai.io
  google.io
  openai.io
  usegoogle.io
  useopenai.io
```

## MCP

`tldx` includes an MCP server for use with AI agents and IDEs.

```sh
tldx mcp
```

Example config (`mcp.json` / Claude Desktop / VS Code):

```json
{
  "mcpServers": {
    "tldx": {
      "command": "tldx",
      "args": ["mcp"]
    }
  }
}
```

Available tools: `check_domain`, `check_domains`, `generate_and_check`, `list_tld_presets`.

## 📦 Installation
#### macOS (Homebrew)
```sh
brew install tldx
```
or
```sh
brew tap brandonyoungdev/tldx
brew install tldx
```


#### Windows (winget)

```sh
winget install --id=brandonyoungdev.tldx  -e
```

#### Arch Linux (AUR)

Two options are available for Arch Linux users:

- [tldx](https://aur.archlinux.org/packages/tldx/) - Build the package from source.
- [tldx-bin](https://aur.archlinux.org/packages/tldx-bin/) - Build the package from releases.

#### Linux and Windows (Manual)
Visit the [Releases page](https://github.com/brandonyoungdev/tldx/releases).

Download the archive for your OS and architecture:

- macOS / Linux: `tldx_<version>_<os>_<arch>.tar.gz`
- Windows: `tldx_<version>_windows_<arch>.zip`

```sh
tar -xzf tldx_<version>_<os>_<arch>.tar.gz
mv tldx /usr/local/bin/
```

#### Go (Install from Source)
```sh
go install github.com/brandonyoungdev/tldx@latest
```
