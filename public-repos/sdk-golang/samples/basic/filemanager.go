package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// FileManager is a simple class that can create file systems with permissions.
type FileManager struct {
	authZClient *authz.Client
}

// NewFileManager returns a new FileManager
func NewFileManager(authZClient *authz.Client) *FileManager {
	return &FileManager{
		authZClient: authZClient,
	}
}

// File represents a file/directory that is part of a tree. Only directories can have child files.
// The ID of the file is used as a key in the permission system to manage ACLs.
type File struct {
	id   uuid.UUID
	name string

	isDir bool

	parent   *File
	children []*File
}

// FullPath returns the full path of the File
func (f File) FullPath() string {
	if f.parent == nil {
		return fmt.Sprintf("file://%s", f.id.String())
	}
	return fmt.Sprintf("%s/%s", f.parent.FullPath(), f.name)
}

// FindFile finds a child file by name
func (f File) FindFile(name string) *File {
	if !f.isDir {
		return nil
	}

	for _, v := range f.children {
		if v.name == name {
			return v
		}
	}
	return nil
}

// HasWriteAccess returns true if the given user has write access to the file
func (fm *FileManager) HasWriteAccess(ctx context.Context, f *File, userID uuid.UUID) error {
	resp, err := fm.authZClient.CheckAttribute(ctx, userID, f.id, "write")
	if err != nil {
		return ucerr.Wrap(err)
	}
	if resp.HasAttribute {
		return nil
	}
	return ucerr.Errorf("user %v does not have write permissions on file %s (id: %s)", userID, f.FullPath(), f.id)
}

// HasReadAccess returns true if the given user has read access to the file
func (fm *FileManager) HasReadAccess(ctx context.Context, f *File, userID uuid.UUID) error {
	resp, err := fm.authZClient.CheckAttribute(ctx, userID, f.id, "read")
	if err != nil {
		return ucerr.Wrap(err)
	}
	if resp.HasAttribute {
		return nil
	}
	return ucerr.Errorf("user %v does not have read permissions on file %s (id: %s)", userID, f.FullPath(), f.id)
}

// NewRoot creates a new root directory
func (fm *FileManager) NewRoot(ctx context.Context, creatorUserID uuid.UUID) (*File, error) {
	f := &File{
		id:       uuid.Must(uuid.NewV4()),
		name:     "",
		isDir:    true,
		parent:   nil,
		children: []*File{},
	}

	// If the first operation fails, nothing is created and the operation fails.
	// If the first succeeds but the second fails, we'll have an orphan authz object to clean-up that is harmless and could be reaped later.
	// NB: We will eventually support Transactions for this, which avoids orphans.
	if _, err := fm.authZClient.CreateObject(ctx, f.id, fileTypeID, f.FullPath()); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Give the creator of the file Editor permission by default (you could get fancy and have "owner"/"creator"/"admin" access on top too)
	if _, err := fm.authZClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), creatorUserID, f.id, fileEditorTypeID); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return f, nil
}

func (fm *FileManager) newFileHelper(ctx context.Context, name string, isDir bool, parent *File, creatorUserID uuid.UUID) (*File, error) {
	if !parent.isDir {
		return nil, ucerr.Errorf("cannot create files or directories under a file: %+v", parent)
	}

	if parent.FindFile(name) != nil {
		return nil, ucerr.Errorf("file with name '%s' already exists in %+v", name, parent)
	}

	if err := fm.HasWriteAccess(ctx, parent, creatorUserID); err != nil {
		return nil, ucerr.Wrap(err)
	}

	f := &File{
		id:       uuid.Must(uuid.NewV4()),
		name:     name,
		isDir:    isDir,
		parent:   parent,
		children: []*File{},
	}

	if _, err := fm.authZClient.CreateObject(ctx, f.id, fileTypeID, f.FullPath()); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Make the parent file a 'container' of the new file
	if _, err := fm.authZClient.CreateEdge(ctx, uuid.Must(uuid.NewV4()), parent.id, f.id, fileContainerTypeID); err != nil {
		return nil, ucerr.Wrap(err)
	}

	parent.children = append(parent.children, f)

	// NB: since the creator has write access to the parent, we don't need to explicitly grant it on the child

	return f, nil
}

