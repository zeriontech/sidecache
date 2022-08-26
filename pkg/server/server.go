package server

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zeriontech/sidecache/pkg/cache"
	"github.com/zeriontech/sidecache/pkg/lock"
	"go.uber.org/zap"
)

const CacheHeaderEnabledKey = "sidecache-headers-enabled"
const applicationDefaultPort = ":9191"

type CacheServer struct {
	Repo           cache.Repository
	LockMgr        lock.Lock
	Proxy          *httputil.ReverseProxy
	Prometheus     *Prometheus
	Logger         *zap.Logger
	CacheKeyPrefix string
}

type CacheData struct {
	Body       []byte
	Headers    map[string]string
	StatusCode int
}

func NewServer(repo cache.Repository, lockMgr lock.Lock, proxy *httputil.ReverseProxy, prom *Prometheus, logger *zap.Logger) *CacheServer {
	return &CacheServer{
		Repo:           repo,
		LockMgr:        lockMgr,
		Proxy:          proxy,
		Prometheus:     prom,
		Logger:         logger,
		CacheKeyPrefix: os.Getenv("CACHE_KEY_PREFIX"),
	}
}

func (server CacheServer) Start(stopChan chan int) {
	server.Proxy.ModifyResponse = func(r *http.Response) error {
		if r.StatusCode >= 500 {
			return nil
		}

		cacheHeadersEnabled := r.Header.Get(CacheHeaderEnabledKey)

		r.Header.Del("Content-Length") // https://github.com/golang/go/issues/14975
		b, err := ioutil.ReadAll(r.Body)

		if err != nil {
			server.Logger.Error("Error while reading response body", zap.Error(err))
			return err
		}

		go func(reqUrl *url.URL, data []byte, statusCode int, ttl time.Duration, cacheHeadersEnabled string) {
			hashedURL := server.HashURL(server.ReorderQueryString(reqUrl))
			cacheData := CacheData{Body: data, StatusCode: statusCode}

			if cacheHeadersEnabled == "true" {
				headers := make(map[string]string)
				for h, v := range r.Header {
					headers[h] = strings.Join(v, ";")
				}
				cacheData.Headers = headers
			}

			cacheDataBytes, _ := json.Marshal(cacheData)
			server.Repo.SetKey(hashedURL, cacheDataBytes, ttl)
		}(r.Request.URL, b, r.StatusCode, CacheTtl, cacheHeadersEnabled)

		err = r.Body.Close()
		if err != nil {
			server.Logger.Error("Error while closing response body", zap.Error(err))
			return err
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(b))

		return nil
	}

	http.HandleFunc("/", server.CacheHandler)
	http.Handle("/metrics", promhttp.Handler())

	port := determinatePort()
	httpServer := &http.Server{Addr: port}
	server.Logger.Info("SideCache process started port: ", zap.String("port", port))

	go func() {
		server.Logger.Warn("Server closed: ", zap.Error(httpServer.ListenAndServe()))
	}()

	<-stopChan

	err := httpServer.Shutdown(context.Background())
	if err != nil {
		server.Logger.Error("shutdown hook error", zap.Error(err))
	}
}

func determinatePort() string {
	customPort := os.Getenv("SIDE_CACHE_PORT")
	if customPort == "" {
		return applicationDefaultPort

	}
	return ":" + customPort
}

func (server CacheServer) CacheHandler(w http.ResponseWriter, r *http.Request) {
	server.Logger.Info("handling request", zap.String("url", r.URL.String()))
	server.Prometheus.TotalRequestCounter.Inc()

	defer func() {
		if rec := recover(); rec != nil {
			var err error
			switch x := rec.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}

			server.Logger.Info("recovered from panic", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()

	path := strings.Split(r.URL.Path, "/")
	key := "lock:" + path[1] + "/" + path[2]
	resultKey := server.HashURL(server.ReorderQueryString(r.URL))

	if UseLock {
		attempt := 1
		for {
			// check the cache
			server.Logger.Info("checking the cache", zap.String("resultKey", resultKey), zap.Int("attempt", attempt))
			if cachedDataBytes := server.CheckCache(resultKey); cachedDataBytes != nil {
				serveFromCache(cachedDataBytes, server, w, r)
				return
			}

			// try to acquire the lock
			server.Logger.Info("acquiring the lock", zap.String("key", key))
			if err := server.LockMgr.Acquire(key, LockTtl); err == nil {
				server.Prometheus.LockAcquiringAttemptsHistogram.Observe(float64(attempt))
				server.Logger.Info("lock acquired", zap.String("key", key), zap.Int("attempt", attempt))
				defer func() {
					// release the lock
					if err := server.LockMgr.Release(key); err != nil {
						server.Logger.Error("could not unlock the lock", zap.Error(err))
					}
				}()
				serve(server, w, r)
				return
			} else {
				server.Logger.Error("lock is locked", zap.Error(err))
			}

			// wait a bit
			backoff := server.GetBackoff(attempt)
			if backoff >= LockTtl {
				// failed to acquire the lock for too long
				server.Logger.Error("failed to acquire the lock", zap.String("url", r.URL.String()))
				w.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			server.Logger.Info("sleeping", zap.String("key", key), zap.Duration("backoff", backoff))
			time.Sleep(backoff)
			attempt++
		}
	} else {
		serve(server, w, r)
	}
}

func serve(server CacheServer, w http.ResponseWriter, r *http.Request) {
	hashedURL := server.HashURL(server.ReorderQueryString(r.URL))
	cachedDataBytes := server.CheckCache(hashedURL)

	if cachedDataBytes != nil {
		serveFromCache(cachedDataBytes, server, w, r)
	} else {
		server.Logger.Info("proxying request", zap.String("url", r.URL.String()))
		server.Proxy.ServeHTTP(w, r)
	}
}

func serveFromCache(cachedDataBytes []byte, server CacheServer, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Cache-Response-For", r.URL.String())
	w.Header().Add("Content-Type", "application/json;charset=UTF-8")

	server.Logger.Info("serving from cache", zap.String("url", r.URL.String()))
	var cachedData CacheData
	err := json.Unmarshal(cachedDataBytes, &cachedData)
	if err != nil {
		server.Logger.Error("Can not unmarshal cached data", zap.Error(err))
		return
	}

	writeHeaders(w, cachedData.Headers)
	w.WriteHeader(cachedData.StatusCode)

	if _, err := io.Copy(w, bytes.NewReader(cachedData.Body)); err != nil {
		server.Logger.Error("IO error", zap.Error(err))
		return
	}

	server.Prometheus.CacheHitCounter.Inc()
}

func writeHeaders(w http.ResponseWriter, headers map[string]string) {
	if headers != nil {
		for h, v := range headers {
			w.Header().Set(h, v)
		}
	}
}

func (server CacheServer) HashURL(url string) string {
	hasher := md5.New()
	hasher.Write([]byte(server.CacheKeyPrefix + "/" + url))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (server CacheServer) CheckCache(url string) []byte {
	if server.Repo == nil {
		return nil
	}
	return server.Repo.Get(url)
}

func (server CacheServer) ReorderQueryString(url *url.URL) string {
	return url.Path + "?" + url.Query().Encode()
}

func (server CacheServer) GetBackoff(attempt int) time.Duration {
	if attempt <= 10 {
		return 100 * time.Millisecond
	}
	return 500 * time.Millisecond
}
