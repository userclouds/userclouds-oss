package uchttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucerr"
	"userclouds.com/test/testlogtransport"
)

const testPath = "/somepath"
const testGetQuery = "query=foo"

// testBaseURL is used in many tests because we don't want a trailing
// slash in the URL (as it would have been stripped off in real life
// by a parent handler), and you can't use "" as a URL.
const testBaseURL = "http://contoso.com"

type handlerResponse struct {
	Success bool `json:"success"`
}

func ensureSuccess(t *testing.T, h http.Handler, method, path string, code int) {
	t.Helper()
	ensureSuccessWithHeaders(t, h, method, path, code, map[string]string{})
}

func ensureSuccessWithHeaders(t *testing.T, h http.Handler, method, path string, code int, headers map[string]string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	r := w.Result()
	assert.Equal(t, r.StatusCode, code, assert.Must())
	defer r.Body.Close()
	if code == http.StatusOK {
		var resp handlerResponse
		err := json.NewDecoder(r.Body).Decode(&resp)
		assert.NoErr(t, err)
		assert.True(t, resp.Success)
	}
}

func ensureMethodNotAllowed(t *testing.T, h http.Handler, method, path string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, w.Result().StatusCode, http.StatusMethodNotAllowed)
}

func ensureNotFound(t *testing.T, h http.Handler, method, path string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	assert.Equal(t, w.Result().StatusCode, http.StatusNotFound)
}

func ensureForbidden(t *testing.T, h http.Handler, method, path, reason string) {
	t.Helper()
	ensureForbiddenWithHeaders(t, h, method, path, reason, map[string]string{})
}

func ensureForbiddenWithHeaders(t *testing.T, h http.Handler, method, path, reason string, headers map[string]string) {
	t.Helper()
	tt := testlogtransport.InitLoggerAndTransportsForTests(t)
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	r := w.Result()
	assert.Equal(t, r.StatusCode, http.StatusForbidden, assert.Must())
	tt.AssertLogsContainString(reason)
}

// MethodHandler tests

func serveHTTPGet(t *testing.T, w http.ResponseWriter, r *http.Request, expectedPath, expectedQuery string) {
	// Make sure handler doesn't mess up basics on the way in or out.
	assert.Equal(t, r.Method, http.MethodGet)
	assert.Equal(t, r.URL.Path, expectedPath)
	assert.Equal(t, r.URL.RawQuery, expectedQuery)

	writeSuccess(w)
}

func serveHTTPPost(t *testing.T, w http.ResponseWriter, r *http.Request) {
	assert.Equal(t, r.Method, http.MethodPost)
	// Allow a trailing slash or not, but nothing else.
	// If an ID or sub-handler was specified in the path, we should have been
	// routed elsewhere.
	assert.Equal(t, strings.TrimPrefix(r.URL.Path, "/"), "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func serveHTTPPut(t *testing.T, w http.ResponseWriter, r *http.Request) {
	assert.Equal(t, r.Method, http.MethodPut)
	w.WriteHeader(http.StatusNoContent)
}

func serveHTTPDelete(t *testing.T, w http.ResponseWriter, r *http.Request) {
	assert.Equal(t, r.Method, http.MethodDelete)
	w.WriteHeader(http.StatusNoContent)
}

func writeSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(handlerResponse{Success: true})
}

func TestGet(t *testing.T) {
	h := &methodHandler{
		Get: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPGet(t, w, r, testPath, testGetQuery)
		},
	}
	url := fmt.Sprintf("%s?%s", testPath, testGetQuery)
	ensureSuccess(t, h, http.MethodGet, url, http.StatusOK)
}

