# Sensu Google Chat Handler

[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/grant-singleton-nz/sensu-simple-google-chat-handler)
[![Go Test](https://github.com/grant-singleton-nz/sensu-simple-google-chat-handler/workflows/Go%20Test/badge.svg)](https://github.com/grant-singleton-nz/sensu-simple-google-chat-handler/actions?query=workflow%3A%22Go+Test%22)
[![goreleaser](https://github.com/grant-singleton-nz/sensu-simple-google-chat-handler/workflows/goreleaser/badge.svg)](https://github.com/grant-singleton-nz/sensu-simple-google-chat-handler/actions?query=workflow%3Agoreleaser)

## Table of Contents
- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Handler definition](#handler-definition)
  - [Environment variables](#environment-variables)
- [Installation from source](#installation-from-source)
- [Contributing](#contributing)

## Overview

The Sensu Google Chat Handler is a [Sensu Handler][1] that sends notifications to a
Google Chat space via webhooks. Messages include a link to the event in the Sensu dashboard
and organize messages by entity in Google Chat threads.

## Files

- `main.go`: The main Go file that implements the handler functionality
- `go.mod`: Dependency management file for Go modules
- `.goreleaser.yml`: Configuration for building and releasing the handler via GoReleaser
- `.github/workflows/`: GitHub Actions workflow files for testing and releasing the handler

## Usage examples

### Help output

```
The Sensu Google Chat Handler is a Sensu Handler that sends alert notifications to Google Chat

Usage:
  sensu-simple-google-chat-handler [flags]
  sensu-simple-google-chat-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -d, --dashboard string   URL prefix to dashboard with namespace
  -h, --help               help for sensu-simple-google-chat-handler
  -w, --webhook string     The webhook URL to post the message to

Use "sensu-simple-google-chat-handler [command] --help" for more information about a command.
```

## Configuration

### Asset registration

[Sensu Assets][2] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! You can use the following command to add the asset:

```
sensuctl asset add grant-singleton-nz/sensu-simple-google-chat-handler
```

You can also find the asset on the [Bonsai Asset Index][3].

### Handler definition

```yml
---
type: Handler
api_version: core/v2
metadata:
  name: google-chat
  namespace: default
spec:
  command: sensu-simple-google-chat-handler --webhook $GOOGLE_CHAT_WEBHOOK --dashboard $SENSU_DASHBOARD
  type: pipe
  runtime_assets:
    - grant-singleton-nz/sensu-simple-google-chat-handler
  secrets:
    - name: GOOGLE_CHAT_WEBHOOK
      secret: google-chat-webhook
  env_vars:
    - SENSU_DASHBOARD=https://sensu.example.com
```

### Environment variables

|Argument    |Environment Variable |Default|Required|Description                                  |
|------------|---------------------|-------|--------|---------------------------------------------|
|--webhook   |GOOGLE_CHAT_WEBHOOK  |       |true    |The webhook URL to post the message to       |
|--dashboard |SENSU_DASHBOARD      |       |true    |URL prefix to dashboard with namespace       |

## Installation from source

### Download

Download the latest version of the sensu-simple-google-chat-handler from [releases][4],
or create an executable from this source.

For Linux systems with `go` installed:

```shell
go install github.com/grant-singleton-nz/sensu-simple-google-chat-handler@latest
```

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://docs.sensu.io/sensu-go/latest/reference/handlers/
[2]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[3]: https://bonsai.sensu.io/assets/grant-singleton-nz/sensu-simple-google-chat-handler
[4]: https://github.com/grant-singleton-nz/sensu-simple-google-chat-handler/releases