package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pilu/go-base62"
)

const (
	// URLIDKEY *
	URLIDKEY = "next.url.id"
	// ShortlinkKey *
	ShortlinkKey = "shortlink:%s:url"
	// URLHashKey *
	URLHashKey = "urlhash:%s:url"
	// ShortlinkDetailKey *
	ShortlinkDetailKey = "shortlink:%s:detail"
)

var ctx = context.Background()

// RedisCli *
type RedisCli struct {
	Cli *redis.Client
}

// URLDetail *
type URLDetail struct {
	URL                 string        `json:"url"`
	CreatedAt           string        `json:"created_at"`
	ExpirationInMinutes time.Duration `json:"expiration_in_minutes"`
}

// NewReidsCli *
func NewReidsCli(addr string, password string, db int) *RedisCli {
	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if _, err := c.Ping(ctx).Result(); err != nil {
		panic(err)
	}
	return &RedisCli{c}
}

func toSha1(url string) string {
	h := sha1.New()
	h.Write([]byte(url))
	bs := h.Sum(nil)
	h.Reset()
	return hex.EncodeToString(bs)
}

// Shorten *
func (r *RedisCli) Shorten(url string, exp int64) (string, error) {
	h := toSha1(url)

	d, err := r.Cli.Get(ctx, fmt.Sprintf(URLHashKey, h)).Result()

	if err == redis.Nil {
		// not existed, nothing to do
	} else if err != nil {
		return "", err
	} else {
		if d == "{}" {
			// expiration, nothing to do
		} else {
			return d, nil
		}
	}

	// increase the global counter
	err = r.Cli.Incr(ctx, URLIDKEY).Err()
	if err != nil {
		return "", err
	}

	// encode global counter to base64
	id, err := r.Cli.Get(ctx, URLIDKEY).Int()
	if err != nil {
		return "", err
	}
	eid := base62.Encode(id)

	err = r.Cli.Set(ctx, fmt.Sprintf(ShortlinkKey, eid), url, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	err = r.Cli.Set(ctx, fmt.Sprintf(URLHashKey, h), eid, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	detail, err := json.Marshal(&URLDetail{
		URL:                 url,
		CreatedAt:           time.Now().String(),
		ExpirationInMinutes: time.Duration(exp),
	})
	if err != nil {
		return "", err
	}

	err = r.Cli.Set(ctx, fmt.Sprintf(ShortlinkDetailKey, eid), detail, time.Minute*time.Duration(exp)).Err()
	if err != nil {
		return "", err
	}

	return eid, nil
}

// ShortlinkInfo *
func (r *RedisCli) ShortlinkInfo(eid string) (string, error) {
	d, err := r.Cli.Get(ctx, fmt.Sprintf(ShortlinkDetailKey, eid)).Result()
	if err == redis.Nil {
		return "", StatusError{404, errors.New("Unknown short URL")}
	} else if err != nil {
		return "", err
	} else {
		return d, nil
	}
}

// Unshorten *
func (r *RedisCli) Unshorten(eid string) (string, error) {
	url, err := r.Cli.Get(ctx, fmt.Sprintf(ShortlinkKey, eid)).Result()
	if err == redis.Nil {
		return "", StatusError{404, err}
	} else if err != nil {
		return "", err
	} else {
		return url, nil
	}
}