func TestMultiMethod(t *testing.T) {
	h := &methodHandler{
		Post: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPPost(t, w, r)
		},
		Put: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPPut(t, w, r)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPDelete(t, w, r)
		},
	}

	ensureSuccess(t, h, http.MethodPost, testBaseURL, http.StatusCreated)
	ensureSuccess(t, h, http.MethodPut, testBaseURL, http.StatusNoContent)
	ensureSuccess(t, h, http.MethodDelete, testBaseURL, http.StatusNoContent)
	ensureMethodNotAllowed(t, h, http.MethodGet, testBaseURL)
}

func TestTrailingSlash(t *testing.T) {
	h := &methodHandler{
		Get: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPGet(t, w, r, testPath, "")
		},
	}

	ensureNotFound(t, h, http.MethodGet, fmt.Sprintf("%s/", testPath))
}

// CollectionHandler tests

func serveHTTPGetAll(t *testing.T, w http.ResponseWriter, r *http.Request) {
	assert.Equal(t, r.Method, http.MethodGet)

	writeSuccess(w)
}

func serveHTTPDeleteAll(t *testing.T, w http.ResponseWriter, r *http.Request) {
	assert.Equal(t, r.Method, http.MethodDelete)

	w.WriteHeader(http.StatusNoContent)
}

func serveHTTPGetOne(t *testing.T, w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	assert.Equal(t, r.Method, http.MethodGet)
	assert.Equal(t, mustGetUUID(r.Context()), id)

	writeSuccess(w)
}

func serveHTTPPutCollection(t *testing.T, w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	assert.Equal(t, r.Method, http.MethodPut)
	assert.Equal(t, mustGetUUID(r.Context()), id)
	w.WriteHeader(http.StatusNoContent)
}

func serveHTTPDeleteCollection(t *testing.T, w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	assert.Equal(t, r.Method, http.MethodDelete)
	assert.Equal(t, mustGetUUID(r.Context()), id)
	w.WriteHeader(http.StatusNoContent)
}

func TestGetAll(t *testing.T) {
	h := &CollectionHandler{
		GetAll: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPGetAll(t, w, r)
		},
		Authorizer: NewAllowAllAuthorizer(),
	}

	ensureSuccess(t, h, http.MethodGet, testBaseURL, http.StatusOK)
}

func TestGetOne(t *testing.T) {
	mux := NewServeMux()
	testUUID := uuid.Must(uuid.NewV4())

	// NOTE: it doesn't matter if you include a trailing slash on this
	// path, we set up handlers both ways.
	mux.Handle("/objects/", &CollectionHandler{
		GetOne: func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
			assert.Equal(t, id, testUUID)
			serveHTTPGetOne(t, w, r, id)
		},
		Authorizer: NewAllowAllAuthorizer(),
	})

	// Test with/without trailing slash
	ensureSuccess(t, mux, http.MethodGet, fmt.Sprintf("/objects/%s", testUUID), http.StatusOK)
	ensureNotFound(t, mux, http.MethodGet, fmt.Sprintf("/objects/%s/", testUUID))
}

func TestPost(t *testing.T) {
	h := &CollectionHandler{
		Post: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPPost(t, w, r)
		},
		Authorizer: NewAllowAllAuthorizer(),
	}

	ensureSuccess(t, h, http.MethodPost, testBaseURL, http.StatusCreated)
}

func TestUnimplementedMethodCH(t *testing.T) {
	h := &CollectionHandler{
		Post: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPPost(t, w, r)
		},
		Authorizer: NewAllowAllAuthorizer(),
	}

	ensureMethodNotAllowed(t, h, http.MethodGet, testBaseURL)
	ensureMethodNotAllowed(t, h, http.MethodPut, testBaseURL)
	ensureMethodNotAllowed(t, h, http.MethodDelete, testBaseURL)
}

