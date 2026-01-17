package wireguard

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// parseWgShowOutput parses the output of "wg show" command.
// Example output:
//
//	interface: my-own-vpn
//	  public key: <key>
//	  private key: (hidden)
//	  listening port: <port>
//
//	peer: <server-public-key>
//	  endpoint: <ip>:<port>
//	  allowed ips: 0.0.0.0/0, ::/0
//	  latest handshake: 1 minute, 5 seconds ago
//	  transfer: 1.23 MiB received, 456.78 KiB sent
func parseWgShowOutput(output string) (bytesSent, bytesReceived uint64, lastHandshake time.Time) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "transfer:") {
			bytesSent, bytesReceived = parseTransferLine(line)
		} else if strings.HasPrefix(line, "latest handshake:") {
			lastHandshake = parseHandshakeLine(line)
		}
	}

	return bytesSent, bytesReceived, lastHandshake
}

// parseTransferLine parses a line like "transfer: 1.23 MiB received, 456.78 KiB sent"
func parseTransferLine(line string) (sent, received uint64) {
	// Remove prefix
	line = strings.TrimPrefix(line, "transfer:")
	line = strings.TrimSpace(line)

	// Split by comma
	parts := strings.Split(line, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasSuffix(part, "received") {
			received = parseBytes(strings.TrimSuffix(part, "received"))
		} else if strings.HasSuffix(part, "sent") {
			sent = parseBytes(strings.TrimSuffix(part, "sent"))
		}
	}

	return sent, received
}

// parseBytes parses a string like "1.23 MiB" or "456.78 KiB" to bytes
func parseBytes(s string) uint64 {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToLower(parts[1])
	switch unit {
	case "b":
		return uint64(value)
	case "kib":
		return uint64(value * 1024)
	case "mib":
		return uint64(value * 1024 * 1024)
	case "gib":
		return uint64(value * 1024 * 1024 * 1024)
	case "tib":
		return uint64(value * 1024 * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

// parseHandshakeLine parses a line like "latest handshake: 1 minute, 5 seconds ago"
func parseHandshakeLine(line string) time.Time {
	line = strings.TrimPrefix(line, "latest handshake:")
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, "ago")
	line = strings.TrimSpace(line)

	if line == "" || line == "(none)" {
		return time.Time{}
	}

	// Parse duration like "1 minute, 5 seconds" or "2 hours, 30 minutes, 45 seconds"
	duration := parseDuration(line)
	if duration == 0 {
		return time.Time{}
	}

	return time.Now().Add(-duration)
}

// parseDuration parses a duration string like "1 minute, 5 seconds"
func parseDuration(s string) time.Duration {
	var total time.Duration

	// Match patterns like "1 minute" or "5 seconds"
	re := regexp.MustCompile(`(\d+)\s+(second|minute|hour|day)s?`)
	matches := re.FindAllStringSubmatch(s, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		value, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}

		unit := match[2]
		switch unit {
		case "second":
			total += time.Duration(value) * time.Second
		case "minute":
			total += time.Duration(value) * time.Minute
		case "hour":
			total += time.Duration(value) * time.Hour
		case "day":
			total += time.Duration(value) * 24 * time.Hour
		}
	}

	return total
}
