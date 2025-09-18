package main

import (
	"fmt"
	"math/rand"

	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const FieldPK = "pk"
const FieldSK = "sk"
const FieldNameFormat = "a%d"
const FieldTTL = "ttl"

func MakeItem(pk, sk, size, numFields uint, withTTL bool) map[string]types.AttributeValue {
	out := make(map[string]types.AttributeValue)

	remaining := int(size)

	pkStr := fmt.Sprintf("%d", pk)
	out[FieldPK] = &types.AttributeValueMemberN{
		Value: pkStr,
	}
	remaining -= len(FieldPK)
	remaining -= len(pkStr)

	skStr := fmt.Sprintf("%d", sk)
	out[FieldSK] = &types.AttributeValueMemberN{
		Value: skStr,
	}
	remaining -= len(FieldSK)
	remaining -= len(skStr)

	if withTTL {
		ttl := fmt.Sprintf("%d", time.Now().UTC().Unix()+3600)
		out[FieldTTL] = &types.AttributeValueMemberN{
			Value: ttl,
		}
		remaining -= len(FieldTTL)
		remaining -= len(ttl)
	}

	valueSize := remaining / int(numFields)

	for c := uint(1); c <= numFields; c++ {
		fieldName := fmt.Sprintf(FieldNameFormat, c)
		fieldValue := strings.Repeat(randomLetter(), min(valueSize, remaining))

		out[fieldName] = &types.AttributeValueMemberS{
			Value: fieldValue,
		}
		remaining -= len(fieldName)
		remaining -= len(fieldValue)
	}

	return out
}

func randomLetter() string {
	rand.Seed(time.Now().UnixNano())
	letter := 'a' + rune(rand.Intn(26))
	return string(letter)
}
