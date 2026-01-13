# Releasing a New Version

## Versioning
This project follows [Semantic Versioning](https://semver.org/):
- MAJOR version for incompatible changes
- MINOR version for new functionality (backwards compatible)
- PATCH version for bug fixes (backwards compatible)

## Creating a Release

1. Ensure all changes are merged to main/master
2. Update CHANGELOG.md (if exists)
3. Create and push a new tag:

```bash
# For a regular release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# For a pre-release
git tag -a v1.0.0-rc1 -m "Release candidate 1 for v1.0.0"
git push origin v1.0.0-rc1
```

4. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Generate checksums
   - Create a GitHub Release with auto-generated notes

## Verifying a Release

Users can verify downloads using checksums:
```bash
# On Linux/macOS
sha256sum -c checksums.txt

# On Windows (PowerShell)
Get-FileHash my-own-vpn-windows-amd64.exe -Algorithm SHA256
```

## Release Artifacts

Each release includes:
- `my-own-vpn-windows-amd64.exe` - Windows binary
- `my-own-vpn-darwin-amd64` - macOS Intel binary
- `my-own-vpn-darwin-arm64` - macOS Apple Silicon binary
- `checksums.txt` - SHA256 checksums for all binaries
