package common

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// SSRFProtection is the SSRF protection configuration
type SSRFProtection struct {
	AllowPrivateIp         bool
	DomainFilterMode       bool     // true: allowlist, false: blocklist
	DomainList             []string // domain format, e.g. example.com, *.example.com
	IpFilterMode           bool     // true: allowlist, false: blocklist
	IpList                 []string // CIDR or single IP
	AllowedPorts           []int    // Allowed port ranges
	ApplyIPFilterForDomain bool     // Apply IP filtering to domains
}

// DefaultSSRFProtection is the default SSRF protection configuration
var DefaultSSRFProtection = &SSRFProtection{
	AllowPrivateIp:   false,
	DomainFilterMode: true,
	DomainList:       []string{},
	IpFilterMode:     true,
	IpList:           []string{},
	AllowedPorts:     []int{},
}

// privateIPv4Nets contains private, reserved, and special-purpose IPv4 ranges
// See the IANA IPv4 Special-Purpose Address Registry
// https://www.iana.org/assignments/iana-ipv4-special-registry/
var privateIPv4Nets = []net.IPNet{
	{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(8, 32)},          // 0.0.0.0/8 ("This network" / unspecified)
	{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},         // 10.0.0.0/8 (private)
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)},      // 100.64.0.0/10 (carrier-grade NAT / CGNAT)
	{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},        // 127.0.0.0/8 (loopback)
	{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)},     // 169.254.0.0/16 (link-local)
	{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},      // 172.16.0.0/12 (private)
	{IP: net.IPv4(192, 0, 0, 0), Mask: net.CIDRMask(24, 32)},       // 192.0.0.0/24 (IETF protocol assignments)
	{IP: net.IPv4(192, 0, 2, 0), Mask: net.CIDRMask(24, 32)},       // 192.0.2.0/24 (TEST-NET-1)
	{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},     // 192.168.0.0/16 (private)
	{IP: net.IPv4(198, 18, 0, 0), Mask: net.CIDRMask(15, 32)},      // 198.18.0.0/15 (benchmarking)
	{IP: net.IPv4(198, 51, 100, 0), Mask: net.CIDRMask(24, 32)},    // 198.51.100.0/24 (TEST-NET-2)
	{IP: net.IPv4(203, 0, 113, 0), Mask: net.CIDRMask(24, 32)},     // 203.0.113.0/24 (TEST-NET-3)
	{IP: net.IPv4(224, 0, 0, 0), Mask: net.CIDRMask(4, 32)},        // 224.0.0.0/4 (multicast)
	{IP: net.IPv4(240, 0, 0, 0), Mask: net.CIDRMask(4, 32)},        // 240.0.0.0/4 (reserved)
	{IP: net.IPv4(255, 255, 255, 255), Mask: net.CIDRMask(32, 32)}, // 255.255.255.255/32 (limited broadcast)
}

// privateIPv6Nets contains private, reserved, and special-purpose IPv6 ranges
// See the IANA IPv6 Special-Purpose Address Registry
// https://www.iana.org/assignments/iana-ipv6-special-registry/
var privateIPv6Nets = func() []net.IPNet {
	cidrs := []string{
		"::/128",        // Unspecified address
		"::1/128",       // Loopback
		"::ffff:0:0/96", // IPv4-mapped
		"64:ff9b::/96",  // IPv4/IPv6 translation
		"100::/64",      // Discard-Only
		"2001::/23",     // IETF Protocol Assignments
		"2001:db8::/32", // Documentation
		"fc00::/7",      // Unique Local Address (ULA)
		"fe80::/10",     // Link-local
		"ff00::/8",      // Multicast
	}
	nets := make([]net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil && n != nil {
			nets = append(nets, *n)
		}
	}
	return nets
}()

// isPrivateIP checks whether an IP is a private, reserved, or special-purpose address
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// Unspecified address (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}
	// Loopback and link-local (unicast/multicast)
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// Interface-local multicast (e.g. IPv6 ff01::/16)
	if ip.IsInterfaceLocalMulticast() {
		return true
	}

	if v4 := ip.To4(); v4 != nil {
		for _, privateNet := range privateIPv4Nets {
			if privateNet.Contains(v4) {
				return true
			}
		}
		return false
	}

	// IPv6 checks
	for _, privateNet := range privateIPv6Nets {
		if privateNet.Contains(ip) {
			return true
		}
	}
	// Fallback: other private addresses recognized by the Go standard library
	if ip.IsPrivate() {
		return true
	}
	return false
}

// parsePortRanges parses the port range configuration
// Supported formats: "80", "443", "8000-9000"
func parsePortRanges(portConfigs []string) ([]int, error) {
	var ports []int

	for _, config := range portConfigs {
		config = strings.TrimSpace(config)
		if config == "" {
			continue
		}

		if strings.Contains(config, "-") {
			// Handle a port range such as "8000-9000"
			parts := strings.Split(config, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid port range format: %s", config)
			}

			startPort, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port in range %s: %v", config, err)
			}

			endPort, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port in range %s: %v", config, err)
			}

			if startPort > endPort {
				return nil, fmt.Errorf("invalid port range %s: start port cannot be greater than end port", config)
			}

			if startPort < 1 || startPort > 65535 || endPort < 1 || endPort > 65535 {
				return nil, fmt.Errorf("port range %s contains invalid port numbers (must be 1-65535)", config)
			}

			// Add all ports in the range
			for port := startPort; port <= endPort; port++ {
				ports = append(ports, port)
			}
		} else {
			// Handle a single port such as "80"
			port, err := strconv.Atoi(config)
			if err != nil {
				return nil, fmt.Errorf("invalid port number: %s", config)
			}

			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("invalid port number %d (must be 1-65535)", port)
			}

			ports = append(ports, port)
		}
	}

	return ports, nil
}

