package tfconfig

import (
	"reflect"
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"userclouds.com/infra/ucerr"
)

// This function takes an arbitrary value and generates HCL lexer tokens to represent that value in
// HCL. The hclwrite package does have SetAttributeValue and NewExpressionLiteral functions that
// generate tokens for a value, but (1) it takes cty values, and it already takes some work to
// convert an arbitrary golang value into a cty value, and (2) as of writing, hclwrite only supports
// setting *literal* values, which prevents us from generating Terraform with expressions and
// references to other resources. (SetAttributeTraversal supports generating references to other
// resources, but only for top-level attributes, not inside of nested objects or arrays.)
func toHclTokens(val any, ctx *GenerationContext) (hclwrite.Tokens, error) {
	// Nil
	if val == nil {
		return []*hclwrite.Token{{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte("null"),
		}}, nil
	}

	reflectVal := reflect.ValueOf(val)

	// Pointers
	if reflectVal.Kind() == reflect.Ptr {
		return toHclTokens(reflectVal.Elem(), ctx)
	}

	// Arrays
	if reflectVal.Kind() == reflect.Array || reflectVal.Kind() == reflect.Slice {
		var tokens []hclwrite.Tokens
		for i := 0; i < reflectVal.Len(); i++ {
			itemRawVal := reflectVal.Index(i).Interface()
			itemTokens, err := toHclTokens(itemRawVal, ctx)
			if err != nil {
				return []*hclwrite.Token{}, ucerr.Errorf("error generating tokens for array value %+v at index %s: %v", itemRawVal, i, err)
			}
			tokens = append(tokens, itemTokens)
		}
		return hclwrite.TokensForTuple(tokens), nil
	}

	// Objects/maps (can't easily distinguish the two since our data came from a JSON file, where
	// there is no difference)
	if reflectVal.Kind() == reflect.Map {
		var tokens []hclwrite.ObjectAttrTokens
		keys := reflectVal.MapKeys()
		sort.SliceStable(keys, func(i int, j int) bool { return keys[i].String() < keys[j].String() })
		for _, key := range keys {
			itemRawVal := reflectVal.MapIndex(key).Interface()
			itemTokens, err := toHclTokens(itemRawVal, ctx)
			if err != nil {
				return []*hclwrite.Token{}, ucerr.Errorf("error generating tokens for map value %+v under key %s: %v", itemRawVal, key.String(), err)
			}
			tokens = append(tokens, hclwrite.ObjectAttrTokens{
				Name:  hclwrite.TokensForValue(cty.StringVal(key.String())),
				Value: itemTokens,
			})
		}
		return hclwrite.TokensForObject(tokens), nil
	}

	// ucconfig functions:
	if reflectVal.Kind() == reflect.String {
		if invocation := parseFunctionInvocation(reflectVal.String()); invocation != nil {
			tokens, err := invocation.dispatch(ctx)
			if err != nil {
				return tokens, ucerr.Wrap(err)
			}
			return tokens, nil
		}
	}

	// Primitive types: use out-of-the-box TokensForValue with cty values
	ctyType, err := gocty.ImpliedType(val)
	if err != nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not infer cty type for %+v: %v", val, err)
	}
	ctyVal, err := gocty.ToCtyValue(val, ctyType)
	if err != nil {
		return []*hclwrite.Token{}, ucerr.Errorf("could not convert value %+v to a cty value: %v", val, err)
	}
	return hclwrite.TokensForValue(ctyVal), nil
}
