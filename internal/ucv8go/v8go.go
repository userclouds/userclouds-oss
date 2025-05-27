package ucv8go

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/dongri/phonenumber"
	"github.com/gofrs/uuid"
	"github.com/jackc/puddle/v2"
	"rogchap.com/v8go"

	"userclouds.com/authz"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
)

// SecretResolver is an interface for resolving secrets by name
type SecretResolver interface {
	ResolveSecret(ctx context.Context, name string) (string, error)
}

const maxPoolSize = 30

func newIsolatePool() *puddle.Pool[*Isolate] {
	constructor := func(context.Context) (*Isolate, error) {
		return &Isolate{Isolate: v8go.NewIsolate()}, nil
	}
	destructor := func(isolate *Isolate) {
		isolate.Dispose()
	}
	pool, err := puddle.NewPool(&puddle.Config[*Isolate]{Constructor: constructor, Destructor: destructor, MaxSize: maxPoolSize})
	if err != nil {
		panic(err)
	}
	return pool
}

var poolCreateOnce sync.Once
var pool *puddle.Pool[*Isolate]

// Context wraps a v8go context to include pointer to pool resource
type Context struct {
	*v8go.Context
	Res *puddle.Resource[*Isolate]
}

// Isolate is a wrapper around v8go.Isolate to track usage
type Isolate struct {
	*v8go.Isolate
	NumUsed int
}

// NewJSContext spins up a consistent v8go isolation context
// including things like our helper functions
func NewJSContext(ctx context.Context, authZClient *authz.Client, auditLogEventType auditlog.EventType, secretResolver SecretResolver) (*Context, *strings.Builder, error) {

	poolCreateOnce.Do(func() {
		pool = newIsolatePool()
	})

	var poolIsolate *puddle.Resource[*Isolate]
	var err error
	for {
		poolIsolate, err = pool.Acquire(ctx)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
		if poolIsolate.Value().NumUsed > 9 {
			poolIsolate.Destroy()
		} else {
			poolIsolate.Value().NumUsed++
			break
		}
	}

	iso := poolIsolate.Value().Isolate

	global := v8go.NewObjectTemplate(iso)
	if err := global.Set("networkRequest", newNetworkRequestFunction(ctx, iso)); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	if err := global.Set("checkAttribute", newCheckAttributeFunction(ctx, iso, authZClient)); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	if err := global.Set("getCountryNameForPhoneNumber", newGetCountryNameForPhoneNumber(ctx, iso)); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	if err := global.Set("getSecret", newGetSecret(ctx, iso, secretResolver)); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	if err := global.Set("auditLog", newAuditLogFunction(ctx, iso, auditLogEventType)); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	jsContext := v8go.NewContext(iso, global)
	ctxGlobal := jsContext.Global()

	consoleBuilder := &strings.Builder{}
	console := v8go.NewObjectTemplate(iso)
	logfn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		consoleBuilder.Write(fmt.Appendln(nil, info.Args()[0]))
		return nil
	})
	if err := console.Set("log", logfn); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	consoleObj, err := console.NewInstance(jsContext)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	if err := ctxGlobal.Set("console", consoleObj); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	return &Context{Context: jsContext, Res: poolIsolate}, consoleBuilder, nil
}

func throwException(ctx context.Context, iso *v8go.Isolate, err error) *v8go.Value {
	res, err := v8go.NewValue(iso, ucerr.UserFriendlyMessage(err))
	if err != nil {
		uclog.Errorf(ctx, "failed to create error value: %v", err)
		return nil
	}
	return iso.ThrowException(res)
}

func newNetworkRequestFunction(ctx context.Context, iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) != 1 {
			uclog.Errorf(ctx, "js network request function called with wrong number of args: %d", len(info.Args()))
			return throwException(ctx, iso, ucerr.Friendlyf(nil, "wrong number of args (got %d, expected 1)", len(info.Args())))
		}

		args := info.Args()[0].Object()
		method, err := args.Get("method")
		if err != nil {
			return throwException(ctx, iso, err)
		}

		url, err := args.Get("url")
		if err != nil {
			return throwException(ctx, iso, err)
		}

		var body string
		if args.Has("body") {
			bodyV, err := args.Get("body")
			if err != nil {
				return throwException(ctx, iso, err)
			}
			body = bodyV.String()
		}

		var headers map[string][]string
		if args.Has("headers") {
			headersV, err := args.Get("headers")
			if err != nil {
				return throwException(ctx, iso, err)
			}

			// TODO (sgarrity 7/24): there seems to be no methods in v8go for accessing the contents of a map
			// so we're just JSON-serializing it and then unmarshalling it into a map we can use ... the alternative
			// seems to be running more Javascript code to do the same thing (ChatGPT's best suggestion), or
			// probably patching v8go (we should do this someday?)
			bs, err := headersV.MarshalJSON()
			if err != nil {
				return throwException(ctx, iso, err)
			}
			headers = make(map[string][]string)
			if err := json.Unmarshal(bs, &headers); err != nil {
				return throwException(ctx, iso, err)
			}
		}

		// support basic auth
		var user, pass string
		if args.Has("auth") {
			authV, err := args.Get("auth")
			if err != nil {
				return throwException(ctx, iso, err)
			}

			authObj := authV.Object()
			userV, err := authObj.Get("user")
			if err != nil {
				return throwException(ctx, iso, err)
			}
			user = userV.String()

			passV, err := authObj.Get("password")
			if err != nil {
				return throwException(ctx, iso, err)
			}
			pass = passV.String()
		}

		r, err := http.NewRequest(method.String(), url.String(), bytes.NewReader([]byte(body)))
		if err != nil {
			return throwException(ctx, iso, err)
		}
		r = r.WithContext(ctx)
		if headers != nil {
			r.Header = http.Header(headers)
		}
		if user != "" || pass != "" {
			r.SetBasicAuth(user, pass)
		}

		res, err := http.DefaultClient.Do(r)
		if err != nil {
			return throwException(ctx, iso, err)
		}
		defer res.Body.Close()
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return throwException(ctx, iso, err)
		}

		// we don't want to assume this is valid JSON (it could just be a single string)
		v, err := v8go.NewValue(iso, string(bodyBytes))
		if err != nil {
			return throwException(ctx, iso, err)
		}
		return v
	})
}

