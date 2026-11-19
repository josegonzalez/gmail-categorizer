package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
)

func TestNewWebCmd(t *testing.T) {
	cmd := newWebCmd()

	assert.Equal(t, "web", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("username"))
	assert.NotNil(t, flags.Lookup("password"))
	assert.NotNil(t, flags.Lookup("keychain"))
	assert.NotNil(t, flags.Lookup("delete-keychain"))
	assert.NotNil(t, flags.Lookup("listen"))
}

func TestWebCmd_Flags(t *testing.T) {
	cmd := newWebCmd()

	listen, _ := cmd.Flags().GetString("listen")
	assert.Equal(t, config.DefaultWebListen, listen)

	keychain, _ := cmd.Flags().GetBool("keychain")
	assert.False(t, keychain)

	deleteKeychain, _ := cmd.Flags().GetBool("delete-keychain")
	assert.False(t, deleteKeychain)
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		addr net.Addr
		want bool
	}{
		{&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}, true},
		{&net.TCPAddr{IP: net.ParseIP("::1"), Port: 8080}, true},
		{&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8080}, false},
		{&net.TCPAddr{IP: net.ParseIP("192.168.1.10"), Port: 8080}, false},
		{&net.UnixAddr{Name: "/tmp/sock", Net: "unix"}, false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, isLoopback(tt.addr), tt.addr.String())
	}
}