func TestPutCH(t *testing.T) {
	testUUID := uuid.Must(uuid.NewV4())
	ch := &CollectionHandler{
		Put: func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
			assert.Equal(t, id, testUUID)
			serveHTTPPutCollection(t, w, r, id)
		},
		Authorizer: NewAllowAllAuthorizer(),
	}
	h := http.StripPrefix("/collection", ch)

	ensureSuccess(t, h, http.MethodPut, fmt.Sprintf("/collection/%s", testUUID), http.StatusNoContent)
	// We don't allow Put on the collection without an ID or with a trailing slash.
	ensureNotFound(t, h, http.MethodPut, fmt.Sprintf("/collection/%s/", testUUID))
	ensureNotFound(t, h, http.MethodPut, "/collection/")
}

func TestDeleteCH(t *testing.T) {
	testUUID := uuid.Must(uuid.NewV4())
	ch := &CollectionHandler{
		Delete: func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
			assert.Equal(t, id, testUUID)
			serveHTTPDeleteCollection(t, w, r, id)
		},
		Authorizer: NewAllowAllAuthorizer(),
	}
	h := http.StripPrefix("/collection", ch)

	ensureSuccess(t, h, http.MethodDelete, fmt.Sprintf("/collection/%s", testUUID), http.StatusNoContent)
	// We don't allow Delete on the collection without an ID or with a trailing slash.
	ensureNotFound(t, h, http.MethodDelete, fmt.Sprintf("/collection/%s/", testUUID))
	ensureNotFound(t, h, http.MethodDelete, "/collection/")
}

func TestDeleteAll(t *testing.T) {
	ch := &CollectionHandler{
		DeleteAll: func(w http.ResponseWriter, r *http.Request) {
			serveHTTPDeleteAll(t, w, r)
		},
		Authorizer: NewAllowAllAuthorizer(),
	}
	h := http.StripPrefix("/collection", ch)

	ensureSuccess(t, h, http.MethodDelete, "/collection", http.StatusNoContent)
}

func TestNestedMH(t *testing.T) {
	testUUID := uuid.Must(uuid.NewV4())
	nestedMux := NewServeMux()
	nestedMux.Handle("/foobar", &nestedMethodHandler{
		Get: func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
			assert.Equal(t, id, testUUID)
			serveHTTPGetOne(t, w, r, id)
		},
	})

	h := &CollectionHandler{
		NestedItemHandler: nestedMux,
		Authorizer:        NewAllowAllAuthorizer(),
	}

	// No GET should be registered for the collection returns 405.
	ensureMethodNotAllowed(t, h, http.MethodGet, fmt.Sprintf("/%s", testUUID))

	// Trailing slashes and invalid sub-handlers return 404.
	ensureNotFound(t, h, http.MethodGet, "/")
	ensureNotFound(t, h, http.MethodGet, fmt.Sprintf("/%s/", testUUID))
	ensureNotFound(t, h, http.MethodGet, fmt.Sprintf("/%s/foo", testUUID))
	ensureNotFound(t, h, http.MethodGet, fmt.Sprintf("/%s/foo/", testUUID))

	ensureSuccess(t, h, http.MethodGet, fmt.Sprintf("/%s/foobar", testUUID), http.StatusOK)
	ensureSuccess(t, h, http.MethodGet, fmt.Sprintf("/%s/foobar?%s", testUUID, testGetQuery), http.StatusOK)

	// Trailing slash on valid handler is a 404.
	ensureNotFound(t, h, http.MethodGet, fmt.Sprintf("/%s/foobar/", testUUID))
}

