# Security Policy

## Reporting a vulnerability

If you find a security issue in skill-validator, please report it privately
via the [contact form](https://dacharycarey.com/contact/) rather than opening
a public issue.

Include as much detail as you can: what you found, how to reproduce it, and
what impact you think it has. You should expect a response within a few days.

## Scope

skill-validator processes untrusted skill packages on the user's machine.
Security-relevant issues include (but aren't limited to):

- Path traversal (reading or writing files outside the skill directory)
- Request forgery via link validation (probing internal network addresses)
- Input that causes the tool to hang, crash, or consume excessive resources
- Command injection through skill content
