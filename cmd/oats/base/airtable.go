package base

import (
	"fmt"

	"github.com/mehanizm/airtable"
)

const (
	COL_ID        = "ID"
	COL_AI_ID     = "AI_ID"
	COL_STATUS    = "Status"
	COL_DOI       = "DOI"
	COL_DOI_CONF  = "DOI_Confirmed"
	COL_OA_STATUS = "OA_status"
	COL_OA_LINK   = "OA_Link"
)

type atIndex map[string][]*airtable.Record

func (cmd *Oats) GetRecord(tableName string, id string) (*airtable.Record, error) {
	table := cmd.atClient.GetTable(cmd.AirtableBase(), tableName)
	return table.GetRecord(id)
}

// downloads all records in the Airtable
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

// func (cmd *Oats) getRecords(tableName string) ([]*airtable.Record, error) {
// 	return cmd.getRecordsFilterFields(tableName, ``, nil)
// }

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

// patchRecords does PUT on records in an Airtable.
// A PUT request will perform a destructive update and clear all unspecified cell values.
// func (cmd *Oats) putRecords(tableName string, records []*airtable.Record) ([]*airtable.Record, error) {
// 	table := cmd.atClient.GetTable(cmd.config.Airtable.BaseID, tableName)
// 	return cmd._commitRecords(records, table.UpdateRecords)
// }

// postRecords does POST for records in an Airtable.
// A POST request will create new rows for each record.
func (cmd *Oats) PostRecords(tableName string, records []*airtable.Record) ([]*airtable.Record, error) {
	table := cmd.atClient.GetTable(cmd.AirtableBase(), tableName)
	return cmd._commitRecords(records, table.AddRecords)
}

// postRecord does POST for a seing record in an Airtable.
// A POST request will create new rows for each record.
// func (cmd *Oats) postRecord(tableName string, record *airtable.Record) (*airtable.Record, error) {
// 	table := cmd.atClient.GetTable(cmd.config.Airtable.BaseID, tableName)
// 	resp, err := cmd._commitRecords([]*airtable.Record{record}, table.AddRecords)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(resp) != 1 {
// 		return nil, errors.New(`no record received`)
// 	}
// 	return resp[0], nil
// }

// type _atRecords []*airtable.Record

// func (atr _atRecords) recordIDMap(idField string) (idTable, error) {
// 	var recs []row
// 	for i := range atr {
// 		recs = append(recs, row(atr[i].Fields))

// 	}
// 	return table(recs).asIDTable(idField)
// }

// create
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
