package server_test

import (
	"net/url"
	"testing"

	"github.com/zeriontech/sidecache/pkg/server"
	"go.uber.org/zap"
)

func TestGetLockKey(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.FailNow()
	}
	testData := []struct {
		name         string
		urlStr       string
		expected     string
		lockLocation server.Location
		lockIndex    int
		lockKey      string
	}{
		{
			"path-1",
			"http://localhost:1234/0x2d03f2b283fc90da454383afa8080293c8336448/info/?address=0x2d03f2b283fc90da454383afa8080293c8336448",
			"lock:0x2d03f2b283fc90da454383afa8080293c8336448",
			server.LocationPath,
			1,
			"",
		},
		{
			"query-address",
			"http://localhost:1234/api/v1/actions/?address=0x49131d39ead64a9e4912e641a6bd5fa7ae452f3f&currency=usd&limit=500&offset=0&search_query=receive",
			"lock:0x49131d39ead64a9e4912e641a6bd5fa7ae452f3f",
			server.LocationQuery,
			0,
			"address",
		},
		{
			"query-multiple-uses-min",
			"http://localhost:1234/api/v1/actions/?address=0x49131d39ead64a9e4912e641a6bd5fa7ae452f3f&address=0x123&address=0x456",
			"lock:0x123",
			server.LocationQuery,
			0,
			"address",
		},
	}
	for _, d := range testData {
		t.Run(d.name, func(t *testing.T) {
			u, err := url.Parse(d.urlStr)
			if err != nil {
				logger.Error("could not parse test url")
				t.FailNow()
			}

			// setup
			server.LockLocation = d.lockLocation
			server.LockIndex = d.lockIndex
			server.LockKey = d.lockKey

			key := server.GetLockKey(logger, u)
			if key != d.expected {
				logger.Error(
					"key mismatch",
					zap.String("expected", d.expected),
					zap.String("actual", key),
				)
				t.Fail()
			}
		})
	}
}
