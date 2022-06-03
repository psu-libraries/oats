package base

// This file includes utility functions for Airtable

import (
	"fmt"

	"github.com/mehanizm/airtable"
)

// GetRecord returns an airtable record associated with the given table name and
// id.
func (cmd *Oats) GetRecord(tableName string, id string) (*airtable.Record, error) {
	table := cmd.atClient.GetTable(cmd.AirtableBase(), tableName)
	return table.GetRecord(id)
}

// GetRecordsFilterFields returns slice of airtable records in the table based
// on the specified filter. The filter string should use the Airtable formula
// syntax. The fields parameter can be used to specify columns in the returned
// records
func (cmd *Oats) GetRecordsFilterFields(tableName string, filter string, fields []string) ([]*airtable.Record, error) {
	var (
		offset  string
		allRecs []*airtable.Record
	)
	table := cmd.atClient.GetTable(cmd.AirtableBase(), tableName)
	cfg := table.GetRecords()
	if filter != "" {
		cfg.WithFilterFormula(filter)
	}
	if len(fields) > 0 {
		cfg.ReturnFields(fields...)
	}
	for {
		if offset != "" {
			cfg.WithOffset(offset)
		}
		results, err := cfg.Do()
		if err != nil {
			return nil, err
		}
		allRecs = append(allRecs, results.Records...)
		if results.Offset == "" {
			break
		}
		offset = results.Offset
	}
	return allRecs, nil
}

// // _commitFunc is common signature of airtable post/put/patch functions
type _commitFunc func(*airtable.Records) (*airtable.Records, error)

// _commitRecords abstracts post/put/patch functions
func (cmd *Oats) _commitRecords(recs []*airtable.Record, f _commitFunc) ([]*airtable.Record, error) {
	var responses []*airtable.Record
	for i := 0; i < len(recs); i += 10 {
		end := i + 10
		if end > len(recs) {
			end = len(recs)
		}
		resp, err := f(&airtable.Records{
			Records: recs[i:end],
		})
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp.Records...)
	}
	return responses, nil
}

// PostRecords does POST for records in an Airtable. A POST request will create
// new rows for each record.
func (cmd *Oats) PostRecords(tableName string, records []*airtable.Record) ([]*airtable.Record, error) {
	table := cmd.atClient.GetTable(cmd.AirtableBase(), tableName)
	return cmd._commitRecords(records, table.AddRecords)
}

type atIndex map[string][]*airtable.Record

// IndexAirtableRecords buils an atIndex (map of airtable records) from recs
// indexed on the specified field. The record values associated with the
// specified field must be resolvable to strings.
func IndexAirtableRecords(recs []*airtable.Record, field string) (atIndex, error) {
	ret := make(atIndex)
	for _, rec := range recs {
		var k string
		if field == "" {
			k = rec.ID
		} else {
			val := rec.Fields[field]
			if val == nil {
				return nil, fmt.Errorf(`Encountered %s=nil while indexing on that column`, field)
			}
			switch val := val.(type) {
			case string:
				k = val
			case []string:
				if len(val) == 1 {
					k = val[0]
				}
			case []interface{}:
				if len(val) == 1 {
					if s, ok := val[0].(string); ok {
						k = s
					}
				}
			}
		}
		if k == `` {
			return nil, fmt.Errorf(`could not index airtable on field: %s`, field)
		}
		ret[k] = append(ret[k], rec)
	}
	return ret, nil
}
