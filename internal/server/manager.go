package server

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/neko-rooms/internal/branding"
	"github.com/m1k1o/neko-rooms/internal/config"
	"github.com/m1k1o/neko-rooms/internal/types"
)

type ServerManagerCtx struct {
	logger zerolog.Logger
	router *chi.Mux
	server *http.Server
	config *config.Server
}

func New(ApiManager types.ApiManager, roomConfig *config.Room, config *config.Server, proxyHandler http.Handler, brandingManager *branding.Manager) *ServerManagerCtx {
	logger := log.With().Str("module", "server").Logger()

	router := chi.NewRouter()
	router.Use(middleware.RequestID) // Create a request ID for each request

	// get real users ip
	if config.Proxy {
		router.Use(middleware.RealIP)
	}

	// add http logger
	router.Use(middleware.RequestLogger(&logformatter{logger}))
	router.Use(middleware.Recoverer) // Recover from panics without crashing server

	// Basic CORS
	if config.CORS {
		// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
		router.Use(cors.Handler(cors.Options{
			AllowOriginFunc: func(r *http.Request, origin string) bool {
				return true
			},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))
	}

	//
	// admin page
	//

	protected := func(next http.Handler) http.Handler {
		// if proxy auth is enabled
		if config.Admin.ProxyAuth != "" {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				req, err := http.NewRequest("GET", config.Admin.ProxyAuth, nil)
				if err != nil {
					logger.Err(err).Msg("proxy auth request `http.NewRequest` err")
					http.Error(w, "proxy auth failed", http.StatusForbidden)
					return
				}

				//
				// copy headers
				//

				for k, vv := range r.Header {
					for _, v := range vv {
						req.Header.Add(k, v)
					}
				}

				//
				// add x-forwarded headers
				//

				if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
					req.Header.Add("X-Forwarded-For", clientIP)
				}

				if r.Method != "" {
					req.Header.Add("X-Forwarded-Method", r.Method)
				} else {
					req.Header.Del("X-Forwarded-Method")
				}

				if r.TLS != nil {
					req.Header.Add("X-Forwarded-Proto", "https")
				} else {
					req.Header.Add("X-Forwarded-Proto", "http")
				}

				if r.Host != "" {
					req.Header.Add("X-Forwarded-Host", r.Host)
				} else {
					req.Header.Del("X-Forwarded-Host")
				}

				if r.URL.RequestURI() != "" {
					req.Header.Add("X-Forwarded-Uri", r.URL.RequestURI())
				} else {
					req.Header.Del("X-Forwarded-Uri")
				}

				client := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}

				res, err := client.Do(req)
				if err != nil {
					logger.Err(err).Msg("proxy auth request `client.Do` err")
					http.Error(w, "proxy auth failed", http.StatusForbidden)
					return
				}
				defer res.Body.Close()

				if res.StatusCode < 200 || res.StatusCode >= 300 {
					// copy headers
					for k, vv := range res.Header {
						for _, v := range vv {
							w.Header().Add(k, v)
						}
					}

					// copy status & body
					w.WriteHeader(res.StatusCode)
					_, err := io.Copy(w, res.Body)
					if err != nil {
						logger.Err(err).Msg("proxy auth request `io.Copy` err")
					}
					return
				} else {
					// discard body so that the connection can be reused
					_, _ = io.Copy(io.Discard, res.Body)
				}

				next.ServeHTTP(w, r)
			})
		}

		return next
	}

	// serves a file from the static directory, index.html is rendered with branding
	serveStatic := func(w http.ResponseWriter, r *http.Request, filePath string) {
		filePath = path.Clean("/" + filePath)
		if filePath == "/" || filePath == "/index.html" {
			raw, err := os.ReadFile(filepath.Join(config.Admin.Static, "index.html"))
			if err != nil {
				http.Error(w, "admin client not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(brandingManager.RenderIndex(raw))
			return
		}

		fullPath := filepath.Join(config.Admin.Static, filepath.FromSlash(filePath))
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, fullPath)
		} else if err == nil || errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	// DEPRECATED: admin should not be served from the same path as rooms
	if config.Admin.PathPrefix == "/" {
		// cache static file paths
		staticFiles := map[string]struct{}{}
		if config.Admin.Static != "" {
			filepath.Walk(config.Admin.Static,
				func(p string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					staticFiles[path.Clean(p)] = struct{}{}
					return nil
				})
		}

		// serve static files
		router.Use(func(next http.Handler) http.Handler {
			// if static files are disabled
			if config.Admin.Static == "" {
				return next
			}

			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				filePath := path.Join(config.Admin.Static, r.URL.Path)

				// check if file exists to serve it
				if _, ok := staticFiles[filePath]; ok {
					protected(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						serveStatic(w, r, r.URL.Path)
					})).ServeHTTP(w, r)
					return
				}

				next.ServeHTTP(w, r)
			})
		})

		// serve API
		router.With(protected).Route("/api", ApiManager.Mount)
	} else {
		router.With(protected).Route(config.Admin.PathPrefix+"/", func(r chi.Router) {
			// serve API
			r.Route("/api", ApiManager.Mount)

			// serve static files
			r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
				serveStatic(w, r, chi.URLParam(r, "*"))
			})
		})

		// the homepage lives in the admin client, send visitors of the root there
		if config.Admin.PathPrefix != "" {
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, config.Admin.PathPrefix+"/", http.StatusFound)
			})
		}
	}

	// mount pprof endpoint
	if config.PProf {
		router.Mount("/debug", middleware.Profiler())
		logger.Info().Msgf("with pprof endpoint")
	}

	// mount metrics prometheus endpoint
	if config.Metrics {
		router.Mount("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
		logger.Info().Msgf("with metrics endpoint")
	}

	// handle all remaining paths with proxy
	router.Handle("/*", proxyHandler)

	return &ServerManagerCtx{
		logger: logger,
		router: router,
		server: &http.Server{
			Addr:    config.Bind,
			Handler: router,
		},
		config: config,
	}
}

func (s *ServerManagerCtx) Start() {
	if s.config.Cert != "" && s.config.Key != "" {
		go func() {
			if err := s.server.ListenAndServeTLS(s.config.Cert, s.config.Key); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start https server")
			}
		}()
		s.logger.Info().Msgf("https listening on %s", s.server.Addr)
	} else {
		go func() {
			if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start http server")
			}
		}()
		s.logger.Info().Msgf("http listening on %s", s.server.Addr)
	}
}

func (s *ServerManagerCtx) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
