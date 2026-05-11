# Security Policy

## Reporting a vulnerability

Email **security@larm.dev** with reproduction steps and affected versions. Please do not file public GitHub issues for security reports.

## Supported versions

During the `0.x` series, only the latest minor release receives security fixes. After `1.0`, the latest two minor releases will receive fixes.

## Scope

This SDK handles bearer tokens for the Larm public API. Vulnerabilities of interest include:

- Token leakage (e.g. logging credentials, sending them to wrong endpoints)
- Request smuggling, header injection
- TLS or transport-layer issues introduced by the SDK
- Vulnerabilities in retry / backoff that could be abused

For vulnerabilities in the Larm backend or product itself, see the security policy at [larm.dev](https://larm.dev).
