/*
 * govm p2p api
 *
 * govm的分布式节点间交互的api
 *
 * API version: 1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package api

import (
	"expvar"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Route 单个http路由信息的结构体
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes 路由列表
type Routes []Route

// NewRouter 创建http路由
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	router.Methods("GET").Path("/api/v1/").Name("Index").HandlerFunc(Index)
	router.Handle("/debug/vars", Logger(expvar.Handler(), "expvar"))
	router.Methods("GET").Name("static").Handler(http.FileServer(http.Dir("./static/")))

	return router
}

// Index api接口的默认的处理函数
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!\n")
	for _, route := range routes {
		fmt.Fprintf(w, "    %s: %s %s\n", route.Name, route.Method, route.Pattern)
	}
}

var routes = Routes{
	Route{
		"AccountGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/account",
		AccountGet,
	},

	Route{
		"TransactionNew",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/transaction/new",
		TransactionNew,
	},

	Route{
		"TransactionMovePost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/transaction/move",
		TransactionMovePost,
	},

	Route{
		"TransactionTranferPost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/transaction/tranfer",
		TransactionTranferPost,
	},

	Route{
		"TransactionMinerGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/transaction/miner",
		TransactionMinerGet,
	},

	Route{
		"TransactionMinerPost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/transaction/miner",
		TransactionMinerPost,
	},

	Route{
		"TransactionNewAppPost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/transaction/app/new",
		TransactionNewAppPost,
	},

	Route{
		"TransactionRunAppPost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/transaction/app/run",
		TransactionRunAppPost,
	},

	Route{
		"TransactionAppLifePost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/transaction/app/life",
		TransactionAppLifePost,
	},

	Route{
		"TransactionAppInfoGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/transaction/app/info",
		TransactionAppInfoGet,
	},

	Route{
		"TransactionInfoGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/transaction/info",
		TransactionInfoGet,
	},

	Route{
		"HistoryInGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/transaction/history/input",
		HistoryInGet,
	},

	Route{
		"HistoryOutGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/transaction/history/output",
		HistoryOutGet,
	},

	Route{
		"BlockMinePost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/block/mine",
		BlockMinePost,
	},

	Route{
		"BlockInfoGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/block/info",
		BlockInfoGet,
	},

	Route{
		"BlockRollback",
		strings.ToUpper("Delete"),
		"/api/v1/{chain}/block/rollback",
		BlockRollback,
	},

	Route{
		"ChainNew",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/chain",
		ChainNew,
	},

	Route{
		"DataRead",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/data",
		DataGet,
	},

	Route{
		"DataNextKeyGet",
		strings.ToUpper("Get"),
		"/api/v1/{chain}/data/visit",
		DataNextKeyGet,
	},

	Route{
		"EventPost",
		strings.ToUpper("Post"),
		"/api/v1/{chain}/event",
		EventPost,
	},

	Route{
		"AddNode",
		strings.ToUpper("Post"),
		"/api/v1/node",
		NodePost,
	},
	Route{
		"NodesGet",
		strings.ToUpper("Get"),
		"/api/v1/nodes",
		NodesGet,
	},
	Route{
		"VersionGet",
		strings.ToUpper("Get"),
		"/api/v1/version",
		VersionGet,
	},
	Route{
		"CryptoSign",
		strings.ToUpper("Post"),
		"/api/v1/crypto/sign",
		CryptoSign,
	},
	Route{
		"CryptoCheck",
		strings.ToUpper("Post"),
		"/api/v1/crypto/check",
		CryptoCheck,
	},
	Route{
		"OSExit",
		strings.ToUpper("Delete"),
		"/api/v1/os/exit",
		OSExit,
	},
}
