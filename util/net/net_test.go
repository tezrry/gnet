package net

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetIPv4(t *testing.T) {
	ip, err := GetIPv4()
	require.NoError(t, err)
	t.Log(ip)

	ip, err = GetIPv4("10.*")
	require.NoError(t, err)
	t.Log(ip)

	ip, err = GetIPv4("192.168.*", "10.*")
	require.NoError(t, err)
	t.Log(ip)
}
