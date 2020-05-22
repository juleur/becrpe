package graph

import (
	"github.com/jmoiron/sqlx"
	"github.com/juleur/becrpe/cache"
	"github.com/juleur/becrpe/graph/model"
	"github.com/sirupsen/logrus"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB                *sqlx.DB
	SecretKey         string
	RedisCache        *cache.Cache
	UploadFileManager *model.UploadFileManager
	Logger            *logrus.Logger
}
