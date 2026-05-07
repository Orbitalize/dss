# Fork

This file aggregates the changes specific to this fork.

Its goal is to facilitate the rebase process with the upstream repository.

## Changelog

### [scd] Add flag for relaxed constraints support

Relax time bound validation on constraints so start/end times are not required

### [surveillance] Add new surveillance core-service

Add surveillance core-service as defined in [cis-interfaces](https://github.com/skyguide-ansp/cis-interfaces).

This core-service works as remote-id and reuses some of its internals (store) to discover air traffic surveillance providers instead of UAS telemetry.

### [surveillance] add surveillance datastore

Add surveillance datastore.

### [surveillance] add surveillance app

Add surveillance application and server implementation.

### [surveillance] add db-manager evict

Add surveillance support to db-manager evict.
