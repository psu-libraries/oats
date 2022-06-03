# `oats` command

This is a commandline tool for managing PSU Libraries' OA workflow. It's mostly
used to automate updates to a project-specific Airtable Database using
information from various APIs (RMD, CrossRef, Unpaywall, and Open Access
Button).

## Overview:

- The `cmd` package defines each subcommand. 
- The `base` package has exports shared by all subcommands. 
- `config-example.yml`: config template (copty to `config.yml`)