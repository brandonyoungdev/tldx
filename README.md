![tldx logo](https://github.com/brandonyoungdev/tldx/raw/main/assets/logo.png)

# tldx

![GitHub release (latest by date)](https://img.shields.io/github/v/release/brandonyoungdev/tldx)
![Tests](https://img.shields.io/github/actions/workflow/status/brandonyoungdev/tldx/test.yml?branch=main)
![GitHub](https://img.shields.io/github/license/brandonyoungdev/tldx)
[![Go Report Card](https://goreportcard.com/badge/github.com/brandonyoungdev/tldx)](https://goreportcard.com/report/github.com/brandonyoungdev/tldx)
[![codecov](https://codecov.io/gh/brandonyoungdev/tldx/branch/main/graph/badge.svg)](https://codecov.io/gh/brandonyoungdev/tldx)
[![MCP Toplist](https://mcptoplist.com/badge/glama%2Fbrandonyoungdev%2Ftldx.svg)](https://mcptoplist.com/server/glama%2Fbrandonyoungdev%2Ftldx)


`tldx` helps you brainstorm available domain names fast.

```sh
tldx openai -p get,use -s ly,hub -t com,io,ai --only-available
✅ getopenaily.com is available
✅ useopenaihub.io is available
  ...
```


![tldx demo](https://github.com/brandonyoungdev/tldx/raw/main/tapes/demo.gif)

## Table of Contents

- [Features](#features)
- [Usage](#usage)
- [Examples](#examples)
  - [Domain Availability](#domain-availability)
  - [Regex Domain Selection](#regex-domain-selection)
  - [Presets](#presets)
  - [Custom Presets](#custom-presets)
  - [Defaults and Config File](#defaults-and-config-file)
  - [Permutations](#permutations)
  - [Brace Expansion](#brace-expansion-macos-linux)
  - [Domains For Sale (RFC 10023)](#domains-for-sale-rfc-10023)
  - [Show Only Available Domains](#show-only-available-domains)
  - [Limit Results](#limit-results)
  - [Dry Run](#dry-run)
  - [Input from File or Stdin](#input-from-file-or-stdin)
  - [Output Formats](#output-formats)
- [MCP](#mcp)
- [Installation](#installation)

## Features

- Keyword permutations across prefixes, suffixes, and TLDs
- Regex patterns for bulk combinations (e.g., all 3-letter domains)
- Fast, concurrent availability checks over RDAP
- Results stream as they are found
- Output as `text`, `json`, `json-stream`, `json-array`, `csv`, `grouped`, or `grouped-tld`
- Finds taken domains advertised for sale via [RFC 10023](https://www.rfc-editor.org/info/rfc10023/)
- Built-in and custom TLD presets
- A config file for your usual TLDs, preset, and flags
- An MCP server (`tldx mcp`) for AI agents


## Usage

```sh
Usage:
  tldx [keywords] [flags]
  tldx [command]

Available Commands:
  completion       Generate the autocompletion script for the specified shell
  config           Inspect and manage the tldx config file
  help             Help about any command
  mcp              Start an MCP (Model Context Protocol) server over stdio
  preset           Manage custom TLD presets

Flags:
      --dry-run                 Print domains that would be checked without making network calls
      --for-sale                Check taken domains for an RFC 10023 _for-sale TXT record
  -f, --format string           Format of output (text, json, json-stream, json-array, csv, grouped, grouped-tld) (default "text")
  -h, --help                    help for tldx
  -i, --input string            File to read keywords from. Use "-" to read from stdin.
  -l, --limit int               Stop after finding this many available domains (0 = no limit)
  -m, --max-domain-length int   Maximum length of domain name (default 64)
      --no-color                Disable colored output
  -a, --only-available          Show only available domains
      --only-for-sale           Show only taken domains that are for sale (implies --for-sale)
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

## Examples

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

`--regex` treats each keyword as a pattern and checks every combination it expands to:

```sh
# All 3-letter .com domains
$ tldx '[a-z]{3}' --regex --tlds com --only-available
  ✅ aaa.com is available
  ✅ aab.com is available
  ...
```

```sh
# All 2-letter domains on specific TLDs
$ tldx '[a-z]{2}' --regex --tlds io,ai --only-available
  ✅ qa.io is available
  ✅ zx.ai is available
  ...
```

```sh
# Patterns combine with prefixes
$ tldx '[a-z]{2}' --regex --prefixes my,get --tlds app --only-available
  ✅ myaa.app is available
  ✅ getab.app is available
  ...
```

Patterns that expand to more than 500,000 combinations are skipped.

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
$ tldx preset list

TLD Presets  (* = custom):

all                     (use all available TLDs)

cheap                   pw fun icu top xyz blog info shop site click
                        space store online website

popular                 ai me app com dev net org

tech                    io ai gg app dev tech codes tools cloud games
                        software digital network security systems
                        data technology
...

```

### Custom Presets

Save your own TLD presets and reuse them across runs:

```sh
# Create a preset
$ tldx preset add myteam com io ai
Saved preset "myteam" (com, io, ai) → ~/.config/tldx/config.toml

# Comma-separated also works
$ tldx preset add myteam com,io,ai

# Use it just like any built-in preset
$ tldx mystartup --tld-preset myteam
  ❌ mystartup.com is not available
  ✅ mystartup.io is available
  ✅ mystartup.ai is available
```

```sh
# List all presets.
$ tldx preset list

TLD Presets  (* = custom):

all                       (use all available TLDs)

myteam *                  ai io com
popular                   ai io app com dev net org
...

Config file: ~/.config/tldx/config.toml
```

```sh
# Remove a custom preset
$ tldx preset remove myteam
Removed preset "myteam"
```

### Defaults and Config File

Set a default preset and it applies to every run:

```sh
$ tldx preset default nordic
Default preset set to "nordic" → ~/.config/tldx/config.toml

$ tldx mystartup
  ✅ mystartup.se is available
  ✅ mystartup.nu is available
  ❌ mystartup.dk is not available

# Show the current default, or drop it
$ tldx preset default
Default preset: nordic

$ tldx preset default --clear
Cleared default preset (was "nordic") → ~/.config/tldx/config.toml
```

Everything else goes in the config file. `tldx config init` writes a commented
template:

```toml
[defaults]
# TLDs checked when neither --tlds nor --tld-preset is given
tlds = ["com", "se", "nu"]

# ...or a preset name instead
# tld_preset = "nordic"

prefixes = ["get", "my"]
suffixes = ["ly"]
max_domain_length = 20
format = "json"
limit = 10
only_available = true
show_stats = true
no_color = false
verbose = false

# Read RFC 10023 "_for-sale" records on taken domains.
# only_for_sale implies for_sale.
for_sale = true
only_for_sale = false

# Custom presets, usable via --tld-preset nordic
[presets.nordic]
tlds = ["se", "nu", "dk", "no", "fi"]
```

Each key under `[defaults]` matches the flag of the same name, and flags passed
on the command line win. Since `tlds` and `tld_preset` both answer "which
TLDs?", passing either `--tlds` or `--tld-preset` ignores both configured
values rather than merging with them.

```sh
$ tldx config path    # where the file lives
$ tldx config show    # what's currently configured
$ tldx config init    # write a commented template (--force to overwrite)
```

`tldx config init --force` replaces the whole file, custom presets included.

The file lives at `~/.config/tldx/config.toml` (macOS/Linux) or
`%APPDATA%\tldx\config.toml` (Windows). A `presets.toml` from an earlier
version still works — add a `[defaults]` section to it. Set `TLDX_CONFIG` to
use a different file.

### Permutations

```sh
$ tldx google --prefixes get,my --suffixes ly,hub --tlds com,io,ai
  ✅ mygooglely.com is available
  ✅ getgooglely.ai is available
  ❌ mygoogle.ai is not available
  ...
```


### Brace Expansion (macOS, Linux)

[Brace expansion](https://www.gnu.org/software/bash/manual/html_node/Brace-Expansion.html) works out of the box in bash/zsh:

```sh
tldx {get,use}{tldx,domains} {star,fork}ongithub
  ✅ gettldx.com is available
  ✅ usetldx.com is available
  ❌ getdomains.com is not available
  ...
```


### Domains For Sale (RFC 10023)

[RFC 10023](https://www.rfc-editor.org/info/rfc10023/) defines a `_for-sale` TXT record that a domain holder can
publish to advertise that the name is for sale, optionally with an asking price and a way to get in touch.
`--for-sale` reads it:

```sh
$ tldx acme -t com,io --for-sale
  ❌ acme.io is not available
  💰 acme.com is taken but for sale — USD 750 · https://fs.example.com/
```

`--only-for-sale` keeps just the purchasable ones. Combine it with `--only-available` to see both kinds of
opportunity at once:

```sh
$ tldx acme wile coyote -t com,io --only-for-sale
$ tldx acme wile coyote -t com,io --only-available --only-for-sale
```

`--verbose` also prints the free text and broker codes from the record. The lookup costs one extra DNS query
per taken domain and never runs on domains that are already available, so it stays off unless you ask for it.
A holder publishes it like this:

```dns
_for-sale.acme.com. IN TXT "v=FORSALE1;fval=USD750"
                    IN TXT "v=FORSALE1;furi=https://fs.example.com/"
```

These records are written by the domain holder, so `tldx` treats them as untrusted: control characters are
stripped, and only `http`, `https`, `mailto` and `tel` links are shown by default. Any other scheme appears
under `--verbose`, flagged as unverified.

### Show Only Available Domains

```sh
$ tldx google reddit facebook -p get,my -s ly,hub -t com,io,ai --only-available
  ✅ getgooglely.ai is available
  ✅ getreddithub.com is available
  ...
```

### Limit Results

```sh
$ tldx stripe -p get,use -t com,io,ai --only-available --limit 3
  ✅ getstripe.io is available
  ✅ usestripe.ai is available
  ✅ stripe.ai is available
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

Output is human-readable (`text`) by default. Change it with `--format` / `-f`.
Color is disabled automatically when stdout is not a terminal.

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
  ...
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
  ...
```

## MCP

`tldx` includes an MCP server for AI agents and IDEs.

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

Two tools, both read-only and returning the same result shape:

| Tool | Use it when |
| --- | --- |
| `check_domains` | You already know the exact names to test. |
| `generate_and_check` | You want names built from keywords, prefixes, suffixes, and TLDs. |

Your custom presets and `[defaults]` from the [config file](#defaults-and-config-file) apply here just as they
do on the command line. `generate_and_check` advertises every preset name in its `tld_preset` schema, so no
separate lookup call is needed.

### Result shape

Each result carries a `status` of `available`, `taken`, or `unknown`. `unknown` means the lookup failed, not
that the domain is free, and the `available` field is omitted entirely in that case.

Each response also reports `checked`, `available_count`, `taken_count`, and — when the search stopped early —
`truncated: true` plus a `note` explaining what to change.

### Call budget

One call resolves at most 1000 domains, and `generate_and_check` is rejected above that before any lookup
happens, naming the argument to shrink. Two ways to search a larger space:

- `only_available: true` with `limit: N` stops the sweep once N available domains are found, which makes a
  wide search cheap. This is usually what you want.
- `dry_run: true` returns the exact domain list and count with no network requests, so an agent can price a
  call before making it, or see which TLDs a preset expands to.

Collection also stops at 45 seconds, under the typical client timeout, returning partial results marked
`truncated` rather than failing outright.

### Domains for sale

Both tools accept `check_for_sale: true`, which adds a `for_sale` object to any taken domain that advertises
itself for sale, and `only_for_sale: true` to return just those. See
[Domains For Sale](#domains-for-sale-rfc-10023).

## Installation
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
