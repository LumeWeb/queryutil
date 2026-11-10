## 0.3.27 (2026-11-10)

### Features

- cap filter tree width

### Fixes

- reject malformed JSON path bytes
- reject DEL in JSON path fields
- isolate node budgets per parse

## 0.3.26 (2026-11-10)

### Fixes

- hide policy list server errors

## 0.3.25 (2026-11-10)

### Features

- add policy-aware list boundaries

### Fixes

- reject malformed query escapes

## 0.3.24 (2026-11-10)

### Fixes

- propagate filter builder errors to GORM
- normalize q equality as contains
- avoid poisoning shared GORM handles
- preserve search query text
- keep search terms raw

## 0.3.23 (2026-11-09)

### Features

- add comparable operator preset

## 0.3.22 (2026-11-09)

### Features

- expose scalar operator policy helpers

## 0.3.21 (2026-11-09)

### Fixes

- preserve configured query field text
- reject numeric ops on preserved text

## 0.3.20 (2026-11-09)

### Features

- add opt-in query policy

### Fixes

- reject unsupported policy operations

## 0.3.19 (2026-08-29)

### Fixes

- disambiguate JSON paths vs columns
- resolve model tables and remove builder race
- resolve model table without mutating query
- correct JSON filtering and parser order