func TestNestedCH(t *testing.T) {
	wrongUUID := uuid.Must(uuid.NewV4())
	parentUUID := uuid.Must(uuid.NewV4())
	nestedUUID := uuid.Must(uuid.NewV4())

	getAllCalls := 0
	deleteAllCalls := 0
	putCalls := 0

	nestedMux := NewServeMux()
	nestedMux.Handle("/foobar", &nestedCollectionHandler{
		GetAll: func(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) {
			assert.Equal(t, parentID, parentUUID)
			getAllCalls++
			writeSuccess(w)
		},
		Put: func(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, nestedID uuid.UUID) {
			assert.Equal(t, parentID, parentUUID)
			assert.Equal(t, nestedID, nestedUUID)
			putCalls++
			writeSuccess(w)
		},
		Delete: func(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, nestedID uuid.UUID) {
			// Unreachable
			assert.True(t, false)
		},
		DeleteAll: func(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) {
			assert.Equal(t, parentID, parentUUID)
			deleteAllCalls++
			serveHTTPDelete(t, w, r)
		},
		Authorizer: &NestedMethodAuthorizer{
			GetAllF: func(r *http.Request, parentID uuid.UUID) error {
				return nil
			},
			PutF: func(r *http.Request, parentID, nestedID uuid.UUID) error {
				if nestedID != nestedUUID {
					return ucerr.New("bad nested id")
				}
				// Assert these are equal because it should be impossible to even get here
				// if the parent authorizer doesn't pass first.
				assert.Equal(t, parentID, parentUUID)
				return nil
			},
			DeleteAllF: func(r *http.Request, parentID uuid.UUID) error {
				return nil
			},
		},
	})

	h := &CollectionHandler{
		NestedItemHandler: nestedMux,
		Authorizer: &MethodAuthorizer{
			NestedF: func(r *http.Request, id uuid.UUID) error {
				if id != parentUUID {
					return ucerr.New("bad parent id")
				}
				return nil
			},
		},
	}

	t.Run("GetAll", func(t *testing.T) {
		ensureForbidden(t, h, http.MethodGet, fmt.Sprintf("/%s/foobar", wrongUUID), "bad parent id")
		assert.Equal(t, getAllCalls, 0)
		ensureSuccess(t, h, http.MethodGet, fmt.Sprintf("/%s/foobar", parentUUID), http.StatusOK)
		assert.Equal(t, getAllCalls, 1)
	})

	t.Run("DeleteAll", func(t *testing.T) {
		ensureForbidden(t, h, http.MethodDelete, fmt.Sprintf("/%s/foobar", wrongUUID), "bad parent id")
		assert.Equal(t, deleteAllCalls, 0)
		ensureSuccess(t, h, http.MethodDelete, fmt.Sprintf("/%s/foobar", parentUUID), http.StatusNoContent)
		assert.Equal(t, deleteAllCalls, 1)
	})

	t.Run("Put", func(t *testing.T) {
		ensureForbidden(t, h, http.MethodPut, fmt.Sprintf("/%s/foobar/%s", wrongUUID, nestedUUID), "bad parent id")
		ensureForbidden(t, h, http.MethodPut, fmt.Sprintf("/%s/foobar/%s", parentUUID, wrongUUID), "bad nested id")
		ensureForbidden(t, h, http.MethodPut, fmt.Sprintf("/%s/foobar/%s", wrongUUID, wrongUUID), "bad parent id")
		assert.Equal(t, putCalls, 0)
		ensureSuccess(t, h, http.MethodPut, fmt.Sprintf("/%s/foobar/%s", parentUUID, nestedUUID), http.StatusOK)
		assert.Equal(t, putCalls, 1)
	})

	t.Run("FailClosed", func(t *testing.T) {
		ensureForbidden(t, h, http.MethodDelete, fmt.Sprintf("/%s/foobar/%s", parentUUID, nestedUUID), "unauthorized")
	})
}

// Route parsing test

