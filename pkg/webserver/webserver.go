package webserver

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/fortnoxab/ginprometheus"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/jonaz/ginlogrus"
	"github.com/sirupsen/logrus"
)

type Webserver struct {
	Port          string
	EnableMetrics bool
}

func (ws *Webserver) Init() *gin.Engine {
	router := gin.New()
	if ws.EnableMetrics {
		p := ginprometheus.New("http")
		p.Use(router)
	}

	logIgnorePaths := []string{
		"/health",
		"/metrics",
		"/readiness",
	}
	router.Use(ginlogrus.New(logrus.StandardLogger(), logIgnorePaths...), gin.Recovery())

	// router.GET("/", func(c *gin.Context) {
	// 	fmt.Fprintf(c.Writer, `todo`)
	// })

	pprof.Register(router)
	return router
}

func (ws *Webserver) Start(ctx context.Context) {
	srv := &http.Server{
		ReadTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		Addr:              ":" + ws.Port,
		Handler:           ws.Init(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("error starting webserver %s", err)
		}
	}()

	logrus.Debug("webserver started")

	<-ctx.Done()

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		logrus.Debug("sleeping 5 sec before shutdown") // to give k8s ingresses time to sync
		time.Sleep(5 * time.Second)
	}
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutDown); !errors.Is(err, http.ErrServerClosed) && err != nil {
		logrus.Error(err)
	}
}
