# release-please (Go)

A Go port of the core of [googleapis/release-please](https://github.com/googleapis/release-please),
which is open source under Apache-2.0 (see [NOTICE](NOTICE)). Standard library only.

It works the same way as the original:

1. **`release-pr`** reads the commits since the last release from the GitHub API and parses them as
   [Conventional Commits](https://www.conventionalcommits.org). It then works out the next version
   and updates the changelog and version files. Finally it force-pushes those changes to
   `release-please--branches--<branch>` and opens or updates a PR titled
   `chore(<branch>): release X.Y.Z` with the label `autorelease: pending`.
2. When that PR is merged, **`github-release`** creates the tag and GitHub release from the merge commit,
   using the PR's notes. It then relabels the PR `autorelease: tagged`.
3. **`all`** runs both commands, in that order, like the upstream GitHub Action.

## Release types

`--release-type auto` (the default) picks one from the files on the target branch:

| Type | Detected by | Files updated |
|---|---|---|
| `go` | `go.mod` | `CHANGELOG.md` only. Go versions *are* the `vX.Y.Z` tags, so there is no version file. |
| `java-gradle` | `build.gradle(.kts)`, `settings.gradle(.kts)` or `gradle.properties` | `CHANGELOG.md`, plus `version=` in `gradle.properties` and `version = "…"` in the root `build.gradle` / `build.gradle.kts`, whichever exist. A `-SNAPSHOT` suffix is replaced by the release version. |
| `simple` | anything else | `CHANGELOG.md`, `version.txt` |

All types also update any `--extra-files`, for example a Go `const Version` or a subproject's `build.gradle`.

**Go v2+:** a Go module must have `/vN` at the end of its module path from v2 onwards
([docs](https://go.dev/doc/modules/major-version)). When a breaking change would release v2 or later and
`go.mod` lacks that suffix, the release PR shows a warning so you can fix `go.mod` before merging. To stay
on v0 while the API is unstable, set `--initial-version=0.1.0 --bump-minor-pre-major`.

## Token: no custom secret needed

The workflow's built-in `GITHUB_TOKEN` is enough. The action uses it by default, with two one-time setup steps:

1. Give the job these `permissions`: `contents: write` (branch, tag, release), `pull-requests: write`, and
   `issues: write` (PR labels).
2. Turn on **Settings → Actions → General → Workflow permissions → "Allow GitHub Actions to create and
   approve pull requests"**. Without it, opening the PR fails with a 403.

**The one catch:** GitHub doesn't start new workflow runs from events caused by `GITHUB_TOKEN`, which affects two things:

- **CI doesn't run automatically on the release PR.** That's usually fine, because the PR only changes the
  changelog and version, and `main` has already passed CI. But if branch protection *requires* status checks,
  the PR can't merge until they run. Either add `workflow_dispatch` to CI and run it on the release branch,
  or close and reopen the PR yourself.
- **Workflows on `release: published` or tag pushes won't fire.** Do the publishing in the same workflow
  instead, gated on the `release_created` output (see the examples below).

Use a PAT or GitHub App token only if those limits get in the way: `with: { token: ${{ secrets.X }} }`.

## Example workflows

### Go

```yaml
name: release
on:
  push:
    branches: [master]
permissions:
  contents: write
  pull-requests: write
  issues: write
jobs:
  release-please:
    runs-on: ubuntu-latest
    outputs:
      release_created: ${{ steps.rp.outputs.release_created }}
      tag_name: ${{ steps.rp.outputs.tag_name }}
    steps:
      - id: rp
        uses: willvk/go-play/releasePlease@master
        # with:
        #   args: --extra-files=internal/version/version.go

  # Optional: build binaries once a release is tagged. Pushing the tag is enough
  # for `go get module@vX.Y.Z` on its own.
  goreleaser:
    needs: release-please
    if: needs.release-please.outputs.release_created == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ github.token }}
```

### Java (Gradle)

```yaml
name: release
on:
  push:
    branches: [master]
permissions:
  contents: write
  pull-requests: write
  issues: write
  packages: write # only for publishing to GitHub Packages
jobs:
  release-please:
    runs-on: ubuntu-latest
    outputs:
      release_created: ${{ steps.rp.outputs.release_created }}
    steps:
      - id: rp
        uses: willvk/go-play/releasePlease@master

  publish:
    needs: release-please
    if: needs.release-please.outputs.release_created == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4 # the merge commit, so the release version is in place
      - uses: actions/setup-java@v4
        with:
          distribution: temurin
          java-version: "21"
      - uses: gradle/actions/setup-gradle@v4
      - run: ./gradlew build publish
        env:
          GITHUB_TOKEN: ${{ github.token }}
```

Action inputs: `release-type` (default `auto`), `command` (default `all`), `token` (default `github.token`),
and `args` for any CLI flag below. Outputs: `release_created`, `tag_name`, `version`, `html_url`, `pr`, `pr_url`.

## CLI

```sh
go run . --repo owner/name --dry-run release-pr          # preview; a token is optional for public repos
GITHUB_TOKEN=... go run . --repo owner/name all
```

| Flag | Default | |
|---|---|---|
| `--release-type` | `auto` | `simple`, `go`, `java-gradle` |
| `--branch` | repo default | target branch |
| `--tag-prefix` | `v` | |
| `--initial-version` | `1.0.0` | used when there are no releases yet |
| `--changelog-path` | `CHANGELOG.md` | empty to disable |
| `--version-file` | `version.txt` for `simple` | plain-text file holding just the version |
| `--extra-files` | | comma-separated; see annotations below |
| `--bump-minor-pre-major` | false | before 1.0, breaking changes bump minor |
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

## Not ported (yet)

- other language strategies (node, python, rust, maven, …)
- Java `-SNAPSHOT` follow-up PRs after a release (upstream's `java` strategy). Here the release simply replaces the snapshot version.
- monorepos (`release-please-config.json` and `.release-please-manifest.json`), including Go modules in subdirectories that need `subdir/vX.Y.Z` tags
- prerelease versioning, custom changelog sections, and splitting squash commits that contain several conventional messages
