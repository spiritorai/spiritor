# Developer Notes

[![Go Reference](https://pkg.go.dev/badge/github.com/spiritorai/spiritor.svg)](https://pkg.go.dev/github.com/spiritorai/spiritor)

## Versioning

I do not yet have a good versioning strategy in place, but I do plan to add proper semver module versions, releases and release notes later.

## Contributions

I will be gladly accepting PR contributions, but at this point it would be best to open an issue and discuss the planned changes beforehand, to make sure your implementation lines up with the intended roadmap.

I take a design-first approach to engineering which means we need to first discuss and agree on the changes before you write the code. I do not yet have a style guide, but all PRs should conform to existing design principles, conventions and architecture. These things can of course be improved upon also, but the new direction needs to be proposed and agreed upon first. See: [Chesterton's Fence](https://medium.com/@mesw1/understanding-chestertons-fence-a-guiding-principle-in-software-engineering-7459e1fb7bf1).

Please ensure that all final PRs are squashed into a single commit rebased against `main` branch, as well as a separate single commit for any vendor changes. Commit messages should be clear and succinct yet descriptive of the changes.

# Quick Reference Commands

Local install:
```sh
GOFLAGS=-mod=vendor && go install spiritor.go
```

Update the vendor directory:
```sh
go mod vendor -u 
```

