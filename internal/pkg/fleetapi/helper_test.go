// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

package fleetapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/internal/pkg/fleetapi/client"
	"github.com/elastic/elastic-agent/internal/pkg/remote"
	"github.com/elastic/elastic-agent/pkg/core/logger"
)

func authHandler(handler http.HandlerFunc, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const key = "Authorization"
		const prefix = "ApiKey "

		v := strings.TrimPrefix(r.Header.Get(key), prefix)
		if v != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}

func withServer(m func(t *testing.T) *http.ServeMux, test func(t *testing.T, host string)) func(t *testing.T) {
	return func(t *testing.T) {
		s := httptest.NewServer(m(t))
		defer s.Close()
		test(t, s.Listener.Addr().String())
	}
}

func withServerWithAuthClient(
	m func(t *testing.T) *http.ServeMux,
	apiKey string,
	test func(t *testing.T, client client.Sender),
	configMod ...func(*remote.Config),
) func(t *testing.T) {

	return withServer(m, func(t *testing.T, host string) {
		log, _ := logger.New("", false)
		cfg := remote.Config{
			Host: host,
		}
		for _, mod := range configMod {
			mod(&cfg)
		}

		client, err := client.NewAuthWithConfig(log, apiKey, cfg)
		require.NoError(t, err)
		test(t, client)
	})
}
