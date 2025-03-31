# Contributing to MCP Subfinder Server

Thank you for considering contributing to MCP Subfinder Server! This document outlines the process for contributing to this project.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## How Can I Contribute?

### Reporting Bugs

This section guides you through submitting a bug report. Following these guidelines helps maintainers understand your report, reproduce the behavior, and find related reports.

* Use the bug report template when creating an issue
* Include detailed steps to reproduce the problem
* Include relevant logs and JSON payloads
* Describe the behavior you observed and what you expected to see

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion, including completely new features and minor improvements to existing functionality.

* Use the feature request template when creating an issue
* Provide a clear and detailed explanation of the feature
* Include examples of how the feature would work (JSON payloads, etc.)
* Explain why this enhancement would be useful

### Pull Requests

* Fill out the pull request template completely
* Include relevant issue numbers in the PR description
* Update documentation as needed
* Add tests for new features
* Follow the code style of the project
* Write meaningful commit messages

## Development Setup

1. Fork and clone the repository
2. Install Go (1.20+)
3. Run `go mod download` to install dependencies
4. Make your changes
5. Run tests: `go test ./...`
6. Build: `go build -o mcp-subfinder-server main.go`

## Style Guidelines

### Git Commit Messages

* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less
* Reference issues and pull requests after the first line

### Go Style Guide

* Follow standard Go conventions and [Effective Go](https://golang.org/doc/effective_go)
* Use `gofmt` to format your code
* Comment public functions and types
* Keep functions focused on a single responsibility
