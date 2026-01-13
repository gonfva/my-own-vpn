# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please follow these steps:

1. **Do NOT** create a public GitHub issue
2. Use GitHub's private vulnerability reporting feature (Security tab > Report a vulnerability)
3. Provide detailed information about the vulnerability
4. Allow reasonable time for us to address the issue before public disclosure

## Security Measures

This project implements the following security measures:

- **Dependency Scanning**: Automated scanning for known vulnerabilities in dependencies using govulncheck
- **Static Analysis**: Code is scanned for security issues using gosec
- **Filesystem Scanning**: Trivy scans for additional vulnerabilities and misconfigurations
- **Credential Storage**: Sensitive data is stored in OS keychain or encrypted files
- **No Hardcoded Secrets**: All credentials are provided by users at runtime
- **Minimal Permissions**: Cloud resources use least-privilege security groups

## Credential Security

- Cloud provider credentials are stored in OS keychain where available
- Fallback storage uses encryption (NaCl secretbox)
- Credentials are never logged
- WireGuard keys are generated fresh for each session

## Security Scanning

This project runs automated security scans:

- **govulncheck**: Official Go vulnerability checker for dependencies
- **gosec**: Comprehensive Go security analyzer
- **trivy**: Filesystem and configuration scanner

Scans run on every push, pull request, and weekly on a schedule.
