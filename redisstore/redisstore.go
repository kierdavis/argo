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

func (store *RedisStore) SupportsIndexes() (result bool) {
	return false
}

func (store *RedisStore) Add(triple *argo.Triple) (index int) {
	s := HashTerm(triple.Subject)
	p := HashTerm(triple.Predicate)
	o := HashTerm(triple.Object)

	store.db.MultiCommand(func(mc *redis.MultiCommand) {
		mc.Command("SADD", "sp"+s+p, triple.Object.String())
		mc.Command("SADD", "so"+s+o, triple.Predicate.String())
		mc.Command("SADD", "po"+p+o, triple.Subject.String())
		mc.Command("RPUSH", "triples", triple.String())
	})

	return 0

	/*
		index, err := result.ResultSetAt(3).ValueAsInt()
		if err != nil {
			panic(err)
		}

		return index
	*/
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

func (store *RedisStore) RemoveIndex(index int) {
	panic("not implemented!")
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
