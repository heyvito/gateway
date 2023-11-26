package gateway

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"strings"
	"testing"
)

func fixtureFile(t *testing.T, name string) []byte {
	t.Helper()
	file, err := os.ReadFile(fixtureFilePath(name))
	require.NoError(t, err)
	return file
}

func fixtureFilePath(name string) string {
	return path.Join("fixtures", name+".txt")
}

func setNetstatSource(t *testing.T, source string) {
	t.Helper()
	prevRoutes := getRoutes
	getRoutes = func() (NetRouteList, error) {
		parser := newNetstatParser()
		for _, line := range strings.Split(string(fixtureFile(t, source)), "\n") {
			if err := parser.feed(line); err != nil {
				return nil, err
			}
		}
		return parser.netData, nil
	}
	t.Cleanup(func() {
		getRoutes = prevRoutes
	})
}

func setProcSource(t *testing.T, ipv4, ipv6 string) {
	t.Helper()
	prevRoutes := getRoutes
	getRoutes = func() (NetRouteList, error) {
		var ip6List, ip4List NetRouteList
		var err error
		if ipv6 != "" {
			ip6List, err = getRoutesIPv6(fixtureFilePath(ipv6))
		}
		if err != nil {
			return nil, err
		}

		if ipv4 != "" {
			ip4List, err = getRoutesIPv4(fixtureFilePath(ipv4))
		}
		if err != nil {
			return nil, err
		}

		return append(ip4List, ip6List...), nil
	}
	t.Cleanup(func() {
		getRoutes = prevRoutes
	})
}

func TestDarwin(t *testing.T) {
	t.Run("Sane", func(t *testing.T) {
		setNetstatSource(t, "darwin")
		ifaces, err := FindDefaultInterfaces()
		require.NoError(t, err)
		assert.Len(t, ifaces, 1)
		assert.Equal(t, "en0", ifaces[0])
	})

	t.Run("Bad Route", func(t *testing.T) {
		setNetstatSource(t, "darwinBadRoute")
		ifaces, err := FindDefaultInterfaces()
		require.Error(t, err)
		assert.Len(t, ifaces, 0)
	})

	t.Run("No Route", func(t *testing.T) {
		setNetstatSource(t, "darwinNoRoute")
		ifaces, err := FindDefaultInterfaces()
		require.NoError(t, err)
		assert.Len(t, ifaces, 0)
	})

	t.Run("Bad Data", func(t *testing.T) {
		setNetstatSource(t, "randomData")
		ifaces, err := FindDefaultInterfaces()
		require.Error(t, err)
		assert.Len(t, ifaces, 0)
	})
}

func TestLinux(t *testing.T) {
	t.Run("Sane", func(t *testing.T) {
		setProcSource(t, "linuxipv4", "linuxipv6")
		ifaces, err := FindDefaultInterfaces()
		require.NoError(t, err)
		assert.Len(t, ifaces, 1)
		assert.Equal(t, "wlp4s0", ifaces[0])
	})

	t.Run("No Route", func(t *testing.T) {
		setProcSource(t, "linuxNoRoute", "")
		ifaces, err := FindDefaultInterfaces()
		require.NoError(t, err)
		assert.Len(t, ifaces, 0)
	})

	t.Run("Bad IPv4 data", func(t *testing.T) {
		setProcSource(t, "randomData", "")
		ifaces, err := FindDefaultInterfaces()
		require.Error(t, err)
		assert.Len(t, ifaces, 0)
	})

	t.Run("Bad IPv6 data", func(t *testing.T) {
		setProcSource(t, "", "randomData")
		ifaces, err := FindDefaultInterfaces()
		require.Error(t, err)
		assert.Len(t, ifaces, 0)
	})
}
