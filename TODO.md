# Turbo Mocker

## Features

- fast mock server that matches expectations and can act as a reverse proxy
- web app
- desktop app?
- tui app?
- test runner against expectations using snapshots
- cloud sync of tests / mocks
- mock context sharing
- github integration where tests do not need to run again based on cryptography

## Expectations

- [o] matchers:
     - [x] URL
     - [x] body
     - [x] headers
     - [ ] query params
- [o] API to insert expectations
- [ ] if client sends `Proxy-Connection` header, use as proxy server, otherwise fail on unmatched expectation 
- [ ] raise server with default expectations

## Dashboard

- check matched / unmatched expectations
- define mocks

## Performance

- focus on performance

