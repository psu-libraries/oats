# `oats` command

This is a commandline tool for managing PSU Libraries' OA workflow. It's mostly
used to automate updates to a project-specific Airtable Database using
information from various APIs (RMD, CrossRef, Unpaywall, and Open Access
Button).

Organization:
- The `cmd` package defines each subcommand. 
- The `base` package has exports shared by all subcommands. 

```
OA Tools: a collection of programs for managing the OA workflow

Usage:
  oats [command]

Available Commands:
  deposit     Deposit to ScholarSphere
  dois        Confirms unconfirmed DOIs in Airtable using CrossRef and RMD
  help        Help about any command
  import      Import Activity Insight Records to Airtable
  merge       Updates Tasks on Airtable with data from a csv file
  oastatus    Updates OA_status in Airtable using Unpaywall API
  permissions Updates deposit permissions in Airtable using Open Access Button's Permissions API
  rmdupdated  Updates Tasks' RMD_Updated column in Airtable
  sslink      Find ScholarSphere Links for Tasks in Airtable
  tasks       Creates new Tasks in Airtable

Flags:
  -c, --config string   config file (default "config.yml")
  -h, --help            help for oats
  -p, --production      run in production mode

Use "oats [command] --help" for more information about a command.
```

## Installation

If you already have [Go installed](https://go.dev/doc/install) (v1.17 or later):
```sh
# install with go
go install github.com/psu-libraries/oats/cmd/oats@latest
# run it with:
~/go/bin/oats --help
```

Alternatively, you can directly [download unsigned binaries](https://github.com/psu-libraries/oats/-/releases). A caveat of this approach is that your operating system may refuse to run unsigned binaries.  

## Configuration

The `oats` command requires a configuration file. By default it looks for a file called `config.yml` in the working directory where `oats` is run.

```yml
## config.yml template 

airtable:
  base:
    # Production Airtable BasedID
    production: "fixme"
    # Testing Airtable BasedID
    test: "fixme"
  # Your Airtable API Key
  apikey: "fixme"
  # Tasks Table - shouldn't need to change this
  tasks: "Tasks"
  # Activity Insight Table - shouldn't need to change this
  activity_insight: "Activity Insight"

unpaywall:
  # This email is sent with request to Unpaywall API
  email: "fixme"

openaccessbutton:
  # API key for open access button: https://openaccessbutton.org/account?next=/api
  key: "fixme"

scholarsphere:
  apikey: "fixme"
  production: "fixme"
  test: "fixme"

rmdb:
 apikey: "fixme"
 production: "fixme"
 test: "fixme"

# Absolute path to directory to search for files (used by deposit)
article_path: "fixme"
```


## Additional Notes

### Depositing Multiple IDs

The deposit command only deposits one item at a time. To deposit many IDs automatically, you can do the following:

1. Create a text file with one ID on each line *and an additional empty line at the end of the file*. (Without the terminating newline, the last ID won't be deposited.):

```
128153 
128829
101241

```

2. If the file is called `deposit-ids.txt`, you can deposit each ID in the file with one of the commands:
```sh
# run in test mode:
while read id; do oats deposit $id; done < deposit-ids.txt
# run in production:
while read id; do oats -p deposit $id; done < deposit-ids.txt
```
