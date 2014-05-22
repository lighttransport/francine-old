package main

import (
  "log"
  "fmt"
  "github.com/bradfitz/gomemcache/memcache"
)

func main() {
   mc := memcache.New("10.0.0.1:11211")
   mc.Set(&memcache.Item{Key: "foo", Value: []byte("my value")})

   it, err := mc.Get("foo")
   if err != nil {
     log.Fatal(err)
   }
 
   fmt.Printf(it.Key)
}