func TestParseRoute(t *testing.T) {
	id, prefix, err := parseRoute("")
	assert.NoErr(t, err)
	assert.Equal(t, id, uuid.Nil)
	assert.Equal(t, prefix, "")

	// Trailing slashes are bad; we assume that any nested collection handlers
	// should have rules set up to match the non-trailing slash version exactly,
	// and if there's a slash there should be some kind of ID after.
	_, _, err = parseRoute("/")
	assert.NotNil(t, err)
	_, _, err = parseRoute("/foo/")
	assert.NotNil(t, err)

	// This is bad because foo isn't an ID
	_, _, err = parseRoute("/foo")
	assert.NotNil(t, err)

	// uuid.Nil is not allowed
	_, _, err = parseRoute(fmt.Sprintf("/%s", uuid.Nil))
	assert.NotNil(t, err)

	// Expect a leading slash, but no trailing slash with IDs.
	testUUID := uuid.Must(uuid.NewV4())
	_, _, err = parseRoute(testUUID.String())
	assert.NotNil(t, err)
	_, _, err = parseRoute(fmt.Sprintf("%s/", testUUID))
	assert.NotNil(t, err)
	_, _, err = parseRoute(fmt.Sprintf("/%s/", testUUID))
	assert.NotNil(t, err)
	_, _, err = parseRoute(fmt.Sprintf("/%s/foo/bar/", testUUID))
	assert.NotNil(t, err)

	// These are all textbook good
	id, prefix, err = parseRoute(fmt.Sprintf("/%s", testUUID))
	assert.NoErr(t, err)
	assert.Equal(t, id, testUUID)
	assert.Equal(t, prefix, fmt.Sprintf("/%s", testUUID))
	id, prefix, err = parseRoute(fmt.Sprintf("/%s/foo", testUUID))
	assert.NoErr(t, err)
	assert.Equal(t, id, testUUID)
	assert.Equal(t, prefix, fmt.Sprintf("/%s", testUUID))
	id, prefix, err = parseRoute(fmt.Sprintf("/%s/foo/bar", testUUID))
	assert.NoErr(t, err)
	assert.Equal(t, id, testUUID)
	assert.Equal(t, prefix, fmt.Sprintf("/%s", testUUID))

	// Malformed IDs
	_, _, err = parseRoute(fmt.Sprintf("/1%s", testUUID))
	assert.NotNil(t, err)
	_, _, err = parseRoute(fmt.Sprintf("/-%s", testUUID))
	assert.NotNil(t, err)
	_, _, err = parseRoute(fmt.Sprintf("/%sa", testUUID))
	assert.NotNil(t, err)
	_, _, err = parseRoute(fmt.Sprintf("/%sz", testUUID))
	assert.NotNil(t, err)

	// Turns out this is ok too (no hyphens)
	id, _, err = parseRoute("/12345678123456781234567812345678")
	assert.NoErr(t, err)
	assert.Equal(t, id, uuid.FromStringOrNil("12345678-1234-5678-1234-567812345678"))
	// But bad hyphens are no good
	_, _, err = parseRoute("/12345678-123456781234567812345678")
	assert.NotNil(t, err)
}