// isAllowedPort checks whether a port is allowed
func (p *SSRFProtection) isAllowedPort(port int) bool {
	if len(p.AllowedPorts) == 0 {
		return true // If no port restriction is configured, allow all ports
	}

	for _, allowedPort := range p.AllowedPorts {
		if port == allowedPort {
			return true
		}
	}
	return false
}

// isDomainWhitelisted checks whether a domain is in the allowlist
func isDomainListed(domain string, list []string) bool {
	if len(list) == 0 {
		return false
	}

	domain = strings.ToLower(domain)
	for _, item := range list {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		// Exact match
		if domain == item {
			return true
		}
		// Wildcard match (*.example.com)
		if strings.HasPrefix(item, "*.") {
			suffix := strings.TrimPrefix(item, "*.")
			if strings.HasSuffix(domain, "."+suffix) || domain == suffix {
				return true
			}
		}
	}
	return false
}

func (p *SSRFProtection) isDomainAllowed(domain string) bool {
	listed := isDomainListed(domain, p.DomainList)
	if p.DomainFilterMode { // allowlist
		return listed
	}
	// blocklist
	return !listed
}

// isIPWhitelisted checks whether an IP is in the allowlist

func isIPListed(ip net.IP, list []string) bool {
	if len(list) == 0 {
		return false
	}

	return IsIpInCIDRList(ip, list)
}

// IsIPAccessAllowed checks whether an IP is allowed to access
func (p *SSRFProtection) IsIPAccessAllowed(ip net.IP) bool {
	// Private IP restriction
	if isPrivateIP(ip) && !p.AllowPrivateIp {
		return false
	}

	listed := isIPListed(ip, p.IpList)
	if p.IpFilterMode { // allowlist
		return listed
	}
	// blocklist
	return !listed
}

// ValidateURL verifies whether a URL is safe
func (p *SSRFProtection) ValidateURL(urlStr string) error {
	// Parse the URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Only HTTP/HTTPS protocols are allowed
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported protocol: %s (only http/https allowed)", u.Scheme)
	}

	// Parse the host and port
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		// No port specified; use the default port
		host = u.Hostname()
		if u.Scheme == "https" {
			portStr = "443"
		} else {
			portStr = "80"
		}
	}

	// Validate the port
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s", portStr)
	}

	if !p.isAllowedPort(port) {
		return fmt.Errorf("port %d is not allowed", port)
	}

	// If the host is an IP, skip the domain check
	if ip := net.ParseIP(host); ip != nil {
		if !p.IsIPAccessAllowed(ip) {
			if isPrivateIP(ip) {
				return fmt.Errorf("private IP address not allowed: %s", ip.String())
			}
			if p.IpFilterMode {
				return fmt.Errorf("ip not in whitelist: %s", ip.String())
			}
			return fmt.Errorf("ip in blacklist: %s", ip.String())
		}
		return nil
	}

	// Apply domain filtering first
	if !p.isDomainAllowed(host) {
		if p.DomainFilterMode {
			return fmt.Errorf("domain not in whitelist: %s", host)
		}
		return fmt.Errorf("domain in blacklist: %s", host)
	}

	// If IP filtering for domains is disabled, the URL passes at this point
	if !p.ApplyIPFilterForDomain {
		return nil
	}

	// Resolve the domain's IPs and check them
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %v", host, err)
	}
	for _, ip := range ips {
		if !p.IsIPAccessAllowed(ip) {
			if isPrivateIP(ip) && !p.AllowPrivateIp {
				return fmt.Errorf("private IP address not allowed: %s resolves to %s", host, ip.String())
			}
			if p.IpFilterMode {
				return fmt.Errorf("ip not in whitelist: %s resolves to %s", host, ip.String())
			}
			return fmt.Errorf("ip in blacklist: %s resolves to %s", host, ip.String())
		}
	}
	return nil
}

// ValidateURLWithFetchSetting validates a URL using the FetchSetting configuration
func ValidateURLWithFetchSetting(urlStr string, enableSSRFProtection, allowPrivateIp bool, domainFilterMode bool, ipFilterMode bool, domainList, ipList, allowedPorts []string, applyIPFilterForDomain bool) error {
	// If SSRF protection is disabled, return success immediately
	if !enableSSRFProtection {
		return nil
	}

	// Parse the port range configuration
	allowedPortInts, err := parsePortRanges(allowedPorts)
	if err != nil {
		return fmt.Errorf("request reject - invalid port configuration: %v", err)
	}

	protection := &SSRFProtection{
		AllowPrivateIp:         allowPrivateIp,
		DomainFilterMode:       domainFilterMode,
		DomainList:             domainList,
		IpFilterMode:           ipFilterMode,
		IpList:                 ipList,
		AllowedPorts:           allowedPortInts,
		ApplyIPFilterForDomain: applyIPFilterForDomain,
	}
	return protection.ValidateURL(urlStr)
}
