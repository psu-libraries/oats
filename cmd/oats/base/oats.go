package base

import "github.com/mehanizm/airtable"

type Oats struct {
	*Config
	Production bool // run in production mode or not
	atClient   *airtable.Client
}

func (oats *Oats) AirtableBase() string {
	if oats.Production {
		return oats.Airtable.Base.Production
	}
	return oats.Airtable.Base.Test
}

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
