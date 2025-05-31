![tldx logo](https://github.com/brandonyoungdev/tldx/raw/main/assets/tldx-logo.png)

# tldx

![GitHub release (latest by date)](https://img.shields.io/github/v/release/brandonyoungdev/tldx)
![Tests](https://img.shields.io/github/actions/workflow/status/brandonyoungdev/tldx/test.yml?branch=main)
![GitHub](https://img.shields.io/github/license/brandonyoungdev/tldx)
[![Go Report Card](https://goreportcard.com/badge/github.com/brandonyoungdev/tldx)](https://goreportcard.com/report/github.com/brandonyoungdev/tldx)

`tldx` is a fast, developer-first CLI tool for **researching available domains** across multiple TLDs with intelligent permutations.

Use it to quickly explore domain name ideas for your next product, startup, or side project — without waiting or guessing.

---

## ⚡ Features

- 🔍 Smart keyword-based domain permutations (prefixes, suffixes, TLDs)
- 🚀 Fast and concurrent WHOIS availability checks
- 📤 Streams results as they're found
- 📏 Optional filtering by domain length
- 🧠 Great for technical founders, indie hackers, and naming brainstorms

---

## 🛠️ Usage

```bash
Usage:
  tldx [keywords] [flags]
  tldx [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print the version

Flags:
  -h, --help                    help for tldx
  -m, --max-domain-length int   Maximum length of domain name (default 64)
  -a, --only-available          Show only available domains
  -p, --prefixes strings        Prefixes to add (e.g. get,my,use)
      --show-stats              Show statistics
  -s, --suffixes strings        Suffixes to add (e.g. ify,ly)
  -t, --tlds strings            TLDs to check (e.g. com,io,ai)
  -v, --verbose                 Show verbose output
```

## 🔗 Examples

### Checking Domain Availability

#### `tldx google` 
```bash
❌ google.com is not available
```

#### `tldx google youtube reddit`
 
```bash
❌ google.com is not available
```

#### `tldx google youtube reddit`
```bash
  ❌ reddit.com is not available
  ❌ google.com is not available
  ❌ youtube.com is not available
```

### Permutations

#### `tldx google --prefixes get,my --suffixes ly,hub --tlds com,io,ai`

This permutates the keywords with the specified prefixes, suffixes, and TLDs, checking for availability:
```bash
  ✔️  mygooglely.com is available
  ✔️  getgooglely.ai is available
  ✔️  getgooglehub.io is available
  ✔️  mygooglehub.io is available
  ❌ mygoogle.ai is not available
```

### Show Only Available Domains

#### `tldx google reddit facebook -p get,my -s ly,hub -t com,io,ai --only-available`

```bash
  ✔️  getgooglely.ai is available
  ✔️  getgooglehub.ai is available
  ✔️  mygooglely.io is available
  ✔️  getreddithub.com is available
  ✔️  getreddit.ai is available
  ✔️  googlely.ai is available
  ✔️  getredditly.com is available
  ✔️  getreddithub.io is available
  ✔️  getredditly.io is available
  ✔️  getredditly.ai is available
  ✔️  myredditly.io is available
  ✔️  myreddithub.io is available
  ✔️  myreddithub.com is available
  ✔️  getfacebookhub.com is available
  ✔️  myfacebookly.com is available
  ✔️  myfacebookhub.ai is available
  ✔️  getfacebookly.ai is available
  ✔️  facebookly.io is available
```


