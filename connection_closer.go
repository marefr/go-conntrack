// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package conntrack

import (
	"context"
	"net"
	"time"
)

type connectionCloseTracker struct {
	net.Conn
	ctx       context.Context
	isClient  bool
	tracker   ConnectionTracker
	beginTime time.Time
}

func newConnectionCloseTracker(ctx context.Context, inner net.Conn, isClient bool, tracker ConnectionTracker, beginTime time.Time) net.Conn {
	return &connectionCloseTracker{
		Conn:      inner,
		ctx:       ctx,
		isClient:  isClient,
		tracker:   tracker,
		beginTime: beginTime,
	}
}

func (ct *connectionCloseTracker) Close() error {
	err := ct.Conn.Close()
	ct.tracker.TrackConnection(ct.ctx, &ConnectionClosed{
		Client:    ct.isClient,
		Error:     err,
		BeginTime: ct.beginTime,
		EndTime:   time.Now(),
	})

	return err
}
