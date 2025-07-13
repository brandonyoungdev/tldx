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
	InputFile        string
	MaxDomainLength  int
	Verbose          bool
	OnlyAvailable    bool
	ShowStats        bool
	OutputFormat     string
	NoColor          bool
	MaxRetries       int
	InitialBackoff   time.Duration
	MaxBackoff       time.Duration
	BackoffFactor    float64
	ContextTimeout   time.Duration
	ConcurrencyLimit int
}

func NewTldxContext() *TldxContext {
	return &TldxContext{
		Config: &TldxConfigOptions{
			MaxRetries:       3,
			InitialBackoff:   1500 * time.Millisecond,
			MaxBackoff:       5 * time.Second,
			BackoffFactor:    1.5,
			ContextTimeout:   15 * time.Second,
			ConcurrencyLimit: 15,
		},
	}
}