// NewFile creates a new file under the given parent
func (fm *FileManager) NewFile(ctx context.Context, name string, parent *File, creatorUserID uuid.UUID) (*File, error) {
	f, err := fm.newFileHelper(ctx, name, false, parent, creatorUserID)
	return f, ucerr.Wrap(err)
}

// NewDir creates a new directory under the given parent
func (fm *FileManager) NewDir(ctx context.Context, name string, parent *File, creatorUserID uuid.UUID) (*File, error) {
	f, err := fm.newFileHelper(ctx, name, true, parent, creatorUserID)
	return f, ucerr.Wrap(err)
}

// ReadFile returns the contents of the file
func (fm *FileManager) ReadFile(ctx context.Context, f *File, readerUserID uuid.UUID) (string, error) {
	if err := fm.HasReadAccess(ctx, f, readerUserID); err != nil {
		return "", ucerr.Wrap(err)
	}

	return fmt.Sprintf("contents of file %s", f.FullPath()), nil
}

// mustFile panics if a file-producing operation returns an error, otherwise it returns the file
func mustFile(f *File, err error) *File {
	if err != nil {
		log.Fatalf("mustFile error: %v", err)
	}
	if f == nil {
		log.Fatal("mustFile error: unexpected nil file")
	}
	return f
}

const leftColWidth = 50

func summarizePermissions(ctx context.Context, idpClient *idp.Client, authZClient *authz.Client, f *File) string {
	permsList := ""

	cursor := pagination.CursorBegin
	for {
		resp, err := authZClient.ListEdgesOnObject(ctx, f.id, authz.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return "<error fetching edges>"
		}
		for _, e := range resp.Data {
			et, err := authZClient.GetEdgeType(ctx, e.EdgeTypeID)
			if err != nil {
				return "<error fetching edge type>"
			}
			// Only look at editor/viewer relationships
			if !strings.Contains(et.TypeName, "viewer") && !strings.Contains(et.TypeName, "editor") {
				continue
			}
			var otherID uuid.UUID
			if e.SourceObjectID == f.id {
				otherID = e.TargetObjectID
			} else {
				otherID = e.SourceObjectID
			}
			obj, err := authZClient.GetObject(ctx, otherID)
			if err != nil {
				return "<error fetching object>"
			}
			displayName := obj.Alias
			if obj.TypeID == authz.UserObjectTypeID {
				user, err := idpClient.GetUser(ctx, obj.ID)
				if err != nil {
					return "<error fetching user>"
				}
				name := ""
				if user.Profile["name"] != nil {
					name = user.Profile["name"].(string)
				}
				displayName = &name
			}
			perm := fmt.Sprintf("%s (%s)", *displayName, et.TypeName)
			if len(permsList) == 0 {
				permsList = perm
			} else {
				permsList = fmt.Sprintf("%s, %s", permsList, perm)
			}
		}
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}
	return permsList
}

func renderFileTree(ctx context.Context, idpClient *idp.Client, authZClient *authz.Client, f *File, indentLevel int) {
	outStr := ""
	if f.parent == nil {
		outStr += "/"
	} else {
		for i := 0; i < indentLevel-1; i++ {
			outStr += "      "
		}
		outStr += "^---> "
		outStr += f.name
	}

	for i := len(outStr); i < leftColWidth; i++ {
		outStr += " "
	}
	outStr += fmt.Sprintf("| %s", summarizePermissions(ctx, idpClient, authZClient, f))
	fmt.Println(outStr)

	for _, v := range f.children {
		renderFileTree(ctx, idpClient, authZClient, v, indentLevel+1)
	}
}
