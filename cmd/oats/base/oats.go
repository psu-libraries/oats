package base

import "github.com/mehanizm/airtable"

// Oats represents primary application state: configuation an airtable client
type Oats struct {
	*Config
	Production bool // run in production mode or not
	atClient   *airtable.Client
}

// NewOats returns new Oats object from config file
func NewOats(config string) (*Oats, error) {
	cfg, err := loadConfig(config)
	if err != nil {
		return nil, err
	}
	return &Oats{
		Config:   cfg,
		atClient: airtable.NewClient(cfg.Airtable.APIKey),
	}, nil
}

// AirtableBase return reference to appropriate base based on whether oats is
// running in production or testing mode.
func (oats *Oats) AirtableBase() string {
	if oats.Production {
		return oats.Airtable.Base.Production
	}
	return oats.Airtable.Base.Test
}
