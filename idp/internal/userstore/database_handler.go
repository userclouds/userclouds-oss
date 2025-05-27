package userstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/sqlshim"
)

func saveDatabasePasswordSecret(ctx context.Context, password string, tenantID, dbID uuid.UUID) (*secret.String, error) {
	passwordSecret, err := secret.NewString(ctx, "dbproxy", fmt.Sprintf("db_password_%s_%s_%s", tenantID, dbID, crypto.MustRandomHex(6)), password)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return passwordSecret, nil
}

func (h *handler) listDatabases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	pager, err := storage.NewSQLShimDatabasePaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	databases, respFields, err := s.ListSQLShimDatabasesPaginated(ctx, *pager)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := make([]userstore.SQLShimDatabase, 0, len(databases))
	for _, db := range databases {
		resp = append(resp, db.ToClientModel())
	}

	jsonapi.Marshal(w, &idp.ListDatabasesResponse{Data: resp, ResponseFields: *respFields})
}

func (h *handler) getDatabase(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	database, err := s.GetSQLShimDatabase(ctx, id)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, database.ToClientModel())
}

func (h *handler) createDatabase(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	var req idp.CreateDatabaseRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := req.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	db := &storage.SQLShimDatabase{
		BaseModel: ucdb.NewBase(),
		Name:      req.Database.Name,
		Type:      sqlshim.DatabaseType(req.Database.Type),
		Host:      req.Database.Host,
		Port:      req.Database.Port,
		Username:  req.Database.Username,
		Schemas:   pq.StringArray{},
	}
	if req.Database.ID != uuid.Nil {
		db.ID = req.Database.ID
	}
	if req.Database.Schemas != nil {
		db.Schemas = req.Database.Schemas
	}

	if req.Database.Password != "" {
		ts := multitenant.MustGetTenantState(ctx)
		password, err := saveDatabasePasswordSecret(ctx, req.Database.Password, ts.ID, db.ID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		db.Password = *password
	}

	if err := s.SaveSQLShimDatabase(ctx, db); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, db.ToClientModel(), jsonapi.Code(http.StatusCreated))
}

func (h *handler) updateDatabase(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	var req idp.UpdateDatabaseRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := req.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	db, err := s.GetSQLShimDatabase(ctx, id)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	db.Name = req.Database.Name
	db.Type = sqlshim.DatabaseType(req.Database.Type)
	db.Host = req.Database.Host
	db.Port = req.Database.Port
	db.Username = req.Database.Username
	if req.Database.Schemas != nil {
		db.Schemas = req.Database.Schemas
	}

	oldPassword, err := db.Password.Resolve(ctx)
	if err != nil {
		uclog.Warningf(ctx, "failed to resolve password secret for database %s: %v", db.ID, err)
		oldPassword = ""
	}
	if req.Database.Password != "" && req.Database.Password != oldPassword {
		if err := db.Password.Delete(ctx); err != nil {
			uclog.Warningf(ctx, "failed to delete password secret for database %s: %v", db.ID, err)
		}
		ts := multitenant.MustGetTenantState(ctx)
		password, err := saveDatabasePasswordSecret(ctx, req.Database.Password, ts.ID, db.ID)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		db.Password = *password
	}

	if err := s.SaveSQLShimDatabase(ctx, db); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, db.ToClientModel())
}

func (h *handler) deleteDatabase(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	if err := s.DeleteSQLShimDatabase(ctx, id); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(http.StatusNoContent))
}

func (h *handler) testDatabaseConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req idp.TestDatabaseRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	password := req.Database.Password

	// If the password is empty, try to get it from the previously stored database (if it exists)
	if password == "" && req.Database.ID != uuid.Nil {
		s := storage.MustCreateStorage(ctx)

		if db, err := s.GetSQLShimDatabase(ctx, req.Database.ID); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
		} else {
			p, err := db.Password.Resolve(ctx)
			if err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
			password = p
		}
	}

	if req.Database.Type == string(sqlshim.DatabaseTypePostgres) {
		db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable", req.Database.Host, req.Database.Port, req.Database.Username, password))
		if err != nil {
			jsonapi.Marshal(w, idp.TestDatabaseResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
		defer func() {
			if err := db.Close(); err != nil {
				uclog.Warningf(ctx, "failed to close database connection: %v", err)
			}
		}()
	} else if req.Database.Type == string(sqlshim.DatabaseTypeMySQL) {
		db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@(%s:%d)/", req.Database.Username, password, req.Database.Host, req.Database.Port))
		if err != nil {
			jsonapi.Marshal(w, idp.TestDatabaseResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}
		defer func() {
			if err := db.Close(); err != nil {
				uclog.Warningf(ctx, "failed to close database connection: %v", err)
			}
		}()
	} else {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("unsupported database type: %s", req.Database.Type))
		return
	}

	jsonapi.Marshal(w, idp.TestDatabaseResponse{
		Success: true,
	})
}
