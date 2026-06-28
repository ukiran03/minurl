package data

import (
	"errors"

	"github.com/bwmarrin/snowflake"
)

const encodeBase62Map = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var (
	decodeBase62Map  [256]byte
	ErrInvalidBase62 = errors.New("invalid base62")
)

type flake snowflake.ID

func init() {
	for i := range len(decodeBase62Map) {
		decodeBase62Map[i] = 0xFF
	}

	for i := range len(encodeBase62Map) {
		decodeBase62Map[encodeBase62Map[i]] = byte(i)
	}
}

func NewFlake(sfnid int64) flake {
	node, _ := snowflake.NewNode(sfnid)
	return flake(node.Generate())
}

// Base62 returns a base62 string of the snowflake ID
func (f flake) Base62() string {
	num := int64(f)

	if num < 62 {
		return string(encodeBase62Map[num])
	}

	var buf [11]byte
	i := 10
	for num >= 62 {
		buf[i] = encodeBase62Map[num%62]
		num /= 62
		i--
	}

	buf[i] = encodeBase62Map[num]
	return string(buf[i:])
}

// ParseBase62 returns int64 representation of string (slug)
func ParseBase62(s string) (flake, error) {
	if len(s) == 0 || len(s) > 11 {
		return 0, ErrInvalidBase62
	}

	var result int64
	var maxInt64 int64 = 9223372036854775807

	for i := 0; i < len(s); i++ {
		val := decodeBase62Map[s[i]]
		if val == 0xFF {
			return 0, ErrInvalidBase62
		}
		if result > (maxInt64-int64(val))/62 {
			return 0, ErrInvalidBase62
		}
		result = (result * 62) + int64(val)
	}

	return flake(result), nil
}
