package database

import (
	"github.com/lengzhao/database/client"
	"github.com/lengzhao/govm/conf"
)

var dfDB *client.Client

// GetClient get database client
func GetClient() *client.Client {
	if dfDB == nil {
		c := conf.GetConf()
		dfDB = client.New(c.DbAddrType, c.DbServerAddr, 10)
	}
	return dfDB
}
