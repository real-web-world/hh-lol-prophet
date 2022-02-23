package models

import (
	"context"

	"github.com/real-web-world/hh-lol-prophet/global"
	"gorm.io/gorm"
)

type (
	Config struct {
		ID  int64           `json:"id" gorm:"primaryKey"`
		Key string          `json:"key" gorm:"column:k"`
		Val string          `json:"val" gorm:"column:v"`
		Ctx context.Context `json:"-" gorm:"-"`
	}
)

const (
	LocalClientConfKey = "localClient"
	InitLocalClientSql = `
create table config
(
    id integer     not null
        constraint config_pk
            primary key autoincrement,
    k  varchar(32) not null,
    v  TEXT        not null
);
create unique index config_k_uindex
    on config (k);
INSERT INTO config (k, v)
VALUES (?, ?);
`
)

func (m Config) TableName() string {
	return "config"
}
func (m Config) GetGormQuery() *gorm.DB {
	db := global.SqliteDB
	if m.Ctx != nil {
		db = db.WithContext(m.Ctx)
	}
	return db.Model(m)
}
func (m Config) Update(k, v string) error {
	return m.GetGormQuery().Where("k = ?", k).Update("v", v).Error
}
