# GITHUB_REPO_NAME

[![Build Status](https://github.com/dogmatiq/GITHUB_REPO_NAME/workflows/CI/badge.svg)](https://github.com/dogmatiq/GITHUB_REPO_NAME/actions?workflow=CI)
[![Code Coverage](https://img.shields.io/codecov/c/github/dogmatiq/GITHUB_REPO_NAME/main.svg)](https://codecov.io/github/dogmatiq/GITHUB_REPO_NAME)
[![Latest Version](https://img.shields.io/github/tag/dogmatiq/GITHUB_REPO_NAME.svg?label=semver)](https://semver.org)
[![Documentation](https://img.shields.io/badge/go.dev-reference-007d9c)](https://pkg.go.dev/github.com/dogmatiq/GITHUB_REPO_NAME)
[![Go Report Card](https://goreportcard.com/badge/github.com/dogmatiq/GITHUB_REPO_NAME)](https://goreportcard.com/report/github.com/dogmatiq/GITHUB_REPO_NAME)

This repository is a template for Dogmatiq Go modules.

[Click here](https://github.com/dogmatiq/template/generate) to create a new
repository from this template.

After creating a repository from this template, follow these steps:

- On the settings page (https://github.com/dogmatiq/GITHUB_REPO_NAME/settings):
  - Disable the "Wiki" feature
  - Enable "Automatically delete head branches" under the "Merge button" section
- Replace the string `GITHUB_REPO_NAME` in all files with the actual repo name.
- Run the commands below to rename `.template` files to their proper names:

    ```
    mv .github/dependabot.yml.template .github/dependabot.yml
    mv .gitignore.template .gitignore
    ```

These renames are necessary because:
- Dependabot doe not seem to inspect the commits that are copied from the
  template, and hence does not find the config for new repositories.
- The GitHub template system does not retain the `.gitignore` file from the
  template in the new repository.
