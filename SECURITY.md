# Security Policy

## Supported Versions

Qubex is pre-1.0; security fixes are applied to the latest `main` and the most
recent release.

## Reporting a Vulnerability

Please report security issues privately rather than opening a public issue:

- Preferred: GitHub's [private vulnerability reporting][gh] on this repository
  (Security → Report a vulnerability).
- Alternatively, email **agrimbuilds@gmail.com** with details and, if possible,
  a minimal reproduction.

Please include the affected version/commit, the impact, and reproduction steps.
You can expect an initial acknowledgement within a few days. Once a fix is
available, we will coordinate disclosure and credit reporters who wish to be
named.

## Scope and boundaries

Qubex decodes a single JSON document in memory and executes queries through an
embedded gojq interpreter. The runtime does **not** enable modules, environment
access, or external command execution. Reports about resource consumption on
hostile input, terminal escape handling, or history-file persistence are in
scope and welcome.

[gh]: https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability
