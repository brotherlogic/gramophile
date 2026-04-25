# Gramophile

Gramophile is a sophisticated system for managing and organizing your vinyl record collection, deeply integrated with the Discogs API.

## Features

- **Collection Organization**: Support for complex organization rules, snapshots, and moving records between virtual folders.
- **Deep Discogs Integration**: Comprehensive syncing of collections, wantlists, and marketplace data.
- **Stateful History**: Tracks historical changes to records and wantlists.
- **Sales Management**: Tools for tracking sales and updating prices based on marketplace data.
- **Advanced Wantlists**: Custom logic for managing wants with historical tracking.

## Components

- **Prober**: Validates the login process and ensures system stability.
- **Background Runner**: Manages long-running tasks and eventual consistency with Discogs.
- **Database**: Custom persistence layer using Protobuf messages.

## Development

Gramophile is built with Go.

**Prerequisites:**
- Go 1.26.2
- Protobuf Compiler

**Build:**
Run `./build.sh` to generate Protobuf code.
