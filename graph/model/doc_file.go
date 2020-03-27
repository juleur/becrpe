package model

import "github.com/99designs/gqlgen/graphql"

type DocFile struct {
	Title string         `json:"title"`
	File  graphql.Upload `json:"file"`
}
