package domain

import "time"

type ConfigOptions struct {
	TLDs            []string
	Prefixes        []string
	TLDPreset       string
	Suffixes        []string
	MaxDomainLength int
	Verbose         bool
	OnlyAvailable   bool
	ShowStats       bool
}

const (
	maxRetries       = 3
	initialBackoff   = 500 * time.Millisecond
	backoffFactor    = 5.0
	jitterFraction   = 0.7 // +/-70% randomness
	contextTimeout   = 15 * time.Second
	concurrencyLimit = 20
)

var Config = ConfigOptions{}
