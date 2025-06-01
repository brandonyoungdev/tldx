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
```


![tldx demo](https://github.com/brandonyoungdev/tldx/raw/main/tapes/demo.gif)

## ⚡ Features

- 🔍 Smart keyword-based domain permutations (prefixes, suffixes, TLDs)
- 🚀 Fast and concurrent WHOIS availability checks
- 📤 Streams results as they're found
- 📏 Optional filtering by domain length
- 🧠 Great for technical founders, indie hackers, and naming brainstorms


## 🛠️ Usage

```sh
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
```sh
❌ google.com is not available
```

#### `tldx google youtube reddit`
 
```sh
❌ google.com is not available
```

#### `tldx google youtube reddit`
```sh
  ❌ reddit.com is not available
  ❌ google.com is not available
  ❌ youtube.com is not available
```

### Permutations

#### `tldx google --prefixes get,my --suffixes ly,hub --tlds com,io,ai`

This permutates the keywords with the specified prefixes, suffixes, and TLDs, checking for availability:
```sh
  ✔️  mygooglely.com is available
  ✔️  getgooglely.ai is available
  ❌ mygoogle.ai is not available
  ...
```

### Show Only Available Domains

#### `tldx google reddit facebook -p get,my -s ly,hub -t com,io,ai --only-available`

```sh
  ✔️  getgooglely.ai is available
  ✔️  getreddithub.com is available
  ✔️  getreddit.ai is available
  ✔️  googlely.ai is available
  ✔️  getredditly.com is available
  ✔️  facebookly.io is available
  ...
```

## 📦 Installation
#### macOS (Homebrew)
```sh
brew install brandonyoungdev/tldx/tldx
```
or
```sh
brew tap brandonyoungdev/tldx
brew install tldx
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

Extract the binary and move it to a directory in your `$PATH`:

```sh
# Example for Linux/macOS
tar -xzf tldx_<version>_<os>_<arch>.tar.gz
mv tldx /usr/local/bin/
```

#### Go (Install from Source)
```sh
go install github.com/brandonyoungdev/tldx@latest
```
