# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## TODOs

- analyze the Makefile's logic for creating the version
- initiate a new go module for this repo
- reimplement the logic from the makefile in go using appropriate library in pure go (e.g. github.com/go-git/go-git)
- create a simple cli tool `gitversion`, which prints the current version.

### Next Steps

- think about customization but stick to convention over configuration!
- list usefull flags with their respective usage
- think about providing customization also via a yaml file, e.g. gitversion.yaml, which may be references automatically if present
- the tool should be a no-brainer to use in ci or manual build processes

## Project Overview

This is a Git versioning utility project (gitversion). The repository is in early initialization stage.

## Current State

The repository currently contains:
- A Makefile that appears to be a template from another project (commerce-data-service) and needs to be replaced or removed
- MIT License (Copyright 2025 Steve)

## Important Notes

**The existing Makefile is not appropriate for this repository.** It contains commands and build instructions for a Go-based commerce-data-service with Docker, Redis, Service Bus, and Terraform dependencies that don't exist in this project. When implementing the actual gitversion tool, the Makefile should be rewritten to match the actual project structure and requirements.

## Development Setup

Once the project structure is established, this section should be updated with:
- Build commands
- Test commands
- Installation instructions
- Usage examples
