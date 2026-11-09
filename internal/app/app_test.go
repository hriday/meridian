package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAppHasNoContextBeforeStartup(t *testing.T) {
	a := New()
	require.NotNil(t, a)
	require.Nil(t, a.ctx)
}

func TestStartupSetsContext(t *testing.T) {
	a := New()
	OnStartup(a)(context.Background())
	require.NotNil(t, a.ctx)
}
