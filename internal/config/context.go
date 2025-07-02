package config

import "time"

type TldxContext struct {
	Config *TldxConfigOptions
}

type TldxConfigOptions struct {
	TLDs             []string
	Prefixes         []string
	TLDPreset        string
	Suffixes         []string
	MaxDomainLength  int
	Verbose          bool
	OnlyAvailable    bool
	ShowStats        bool
	OutputFormat     string
	NoColor          bool
	MaxRetries       int
	InitialBackoff   time.Duration
	BackoffFactor    float64
	JitterFraction   float64
	ContextTimeout   time.Duration
	ConcurrencyLimit int
}

func NewTldxContext() *TldxContext {
	return &TldxContext{
		Config: &TldxConfigOptions{
			MaxRetries:       3,
			InitialBackoff:   500 * time.Millisecond,
			BackoffFactor:    5.0,
			JitterFraction:   0.7, // +/-70% randomness
			ContextTimeout:   15 * time.Second,
			ConcurrencyLimit: 15,
		},
	}
}
