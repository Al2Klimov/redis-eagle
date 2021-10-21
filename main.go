package main

import (
	"flag"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

type rgxFlag struct {
	rgx *regexp.Regexp
}

var _ flag.Value = (*rgxFlag)(nil)

func (rf rgxFlag) String() string {
	if rf.rgx == nil {
		return ""
	}

	return rf.rgx.String()
}

func (rf *rgxFlag) Set(s string) (err error) {
	rf.rgx, err = regexp.Compile(s)
	return
}

func main() {
	m := &rgxFlag{}
	r := flag.String("r", "127.0.0.1:6379", "HOST:PORT")
	w := flag.String("w", "", "REDIS COMMAND")
	wg := &sync.WaitGroup{}

	flag.Var(m, "m", "REGEX")
	flag.Parse()
	log.SetLevel(log.TraceLevel)

	if *w != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cmd := strings.Split(*w, " ")
			var prev interface{} = struct{}{}

			args := make([]interface{}, 0, len(cmd)-1)
			for _, arg := range cmd[1:] {
				args = append(args, arg)
			}

			cn, err := redis.Dial("tcp", *r)
			if err != nil {
				panic(err)
			}

			for {
				res, err := cn.Do(cmd[0], args...)
				if err != nil {
					panic(err)
				}

				if !reflect.DeepEqual(res, prev) {
					prev = res
					log.Info(res)
				}
			}
		}()
	}

	if m.rgx != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cn, err := redis.Dial("tcp", *r)
			if err != nil {
				panic(err)
			}

			if _, err := cn.Do("MONITOR"); err != nil {
				panic(err)
			}

			for {
				resp, err := cn.Receive()
				if err != nil {
					panic(err)
				}

				if m.rgx.MatchString(resp.(string)) {
					log.Debug(resp)
				}
			}
		}()
	}

	wg.Wait()
}
