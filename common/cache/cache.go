package cache

import (
	//"github.com/dgraph-io/ristretto"
	"sync"
)

// 定义一个全局的 Cache  实例
// var globalCache *ristretto.Cache[string, *interface{}]
var once sync.Once

// Init
