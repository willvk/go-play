# release-please (Go)

A Go port of the core of [googleapis/release-please](https://github.com/googleapis/release-please),
which is open source under Apache-2.0 (see [NOTICE](NOTICE)). Standard library only.

It works the same way as the original:

1. **`release-pr`** reads the commits since the last release from the GitHub API and parses them as
   [Conventional Commits](https://www.conventionalcommits.org). It then works out the next version,
   updates `CHANGELOG.md`, `version.txt` and any annotated files, and force-pushes them to
   `release-please--branches--<branch>`. Finally it opens or updates a PR titled
   `chore(<branch>): release X.Y.Z` with the label `autorelease: pending`.
2. When that PR is merged, **`github-release`** creates the tag and GitHub release from the merge commit,
   using the PR's notes. It then relabels the PR `autorelease: tagged`.
3. **`all`** runs both commands, in that order, like the upstream GitHub Action.

## Usage

```sh
go run . --repo owner/name --dry-run release-pr          # preview; a token is optional for public repos
GITHUB_TOKEN=... go run . --repo owner/name all
```

| Flag | Default | |
|---|---|---|
| `--branch` | repo default | target branch |
| `--tag-prefix` | `v` | |
| `--initial-version` | `1.0.0` | used when there are no releases yet |
| `--changelog-path` | `CHANGELOG.md` | empty to disable |
| `--version-file` | `version.txt` | empty to disable |
| `--extra-files` | | comma-separated; see annotations below |
| `--bump-minor-pre-major` | false | before 1.0, `feat!` bumps minor |
| `--bump-patch-for-minor-pre-major` | false | before 1.0, `feat` bumps patch |
| `--dry-run` | false | no writes |

### Versioning

`fix`/`perf`/etc. bump the patch version, `feat` bumps minor, and `!` or a `BREAKING CHANGE:` footer bumps
major. A `Release-As: x.y.z` footer forces the version. Commits that only touch hidden sections
(`chore`, `docs`, `ci`, …) don't trigger a release.

### Annotations for `--extra-files`

```go
const Version = "1.2.3" // x-release-please-version

// x-release-please-start-version
image: app:1.2.3
// x-release-please-end
```

### GitHub Action

```yaml
on:
  push:
    branches: [main]
permissions:
  contents: write
  pull-requests: write
jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: willvk/go-play/releasePlease@main
        with:
          args: --extra-files=version.go
```

## Not ported (yet)

The upstream project is much larger. This port leaves out:

- language strategies (node, python, rust, java, …) and their manifest updaters
- monorepos (`release-please-config.json` and `.release-please-manifest.json`) and plugins
- prerelease versioning, custom changelog sections from config, and changelog types other than the default
- splitting squash-merged commits that contain several conventional messages, and `release-please` comment commands