func TestAuthorize(t *testing.T) {
	testGetOneUUID := uuid.Must(uuid.NewV4())
	testDeleteUUID := uuid.Must(uuid.NewV4())
	testNestedUUID := uuid.Must(uuid.NewV4())
	wrongUUID := uuid.Must(uuid.NewV4())

	// Counters to ensure methods are actually invoked when intended
	getAllCalls := 0
	getOneCalls := 0
	postCalls := 0
	putCalls := 0
	deleteCalls := 0
	nestedCalls := 0

	authHeader := map[string]string{
		"Authorization": "Bearer 1234",
	}

	mux := NewServeMux()

	// Most other tests are validating the default "allow all" behavior,
	// so this test focuses on non-default behavior.
	mux.Handle("/objects/", &CollectionHandler{
		GetAll: func(w http.ResponseWriter, r *http.Request) {
			getAllCalls++
			serveHTTPGetAll(t, w, r)
		},
		GetOne: func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
			getOneCalls++
			serveHTTPGetOne(t, w, r, id)
		},
		Post: func(w http.ResponseWriter, r *http.Request) {
			postCalls++
			serveHTTPPost(t, w, r)
		},
		Put: func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
			putCalls++
			serveHTTPPutCollection(t, w, r, id)
		},
		Delete: func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
			deleteCalls++
			serveHTTPDeleteCollection(t, w, r, id)
		},
		NestedItemHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nestedCalls++
			writeSuccess(w)
		}),
		Authorizer: &MethodAuthorizer{
			GetAllF: func(r *http.Request) error {
				if r.URL.Query().Get("getallkey") != "getall" {
					return ucerr.New("no getall for you")
				}
				return nil
			},
			GetOneF: func(r *http.Request, id uuid.UUID) error {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer 1234" || id != testGetOneUUID {
					return ucerr.New("no getone for you")
				}
				return nil
			},
			PostF: func(r *http.Request) error {
				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer 1234" {
					return ucerr.New("no post for you")
				}
				return nil

			},
			// PutF intentionally omitted
			DeleteF: func(r *http.Request, id uuid.UUID) error {
				if id != testDeleteUUID {
					return ucerr.New("no delete for you")
				}
				return nil
			},
			NestedF: func(r *http.Request, id uuid.UUID) error {
				if id != testNestedUUID {
					return ucerr.New("no nested items for you")
				}
				return nil
			},
		},
	})

	t.Run("GetAll", func(t *testing.T) {
		ensureForbidden(t, mux, http.MethodGet, "/objects", "no getall for you")
		ensureForbidden(t, mux, http.MethodGet, "/objects?foo=bar", "no getall for you")
		ensureForbidden(t, mux, http.MethodGet, "/objects?getallkey=bar", "no getall for you")
		assert.Equal(t, getAllCalls, 0)
		ensureSuccess(t, mux, http.MethodGet, "/objects?getallkey=getall", http.StatusOK)
		assert.Equal(t, getAllCalls, 1)
	})

	t.Run("GetOne", func(t *testing.T) {
		ensureForbidden(t, mux, http.MethodGet, fmt.Sprintf("/objects/%s", testGetOneUUID), "no getone for you")
		ensureForbiddenWithHeaders(t, mux, http.MethodGet, fmt.Sprintf("/objects/%s", wrongUUID), "no getone for you", authHeader)
		assert.Equal(t, getOneCalls, 0)
		ensureSuccessWithHeaders(t, mux, http.MethodGet, fmt.Sprintf("/objects/%s", testGetOneUUID), http.StatusOK, authHeader)
		assert.Equal(t, getOneCalls, 1)
	})

	t.Run("Post", func(t *testing.T) {
		ensureForbidden(t, mux, http.MethodPost, "/objects", "no post for you")
		assert.Equal(t, postCalls, 0)
		ensureSuccessWithHeaders(t, mux, http.MethodPost, "/objects", http.StatusCreated, authHeader)
		assert.Equal(t, postCalls, 1)
	})

	t.Run("Put", func(t *testing.T) {
		// No Put handler specified, UUID needs to be parseable but isn't used.
		ensureForbidden(t, mux, http.MethodPut, fmt.Sprintf("/objects/%s", wrongUUID), "unauthorized")
		assert.Equal(t, putCalls, 0)
	})

	t.Run("Delete", func(t *testing.T) {
		ensureForbidden(t, mux, http.MethodDelete, fmt.Sprintf("/objects/%s", wrongUUID), "no delete for you")
		assert.Equal(t, deleteCalls, 0)
		ensureSuccess(t, mux, http.MethodDelete, fmt.Sprintf("/objects/%s", testDeleteUUID), http.StatusNoContent)
		assert.Equal(t, deleteCalls, 1)
	})

	t.Run("Nested", func(t *testing.T) {
		ensureForbidden(t, mux, http.MethodGet, fmt.Sprintf("/objects/%s/foo", wrongUUID), "no nested items for you")
		assert.Equal(t, nestedCalls, 0)
		ensureSuccess(t, mux, http.MethodGet, fmt.Sprintf("/objects/%s/foo", testNestedUUID), http.StatusOK)
		assert.Equal(t, nestedCalls, 1)
	})
}
