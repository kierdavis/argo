/*
	Copyright (c) 2012 Kier Davis

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
	associated documentation files (the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge, publish, distribute,
	sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial
	portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
	NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
	NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES
	OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
	CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package redisstore

import (
	"code.google.com/p/tcgl/redis"
	"encoding/hex"
	"github.com/kierdavis/argo"
	"hash/fnv"
)

func HashTerm(term argo.Term) (s string) {
	h := fnv.New64()
	h.Write([]byte(term.String()))
	return hex.EncodeToString(h.Sum(nil))
}

type RedisStore struct {
	db *redis.Database
}

func NewRedisStore(c redis.Configuration) (store *RedisStore) {
	return &RedisStore{
		db: redis.Connect(c),
	}
}

func (store *RedisStore) Add(triple *argo.Triple) {
	s := HashTerm(triple.Subject)
	p := HashTerm(triple.Predicate)
	o := HashTerm(triple.Object)

	store.db.MultiCommand(func(mc *redis.MultiCommand) {
		mc.Command("SADD", "sp"+s+p, triple.Object.String())
		mc.Command("SADD", "so"+s+o, triple.Predicate.String())
		mc.Command("SADD", "po"+p+o, triple.Subject.String())
		mc.Command("RPUSH", "triples", triple.String())
	})
}

func (store *RedisStore) Remove(triple *argo.Triple) {
	s := HashTerm(triple.Subject)
	p := HashTerm(triple.Predicate)
	o := HashTerm(triple.Object)

	store.db.MultiCommand(func(mc *redis.MultiCommand) {
		mc.Command("SREM", "sp"+s+p, triple.Object.String())
		mc.Command("SREM", "so"+s+o, triple.Predicate.String())
		mc.Command("SREM", "po"+p+o, triple.Subject.String())
		mc.Command("LREM", "triples", 0, triple.String())
	})
}

func (store *RedisStore) Clear() {
	store.db.Command("FLUSHDB")
}

func (store *RedisStore) Num() (n int) {
	n, err := store.db.Command("LLEN", "triples").ValueAsInt()
	if err != nil {
		panic(err)
	}

	return n
}

func (store *RedisStore) IterTriples() (ch chan *argo.Triple) {
	return nil
}

func (store *RedisStore) Filter(subjSearch, predSearch, objSearch argo.Term) (ch chan *argo.Triple) {
	return nil
}
