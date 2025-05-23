// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build windows

package install

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// isStopped queries the Windows service manager to see if the state
// of the service is stopped.  It will repeat the query every
// 'interval' until the 'timeout' is reached.  It returns nil if the
// system is stopped within the timeout period.  An error is returned
// if the service doesn't stop before the timeout or if there are
// errors communicating with the service manager.
func isStopped(timeout time.Duration, interval time.Duration, service string) error {
	var err error
	var status svc.Status

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer func() {
		_ = m.Disconnect()
	}()

	s, err := m.OpenService(service)
	if err != nil {
		return fmt.Errorf("failed to open service (%s): %w", service, err)
	}
	defer s.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			status, err = s.Query()
			if err != nil {
				return fmt.Errorf("error querying service (%s): %w", service, err)
			}
			if status.State == svc.Stopped {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("timed out after %s waiting for service (%s) to stop, last state was: %d", timeout, service, status.State)
		}
	}
}

// EnsureServiceRemoved opens the Windows service manager and checks if the
// service is removed. It will repeat this check every 'interval'
// until the 'timeout' is reached.  It returns nil if the service
// is removed within the timeout period.  An error is returned if
// the service is not removed before the timeout or if there are
// errors communicating with the service manager.
func EnsureServiceRemoved(timeout time.Duration, interval time.Duration, service string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer func() {
		_ = m.Disconnect()
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			s, err := m.OpenService(service)
			if s != nil {
				_ = s.Close()
			}
			switch {
			case err == nil:
				// The service is still installed continue waiting
				continue
			case errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST):
				// The service is no longer installed
				return nil
			default:
				// An unknown error occurred trying to open the service
				return fmt.Errorf("error opening service (%s): %w", service, err)
			}
		case <-timer.C:
			return fmt.Errorf("timed out after %s waiting for service (%s) to uninstall", timeout, service)
		}
	}
}
