# rmdcsvfixer

This is a program that normalizes CSV files from RMD by Activity Insight ID. It can be used to extract values relating to Activity Insight IDs from CSV exports from RMD. The output can be used as input for for the `oats merge` command.

## Example

An input file might look like this (see `testdata/input.csv`):

| Id    | DOI                                        | Activity insight postprint status | Id [Imports]       | Source [Imports]                       | Source identifier [Imports]                                    |
|-------|--------------------------------------------|-----------------------------------|--------------------|----------------------------------------|----------------------------------------------------------------|
| 1612  | https://doi.org/10.1080/14747730802500398  | Deposited to ScholarSphere        | 1612,183261        | Pure,Activity Insight                  | 2f9ddc57-f66e-4f69-b012-5c4918dd2602,41074354177               |
| 8871  | https://doi.org/10.1016/j.acra.2014.07.013 | Deposited to ScholarSphere        | 265485,261497,8871 | Activity Insight,Activity Insight,Pure | 195361005568,195357255680,9f678183-7dd0-465d-9001-b94715acf7c4 |
| 59876 | https://doi.org/10.1177/0886260518757226   | Deposited to ScholarSphere        | 59876,293407       | Pure,Activity Insight                  | abad6282-229b-4738-b6bc-85270b8e1eb6,135865448448              |

```sh
# Example: Extract value in "Activity insight postprint status" for each Activity Insight ID in the input, 
# also rename the column "RMD_activity_insight_postprint_status" in the output: 
go run main.go -c "Activity insight postprint status" -r "RMD_activity_insight_postprint_status" testdata/input.csv
ID,RMD_activity_insight_postprint_status
41074354177,Deposited to ScholarSphere
195361005568,Deposited to ScholarSphere
195357255680,Deposited to ScholarSphere
```