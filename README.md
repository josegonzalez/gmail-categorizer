# gmail-categorizer

A CLI tool that connects to Gmail via IMAP and generates statistics about email distribution by recipient address. Designed for catch-all inbox analysis.

## Features

- Connect to Gmail via IMAP with App Password authentication
- Generate statistics by recipient (`to`) address
- Interactive TUI for email triage with batch archiving
- List all mailboxes/labels in your account
- Special handling for catch-all addresses (admin@, hi@, email@) - groups by sender instead
- Multiple output formats: table, JSON, CSV
- Flexible configuration: CLI flags, environment variables, config file
- Secure password storage using OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)

## Installation

```bash
go install github.com/josegonzalez/gmail-categorizer/cmd/gmail-categorizer@latest
```

Or build from source:

```bash
git clone https://github.com/josegonzalez/gmail-categorizer.git
cd gmail-categorizer
make build
```

## Prerequisites

### Gmail App Password

1. Go to [Google Account Security](https://myaccount.google.com/security)
2. Enable 2-Step Verification if not already enabled
3. Go to [App Passwords](https://myaccount.google.com/apppasswords)
4. Select "Mail" and your device, then click "Generate"
5. Copy the 16-character password

## Usage

The CLI has three subcommands: `stats`, `triage`, and `mailboxes`.

### Stats Command

Generate statistics about email distribution:

```bash
# Interactive password prompt
gmail-categorizer stats -u your-email@gmail.com

# With environment variable
export GMAIL_CATEGORIZER_IMAP_PASSWORD="your-app-password"
gmail-categorizer stats -u your-email@gmail.com

# With config file
gmail-categorizer stats -c ~/.gmail-categorizer.yaml

# Different output formats
gmail-categorizer stats -u your-email@gmail.com -f json
gmail-categorizer stats -u your-email@gmail.com -f csv > stats.csv
```

### Triage Command

Interactive TUI for batch email management:

```bash
# Launch triage interface
gmail-categorizer triage -u your-email@gmail.com -k

# With keychain (recommended for repeated use)
gmail-categorizer triage -u your-email@gmail.com --keychain
```

The triage interface allows you to:
- View emails grouped by recipient address
- Browse subject lines within each group
- Archive entire groups with a single action

When archiving, emails are moved to `automated/<local_part>` folders based on the grouping address. For example, emails grouped under `notifications@example.com` are archived to the `automated/notifications` folder.

### Mailboxes Command

List all available mailboxes/labels:

```bash
gmail-categorizer mailboxes -u your-email@gmail.com -k
```

### Keychain Storage

Store your password securely in the OS keychain to avoid entering it each time:

```bash
# First run: prompts for password and stores it in keychain
gmail-categorizer stats -u your-email@gmail.com --keychain

# Subsequent runs: retrieves password from keychain automatically
gmail-categorizer stats -u your-email@gmail.com --keychain

# Store password from environment variable in keychain
GMAIL_CATEGORIZER_IMAP_PASSWORD="your-app-password" gmail-categorizer stats -u your-email@gmail.com --keychain

# Delete stored password from keychain
gmail-categorizer stats -u your-email@gmail.com --delete-keychain
```

The keychain feature uses:
- **macOS**: Keychain Access
- **Windows**: Windows Credential Manager
- **Linux**: Secret Service (GNOME Keyring, KWallet)

### Options

Global flags:
```
  -c, --config string       config file (default: $HOME/gmail-categorizer.yaml)
  -h, --help                help for gmail-categorizer
  -v, --version             version for gmail-categorizer
```

Stats command flags:
```
  -u, --username string     Gmail username/email address
  -p, --password string     Gmail app password (prefer stdin or env var)
  -m, --mailbox string      mailbox to analyze (default "INBOX")
  -k, --keychain            use OS keychain to store/retrieve password
      --delete-keychain     delete stored password from keychain and exit
  -f, --format string       output format: table, json, csv (default "table")
  -l, --limit int           limit number of results (0 = unlimited)
      --sort-by string      sort by: count, address (default "count")
      --sort-order string   sort order: asc, desc (default "desc")
```

Triage command flags:
```
  -u, --username string     Gmail username/email address
  -p, --password string     Gmail app password (prefer stdin or env var)
  -k, --keychain            use OS keychain to store/retrieve password
      --delete-keychain     delete stored password from keychain and exit
```

Mailboxes command flags:
```
  -u, --username string     Gmail username/email address
  -p, --password string     Gmail app password (prefer stdin or env var)
  -k, --keychain            use OS keychain to store/retrieve password
```

## Configuration

Configuration can be provided via:

1. CLI flags (highest priority)
2. Environment variables
3. Config file
4. Defaults (lowest priority)

### Environment Variables

```bash
export GMAIL_CATEGORIZER_IMAP_USERNAME="your-email@gmail.com"
export GMAIL_CATEGORIZER_IMAP_PASSWORD="your-app-password"
export GMAIL_CATEGORIZER_OUTPUT_FORMAT="json"
```

### Config File

Create `~/.gmail-categorizer.yaml` or `./gmail-categorizer.yaml`:

```yaml
imap:
  username: your-email@gmail.com
  # password: set via env var or stdin for security
  mailbox: INBOX

output:
  format: table
  sort_by: count
  sort_order: desc
  limit: 0

special:
  group_by_from:
    - admin@
    - hi@
    - email@
    - support@
```

## Special Address Handling

For catch-all inboxes, emails to certain addresses (like admin@, hi@, email@) are grouped by sender instead of recipient. This helps identify who is sending emails to these generic addresses.

Example output:

```
Emails by Recipient Address:
+----------------------+-------+
| To Address           | Count |
+----------------------+-------+
| user@yourdomain.com  | 150   |
| sales@yourdomain.com | 75    |
+----------------------+-------+
| Total                | 225   |
+----------------------+-------+

Emails to Special Addresses (grouped by sender):
+----------------------+----------------------+-------+
| To Address           | From Address         | Count |
+----------------------+----------------------+-------+
| admin@yourdomain.com | noreply@service.com  | 45    |
| admin@yourdomain.com | alerts@monitor.io    | 23    |
| hi@yourdomain.com    | newsletter@news.com  | 12    |
+----------------------+----------------------+-------+
```

## Development

```bash
# Run tests
make test

# Build
make build

# Format code
make fmt

# Run all checks
make all
```

## License

MIT
