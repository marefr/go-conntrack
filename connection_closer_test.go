package conntrack

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConnectionCloser(t *testing.T) {
	expectedCloseErr := errors.New("boom")

	tcs := []struct {
		desc     string
		client   bool
		closeErr error
	}{
		{
			desc:     "Server connection closed without error",
			client:   false,
			closeErr: nil,
		},
		{
			desc:     "Client connection closed with error",
			client:   true,
			closeErr: expectedCloseErr,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			tracker := &testConnectionTracker{
				stats: []ConnectionStats{},
			}
			conn := &connUnderTest{
				closeErr: tc.closeErr,
			}
			beginTime := time.Now()
			closer := newConnectionCloseTracker(context.Background(), conn, tc.client, tracker, beginTime)
			err := closer.Close()
			require.Len(t, tracker.stats, 1)
			s := tracker.stats[0].(*ConnectionClosed)
			require.NotNil(t, s)

			if tc.closeErr == nil {
				require.NoError(t, err)
				require.Equal(t, tc.client, s.Client)
				require.NoError(t, s.Error)
			} else {
				require.ErrorIs(t, err, expectedCloseErr)
				require.Equal(t, tc.client, s.Client)
				require.ErrorIs(t, s.Error, expectedCloseErr)
			}

			require.Equal(t, s.BeginTime, beginTime)
			require.Greater(t, s.EndTime, beginTime)
		})
	}
}

type connUnderTest struct {
	net.Conn
	closeErr error
}

func (c *connUnderTest) Close() error {
	return c.closeErr
}

type testConnectionTracker struct {
	stats []ConnectionStats
}

func (t *testConnectionTracker) TrackConnection(_ context.Context, stats ConnectionStats) {
	t.stats = append(t.stats, stats)
}
