package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/go-chi/chi"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/juleur/ecrpe/cache"
	"github.com/juleur/ecrpe/graph"
	"github.com/juleur/ecrpe/graph/generated"
	"github.com/juleur/ecrpe/graph/model"
	"github.com/juleur/ecrpe/interceptors"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/vektah/gqlparser/gqlerror"
	"gopkg.in/natefinch/lumberjack.v2"
)

const defaultPort = "6677"

var (
	secretKey  string
	db         *sqlx.DB
	redisCache *cache.Cache
	logger     *logrus.Logger
)

func init() {
	logger = logrus.New()
	logger.SetReportCaller(true)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05-0700",
		PrettyPrint:     true,
	})
	logger.SetOutput(&lumberjack.Logger{
		Filename: fmt.Sprintf("./log/%s.log", time.RFC822),
		MaxAge:   2,
		MaxSize:  30,
	})
}

func init() {
	var err error
	secretKey = "secretKey"
	db, err = sqlx.Connect("mysql", "chermak:pwd@tcp(127.0.0.1:7359)/ecrpe?parseTime=true&time_zone=%27Europe%2FParis%27")
	if err != nil {
		logger.Fatalln(err)
	}
	if err := db.Ping(); err != nil {
		logger.Fatalln(err)
	}
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if redisCache, err = cache.NewCache("localhost:8989", "", 24*time.Hour); err != nil {
		logger.Fatalln(err)
	}
}

func main() {
	defer db.Close()

	uploadFileManager := model.NewUploadFileManager(db, logger)
	go uploadFileManager.DoneProcesses()

	router := chi.NewRouter()
	router.Use(interceptors.JWTCheck(secretKey))
	router.Use(interceptors.GetIPAddress())
	router.Use(interceptors.GetUserAgent())

	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"OPTIONS", "GET", "POST"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)
	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: &graph.Resolver{
			DB:                db,
			SecretKey:         secretKey,
			RedisCache:        redisCache,
			UploadFileManager: uploadFileManager,
			Logger:            logger,
		},
	}))
	srv.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
		logger.Error(err)
		return &gqlerror.Error{
			Message: "Oops, une erreur est survenue",
			Extensions: map[string]interface{}{
				"statusCode": http.StatusInternalServerError,
				"statusText": http.StatusText(http.StatusInternalServerError),
			},
		}
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxUploadSize: 300000000,
	})
	srv.SetQueryCache(lru.New(1000))
	srv.Use(extension.FixedComplexityLimit(30))

	router.Handle("/query", srv)
	if err := http.ListenAndServe(":"+defaultPort, router); err != nil {
		logger.Fatalln(err)
	}
}