func newCheckAttributeFunction(ctx context.Context, iso *v8go.Isolate, authzClient *authz.Client) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) != 3 {
			uclog.Errorf(ctx, "js check attribute function called with wrong number of args: %d", len(info.Args()))
			return throwException(ctx, iso, ucerr.Errorf("wrong number of args (got %d, expected 3)", len(info.Args())))
		}

		if authzClient == nil {
			uclog.Errorf(ctx, "checkAttribute called in v8go without a valid authZClient")

			v, err := v8go.NewValue(iso, false)
			if err != nil {
				return throwException(ctx, iso, err)
			}
			return v
		}

		id1, err := uuid.FromString(info.Args()[0].String())
		if err != nil {
			return throwException(ctx, iso, err)
		}

		id2, err := uuid.FromString(info.Args()[1].String())
		if err != nil {
			return throwException(ctx, iso, err)
		}

		attribute := info.Args()[2].String()

		ret, err := authzClient.CheckAttribute(ctx, id1, id2, attribute)
		if err != nil {
			return throwException(ctx, iso, err)
		}

		v, err := v8go.NewValue(iso, ret.HasAttribute)
		if err != nil {
			return throwException(ctx, iso, err)
		}
		return v
	})
}

func newGetCountryNameForPhoneNumber(ctx context.Context, iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) != 1 {
			uclog.Errorf(ctx, "js getCountryNameForPhoneNumber function called with wrong number of args: %d", len(info.Args()))
			return throwException(ctx, iso, ucerr.Errorf("wrong number of args (got %d, expected 1)", len(info.Args())))
		}
		v8ctx := v8go.NewContext(iso)

		phoneNumber := info.Args()[0].String()
		country := phonenumber.GetISO3166ByNumber(phoneNumber, true)
		countryObjTemp := v8go.NewObjectTemplate(iso)
		countryObj, err := countryObjTemp.NewInstance(v8ctx)
		if err != nil {
			return throwException(ctx, iso, err)
		}
		err = countryObj.Set("alpha_2", country.Alpha2)
		if err != nil {
			return throwException(ctx, iso, err)
		}
		err = countryObj.Set("alpha_3", country.Alpha3)
		if err != nil {
			return throwException(ctx, iso, err)
		}
		err = countryObj.Set("country_name", country.CountryName)
		if err != nil {
			return throwException(ctx, iso, err)
		}
		err = countryObj.Set("country_code", country.CountryCode)
		if err != nil {
			return throwException(ctx, iso, err)
		}

		return countryObj.Value
	})
}

func newGetSecret(ctx context.Context, iso *v8go.Isolate, secretResolver SecretResolver) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) != 1 {
			uclog.Errorf(ctx, "js getSecret function called with wrong number of args: %d", len(info.Args()))
			return throwException(ctx, iso, ucerr.Errorf("wrong number of args (got %d, expected 1)", len(info.Args())))
		}

		secretValue := ""
		if secretResolver == nil {
			uclog.Warningf(ctx, "getSecret called in v8go without a valid secretResolver")
		} else {
			var err error
			secretValue, err = secretResolver.ResolveSecret(ctx, info.Args()[0].String())
			if err != nil {
				return throwException(ctx, iso, ucerr.Friendlyf(err, "failed to resolve secret"))
			}
		}

		v, err := v8go.NewValue(iso, secretValue)
		if err != nil {
			return throwException(ctx, iso, err)
		}
		return v
	})
}

func newAuditLogFunction(ctx context.Context, iso *v8go.Isolate, auditLogEventType auditlog.EventType) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		auditlog.PostMultipleAsync(ctx, auditlog.NewEntryArray(
			auth.GetAuditLogActor(ctx),
			auditLogEventType,
			auditlog.Payload{"message": info.Args()[0].String()},
		))
		return nil
	})
}

// RunScript runs the given source code in a new JS context and returns the result as a string
func RunScript(source string, origin string, authzClient *authz.Client, auditLogEventType auditlog.EventType, secretResolver SecretResolver) (string, error) {
	// spin up a V8 Isolate container
	jsContext, _, err := NewJSContext(context.Background(), authzClient, auditLogEventType, secretResolver)
	if err != nil {
		return "", ucerr.Wrap(err)

	}
	defer jsContext.Res.Release()
	defer jsContext.Close()

	val, err := jsContext.RunScript(source, origin)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	return val.String(), nil
}
