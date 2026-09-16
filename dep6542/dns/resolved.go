package dns

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SetupResolved routes vm3's plain DNS configuration through the local resolver
// so names such as registry.localhost resolve without querying public DNS.
func SetupResolved(ctx context.Context) error {
	const resolvConf = "/etc/resolv.conf"
	info, err := os.Lstat(resolvConf)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	data, err := os.ReadFile(resolvConf)
	if err != nil {
		return err
	}
	servers := plainNameservers(string(data))
	if len(servers) == 0 {
		return nil
	}

	const configDir = "/run/systemd/resolved.conf.d"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	config := "[Resolve]\nDNS=\nDNS=" + strings.Join(servers, " ") + "\nDNSStubListener=yes\n"
	if err := os.WriteFile(filepath.Join(configDir, "agentd.conf"), []byte(config), 0644); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "systemctl", "restart", "systemd-resolved").CombinedOutput(); err != nil {
		return fmt.Errorf("start local resolver: %w: %s", err, out)
	}
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, "127.0.0.53:53")
	}}
	addresses, err := resolver.LookupNetIP(ctx, "ip", "registry.localhost")
	if err != nil {
		return fmt.Errorf("check local resolver: %w", err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("local resolver returned no localhost addresses")
	}
	for _, address := range addresses {
		if !address.IsLoopback() {
			return fmt.Errorf("local resolver returned non-loopback address %s", address)
		}
	}
	if _, err := os.Stat("/run/systemd/resolve/stub-resolv.conf"); err != nil {
		return err
	}

	// Replace atomically only after the resolver is ready; failures retain host DNS.
	dir, err := os.MkdirTemp("/etc", ".agentd-resolved-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	link := filepath.Join(dir, "resolv.conf")
	if err := os.Symlink("/run/systemd/resolve/stub-resolv.conf", link); err != nil {
		return err
	}
	return os.Rename(link, resolvConf)
}

// Only accept vm3's nameserver-only format. Leave managed files, search domains,
// resolver options and existing local resolvers untouched.
func plainNameservers(config string) []string {
	var servers []string
	for _, line := range strings.Split(config, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 || fields[0] != "nameserver" {
			return nil
		}
		address, err := netip.ParseAddr(fields[1])
		if err != nil || !address.IsGlobalUnicast() || address.IsLoopback() || address.Zone() != "" {
			return nil
		}
		servers = append(servers, fields[1])
	}
	return servers
}
